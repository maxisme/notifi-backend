resource "aws_dynamodb_table" "user-table" {
  name         = "user"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "credentials"

  attribute {
    name = "credentials"
    type = "S"
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

  global_secondary_index {
    name            = "credentials-index"
    projection_type = "ALL"
    hash_key        = "credentials"
  }
}