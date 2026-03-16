terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    # Bucket, region, and profile configured via -backend-config flags in justfile.
    key = "demo-incident-response/terraform.tfstate"
  }
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

data "aws_route53_zone" "parent" {
  name = var.route53_zone
}

locals {
  project        = "demo-incident-response"
  app_name       = "demo-order-api"
  agent_name     = "demo-triage-agent"
  table_name     = "demo-orders"
  status_index   = "status-index"
  namespace      = "DemoOrderAPI"
  container_port = 8080
  domain         = var.subdomain
  app_domain     = "orders.${local.domain}"
  azs            = ["${var.aws_region}a", "${var.aws_region}b"]

  common_tags = {
    Project   = local.project
    ManagedBy = "terraform"
  }
}
