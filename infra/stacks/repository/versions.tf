terraform {
  required_version = ">= 1.9"

  # connection string comes from PG_CONN_STR, one workspace per repository
  backend "pg" {}

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.13"
    }
  }
}

provider "github" {
  owner = var.github_owner

  app_auth {
    id              = var.github_app_id
    installation_id = var.github_app_installation_id
    pem_file        = var.github_app_private_key
  }
}
