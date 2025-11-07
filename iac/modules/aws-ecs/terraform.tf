terraform {
  required_providers {
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