# create notifi repo manually - https://console.aws.amazon.com/ecr/repositories

data "aws_ecr_repository" "notifi" {
  name = "notifi"
}
