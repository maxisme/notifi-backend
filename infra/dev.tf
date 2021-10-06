module "cloudflare-develop" {
  source               = "./cloudflare"
  CF_API_KEY           = var.CF_API_KEY
  CF_WORKER_ACCOUNT_ID = var.CF_WORKER_ACCOUNT_ID
  CF_DOMAIN            = var.CF_DOMAIN
  CF_DOMAIN_ZONE_ID    = var.CF_DOMAIN_ZONE_ID
  CF_EMAIL             = var.CF_EMAIL

  HTTP_DOMAIN             = module.aws-develop.HTTP_DOMAIN
  WS_DOMAIN               = module.aws-develop.WS_DOMAIN
  AWS_WS_DOMAIN_GATEWAY   = module.aws-develop.AWS_WS_DOMAIN_GATEWAY
  AWS_HTTP_DOMAIN_GATEWAY = module.aws-develop.AWS_HTTP_DOMAIN_GATEWAY

  IS_DEV = true
}

module "aws-develop" {
  source         = "./aws"
  AWS_REGION     = var.AWS_REGION
  ENCRYPTION_KEY = var.DEV_ENCRYPTION_KEY
  SERVER_KEY     = var.DEV_SERVER_KEY
  CF_DOMAIN      = var.CF_DOMAIN
  IS_DEV         = true
}