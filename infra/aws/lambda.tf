resource "aws_iam_role" "iam_for_lambda" {
  name               = "iam_for_lambda"
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

resource "aws_lambda_function" "connect" {
  function_name = "notifi-connect"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = format("%s:%s", aws_ecr_repository.notifi.repository_url, var.DOCKER_TAG)
  image_config {
    entry_point = ["/main", "connect"]
  }
  environment {
    variables = {
      ENCRYPTION_KEY = var.ENCRYPTION_KEY
      WS_ENDPOINT    = aws_apigatewayv2_api.ws.api_endpoint
    }
  }
  package_type = "Image"
}
resource "aws_lambda_permission" "connect" {
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.connect.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.ws.execution_arn}/**"
}

resource "aws_lambda_function" "disconnect" {
  function_name = "notifi-disconnect"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = format("%s:%s", aws_ecr_repository.notifi.repository_url, var.DOCKER_TAG)
  image_config {
    entry_point = ["/main", "disconnect"]
  }
  package_type = "Image"
}
resource "aws_lambda_permission" "disconnect" {
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.disconnect.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.ws.execution_arn}/**"
}

resource "aws_lambda_function" "message" {
  function_name = "notifi-message"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = format("%s:%s", aws_ecr_repository.notifi.repository_url, var.DOCKER_TAG)
  image_config {
    entry_point = ["/main", "message"]
  }
  environment {
    variables = {
      ENCRYPTION_KEY = var.ENCRYPTION_KEY
      WS_ENDPOINT    = aws_apigatewayv2_api.ws.api_endpoint
    }
  }
  package_type = "Image"
}
resource "aws_lambda_permission" "message" {
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.message.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.ws.execution_arn}/**"
}

resource "aws_lambda_function" "http" {
  function_name = "http"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = format("%s:%s", aws_ecr_repository.notifi.repository_url, var.DOCKER_TAG)
  image_config {
    entry_point = ["/main", "http"]
  }
  environment {
    variables = {
      ENCRYPTION_KEY = var.ENCRYPTION_KEY
      WS_ENDPOINT    = aws_apigatewayv2_api.ws.api_endpoint
    }
  }
  package_type = "Image"
}
resource "aws_lambda_permission" "http" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.http.function_name
  principal     = "apigateway.amazonaws.com"
}


resource "aws_iam_role_policy_attachment" "lambda_policy" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}