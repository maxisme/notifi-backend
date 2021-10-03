variable "DOCKER_TAG" {
  type    = string
  default = "latest"
}

variable "TAG" {
  type    = string
  default = "notifi"
}

variable "SUB_DOMAIN" {
  type    = string
  default = ""
}

# ALL set in https://app.terraform.io/
variable "AWS_REGION" {
  type = string
}

variable "ENCRYPTION_KEY" {
  type = string
}

variable "CF_DOMAIN" {
  type = string
}