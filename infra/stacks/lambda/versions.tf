terraform {
  required_version = ">= 1.9"

  # separate schema so it never collides with the repository stack workspaces
  backend "pg" {
    schema_name = "terraform_lambda_state"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.69"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      managed-by = "forge"
      repository = "${var.github_owner}/${var.name}"
    }
  }
}
