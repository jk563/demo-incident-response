resource "aws_secretsmanager_secret" "git_pat" {
  name                    = "${local.project}/git-pat"
  description             = "Personal access token for Git provider (GitHub or GitLab) issue creation and source file access"
  recovery_window_in_days = 0

  tags = local.common_tags
}
