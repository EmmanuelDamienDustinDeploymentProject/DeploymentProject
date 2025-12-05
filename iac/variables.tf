variable "github_client_id" {
  type        = string
  nullable    = false
  description = "GitHub OAuth App Client ID"
  sensitive   = true
}

variable "github_client_secret" {
  type        = string
  nullable    = false
  description = "GitHub OAuth App Client Secret"
  sensitive   = true
}