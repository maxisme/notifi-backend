terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.1"
    }
  }
}

provider "cloudflare" {
  email      = var.CF_EMAIL
  api_key    = var.CF_API_KEY
  account_id = var.CF_WORKER_ACCOUNT_ID
}