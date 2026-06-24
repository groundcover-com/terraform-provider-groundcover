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

# groundcover_connected_app_json is identical to groundcover_connected_app, except `data`
# is a JSON string (via jsonencode) instead of an HCL object. Prefer it when the config is
# generated/templated or consumed by tooling that can't model dynamic objects (e.g. the
# Crossplane provider). For hand-written HCL, groundcover_connected_app is usually nicer.
resource "groundcover_connected_app_json" "slack" {
  name = "alerts-slack-channel"
  type = "slack-webhook"
  data = jsonencode({
    url = var.slack_webhook_url
  })
}

output "slack_app_id" {
  description = "ID of the Slack connected app"
  value       = groundcover_connected_app_json.slack.id
}
