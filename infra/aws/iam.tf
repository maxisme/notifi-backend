resource "aws_iam_role" "iam_for_lambda" {
  name               = var.IS_DEV ? "iam_for_lambda_dev" : "iam_for_lambda"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

# lambda
resource "aws_iam_role_policy_attachment" "lambda_policy" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# gateway
resource "aws_iam_role_policy_attachment" "api_gateway_policy" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonAPIGatewayInvokeFullAccess"
}

# db
resource "aws_iam_role_policy" "lambda_db_notification_policy" {
  role = aws_iam_role.iam_for_lambda.id
  policy = templatefile("${path.module}/templates/policy.tpl", {
    table_arn = aws_dynamodb_table.notification-table.arn
  })
}

resource "aws_iam_role_policy" "lambda_db_user_policy" {
  role = aws_iam_role.iam_for_lambda.id
  policy = templatefile("${path.module}/templates/policy.tpl", {
    table_arn = aws_dynamodb_table.user-table.arn
  })
}