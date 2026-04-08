locals {
  app_dir      = "${path.module}/../demo-order-api"
  observer_dir = "${path.module}/../observer"
  project_root = "${path.module}/.."

  app_files      = sort(concat(
    tolist(fileset(local.app_dir, "**/*.go")),
    tolist(fileset(local.app_dir, "Dockerfile")),
    tolist(fileset(local.app_dir, "go.mod")),
    tolist(fileset(local.app_dir, "go.sum")),
    tolist(fileset(local.app_dir, "web/**/*")),
  ))
  observer_files = sort(fileset(local.observer_dir, "**/*"))

  app_source_hash = sha1(join("", concat(
    [for f in local.app_files : filesha256("${local.app_dir}/${f}")],
    [for f in local.observer_files : filesha256("${local.observer_dir}/${f}")],
  )))
}

resource "terraform_data" "app_image" {
  triggers_replace = [local.app_source_hash]

  provisioner "local-exec" {
    working_dir = local.project_root
    environment = {
      REPO_URL   = aws_ecr_repository.app.repository_url
      IMAGE_TAG  = var.app_image_tag
      REGION     = data.aws_region.current.name
      ACCOUNT_ID = data.aws_caller_identity.current.account_id
    }
    command = <<-EOT
      aws ecr get-login-password --region "$REGION" \
        | docker login --username AWS --password-stdin "$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com"
      docker build --provenance=false --platform linux/arm64 -f demo-order-api/Dockerfile -t "$REPO_URL:$IMAGE_TAG" .
      docker push "$REPO_URL:$IMAGE_TAG"
    EOT
  }

  depends_on = [aws_ecr_repository.app]
}

# Agent image build.
locals {
  agent_dir = "${path.module}/../agent"

  agent_src = sort(concat(
    tolist(fileset(local.agent_dir, "**/*.py")),
    tolist(fileset(local.agent_dir, "Dockerfile")),
    tolist(fileset(local.agent_dir, "requirements.txt")),
    tolist(fileset(local.agent_dir, "prompts/**/*")),
  ))

  agent_source_hash = sha1(join("", [for f in local.agent_src : filesha256("${local.agent_dir}/${f}")]))
}

resource "terraform_data" "agent_image" {
  triggers_replace = [local.agent_source_hash]

  provisioner "local-exec" {
    working_dir = local.agent_dir
    environment = {
      REPO_URL   = aws_ecr_repository.agent.repository_url
      IMAGE_TAG  = var.agent_image_tag
      REGION     = data.aws_region.current.name
      ACCOUNT_ID = data.aws_caller_identity.current.account_id
    }
    command = <<-EOT
      aws ecr get-login-password --region "$REGION" \
        | docker login --username AWS --password-stdin "$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com"
      docker build --provenance=false --platform linux/arm64 -t "$REPO_URL:$IMAGE_TAG" .
      docker push "$REPO_URL:$IMAGE_TAG"
    EOT
  }

  depends_on = [aws_ecr_repository.agent]
}
