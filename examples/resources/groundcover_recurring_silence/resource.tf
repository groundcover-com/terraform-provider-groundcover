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
  type      = string
  sensitive = true
}

variable "groundcover_backend_id" {
  type = string
}

# Daily: silence a nightly maintenance window every day.
resource "groundcover_recurring_silence" "nightly_maintenance" {
  recurrence_type = "daily"
  timezone        = "UTC"
  comment         = "Nightly maintenance window"

  timeframes = [
    { day = "every_day", start_time = "03:00", end_time = "03:30" },
  ]

  matchers = [
    {
      name  = "env"
      value = "production"
    },
  ]
}

# Weekly: silence business-hours alerts on Wednesday and Thursday.
resource "groundcover_recurring_silence" "weekly_window" {
  recurrence_type = "weekly"
  timezone        = "America/New_York"

  timeframes = [
    { day = "wednesday", start_time = "09:00", end_time = "11:00" },
    { day = "thursday", start_time = "09:00", end_time = "11:00" },
  ]

  matchers = [
    {
      name  = "service"
      value = "checkout"
    },
  ]
}

# Monthly: silence the first and fifteenth of each month.
resource "groundcover_recurring_silence" "monthly_billing" {
  recurrence_type = "monthly"
  timezone        = "UTC"
  enabled         = true

  timeframes = [
    { day = "1", start_time = "00:00", end_time = "02:00" },
    { day = "15", start_time = "00:00", end_time = "02:00" },
  ]

  matchers = [
    {
      name        = "job"
      value       = "billing"
      is_equal    = true
      is_contains = false
    },
  ]
}
