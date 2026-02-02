terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover-com/groundcover"
      version = ">= 1.5.0"
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
  description = "Slack webhook URL"
  sensitive   = true
}

variable "pagerduty_routing_key" {
  type        = string
  description = "PagerDuty routing key"
  sensitive   = true
}

resource "groundcover_connected_app" "slack" {
  name = "alerts-slack"
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

resource "groundcover_notification_route" "critical_alerts" {
  name  = "critical-alerts-route"
  query = "severity:critical"

  routes = [
    {
      status = ["open"]
      connected_apps = [
        {
          type = "pagerduty"
          id   = groundcover_connected_app.pagerduty.id
        },
        {
          type = "slack-webhook"
          id   = groundcover_connected_app.slack.id
        }
      ]
    },
    {
      status = ["resolved"]
      connected_apps = [
        {
          type = "slack-webhook"
          id   = groundcover_connected_app.slack.id
        }
      ]
    }
  ]

  notification_settings = {
    renotification_interval = "1h"
  }
}

resource "groundcover_notification_route" "all_alerts_to_slack" {
  name  = "all-alerts-slack"
  query = "*"

  routes = [
    {
      status = ["open", "resolved"]
      connected_apps = [
        {
          type = "slack-webhook"
          id   = groundcover_connected_app.slack.id
        }
      ]
    }
  ]

  notification_settings = {
    renotification_interval = "30m"
  }
}

output "critical_route_id" {
  description = "ID of the critical alerts notification route"
  value       = groundcover_notification_route.critical_alerts.id
}

output "all_alerts_route_id" {
  description = "ID of the all alerts notification route"
  value       = groundcover_notification_route.all_alerts_to_slack.id
}
