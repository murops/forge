output "name" {
  value = github_repository.this.name
}

output "full_name" {
  value = github_repository.this.full_name
}

output "html_url" {
  value = github_repository.this.html_url
}

output "protected_branches" {
  value = sort(keys(github_repository_ruleset.branch))
}

output "ssh_clone_url" {
  value = github_repository.this.ssh_clone_url
}
