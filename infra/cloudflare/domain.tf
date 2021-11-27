resource "cloudflare_record" "notifi-page" {
  name    = var.IS_DEV ? replace(local.HTTP_DOMAIN, format(".%s", var.CF_DOMAIN), "") : local.HTTP_DOMAIN
  zone_id = var.CF_DOMAIN_ZONE_ID
  value   = var.PAGES_PROXY_URL
  type    = "CNAME"
  proxied = true
}

resource "cloudflare_record" "notifi-api" {
  name    = replace(var.API_DOMAIN, format(".%s", var.CF_DOMAIN), "")
  zone_id = var.CF_DOMAIN_ZONE_ID
  value   = var.AWS_HTTP_DOMAIN_GATEWAY
  type    = "CNAME"
  proxied = true
}

resource "cloudflare_record" "notifi-ws" {
  name    = replace(var.WS_DOMAIN, format(".%s", var.CF_DOMAIN), "")
  zone_id = var.CF_DOMAIN_ZONE_ID
  value   = var.AWS_WS_DOMAIN_GATEWAY
  type    = "CNAME"
  proxied = true
}