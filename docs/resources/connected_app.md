---
page_title: "groundcover_connected_app Resource - groundcover"
subcategory: ""
description: |-
  Manages a groundcover Connected App.
  Connected Apps are integrations with external services (Slack, PagerDuty) that can receive notifications from groundcover. Use them with Notification Routes to route alerts to the appropriate channels.
---

# groundcover_connected_app (Resource)

Manages a groundcover Connected App.

Connected Apps are integrations with external services (Slack, PagerDuty) that can receive notifications from groundcover. Use them with Notification Routes to route alerts to the appropriate channels.

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
```

## Schema

### Required

- `name` (String) Name of the connected app.
- `type` (String) Type of connected app. Valid values: `slack-webhook`, `pagerduty`.
- `data` (Dynamic, Sensitive) Type-specific configuration. Supports nested structures.
  - For `slack-webhook`: `{ url = "https://hooks.slack.com/..." }`
  - For `pagerduty`: `{ routing_key = "...", severity_mapping = { critical = "critical", ... } }`

### Read-Only

- `id` (String) The unique identifier for the connected app.
- `created_by` (String) The user who created the connected app.
- `created_at` (String) The date the connected app was created (RFC3339 format).
- `updated_by` (String) The user who last updated the connected app.
- `updated_at` (String) The date the connected app was last updated (RFC3339 format).

## Import

Connected Apps can be imported using their ID:

```shell
terraform import groundcover_connected_app.example <connected-app-id>
```

**Note:** The `data` attribute is sensitive and will not be populated on import.
