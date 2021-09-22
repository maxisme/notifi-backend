resource "aws_dynamodb_table" "user-table" {
  name         = "user"
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

  global_secondary_index {
    name            = "credentials-index"
    projection_type = "ALL"
    hash_key        = "credentials"
  }
}

resource "aws_dynamodb_table" "notification-table" {
  name         = "notification"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "credentials"

  attribute {
    name = "credentials"
    type = "S"
  }
}

resource "aws_dynamodb_table" "ws-table" {
  name         = "ws"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "connection_id"

  attribute {
    name = "connection_id"
    type = "S"
  }

  attribute {
    name = "device_uuid"
    type = "S"
  }

  global_secondary_index {
    name            = "device_uuid-index"
    projection_type = "ALL"
    hash_key        = "device_uuid"
  }
}

data "template_file" "policy_ws" {
  template = file("${path.module}/templates/policy.tpl")
  vars = {
    table_arn = "${aws_dynamodb_table.ws-table.arn}"
  }
}
resource "aws_iam_role_policy" "lambda_db_ws_policy" {
  role   = aws_iam_role.iam_for_lambda.id
  policy = data.template_file.policy_ws.rendered
}

data "template_file" "policy_notification" {
  template = file("${path.module}/templates/policy.tpl")
  vars = {
    table_arn = "${aws_dynamodb_table.notification-table.arn}"
  }
}
resource "aws_iam_role_policy" "lambda_db_notification_policy" {
  role   = aws_iam_role.iam_for_lambda.id
  policy = data.template_file.policy_notification.rendered
}

data "template_file" "policy_user" {
  template = file("${path.module}/templates/policy.tpl")
  vars = {
    table_arn = "${aws_dynamodb_table.user-table.arn}"
  }
}

resource "aws_iam_role_policy" "lambda_db_user_policy" {
  role   = aws_iam_role.iam_for_lambda.id
  policy = data.template_file.policy_user.rendered
}