variable "name" {
  type = string
}

variable "github_repository" {
  description = "owner/name of the repository allowed to deploy."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$", var.github_repository))
    error_message = "github_repository must be in the owner/name format."
  }
}

variable "deploy_branches" {
  type    = list(string)
  default = ["main"]
}

variable "github_oidc_provider_arn" {
  type = string
}

variable "placeholder_image_uri" {
  description = "Image used until the first deploy from CI."
  type        = string
}

variable "memory_size" {
  type    = number
  default = 512
}

variable "timeout" {
  type    = number
  default = 30
}

variable "log_retention_days" {
  type    = number
  default = 14
}

variable "tags" {
  type    = map(string)
  default = {}
}
