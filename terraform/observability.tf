# --- Log groups ---

resource "aws_cloudwatch_log_group" "app" {
  name              = "/ecs/${local.app_name}"
  retention_in_days = 7

  tags = local.common_tags
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${local.agent_name}"
  retention_in_days = 7

  tags = local.common_tags
}

# --- Metric filters ---

resource "aws_cloudwatch_log_metric_filter" "errors" {
  name           = "${local.app_name}-errors"
  log_group_name = aws_cloudwatch_log_group.app.name
  pattern        = "{ $.level = \"ERROR\" }"

  metric_transformation {
    name          = "LogErrorCount"
    namespace     = local.namespace
    value         = "1"
    default_value = "0"
  }
}

resource "aws_cloudwatch_log_metric_filter" "requests" {
  name           = "${local.app_name}-requests"
  log_group_name = aws_cloudwatch_log_group.app.name
  pattern        = "{ $.status_code = * }"

  metric_transformation {
    name          = "LogRequestCount"
    namespace     = local.namespace
    value         = "1"
    default_value = "0"
  }
}

# --- Error rate alarm (math expression) ---

resource "aws_cloudwatch_metric_alarm" "error_rate" {
  alarm_name        = "${local.project}-error-rate"
  alarm_description = "Error rate exceeds ${var.alarm_threshold}%"

  comparison_operator = "GreaterThanThreshold"
  threshold           = var.alarm_threshold
  evaluation_periods  = 1
  datapoints_to_alarm = 1

  metric_query {
    id          = "e1"
    expression  = "IF(m2 > 0, m1/m2*100, 0)"
    label       = "Error Rate %"
    return_data = true
  }

  metric_query {
    id = "m1"

    metric {
      metric_name = "LogErrorCount"
      namespace   = local.namespace
      period      = 60
      stat        = "Sum"
    }
  }

  metric_query {
    id = "m2"

    metric {
      metric_name = "LogRequestCount"
      namespace   = local.namespace
      period      = 60
      stat        = "Sum"
    }
  }

  alarm_actions = [aws_sns_topic.alarm.arn]
  ok_actions    = [aws_sns_topic.alarm.arn]

  tags = local.common_tags
}

# --- Dashboard ---

resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = local.project

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "alarm"
        x      = 0
        y      = 0
        width  = 24
        height = 3
        properties = {
          title  = "Alarm Status"
          alarms = [aws_cloudwatch_metric_alarm.error_rate.arn]
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 3
        width  = 12
        height = 6
        properties = {
          title  = "Request Rate"
          region = data.aws_region.current.name
          metrics = [
            [local.namespace, "LogRequestCount", { stat = "Sum", period = 60, label = "Requests/min" }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 3
        width  = 12
        height = 6
        properties = {
          title  = "Error Rate %"
          region = data.aws_region.current.name
          metrics = [
            [{ expression = "IF(m2 > 0, m1/m2*100, 0)", label = "Error Rate %", id = "e1" }],
            [local.namespace, "LogErrorCount", { stat = "Sum", period = 60, id = "m1", visible = false }],
            [local.namespace, "LogRequestCount", { stat = "Sum", period = 60, id = "m2", visible = false }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 9
        width  = 12
        height = 6
        properties = {
          title  = "Error Count"
          region = data.aws_region.current.name
          metrics = [
            [local.namespace, "LogErrorCount", { stat = "Sum", period = 60, label = "Errors/min" }],
          ]
          view = "timeSeries"
        }
      },
      {
        type   = "log"
        x      = 0
        y      = 15
        width  = 24
        height = 6
        properties = {
          title  = "Error Logs"
          region = data.aws_region.current.name
          query  = "SOURCE '${aws_cloudwatch_log_group.app.name}' | fields @timestamp, @message | filter level = 'ERROR' | sort @timestamp desc | limit 50"
          view   = "table"
        }
      },
    ]
  })
}

# --- X-Ray sampling rule ---

resource "aws_xray_sampling_rule" "app" {
  rule_name      = local.app_name
  priority       = 1000
  reservoir_size = 1
  fixed_rate     = 1.0
  version        = 1

  host        = "*"
  http_method = "*"
  url_path    = "*"
  service_name = local.app_name
  service_type = "*"
  resource_arn = "*"

  tags = local.common_tags
}
