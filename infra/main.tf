terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "notifi"
    workspaces {
      name = "github-actions"
    }
  }
}

// add this provider due to bug
provider "aws" {
  region = var.AWS_REGION
}

module "aws" {
  source         = "./aws"
  AWS_REGION     = var.AWS_REGION
  ENCRYPTION_KEY = var.ENCRYPTION_KEY
  CF_DOMAIN      = var.CF_DOMAIN
}

module "cloudflare" {
  source               = "./cloudflare"
  CF_API_KEY           = var.CF_API_KEY
  CF_WORKER_ACCOUNT_ID = var.CF_WORKER_ACCOUNT_ID
  CF_DOMAIN            = var.CF_DOMAIN
  CF_DOMAIN_ZONE_ID    = var.CF_DOMAIN_ZONE_ID
  CF_EMAIL             = var.CF_EMAIL
}

