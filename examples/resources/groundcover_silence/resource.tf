# examples/resources/groundcover_silence/resource.tf

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
  description = "groundcover API Key"
  sensitive   = true
}

variable "groundcover_backend_id" {
  type        = string
  description = "groundcover Backend ID"
}

# Example 1: Simple silence for planned maintenance
# Silences all alerts for a specific service during a maintenance window
resource "groundcover_silence" "maintenance_window" {
  starts_at = "2030-01-15T10:00:00Z"
  ends_at   = "2030-01-15T14:00:00Z"
  comment   = "Planned maintenance window for payment-service"

  matchers = [
    {
      name        = "service"
      value       = "payment-service"
      is_equal    = true
      is_contains = false
    }
  ]
}

# Example 2: Silence with multiple matchers
# Silences alerts for a specific workload in a specific environment
resource "groundcover_silence" "deployment_silence" {
  starts_at = "2030-01-20T08:00:00Z"
  ends_at   = "2030-01-20T10:00:00Z"
  comment   = "Deploying new version of api-gateway in staging"

  matchers = [
    {
      name        = "workload"
      value       = "api-gateway"
      is_equal    = true
      is_contains = false
    },
    {
      name        = "environment"
      value       = "staging"
      is_equal    = true
      is_contains = false
    }
  ]
}

# Example 3: Silence with contains pattern (starts_at and ends_at are UTC 0)
# Silences alerts for any service containing the specified value
resource "groundcover_silence" "test_services_silence" {
  starts_at = "2030-02-01T00:00:00Z"
  ends_at   = "2030-02-01T06:00:00Z"
  comment   = "Overnight testing - silence all test services"

  matchers = [
    {
      name        = "service"
      value       = "app-dev"
      is_equal    = true
      is_contains = true
    }
  ]
}

# Example 4: Silence with negation (is_equal = false)
# Silences all alerts EXCEPT for production environment
resource "groundcover_silence" "non_production_silence" {
  starts_at = "2030-03-01T12:00:00Z"
  ends_at   = "2030-03-01T18:00:00Z"
  comment   = "Silence non-production alerts during load testing"

  matchers = [
    {
      name        = "environment"
      value       = "production"
      is_equal    = false # Match everything EXCEPT production
      is_contains = false
    }
  ]
}

output "maintenance_silence_id" {
  description = "The ID of the maintenance window silence"
  value       = groundcover_silence.maintenance_window.id
}

output "deployment_silence_id" {
  description = "The ID of the deployment silence"
  value       = groundcover_silence.deployment_silence.id
}
