mock_provider "github" {}

variables {
  name                       = "payments-api"
  github_owner               = "murops-forge"
  github_app_id              = "1"
  github_app_installation_id = "1"
  github_app_private_key     = "test"
  template = {
    owner      = "murops-forge"
    repository = "template-api-go"
  }
  template_topic = "api-go"
}

run "public_repository" {
  command = plan

  assert {
    condition     = module.repository.protected_branches == tolist(["desen", "homol", "main"])
    error_message = "public repositories must be protected"
  }
}

run "private_repository_on_free_plan" {
  command = plan

  variables {
    visibility = "private"
  }

  assert {
    condition     = length(module.repository.protected_branches) == 0
    error_message = "free plan does not support rulesets on private repositories"
  }
}

run "private_repository_on_paid_plan" {
  command = plan

  variables {
    visibility                 = "private"
    private_rulesets_supported = true
  }

  assert {
    condition     = length(module.repository.protected_branches) == 3
    error_message = "rulesets expected when the plan supports them"
  }
}
