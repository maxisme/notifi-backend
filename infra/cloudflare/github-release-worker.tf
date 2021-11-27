resource "cloudflare_worker_route" "github" {
  zone_id     = var.CF_DOMAIN_ZONE_ID
  pattern     = format("%s/version*", local.HTTP_DOMAIN)
  script_name = cloudflare_worker_script.github_release.name
}

resource "cloudflare_worker_script" "github_release" {
  name    = var.IS_DEV ? "notifi-github-release-dev" : "notifi-github-release"
  content = file("${path.module}/worker-scripts/github-release.js")
}