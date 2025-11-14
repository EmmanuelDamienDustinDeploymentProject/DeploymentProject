terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    github = {
      source  = "integrations/github"
      version = "6.4.0"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = "us-east-1"
}

provider "github" {
  owner = "EmmanuelDamienDustinDeploymentProject"
}

data "github_repository" "main" {
  full_name = "EmmanuelDamienDustinDeploymentProject/DeploymentProject"
}

data "aws_vpc" "default" {}

resource "aws_route53_zone" "private_zone" {
  name = "mcp.alandzes.com"
  vpc {
    vpc_id = data.aws_vpc.default.id
    vpc_region = "us-east-1"
  }
}

resource "aws_acm_certificate" "mcp" {
  domain_name       = "mcp.alandzes.com"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}


module "ecs_cluster" {
  source                 = "./modules/aws-ecs"
  github_repo_name       = data.github_repository.main.full_name
  ecs_task_environment_variables = []
  domain                 = aws_route53_zone.private_zone.name
  acm_certificate_domain = aws_acm_certificate.mcp.domain_name
}