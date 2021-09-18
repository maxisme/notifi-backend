resource "aws_iam_role" "iam_for_lambda" {
  name = "iam_for_lambda"

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


resource "aws_lambda_function" "api" {
  function_name = "notifi-api"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = data.aws_ecr_repository.notifi.repository_url
  image_config {
    command = ["./main", "api"]
  }
  package_type = "Image"
}

resource "aws_lambda_function" "connect" {
  function_name = "notifi-connect"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = data.aws_ecr_repository.notifi.repository_url
  image_config {
    command = ["./main", "connect"]
  }
  package_type = "Image"
}

resource "aws_lambda_function" "disconnect" {
  function_name = "notifi-disconnect"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = data.aws_ecr_repository.notifi.repository_url
  image_config {
    command = ["./main", "disconnect"]
  }
  package_type = "Image"
}

resource "aws_lambda_function" "message" {
  function_name = "notifi-message"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = data.aws_ecr_repository.notifi.repository_url
  image_config {
    command = ["./main", "message"]
  }
  package_type = "Image"
}

resource "aws_lambda_function" "code" {
  function_name = "notifi-code"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = data.aws_ecr_repository.notifi.repository_url
  image_config {
    command = ["./main", "code"]
  }
  package_type = "Image"
}