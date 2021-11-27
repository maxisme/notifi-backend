resource "cloudflare_page_rule" "https-redirects" {
  zone_id  = var.CF_DOMAIN_ZONE_ID
  target   = "*${var.CF_DOMAIN}*"
  priority = 1

  actions {
    always_use_https = true
  }
}

resource "cloudflare_page_rule" "api-redirect" {
  zone_id  = var.CF_DOMAIN_ZONE_ID
  target   = "*${local.HTTP_DOMAIN}/*"
  priority = 2

  actions {
    forwarding_url {
      status_code = 301
      url         = "https://${var.API_DOMAIN}/$2"
    }
  }
}

