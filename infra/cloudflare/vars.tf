variable "IS_DEV" {
  type    = bool
  default = false
}

variable "HTTP_DOMAIN" {
  type = string
}

variable "WS_DOMAIN" {
  type = string
}

variable "AWS_WS_DOMAIN_GATEWAY" {
  type = string
}

variable "AWS_HTTP_DOMAIN_GATEWAY" {
  type = string
}

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