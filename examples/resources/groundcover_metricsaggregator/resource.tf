# examples/resources/groundcover_metricsaggregator/resource.tf

terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover-com/groundcover"
      version = ">= 0.0.0" # Replace with actual version constraint
    }
  }
}

provider "groundcover" {
  # Configure API key and Backend ID via environment variables
  # export TF_VAR_groundcover_api_key="YOUR_API_KEY"
  # export TF_VAR_groundcover_backend_id="YOUR_BACKEND_ID"
  api_key    = var.groundcover_api_key
  backend_id = var.groundcover_backend_id
  # api_url = "..." # Optional: Override default API URL
}

variable "groundcover_api_key" {
  type        = string
  description = "Groundcover API Key"
  sensitive   = true
}

variable "groundcover_backend_id" {
  type        = string
  description = "Groundcover Backend ID"
}

# Example Metrics Aggregator
# This resource configures metrics aggregation rules for reducing cardinality
# and improving query performance.
resource "groundcover_metricsaggregator" "metricsaggregator" {
  value = <<-EOT
- ignore_old_samples: true
  match: '{__name__=~"http_requests_total"}'
  without: [instance, pod]
  interval: 60s
  outputs: [total_prometheus]
- ignore_old_samples: true
  match: '{__name__=~"process_cpu_seconds_total"}'
  without: [instance]
  interval: 30s
  outputs: [total_prometheus]
EOT
}

output "metrics_aggregator_updated_at" {
  description = "The timestamp when the metrics aggregator was last updated."
  value       = groundcover_metricsaggregator.metricsaggregator.updated_at
}
