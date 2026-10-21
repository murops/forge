locals {
  # desen is the integration branch, homol and main only change through PRs
  branch_rules = {
    main  = { require_pr = true, approvals = var.required_approvals, checks = var.required_checks }
    homol = { require_pr = true, approvals = 0, checks = var.required_checks }
    desen = { require_pr = false, approvals = 0, checks = [] }
  }
}

resource "github_repository" "this" {
  name        = var.name
  description = var.description
  visibility  = var.visibility
  topics      = distinct(concat(["forge-managed"], var.topics))

  auto_init = var.template == null

  dynamic "template" {
    for_each = var.template == null ? [] : [var.template]
    content {
      owner      = template.value.owner
      repository = template.value.repository
    }
  }

  has_issues   = true
  has_wiki     = false
  has_projects = false

  allow_merge_commit     = true
  allow_squash_merge     = true
  allow_rebase_merge     = false
  allow_update_branch    = true
  delete_branch_on_merge = var.enable_rulesets # without rulesets it would delete desen/homol after a merge

  archive_on_destroy = true
}

resource "github_repository_vulnerability_alerts" "this" {
  repository = github_repository.this.name
  enabled    = true
}

resource "github_branch" "env" {
  for_each = toset(["homol", "desen"])

  repository    = github_repository.this.name
  branch        = each.key
  source_branch = "main"
}

resource "github_branch_default" "this" {
  repository = github_repository.this.name
  branch     = "main"
}

resource "github_repository_ruleset" "branch" {
  for_each = var.enable_rulesets ? local.branch_rules : {}

  name        = "protect-${each.key}"
  repository  = github_repository.this.name
  target      = "branch"
  enforcement = "active"

  conditions {
    ref_name {
      include = ["refs/heads/${each.key}"]
      exclude = []
    }
  }

  # the API returns no id for OrganizationAdmin, 0 avoids a diff on every plan
  bypass_actors {
    actor_id    = 0
    actor_type  = "OrganizationAdmin"
    bypass_mode = "always"
  }

  rules {
    deletion         = true
    non_fast_forward = true

    dynamic "pull_request" {
      for_each = each.value.require_pr ? [1] : []
      content {
        required_approving_review_count   = each.value.approvals
        dismiss_stale_reviews_on_push     = true
        required_review_thread_resolution = true
      }
    }

    dynamic "required_status_checks" {
      for_each = length(each.value.checks) > 0 ? [1] : []
      content {
        strict_required_status_checks_policy = false

        dynamic "required_check" {
          for_each = each.value.checks
          content {
            context = required_check.value
          }
        }
      }
    }
  }

  depends_on = [github_branch.env]
}

resource "github_actions_variable" "this" {
  for_each = var.actions_variables

  repository    = github_repository.this.name
  variable_name = each.key
  value         = each.value
}
