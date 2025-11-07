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

provider "github" {}

data "github_repository" "main" {
  full_name = "EmmanuelDamienDustinDeploymentProject/DeploymentProject"
}

module "ecs_cluster" {
  source                 = "../aws-ecs"
  github_repo_name       = data.github_repository.main.full_name
  ecs_task_environment_variables = []
}