variable "IS_DEV" {
  type    = bool
  default = false
}

locals {
  DOMAIN = var.IS_DEV ? format("d.%s", var.CF_DOMAIN) : var.CF_DOMAIN
}

# ALL set in https://app.terraform.io/
variable "CF_EMAIL" {
  type = string
}

variable "CF_API_KEY" {
  type = string
}

variable "CF_WORKER_ACCOUNT_ID" {
  type = string
}

variable "CF_DOMAIN_ZONE_ID" {
  type = string
}

variable "CF_DOMAIN" {
  type = string
}