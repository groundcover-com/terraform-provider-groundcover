---
page_title: "groundcover_notification_route Resource - groundcover"
subcategory: ""
description: |-
  Manages a groundcover Notification Route.
  Notification Routes define how alerts are routed to Connected Apps based on issue queries and statuses. Use gcQL queries to match specific issues and route them to Slack, PagerDuty, or other integrations.
---

# groundcover_notification_route (Resource)

Manages a groundcover Notification Route.

Notification Routes define how alerts are routed to Connected Apps based on issue queries and statuses. Use gcQL queries to match specific issues and route them to Slack, PagerDuty, or other integrations.

## Example Usage

```terraform
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
}

output "critical_route_id" {
  description = "ID of the critical alerts notification route"
  value       = groundcover_notification_route.critical_alerts.id
}

output "all_alerts_route_id" {
  description = "ID of the all alerts notification route"
  value       = groundcover_notification_route.all_alerts_to_slack.id
}
```

## Schema

### Required

- `name` (String) Name of the notification route.
- `query` (String) gcQL query to match issues. Use `*` to match all issues.
- `routes` (List of Object) List of routing rules that define which connected apps receive notifications based on issue status.

### Optional

- `notification_settings` (Object) Optional notification settings for this route.
  - `renotification_interval` (String) Duration between renotifications (e.g., `1h`, `30m`).

### Read-Only

- `id` (String) The unique identifier for the notification route.
- `created_by` (String) The user who created the notification route.
- `created_at` (String) The date the notification route was created (RFC3339 format).
- `modified_by` (String) The user who last modified the notification route.
- `modified_at` (String) The date the notification route was last modified (RFC3339 format).

### Nested Schema for `routes`

Required:

- `status` (List of String) List of issue statuses that trigger this route. Valid values: `open`, `resolved`.
- `connected_apps` (List of Object) List of connected apps to notify for this route.

### Nested Schema for `routes.connected_apps`

Required:

- `type` (String) Type of connected app (e.g., `slack-webhook`, `pagerduty`).
- `id` (String) ID of the connected app.

## Import

Notification Routes can be imported using their ID:

```shell
terraform import groundcover_notification_route.example <notification-route-id>
```
