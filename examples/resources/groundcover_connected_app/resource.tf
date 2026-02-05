terraform {
  required_providers {
    groundcover = {
      source = "registry.terraform.io/groundcover-com/groundcover"
    }
  }
}

provider "groundcover" {
  api_key    = var.groundcover_api_key
  backend_id = var.groundcover_backend_id
}

variable "groundcover_api_key" {
  type        = string
  description = "groundcover API Key"
  sensitive   = true
}

variable "groundcover_backend_id" {
  type        = string
  description = "groundcover Backend ID"
}

variable "slack_webhook_url" {
  type        = string
  description = "Slack webhook URL for notifications"
  sensitive   = true
}

variable "pagerduty_routing_key" {
  type        = string
  description = "PagerDuty routing key (32 characters)"
  sensitive   = true
}

resource "groundcover_connected_app" "slack" {
  name = "alerts-slack-channel"
  type = "slack-webhook"
  data = {
    url = var.slack_webhook_url
  }
}

resource "groundcover_connected_app" "pagerduty" {
  name = "oncall-pagerduty"
  type = "pagerduty"
  data = {
    routing_key = var.pagerduty_routing_key
  }
}

resource "groundcover_connected_app" "pagerduty_with_severity" {
  name = "oncall-pagerduty-mapped"
  type = "pagerduty"
  data = {
    routing_key = var.pagerduty_routing_key
    severity_mapping = {
      critical = "critical"
      error    = "error"
      warning  = "warning"
      info     = "info"
    }
  }
}

output "slack_app_id" {
  description = "ID of the Slack connected app"
  value       = groundcover_connected_app.slack.id
}

output "pagerduty_app_id" {
  description = "ID of the PagerDuty connected app"
  value       = groundcover_connected_app.pagerduty.id
}
