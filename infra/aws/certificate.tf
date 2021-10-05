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