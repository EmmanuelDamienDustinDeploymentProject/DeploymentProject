terraform {
  required_version = ">= 1.12"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    github = {
      source  = "integrations/github"
      version = "6.4.0"
    }
    mongodbatlas = {
      source  = "mongodb/mongodbatlas"
      version = "1.32.0"
    }
  }
}