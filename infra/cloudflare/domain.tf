resource "cloudflare_record" "notifi" {
  name    = var.IS_DEV ? replace(var.HTTP_DOMAIN, format(".%s", var.CF_DOMAIN), "") : var.HTTP_DOMAIN
  zone_id = var.CF_DOMAIN_ZONE_ID
  value   = var.AWS_HTTP_DOMAIN
  type    = "CNAME"
  proxied = true
}

resource "cloudflare_record" "notifi-ws" {
  name    = replace(var.WS_DOMAIN, format(".%s", var.CF_DOMAIN), "")
  zone_id = var.CF_DOMAIN_ZONE_ID
  value   = var.AWS_WS_DOMAIN
  type    = "CNAME"
  proxied = false
}