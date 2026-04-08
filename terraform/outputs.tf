output "app_url" {
  description = "Application URL"
  value       = "https://${local.app_domain}"
}

output "observer_url" {
  description = "Observer UI URL"
  value       = "https://${local.observer_domain}"
}

output "dashboard_url" {
  description = "CloudWatch dashboard URL"
  value       = "https://${data.aws_region.current.name}.console.aws.amazon.com/cloudwatch/home?region=${data.aws_region.current.name}#dashboards:name=${local.project}"
}

output "log_group_name" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.app.name
}

output "alarm_name" {
  description = "CloudWatch alarm name"
  value       = aws_cloudwatch_metric_alarm.error_rate.alarm_name
}

output "sns_topic_arn" {
  description = "SNS topic ARN for alarm notifications"
  value       = aws_sns_topic.alarm.arn
}

output "ecr_app_url" {
  description = "ECR repository URL for the order API"
  value       = aws_ecr_repository.app.repository_url
}

output "ecr_agent_url" {
  description = "ECR repository URL for the triage agent"
  value       = aws_ecr_repository.agent.repository_url
}

output "lambda_arn" {
  description = "Triage agent Lambda ARN"
  value       = aws_lambda_function.triage_agent.arn
}

output "dynamodb_table" {
  description = "DynamoDB table name"
  value       = aws_dynamodb_table.orders.name
}

output "events_table" {
  description = "Agent events DynamoDB table name"
  value       = aws_dynamodb_table.agent_events.name
}
