module "aws" {
  source         = "./aws"
  AWS_REGION     = var.AWS_REGION
  ENCRYPTION_KEY = var.ENCRYPTION_KEY
  SERVER_KEY     = var.SERVER_KEY
  CF_DOMAIN      = var.CF_DOMAIN
}

module "cloudflare" {
  source               = "./cloudflare"
  CF_API_KEY           = var.CF_API_KEY
  CF_WORKER_ACCOUNT_ID = var.CF_WORKER_ACCOUNT_ID
  CF_DOMAIN            = var.CF_DOMAIN
  CF_DOMAIN_ZONE_ID    = var.CF_DOMAIN_ZONE_ID
  CF_EMAIL             = var.CF_EMAIL

  # change aws-develop -> aws
  HTTP_DOMAIN             = module.aws-develop.HTTP_DOMAIN
  WS_DOMAIN               = module.aws-develop.WS_DOMAIN
  AWS_WS_DOMAIN_GATEWAY   = module.aws-develop.AWS_WS_DOMAIN_GATEWAY
  AWS_HTTP_DOMAIN_GATEWAY = module.aws-develop.AWS_HTTP_DOMAIN_GATEWAY
}