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
  region  = "us-east-1"

  default_tags {
    tags = {
      "project" : "notifi"
    }
  }
}
