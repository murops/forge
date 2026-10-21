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

  mock_resource "aws_ecr_repository" {
    defaults = {
      arn            = "arn:aws:ecr:us-east-1:123456789012:repository/mock"
      repository_url = "123456789012.dkr.ecr.us-east-1.amazonaws.com/mock"
    }
  }

  mock_resource "aws_lambda_function" {
    defaults = {
      arn = "arn:aws:lambda:us-east-1:123456789012:function:mock"
    }
  }
}

variables {
  name                     = "orders-lambda"
  github_repository        = "murops-forge/orders-lambda"
  github_oidc_provider_arn = "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
  placeholder_image_uri    = "123456789012.dkr.ecr.us-east-1.amazonaws.com/forge-placeholder:latest"
}

run "ecr" {
  command = apply

  assert {
    condition     = aws_ecr_repository.this.image_scanning_configuration[0].scan_on_push
    error_message = "images must be scanned on push"
  }

  assert {
    condition     = aws_ecr_repository.this.image_tag_mutability == "IMMUTABLE"
    error_message = "tags must be immutable"
  }
}

run "lambda" {
  command = apply

  assert {
    condition     = aws_lambda_function.this.package_type == "Image" && aws_lambda_function.this.image_uri == var.placeholder_image_uri
    error_message = "function must start from the placeholder image"
  }

  assert {
    condition     = aws_lambda_function.this.logging_config[0].log_group == "/aws/lambda/orders-lambda"
    error_message = "function must log to its own log group"
  }
}

run "deploy_role_trusts_only_the_repository" {
  command = apply

  variables {
    deploy_branches = ["main", "homol"]
  }

  assert {
    condition = jsondecode(aws_iam_role.deploy.assume_role_policy).Statement[0].Condition.StringEquals["token.actions.githubusercontent.com:sub"] == [
      "repo:murops-forge/orders-lambda:ref:refs/heads/main",
      "repo:murops-forge/orders-lambda:ref:refs/heads/homol",
    ]
    error_message = "deploy role must only trust the repository branches"
  }

  assert {
    condition     = jsondecode(aws_iam_role_policy.deploy.policy).Statement[1].Resource == aws_ecr_repository.this.arn
    error_message = "push permissions must be scoped to the repository"
  }
}

run "invalid_repository" {
  command = plan

  variables {
    github_repository = "orders-lambda"
  }

  expect_failures = [var.github_repository]
}
