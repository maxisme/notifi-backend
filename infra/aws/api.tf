resource "aws_apigatewayv2_api" "ws" {
  name                       = var.IS_DEV ? "notifi-ws-dev" : "notifi-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

resource "aws_apigatewayv2_api" "http" {
  name          = var.IS_DEV ? "notifi-http-dev" : "notifi-http"
  protocol_type = "HTTP"
}

////////////////
// deployment //
////////////////
resource "aws_apigatewayv2_deployment" "api" {
  api_id = aws_apigatewayv2_route.api.api_id
  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_apigatewayv2_deployment" "code" {
  api_id = aws_apigatewayv2_route.code.api_id
  lifecycle {
    create_before_destroy = true
  }
}

////////////
// stages //
////////////
resource "aws_apigatewayv2_stage" "http" {
  name        = var.IS_DEV ? "dev" : "prod"
  api_id      = aws_apigatewayv2_api.http.id
  auto_deploy = true
}
resource "aws_apigatewayv2_stage" "ws" {
  api_id      = aws_apigatewayv2_api.ws.id
  name        = var.IS_DEV ? "dev" : "prod"
  auto_deploy = true

  default_route_settings {
    throttling_rate_limit  = 100
    throttling_burst_limit = 50
  }
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
resource "aws_apigatewayv2_route" "code" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "POST /code"
  target    = "integrations/${aws_apigatewayv2_integration.http.id}"
}
resource "aws_apigatewayv2_route" "api" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /api"
  target    = "integrations/${aws_apigatewayv2_integration.http.id}"
}
resource "aws_apigatewayv2_route" "ws-redirect" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /ws"
  target    = "integrations/${aws_apigatewayv2_integration.http.id}"
}

# website proxy
resource "aws_apigatewayv2_integration" "website" {
  api_id             = aws_apigatewayv2_api.http.id
  integration_type   = "HTTP_PROXY"
  integration_method = "ANY"
  integration_uri    = "${var.PAGES_PROXY_URL}/{proxy}"
}
resource "aws_apigatewayv2_route" "website" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "GET /{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.website.id}"
}

resource "aws_apigatewayv2_api_mapping" "http" {
  api_id      = aws_apigatewayv2_api.http.id
  domain_name = aws_apigatewayv2_domain_name.notifi.id
  stage       = aws_apigatewayv2_stage.http.id
}
resource "aws_apigatewayv2_domain_name" "notifi" {
  domain_name = local.DOMAIN

  domain_name_configuration {
    certificate_arn = !var.IS_DEV ? aws_acm_certificate_validation.notifi[0].certificate_arn : aws_acm_certificate_validation.sub-notifi.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

// Web Socket
resource "aws_apigatewayv2_integration" "message" {
  api_id             = aws_apigatewayv2_api.ws.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.message.invoke_arn
}
resource "aws_apigatewayv2_route" "message" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$default"
  target    = "integrations/${aws_apigatewayv2_integration.message.id}"
}

resource "aws_apigatewayv2_integration" "disconnect" {
  api_id           = aws_apigatewayv2_api.ws.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.disconnect.invoke_arn
}
resource "aws_apigatewayv2_route" "disconnect" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$disconnect"
  target    = "integrations/${aws_apigatewayv2_integration.disconnect.id}"
}

resource "aws_apigatewayv2_integration" "connect" {
  api_id             = aws_apigatewayv2_api.ws.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.connect.invoke_arn
}
resource "aws_apigatewayv2_route" "connect" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$connect"
  target    = "integrations/${aws_apigatewayv2_integration.connect.id}"
}

resource "aws_apigatewayv2_api_mapping" "ws" {
  api_id      = aws_apigatewayv2_api.ws.id
  domain_name = aws_apigatewayv2_domain_name.ws-notifi.id
  stage       = aws_apigatewayv2_stage.ws.id
}
resource "aws_apigatewayv2_domain_name" "ws-notifi" {
  domain_name = local.WS_DOMAIN

  domain_name_configuration {
    certificate_arn = aws_acm_certificate_validation.sub-notifi.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}