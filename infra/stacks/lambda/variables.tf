variable "name" {
  type = string
}

variable "github_owner" {
  type = string
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "github_oidc_provider_arn" {
  type = string
}

variable "placeholder_image_uri" {
  type = string
}
