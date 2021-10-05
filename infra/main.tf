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
