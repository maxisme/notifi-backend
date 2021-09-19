terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.44"
    }
  }
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "notifi"
    workspaces {
      name = "github-actions"
    }
  }
}

provider "aws" {
  profile = "default"
  region  = "us-east-1"

  default_tags {
    tags = {
      "project" : "notifi"
    }
  }
}
