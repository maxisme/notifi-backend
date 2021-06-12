resource "aws_ecr_repository" "notifi" {
  name                 = "notifi"
  image_tag_mutability = "MUTABLE"
}

data "aws_ecr_repository" "notifi" {
  name = "notifi"
}
