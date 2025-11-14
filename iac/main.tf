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

# Subdomain and certificate for HTTPS

data "aws_route53_zone" "public_zone" {
  name = "mcp.alandzes.com"
}

resource "aws_acm_certificate" "mcp" {
  domain_name       = "mcp.alandzes.com"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "validate_certificate" {
  allow_overwrite = true
  name            = tolist(aws_acm_certificate.mcp.domain_validation_options)[0].resource_record_name
  records         = [tolist(aws_acm_certificate.mcp.domain_validation_options)[0].resource_record_value]
  type            = tolist(aws_acm_certificate.mcp.domain_validation_options)[0].resource_record_type
  zone_id         = data.aws_route53_zone.public_zone.zone_id
  ttl             = 60
}

resource "aws_acm_certificate_validation" "validate_certificate" {
  certificate_arn         = aws_acm_certificate.mcp.arn
  validation_record_fqdns = [aws_route53_record.validate_certificate.fqdn]
}

module "ecs_cluster" {
  source                         = "./modules/aws-ecs"
  github_repo_name               = data.github_repository.main.full_name
  ecs_task_environment_variables = []
  domain                         = data.aws_route53_zone.public_zone.name
  acm_certificate_domain         = aws_acm_certificate.mcp.domain_name
}