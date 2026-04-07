# --- ECS execution role ---

resource "aws_iam_role" "ecs_execution" {
  name = "${local.project}-ecs-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "ecs_execution" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# --- ECS task role ---

resource "aws_iam_role" "ecs_task" {
  name = "${local.project}-ecs-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "ecs_task" {
  name = "${local.project}-ecs-task"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "DynamoDB"
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:Query",
          "dynamodb:Scan",
        ]
        Resource = [
          aws_dynamodb_table.orders.arn,
          "${aws_dynamodb_table.orders.arn}/index/*",
        ]
      },
      {
        Sid    = "XRay"
        Effect = "Allow"
        Action = [
          "xray:PutTraceSegments",
          "xray:PutTelemetryRecords",
          "xray:GetSamplingRules",
          "xray:GetSamplingTargets",
        ]
        Resource = "*"
      },
      {
        Sid    = "DynamoDBAgentEvents"
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:Query",
        ]
        Resource = aws_dynamodb_table.agent_events.arn
      },
    ]
  })
}

# --- Lambda role ---

resource "aws_iam_role" "lambda" {
  name = "${local.project}-lambda"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "lambda" {
  name = "${local.project}-lambda"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "CloudWatchLogsRead"
        Effect = "Allow"
        Action = [
          "logs:StartQuery",
          "logs:GetQueryResults",
        ]
        Resource = "${aws_cloudwatch_log_group.app.arn}:*"
      },
      {
        Sid    = "CloudWatchLogsWrite"
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = "${aws_cloudwatch_log_group.lambda.arn}:*"
      },
      {
        Sid    = "CloudWatchAlarms"
        Effect = "Allow"
        Action = [
          "cloudwatch:DescribeAlarms",
          "cloudwatch:GetMetricData",
        ]
        Resource = "*"
      },
      {
        Sid    = "XRayRead"
        Effect = "Allow"
        Action = [
          "xray:GetTraceSummaries",
          "xray:BatchGetTraces",
        ]
        Resource = "*"
      },
      {
        Sid    = "Bedrock"
        Effect = "Allow"
        Action = [
          "bedrock:InvokeModel",
          "bedrock:InvokeModelWithResponseStream",
        ]
        Resource = [
          "arn:aws:bedrock:${data.aws_region.current.name}::foundation-model/*",
          "arn:aws:bedrock:eu-*::foundation-model/*",
          "arn:aws:bedrock:eu-west-2:${data.aws_caller_identity.current.account_id}:inference-profile/*",
        ]
      },
      {
        Sid    = "DynamoDBAgentEvents"
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
        ]
        Resource = aws_dynamodb_table.agent_events.arn
      },
      {
        Sid    = "SecretsManager"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
        ]
        Resource = aws_secretsmanager_secret.git_pat.arn
      },
    ]
  })
}
