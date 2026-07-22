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
      status = ["Alerting"]
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
      status = ["Resolved"]
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
      status = ["Alerting", "Resolved"]
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

# slack-app and linear connected apps are installed through the groundcover UI
# (OAuth flow), so they are referenced by ID rather than created in Terraform.
variable "slack_app_id" {
  type        = string
  description = "ID of an existing slack-app connected app"
}

variable "linear_app_id" {
  type        = string
  description = "ID of an existing linear connected app"
}

resource "groundcover_notification_route" "slack_app_and_linear" {
  name  = "critical-alerts-slack-app-linear"
  query = "severity:critical"

  routes = [
    {
      status = ["Alerting", "Resolved"]
      connected_apps = [
        {
          # slack-app routes require params.channels
          type = "slack-app"
          id   = var.slack_app_id
          params = {
            channels = [
              {
                id   = "C0123456789"
                name = "#alerts"
              }
            ]
          }
        },
        {
          # linear routes require params.team_id, and resolved_status_id
          # unless auto_resolve is set to false
          type = "linear"
          id   = var.linear_app_id
          params = {
            team_id            = "d1b2c3d4-team"
            project_id         = "d1b2c3d4-project"
            label_ids          = ["d1b2c3d4-label"]
            resolved_status_id = "d1b2c3d4-status"
            auto_resolve       = true
          }
        }
      ]
    }
  ]
}

output "critical_route_id" {
  description = "ID of the critical alerts notification route"
  value       = groundcover_notification_route.critical_alerts.id
}

output "all_alerts_route_id" {
  description = "ID of the all alerts notification route"
  value       = groundcover_notification_route.all_alerts_to_slack.id
}
