mock_provider "github" {}

variables {
  name       = "payments-api"
  visibility = "public"
  template = {
    owner      = "murops-forge"
    repository = "template-api-go"
  }
}

run "defaults" {
  command = plan

  assert {
    condition     = contains(github_repository.this.topics, "forge-managed")
    error_message = "repository must be tagged forge-managed"
  }

  assert {
    condition     = !github_repository.this.auto_init && github_repository.this.template[0].repository == "template-api-go"
    error_message = "repository must be generated from the template"
  }

  assert {
    condition     = toset(keys(github_branch.env)) == toset(["homol", "desen"])
    error_message = "homol and desen branches must be created"
  }

  assert {
    condition     = toset(keys(github_repository_ruleset.branch)) == toset(["main", "homol", "desen"])
    error_message = "every long-lived branch needs a ruleset"
  }

  assert {
    condition     = github_repository_ruleset.branch["main"].rules[0].pull_request[0].required_approving_review_count == 1
    error_message = "main must require one approval"
  }

  assert {
    condition     = length(github_repository_ruleset.branch["desen"].rules[0].pull_request) == 0
    error_message = "desen must accept direct pushes"
  }
}

run "without_template" {
  command = plan

  variables {
    template = null
  }

  assert {
    condition     = github_repository.this.auto_init
    error_message = "repository without template must be initialized"
  }
}

run "required_checks" {
  command = plan

  variables {
    required_checks = ["test", "trivy"]
  }

  assert {
    condition     = length(github_repository_ruleset.branch["main"].rules[0].required_status_checks[0].required_check) == 2
    error_message = "main must require the configured checks"
  }

  assert {
    condition     = length(github_repository_ruleset.branch["desen"].rules[0].required_status_checks) == 0
    error_message = "desen must not require checks"
  }
}

run "rulesets_disabled" {
  command = plan

  variables {
    visibility      = "private"
    enable_rulesets = false
  }

  assert {
    condition     = length(github_repository_ruleset.branch) == 0
    error_message = "no rulesets expected"
  }

  assert {
    condition     = !github_repository.this.delete_branch_on_merge
    error_message = "auto delete must be off when branches are unprotected"
  }
}

run "actions_variables" {
  command = plan

  variables {
    actions_variables = { AWS_REGION = "us-east-1" }
  }

  assert {
    condition     = github_actions_variable.this["AWS_REGION"].value == "us-east-1"
    error_message = "actions variable not set"
  }
}

run "invalid_name" {
  command = plan

  variables {
    name = "Payments_API"
  }

  expect_failures = [var.name]
}
