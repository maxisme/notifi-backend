resource "aws_dynamodb_table" "user-table" {
  name         = var.IS_DEV ? "dev-user" : "user"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "device_uuid"

  attribute {
    name = "device_uuid"
    type = "S"
  }

  attribute {
    name = "credentials"
    type = "S"
  }

  attribute {
    name = "connection_id"
    type = "S"
  }

  global_secondary_index {
    name            = "credentials-index"
    projection_type = "ALL"
    hash_key        = "credentials"
  }

  global_secondary_index {
    name            = "connection_id-index"
    projection_type = "ALL"
    hash_key        = "connection_id"
  }
}

resource "aws_dynamodb_table" "notification-table" {
  name         = var.IS_DEV ? "dev-notification" : "notification"
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

data "template_file" "policy_notification" {
  template = file("${path.module}/templates/policy.tpl")
  vars = {
    table_arn = aws_dynamodb_table.notification-table.arn
  }
}
resource "aws_iam_role_policy" "lambda_db_notification_policy" {
  role   = aws_iam_role.iam_for_lambda.id
  policy = data.template_file.policy_notification.rendered
}

data "template_file" "policy_user" {
  template = file("${path.module}/templates/policy.tpl")
  vars = {
    table_arn = aws_dynamodb_table.user-table.arn
  }
}

resource "aws_iam_role_policy" "lambda_db_user_policy" {
  role   = aws_iam_role.iam_for_lambda.id
  policy = data.template_file.policy_user.rendered
}