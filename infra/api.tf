resource "aws_apigatewayv2_api" "ws" {
  name                       = "notifi-websocket"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

resource "aws_apigatewayv2_api" "http" {
  name          = "notifi-http"
  protocol_type = "HTTP"
}


////////////////
// deployment //
////////////////
resource "aws_apigatewayv2_deployment" "develop-api" {
  api_id = aws_apigatewayv2_route.api.api_id

  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_apigatewayv2_deployment" "production-api" {
  api_id = aws_apigatewayv2_route.api.api_id

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_apigatewayv2_deployment" "develop-code" {
  api_id = aws_apigatewayv2_route.code.api_id

  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_apigatewayv2_deployment" "production-code" {
  api_id = aws_apigatewayv2_route.code.api_id

  lifecycle {
    create_before_destroy = true
  }
}

////////////
// stages //
////////////

resource "aws_apigatewayv2_stage" "develop-api" {
  api_id        = aws_apigatewayv2_api.http.id
  name          = "develop-api"
  deployment_id = aws_apigatewayv2_deployment.develop-api.id
}

resource "aws_apigatewayv2_stage" "production-api" {
  api_id        = aws_apigatewayv2_api.http.id
  name          = "production-api"
  deployment_id = aws_apigatewayv2_deployment.production-api.id
}

resource "aws_apigatewayv2_stage" "develop-code" {
  api_id        = aws_apigatewayv2_api.http.id
  name          = "develop-code"
  deployment_id = aws_apigatewayv2_deployment.develop-code.id
}

resource "aws_apigatewayv2_stage" "production-code" {
  api_id        = aws_apigatewayv2_api.http.id
  name          = "production-code"
  deployment_id = aws_apigatewayv2_deployment.production-code.id
}

resource "aws_apigatewayv2_stage" "develop-ws" {
  api_id        = aws_apigatewayv2_api.ws.id
  name          = "develop-ws"
  auto_deploy   = true
  deployment_id = ""
}

resource "aws_apigatewayv2_stage" "production-ws" {
  api_id = aws_apigatewayv2_api.ws.id
  name   = "production-ws"
}

//////////////////
// integrations //
//////////////////
// HTTP
resource "aws_apigatewayv2_integration" "api" {
  api_id           = aws_apigatewayv2_api.http.id
  connection_type  = "INTERNET"
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.api.invoke_arn
}
resource "aws_apigatewayv2_route" "api" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /api"
  target    = "integrations/${aws_apigatewayv2_integration.api.id}"
}

resource "aws_apigatewayv2_integration" "code" {
  api_id           = aws_apigatewayv2_api.http.id
  connection_type  = "INTERNET"
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.code.invoke_arn
}
resource "aws_apigatewayv2_route" "code" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /code"
  target    = "integrations/${aws_apigatewayv2_integration.code.id}"
}

// WS
resource "aws_apigatewayv2_integration" "message" {
  api_id                    = aws_apigatewayv2_api.ws.id
  integration_type          = "AWS"
  content_handling_strategy = "CONVERT_TO_TEXT"
  integration_method        = "POST"
  integration_uri           = aws_lambda_function.message.invoke_arn
}
resource "aws_apigatewayv2_route" "message" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$default"
}


resource "aws_apigatewayv2_integration" "disconnect" {
  api_id                    = aws_apigatewayv2_api.ws.id
  integration_type          = "AWS"
  content_handling_strategy = "CONVERT_TO_TEXT"
  integration_method        = "POST"
  integration_uri           = aws_lambda_function.disconnect.invoke_arn
}
resource "aws_apigatewayv2_route" "disconnect" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$disconnect"
}


resource "aws_apigatewayv2_integration" "connect" {
  api_id                    = aws_apigatewayv2_api.ws.id
  integration_type          = "AWS"
  content_handling_strategy = "CONVERT_TO_TEXT"
  integration_method        = "POST"
  integration_uri           = aws_lambda_function.connect.invoke_arn
}
resource "aws_apigatewayv2_route" "connect" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$connect"
}