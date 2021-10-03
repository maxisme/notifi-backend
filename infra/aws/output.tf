output "AWS_WS_DOMAIN" {
  value = aws_apigatewayv2_domain_name.ws-notifi.domain_name_configuration[0].target_domain_name
}

output "AWS_HTTP_DOMAIN" {
  value = aws_apigatewayv2_domain_name.notifi.domain_name_configuration[0].target_domain_name
}

output "HTTP_DOMAIN" {
  value = local.DOMAIN
}

output "WS_DOMAIN" {
  value = local.WS_DOMAIN
}