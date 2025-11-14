variable "github_repo_name" {
  type        = string
  description = "Name of the github repository to put the secrets in."
  nullable    = false
}

variable "ecs_task_environment_variables" {
  type        = list(object({ name = string, value = string }))
  default     = []
  nullable    = false
  description = "List of objects with name and value keys that define environment variables for the ecs task"
}

variable "domain" {
  type        = string
  nullable    = false
  description = "Route53 Domain"
}

variable "acm_certificate_domain" {
  type        = string
  nullable    = false
  description = "Domain with an ACM certificate created"
}