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
resource "aws_apigatewayv2_deployment" "http" {
  api_id = aws_apigatewayv2_route.http.api_id

  lifecycle {
    create_before_destroy = true
  }
}

////////////
// stages //
////////////

resource "aws_apigatewayv2_stage" "prod" {
  name        = "prod"
  api_id      = aws_apigatewayv2_api.http.id
  auto_deploy = true
}
resource "aws_apigatewayv2_stage" "ws" {
  api_id = aws_apigatewayv2_api.ws.id
  name   = "ws"
}

//////////////////
// integrations //
//////////////////
// HTTP
resource "aws_apigatewayv2_integration" "http" {
  api_id             = aws_apigatewayv2_api.http.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.http.invoke_arn
}
resource "aws_apigatewayv2_route" "http" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "$default"
  target    = "integrations/${aws_apigatewayv2_integration.http.id}"
}

// Web Socket
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