variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "name_prefix" {
  type = string
}

variable "external_domain" {
  description = "Public domain for Zitadel (e.g. auth.zagforge.com)"
  type        = string
}

variable "zitadel_image" {
  description = "Zitadel container image with tag"
  type        = string
  default     = "ghcr.io/zitadel/zitadel:v2.71.6"
}

variable "min_instances" {
  description = "Minimum instances (set to 1 in prod — auth should never cold start)"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum instances"
  type        = number
  default     = 3
}

variable "cpu" {
  description = "CPU limit per instance"
  type        = string
  default     = "1"
}

variable "memory" {
  description = "Memory limit per instance"
  type        = string
  default     = "512Mi"
}
