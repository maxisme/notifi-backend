module "cloudflare-develop" {
  source               = "./cloudflare"
  CF_API_KEY           = var.CF_API_KEY
  CF_WORKER_ACCOUNT_ID = var.CF_WORKER_ACCOUNT_ID
  CF_DOMAIN            = var.CF_DOMAIN
  CF_DOMAIN_ZONE_ID    = var.CF_DOMAIN_ZONE_ID
  CF_EMAIL             = var.CF_EMAIL

  API_DOMAIN              = module.aws-develop.API_DOMAIN
  WS_DOMAIN               = module.aws-develop.WS_DOMAIN
  AWS_WS_DOMAIN_GATEWAY   = module.aws-develop.AWS_WS_DOMAIN_GATEWAY
  AWS_HTTP_DOMAIN_GATEWAY = module.aws-develop.AWS_API_DOMAIN_GATEWAY

  IS_DEV = true
}

module "aws-develop" {
  AWS_REGION          = var.AWS_REGION
  CF_DOMAIN           = var.CF_DOMAIN
  ENCRYPTION_KEY      = var.DEV_ENCRYPTION_KEY
  FIREBASE_SERVER_KEY = var.FIREBASE_SERVER_KEY
  IS_DEV              = true
  SERVER_KEY          = var.DEV_SERVER_KEY
  source              = "./aws"
}