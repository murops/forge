module "lambda" {
  source = "../../modules/aws-lambda"

  name                     = var.name
  github_repository        = "${var.github_owner}/${var.name}"
  github_oidc_provider_arn = var.github_oidc_provider_arn
  placeholder_image_uri    = var.placeholder_image_uri
}
