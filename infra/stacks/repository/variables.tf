variable "name" {
  type = string
}

variable "description" {
  type    = string
  default = ""
}

variable "visibility" {
  type    = string
  default = "public"
}

variable "github_owner" {
  type = string
}

variable "github_app_id" {
  type = string
}

variable "github_app_installation_id" {
  type = string
}

variable "github_app_private_key" {
  type      = string
  sensitive = true
}

variable "template" {
  type = object({
    owner      = string
    repository = string
  })
  default = null
}

variable "template_topic" {
  type    = string
  default = null
}

variable "required_checks" {
  type    = list(string)
  default = []
}

variable "private_rulesets_supported" {
  description = "Set to true when the organization plan allows rulesets on private repositories."
  type        = bool
  default     = false
}

variable "actions_variables" {
  description = "Usually the outputs of the lambda stack."
  type        = map(string)
  default     = {}
}
