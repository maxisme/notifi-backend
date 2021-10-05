resource "aws_ecr_repository" "notifi" {
  name = var.IS_DEV ? "notifi-dev" : "notifi"
}

resource "aws_ecr_lifecycle_policy" "notifi" {
  repository = aws_ecr_repository.notifi.name
  policy     = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Keep only one untagged image, expire all others",
            "selection": {
                "tagStatus": "untagged",
                "countType": "imageCountMoreThan",
                "countNumber": 1
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}
