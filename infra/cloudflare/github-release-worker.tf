resource "cloudflare_worker_route" "github" {
  zone_id     = var.CF_DOMAIN_ZONE_ID
  pattern     = format("%s/version*", var.CF_DOMAIN)
  script_name = cloudflare_worker_script.github_release.name
}

resource "cloudflare_worker_script" "github_release" {
  name    = "notifi-github-release"
  content = file("${path.module}/scripts/github-release.js")
}