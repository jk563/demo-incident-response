resource "aws_sns_topic" "alarm" {
  name = "${local.project}-alarm-topic"

  tags = local.common_tags
}

resource "aws_sns_topic_subscription" "lambda" {
  topic_arn = aws_sns_topic.alarm.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.triage_agent.arn
}
