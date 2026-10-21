mock_provider "aws" {
  mock_data "aws_partition" {
    defaults = {
      partition = "aws"
    }
  }

  mock_resource "aws_iam_role" {
    defaults = {
      arn = "arn:aws:iam::123456789012:role/mock"
    }
  }
}

variables {
  name                     = "orders"
  github_owner             = "murops-forge"
  aws_region               = "sa-east-1"
  github_oidc_provider_arn = "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
  placeholder_image_uri    = "123456789012.dkr.ecr.sa-east-1.amazonaws.com/forge-placeholder:latest"
}

run "actions_variables" {
  command = apply

  assert {
    condition     = output.actions_variables.LAMBDA_FUNCTION == "orders" && output.actions_variables.AWS_REGION == "sa-east-1"
    error_message = "outputs must describe the function for the repository workflows"
  }

  assert {
    condition     = output.actions_variables.AWS_ROLE_ARN == "arn:aws:iam::123456789012:role/mock"
    error_message = "deploy role must be exported"
  }
}
