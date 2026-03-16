resource "aws_ecr_repository" "app" {
  name                 = local.app_name
  image_tag_mutability = "MUTABLE"
  force_delete         = true

  tags = local.common_tags
}

resource "aws_ecr_repository" "agent" {
  name                 = local.agent_name
  image_tag_mutability = "MUTABLE"
  force_delete         = true

  tags = local.common_tags
}
