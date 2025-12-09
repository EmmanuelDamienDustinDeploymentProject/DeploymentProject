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

variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS Region"
}

variable "domain_name" {
  type        = string
  default     = "mcp.alandzes.com"
  description = "Domain name for the MCP server"
}
