variable "docker_tag" {
  type    = string
  default = "latest"
}

# ALL set in https://app.terraform.io/
variable "AWS_REGION" {
  type = string
}

variable "ENCRYPTION_KEY" {
  type = string
}