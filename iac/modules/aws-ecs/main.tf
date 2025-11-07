
data "aws_vpc" "default" {
}

data "aws_subnets" "default" {
}

resource "aws_ecs_cluster" "main" {
  name = format("tf-%s", data.github_repository.main.name)
}

resource "aws_security_group" "ecs_service_sg" {
  name_prefix = "tf-ecs"
  description = format("ECS Service for %s Security group", data.github_repository.main.name)
  vpc_id      = data.aws_vpc.default.id
}


resource "aws_vpc_security_group_ingress_rule" "allow_ingress_to_the_ecs_containers_from_alb" {
  security_group_id            = aws_security_group.ecs_service_sg.id
  referenced_security_group_id = aws_security_group.load_balancer_sg.id

  from_port   = 80
  ip_protocol = "tcp"
  to_port     = 80
}

resource "aws_vpc_security_group_egress_rule" "allow_ecs_service_to_egress_to_everything" {
  security_group_id = aws_security_group.ecs_service_sg.id
  # TODO: Is there a way to limit this to docker hub/ecr and the alb?
  cidr_ipv4 = "0.0.0.0/0"

  # semantically equivalent to all ports
  ip_protocol = "-1"
}


resource "aws_ecs_service" "main" {
  name            = format("tf-ecs-task-%s", data.github_repository.main.name)
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.main.arn
  desired_count   = 1

  launch_type = "FARGATE"

  network_configuration {
    subnets         = [data.aws_subnets.default.ids[0], data.aws_subnets.default.ids[1]]
    security_groups = [aws_security_group.ecs_service_sg.id]

    # TODO: Figure out how to get ECR images without a public IP at a later date
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.main.arn
    container_name   = "server"
    container_port   = 80
  }
}


data "aws_iam_policy_document" "task_execution_role_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]

    # TODO: fix confused deputy problem at a later date
    # https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html
    # condition {
    #   test     = "ArnLike"
    #   variable = "aws:SourceArn"
    #   values = ["arn:aws:ecs:us-east-1:111122223333:*"]
    # }
    # condition {
    #   test     = "StringEquals"
    #   variable = "aws:SourceAccount"
    #   values = ["arn:aws:ecs:us-east-1:111122223333:*"]
    # }
  }
}

resource "aws_iam_role_policy_attachment" "ecs_task" {
  role       = aws_iam_role.task_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"

  depends_on = [aws_iam_role.task_role]
}

resource "aws_iam_role" "task_role" {
  name               = format("tf-ecs-%s", data.github_repository.main.name)
  assume_role_policy = data.aws_iam_policy_document.task_execution_role_policy.json
}

resource "aws_cloudwatch_log_group" "ecs_task" {
  name              = "/ecs/${aws_ecs_cluster.main.name}/${data.github_repository.main.name}"
  retention_in_days = 5
}

resource "aws_ecs_task_definition" "main" {
  family = data.github_repository.main.name

  # TODO: separate roles
  execution_role_arn = aws_iam_role.task_role.arn
  task_role_arn      = aws_iam_role.task_role.arn

  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode([
    {
      name        = "server"
      image       = "dustinalandzes384/deployment-project:latest"
      environment = var.ecs_task_environment_variables
      essential   = true
      portMappings = [
        # TOD: We might need to change this when we use our own code instead of nginx
        {
          containerPort = 80
          hostPort      = 80
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.ecs_task.name
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = data.github_repository.main.name
        }
      }
    }
  ])
}

resource "aws_lb_target_group" "main" {
  name_prefix = "tf-ecs"
  port        = 80
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.default.id
  target_type = "ip"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "load_balancer_sg" {
  name        = "allow_tls"
  description = "Allow HTTP inbound traffic and all outbound traffic"
  vpc_id      = data.aws_vpc.default.id

  tags = {
    Name = "allow_http"
  }
}

resource "aws_vpc_security_group_ingress_rule" "allow_http_ipv4" {
  security_group_id = aws_security_group.load_balancer_sg.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 80
  ip_protocol       = "tcp"
  to_port           = 80
}

resource "aws_vpc_security_group_egress_rule" "allow_egress_all_traffic_ipv4" {
  security_group_id = aws_security_group.load_balancer_sg.id
  cidr_ipv4         = "0.0.0.0/0"

  # semantically equivalent to all ports
  ip_protocol = "-1"
}

resource "aws_lb" "main" {
  name               = format("tf-alb-%s", data.github_repository.main.name)
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.load_balancer_sg.id]
  subnets            = [data.aws_subnets.default.ids[0], data.aws_subnets.default.ids[1]]

  enable_deletion_protection = false

  tags = {
    Environment = "development"
  }
}

resource "aws_lb_listener" "main" {
  load_balancer_arn = aws_lb.main.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }
}

resource "aws_lb_listener_rule" "main" {
  listener_arn = aws_lb_listener.main.arn
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }

  condition {
    path_pattern {
      values = ["/*"]
    }
  }
}

data "github_repository" "main" {
  full_name = var.github_repo_name
}

resource "aws_iam_user" "github_actions_update_ecs" {
  name = "github_actions_update_ecs"
  path = "/system/"
}

# This is the policy the github iam user will have to update ecs
data "aws_iam_policy_document" "github_actions_update_ecs" {
  statement {
    effect = "Allow"
    actions = [
      "ecs:RegisterTaskDefinition",
      "ecs:DescribeServices",
      "ecs:UpdateService",
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "iam:PassRole"
    ]
    resources = [
      aws_ecs_service.main.id
    ]
  }
}

resource "aws_iam_user_policy" "github_actions_update_ecs" {
  name   = "github_actions_update_ecs"
  user   = aws_iam_user.github_actions_update_ecs.name
  policy = data.aws_iam_policy_document.github_actions_update_ecs.json
}

resource "aws_iam_access_key" "github_actions_update_ecs" {
  user = aws_iam_user.github_actions_update_ecs.name
}

resource "github_actions_secret" "aws_secret_access_key" {
  repository      = data.github_repository.main.name
  secret_name     = "AWS_SECRET_ACCESS_KEY"
  plaintext_value = aws_iam_access_key.github_actions_update_ecs.secret
}

resource "github_actions_secret" "aws_access_key_id" {
  repository      = data.github_repository.main.name
  secret_name     = "AWS_ACCESS_KEY_ID"
  plaintext_value = aws_iam_access_key.github_actions_update_ecs.id
}

resource "github_actions_variable" "aws_region" {
  repository    = data.github_repository.main.name
  variable_name = "AWS_REGION"
  value         = "us-east-1"
}

resource "github_actions_variable" "ecs_service_name" {
  repository    = data.github_repository.main.name
  variable_name = "ECS_SERVICE_NAME"
  value         = aws_ecs_service.main.name
}

resource "github_actions_variable" "ecs_cluster_name" {
  repository    = data.github_repository.main.name
  variable_name = "ECS_CLUSTER_NAME"
  value         = aws_ecs_cluster.main.name
}
