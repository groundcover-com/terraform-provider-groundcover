# examples/resources/groundcover_dashboard/resource.tf

terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover-com/groundcover"
      version = ">= 1.0.0"
    }
  }
}

provider "groundcover" {
  # Configure API key and Backend ID via environment variables
  # export GROUNDCOVER_API_KEY="YOUR_API_KEY"
  # export GROUNDCOVER_BACKEND_ID="YOUR_BACKEND_ID"
  api_key    = var.groundcover_api_key
  backend_id = var.groundcover_backend_id
  # api_url = "..." # Optional: Override default API URL
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

# Example Dashboard: Basic metrics dashboard
resource "groundcover_dashboard" "metrics_dashboard" {
  name        = "Terraform Example - Metrics Dashboard"
  description = "Example dashboard showing system metrics"
  team        = "platform"

  # Dashboard preset contains the JSON configuration
  preset = jsonencode({
    duration = "Last 1 hour"
    layout = [
      {
        id   = "A"
        x    = 0
        y    = 0
        w    = 12
        h    = 4
        minH = 2
      },
      {
        id   = "B"
        x    = 0
        y    = 4
        w    = 6
        h    = 4
        minH = 2
      },
      {
        id   = "C"
        x    = 6
        y    = 4
        w    = 6
        h    = 4
        minH = 2
      }
    ]
    widgets = [
      {
        id   = "A"
        type = "widget"
        name = "CPU Usage"
        queries = [
          {
            id         = "A"
            expr       = "avg(rate(container_cpu_usage_seconds_total[5m])) * 100"
            dataType   = "metrics"
            step       = null
            editorMode = "builder"
          }
        ]
        visualizationConfig = {
          type = "time-series"
        }
      },
      {
        id   = "B"
        type = "widget"
        name = "Memory Usage"
        queries = [
          {
            id         = "B"
            expr       = "avg(container_memory_usage_bytes)"
            dataType   = "metrics"
            step       = null
            editorMode = "builder"
          }
        ]
        visualizationConfig = {
          type = "gauge"
        }
      },
      {
        id   = "C"
        type = "text"
        html = "<h3>System Metrics</h3><p>This dashboard shows key system metrics including CPU and memory usage.</p>"
      }
    ]
    variables     = {}
    schemaVersion = 3
  })

}

# Example Dashboard: Simple monitoring dashboard
resource "groundcover_dashboard" "simple_dashboard" {
  name = "Simple Dashboard"

  preset = jsonencode({
    duration = "Last 30 minutes"
    layout = [
      {
        id   = "widget1"
        x    = 0
        y    = 0
        w    = 12
        h    = 6
        minH = 3
      }
    ]
    widgets = [
      {
        id   = "widget1"
        type = "widget"
        name = "Request Rate"
        queries = [
          {
            id         = "query1"
            expr       = "sum(rate(http_requests_total[5m]))"
            dataType   = "metrics"
            step       = null
            editorMode = "code"
          }
        ]
        visualizationConfig = {
          type = "time-series"
          config = {
            showLegend = true
            unit       = "ops"
          }
        }
      }
    ]
    variables     = {}
    schemaVersion = 3
  })
}

output "metrics_dashboard_id" {
  description = "The UUID of the metrics dashboard"
  value       = groundcover_dashboard.metrics_dashboard.id
}

output "metrics_dashboard_owner" {
  description = "The owner of the metrics dashboard"
  value       = groundcover_dashboard.metrics_dashboard.owner
}

output "simple_dashboard_id" {
  description = "The UUID of the simple dashboard"
  value       = groundcover_dashboard.simple_dashboard.id
}

output "simple_dashboard_owner" {
  description = "The owner of the simple dashboard"
  value       = groundcover_dashboard.simple_dashboard.owner
}
