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
}

resource "aws_apigatewayv2_api" "ws" {
  name          = "notifi-websocket"
  protocol_type = "WEBSOCKET"
}

resource "aws_apigatewayv2_api" "http" {
  name          = "notifi-http"
  protocol_type = "HTTP"
}

resource "aws_dynamodb_table" "user-table" {
  name         = "user"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "credentials"

  attribute {
    name = "credentials"
    type = "S"
  }

  tags = {
    "project" : "notifi"
  }
}

resource "aws_dynamodb_table" "notification-table" {
  name         = "notification"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "uuid"

  attribute {
    name = "uuid"
    type = "S"
  }

  attribute {
    name = "credentials"
    type = "S"
  }

  local_secondary_index {
    name            = "credentials-index"
    projection_type = "ALL"
    range_key       = "credentials"
  }

  tags = {
    "project" : "notifi"
  }
}