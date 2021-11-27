module "aws" {
  AWS_REGION          = var.AWS_REGION
  CF_DOMAIN           = var.CF_DOMAIN
  ENCRYPTION_KEY      = var.ENCRYPTION_KEY
  FIREBASE_SERVER_KEY = var.FIREBASE_SERVER_KEY
  SERVER_KEY          = var.SERVER_KEY
  source              = "./aws"
  IS_DEV              = false
}

module "cloudflare" {
  source               = "./cloudflare"
  CF_API_KEY           = var.CF_API_KEY
  CF_WORKER_ACCOUNT_ID = var.CF_WORKER_ACCOUNT_ID
  CF_DOMAIN            = var.CF_DOMAIN
  CF_DOMAIN_ZONE_ID    = var.CF_DOMAIN_ZONE_ID
  CF_EMAIL             = var.CF_EMAIL
  IS_DEV               = false

  HTTP_DOMAIN             = module.aws.HTTP_DOMAIN
  WS_DOMAIN               = module.aws.WS_DOMAIN
  AWS_WS_DOMAIN_GATEWAY   = module.aws.AWS_WS_DOMAIN_GATEWAY
  AWS_HTTP_DOMAIN_GATEWAY = module.aws.AWS_HTTP_DOMAIN_GATEWAY
}