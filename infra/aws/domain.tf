resource "aws_acm_certificate_validation" "notifi" {
  certificate_arn = aws_acm_certificate.notifi.arn
}
resource "aws_acm_certificate" "notifi" {
  domain_name       = var.CF_DOMAIN
  validation_method = "EMAIL"
}

resource "aws_acm_certificate_validation" "sub-notifi" {
  certificate_arn = aws_acm_certificate.sub-notifi.arn
}
resource "aws_acm_certificate" "sub-notifi" {
  domain_name       = format("*.%s", var.CF_DOMAIN)
  validation_method = "EMAIL"
}

resource "aws_apigatewayv2_domain_name" "notifi" {
  domain_name = format("%s%s", var.SUB_DOMAIN, var.CF_DOMAIN)

  domain_name_configuration {
    certificate_arn = var.SUB_DOMAIN == "" ? aws_acm_certificate_validation.notifi.certificate_arn : aws_acm_certificate_validation.sub-notifi.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

resource "aws_apigatewayv2_domain_name" "ws-notifi" {
  domain_name = format("%sws.%s", var.SUB_DOMAIN, var.CF_DOMAIN)

  domain_name_configuration {
    certificate_arn = aws_acm_certificate_validation.sub-notifi.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}