# examples/resources/groundcover_secret/resource.tf

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

# Variables for secret content (should be passed securely, e.g., via environment variables or CI/CD secrets)
variable "external_api_key" {
  type        = string
  description = "External service API key"
  sensitive   = true
}

variable "database_password" {
  type        = string
  description = "Database password"
  sensitive   = true
}

# Example: Create an API key secret
resource "groundcover_secret" "api_key_example" {
  name    = "my-external-api-key"
  type    = "api_key"
  content = var.external_api_key
}

# Example: Create a password secret
resource "groundcover_secret" "password_example" {
  name    = "database-password"
  type    = "password"
  content = var.database_password
}

# The secret ID can be used in other resources like data integrations
# For example:
# resource "groundcover_dataintegration" "example" {
#   type = "prometheus"
#   config = jsonencode({
#     endpoint = "https://prometheus.example.com"
#     password = groundcover_secret.password_example.id  # Use secret reference
#   })
# }

output "api_key_secret_id" {
  description = "The reference ID for the API key secret. Use this in other resources."
  value       = groundcover_secret.api_key_example.id
}

output "password_secret_id" {
  description = "The reference ID for the password secret."
  value       = groundcover_secret.password_example.id
}

