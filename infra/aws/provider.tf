terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.44"
    }
  }
}

provider "aws" {
  profile = "default"
  region  = var.AWS_REGION

  default_tags {
    tags = {
      "project" : "notifi"
    }
  }
}