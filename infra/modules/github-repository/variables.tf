variable "name" {
  type = string

  validation {
    condition     = can(regex("^[a-z0-9]([a-z0-9-]*[a-z0-9])?$", var.name)) && length(var.name) <= 100
    error_message = "name must be lowercase letters, digits and hyphens."
  }
}

variable "description" {
  type    = string
  default = ""
}

variable "visibility" {
  type    = string
  default = "private"

  validation {
    condition     = contains(["public", "private", "internal"], var.visibility)
    error_message = "visibility must be public, private or internal."
  }
}

variable "template" {
  description = "Template repository to generate from. Null creates an empty repository."
  type = object({
    owner      = string
    repository = string
  })
  default = null
}

variable "topics" {
  type    = list(string)
  default = []
}

variable "enable_rulesets" {
  description = "Rulesets on private repositories need a paid plan."
  type        = bool
  default     = true
}

variable "required_approvals" {
  type    = number
  default = 1
}

variable "required_checks" {
  description = "Status check names required on main and homol."
  type        = list(string)
  default     = []
}

variable "actions_variables" {
  type    = map(string)
  default = {}
}
