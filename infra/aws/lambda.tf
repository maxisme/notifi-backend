resource "aws_lambda_function" "connect" {
  function_name = var.IS_DEV ? "notifi-connect-dev" : "notifi-connect"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = local.IMAGE_URI
  package_type  = "Image"
  image_config {
    entry_point = ["/main", "connect"]
  }
  environment {
    variables = {
      ENCRYPTION_KEY          = var.ENCRYPTION_KEY
      NOTIFICATION_TABLE_NAME = aws_dynamodb_table.notification-table.name
      SERVER_KEY              = var.SERVER_KEY
      USER_TABLE_NAME         = aws_dynamodb_table.user-table.name
      WS_ENDPOINT             = local.AWS_WS_ENDPOINT
    }
  }
}
resource "aws_lambda_permission" "connect" {
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.connect.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.ws.execution_arn}/**"
}

resource "aws_lambda_function" "disconnect" {
  function_name = var.IS_DEV ? "notifi-disconnect-dev" : "notifi-disconnect"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = local.IMAGE_URI
  package_type  = "Image"
  image_config {
    entry_point = ["/main", "disconnect"]
  }
  environment {
    variables = {
      SERVER_KEY      = var.SERVER_KEY
      USER_TABLE_NAME = aws_dynamodb_table.user-table.name
    }
  }
}
resource "aws_lambda_permission" "disconnect" {
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.disconnect.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.ws.execution_arn}/**"
}

resource "aws_lambda_function" "message" {
  function_name = var.IS_DEV ? "notifi-message-dev" : "notifi-message"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = local.IMAGE_URI
  package_type  = "Image"
  image_config {
    entry_point = ["/main", "message"]
  }
  environment {
    variables = {
      ENCRYPTION_KEY          = var.ENCRYPTION_KEY
      NOTIFICATION_TABLE_NAME = aws_dynamodb_table.notification-table.name
      SERVER_KEY              = var.SERVER_KEY
      USER_TABLE_NAME         = aws_dynamodb_table.user-table.name
      WS_ENDPOINT             = local.AWS_WS_ENDPOINT
    }
  }
}
resource "aws_lambda_permission" "message" {
  statement_id  = "AllowExecutionFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.message.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.ws.execution_arn}/**"
}

resource "aws_lambda_function" "http" {
  function_name = var.IS_DEV ? "http-dev" : "http"
  role          = aws_iam_role.iam_for_lambda.arn
  image_uri     = local.IMAGE_URI
  package_type  = "Image"
  image_config {
    entry_point = ["/main", "http"]
  }
  environment {
    variables = {
      ENCRYPTION_KEY                = var.ENCRYPTION_KEY
      FIREBASE_CREDENTIALS_JSON_B64 = var.FIREBASE_CREDENTIALS_JSON_B64
      NOTIFICATION_TABLE_NAME       = aws_dynamodb_table.notification-table.name
      SERVER_KEY                    = var.SERVER_KEY
      USER_TABLE_NAME               = aws_dynamodb_table.user-table.name
      WS_ENDPOINT                   = local.AWS_WS_ENDPOINT
      WS_HOST                       = local.WS_DOMAIN
    }
  }
}
resource "aws_lambda_permission" "http" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.http.function_name
  principal     = "apigateway.amazonaws.com"
}