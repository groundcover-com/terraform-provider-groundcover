# examples/resources/groundcover_recurring_silence/resource.tf

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

# Example 1: Daily recurring silence for nightly maintenance
# Silences alerts every day from 2 AM to 4 AM UTC
resource "groundcover_recurring_silence" "nightly_maintenance" {
  comment         = "Nightly maintenance window for payment-service"
  recurrence_type = "daily"
  start_time      = "02:00"
  end_time        = "04:00"
  timezone        = "UTC"

  matchers = [
    {
      name        = "service"
      value       = "payment-service"
      is_equal    = true
      is_contains = false
    }
  ]
}

# Example 2: Weekly recurring silence for deployment windows
# Silences alerts every Monday and Wednesday from 9 AM to 11 AM Eastern
resource "groundcover_recurring_silence" "deployment_window" {
  comment         = "Scheduled deployment window"
  recurrence_type = "weekly"
  start_time      = "09:00"
  end_time        = "11:00"
  timezone        = "America/New_York"
  days_of_week    = [1, 3] # Monday, Wednesday

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

# Example 3: Monthly recurring silence with contains pattern
# Silences alerts on the 1st and 15th of each month from 10 PM to 6 AM (overnight)
resource "groundcover_recurring_silence" "patch_day" {
  comment         = "Monthly patch day - overnight maintenance"
  recurrence_type = "monthly"
  start_time      = "22:00"
  end_time        = "06:00"
  timezone        = "Europe/London"
  days_of_month   = [1, 15]

  matchers = [
    {
      name        = "service"
      value       = "app-dev"
      is_equal    = true
      is_contains = true
    }
  ]
}

# Example 4: Disabled recurring silence with negation (is_equal = false)
# Silences all alerts EXCEPT for production environment
resource "groundcover_recurring_silence" "weekend_silence" {
  comment         = "Weekend silence - currently disabled"
  recurrence_type = "weekly"
  start_time      = "00:00"
  end_time        = "23:59"
  timezone        = "UTC"
  days_of_week    = [0, 6] # Sunday, Saturday
  enabled         = false

  matchers = [
    {
      name        = "environment"
      value       = "production"
      is_equal    = false # Match everything EXCEPT production
      is_contains = false
    }
  ]
}

output "nightly_maintenance_id" {
  description = "The ID of the nightly maintenance recurring silence"
  value       = groundcover_recurring_silence.nightly_maintenance.id
}

output "deployment_window_id" {
  description = "The ID of the deployment window recurring silence"
  value       = groundcover_recurring_silence.deployment_window.id
}
