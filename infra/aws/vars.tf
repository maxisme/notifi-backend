variable "IS_DEV" {
  type    = bool
  default = false
}

locals {
  IMAGE_URI = format("%s:%s", aws_ecr_repository.notifi.repository_url, var.IS_DEV ? "develop" : "latest")
  DOMAIN    = var.IS_DEV ? format("d.%s", var.CF_DOMAIN) : var.CF_DOMAIN
  WS_DOMAIN = var.IS_DEV ? format("dws.%s", var.CF_DOMAIN) : format("ws.%s", var.CF_DOMAIN)
}

variable "AWS_REGION" {
  type = string
}

variable "ENCRYPTION_KEY" {
  type = string
}

variable "CF_DOMAIN" {
  type = string
}