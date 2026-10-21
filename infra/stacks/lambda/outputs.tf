# exposed to the repository workflows as Actions variables
output "actions_variables" {
  value = {
    AWS_REGION      = var.aws_region
    AWS_ROLE_ARN    = module.lambda.deploy_role_arn
    ECR_REPOSITORY  = module.lambda.ecr_repository_url
    LAMBDA_FUNCTION = module.lambda.function_name
  }
}
