resource "aws_lambda_function" "triage_agent" {
  function_name = local.agent_name
  role          = aws_iam_role.lambda.arn
  package_type  = "Image"
  image_uri     = "${aws_ecr_repository.agent.repository_url}:${var.agent_image_tag}"
  memory_size   = var.lambda_memory
  timeout       = var.lambda_timeout

  architectures = ["arm64"]

  environment {
    variables = {
      LOG_GROUP_NAME   = aws_cloudwatch_log_group.app.name
      GIT_PROVIDER     = var.git_provider
      GIT_REPO         = var.git_repo
      GIT_SECRET_NAME  = aws_secretsmanager_secret.git_pat.name
      GITLAB_URL       = var.gitlab_url
      BEDROCK_MODEL    = var.bedrock_model
      ALARM_NAME       = aws_cloudwatch_metric_alarm.error_rate.alarm_name
    }
  }

  tags = local.common_tags

  depends_on = [terraform_data.agent_image]
}

resource "aws_lambda_permission" "sns" {
  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.triage_agent.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.alarm.arn
}
