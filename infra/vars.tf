variable "AWS_REGION" {
  type    = string
  default = "us-east-1"
}

# ALL set in https://app.terraform.io/
variable "ENCRYPTION_KEY" {
  type = string
}

variable "DEV_ENCRYPTION_KEY" {
  type = string
}

variable "SERVER_KEY" {
  type = string
}

variable "DEV_SERVER_KEY" {
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
  # found under API on domain page
  type = string
}

variable "CF_DOMAIN" {
  type = string
}