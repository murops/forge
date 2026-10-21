module "repository" {
  source = "../../modules/github-repository"

  name              = var.name
  description       = var.description
  visibility        = var.visibility
  template          = var.template
  topics            = compact([var.template_topic])
  required_checks   = var.required_checks
  enable_rulesets   = var.visibility == "public" || var.private_rulesets_supported
  actions_variables = var.actions_variables
}
