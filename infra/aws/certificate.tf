# resource "aws_acm_certificate_validation" "notifi" {
#   count           = var.IS_DEV ? 0 : 1
#   certificate_arn = aws_acm_certificate.notifi[0].arn
# }
resource "aws_acm_certificate" "notifi" {
  count             = var.IS_DEV ? 0 : 1
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