variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
}

variable "route53_zone" {
  description = "Root Route53 hosted zone name (e.g. example.com)"
  type        = string
}

variable "subdomain" {
  description = "Subdomain for the demo (e.g. demo.example.com)"
  type        = string
}

variable "app_image_tag" {
  description = "Docker image tag for the order API"
  type        = string
  default     = "latest"
}

variable "agent_image_tag" {
  description = "Docker image tag for the triage agent Lambda"
  type        = string
  default     = "latest"
}

variable "desired_count" {
  description = "Number of ECS tasks to run"
  type        = number
  default     = 1
}

variable "alarm_threshold" {
  description = "Error rate percentage threshold for the alarm"
  type        = number
  default     = 10
}

variable "lambda_memory" {
  description = "Memory allocation for the triage agent Lambda (MB)"
  type        = number
  default     = 1024
}

variable "lambda_timeout" {
  description = "Timeout for the triage agent Lambda (seconds)"
  type        = number
  default     = 300
}

variable "bedrock_model" {
  description = "Bedrock model ID for the triage agent"
  type        = string
  default     = "eu.anthropic.claude-sonnet-4-6"
}

variable "git_provider" {
  description = "Git provider for issue creation and source file access (github or gitlab)"
  type        = string
  default     = "github"

  validation {
    condition     = contains(["github", "gitlab"], var.git_provider)
    error_message = "git_provider must be either 'github' or 'gitlab'."
  }
}

variable "gitlab_url" {
  description = "GitLab instance base URL (only used when git_provider is gitlab)"
  type        = string
  default     = "https://gitlab.com"
}

variable "git_repo" {
  description = "Git repository for issue creation (GitHub: org/repo, GitLab: numeric project ID or URL-encoded path)"
  type        = string
}
