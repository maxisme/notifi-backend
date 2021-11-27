output "AWS_WS_DOMAIN_GATEWAY" {
  value = aws_apigatewayv2_domain_name.ws-notifi.domain_name_configuration[0].target_domain_name
}

output "AWS_API_DOMAIN_GATEWAY" {
  value = aws_apigatewayv2_domain_name.notifi.domain_name_configuration[0].target_domain_name
}

output "API_DOMAIN" {
  value = local.API_DOMAIN
}

output "WS_DOMAIN" {
  value = local.WS_DOMAIN
}