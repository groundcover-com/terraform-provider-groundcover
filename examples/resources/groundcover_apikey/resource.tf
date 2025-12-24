# examples/resources/groundcover_apikey/resource.tf

terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover-com/groundcover"
      version = ">= 0.0.0" # Replace with actual version constraint
    }
  }
}

provider "groundcover" {
  # Configure API key and Org Name via environment variables
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

# Example Policy (required for Service Account)
resource "groundcover_policy" "example" {
  name = "terraform-provider-apikey-example-policy"
  role = {
    read = "read"
  }
  description = "Example policy for api key resource example."
}

# Example Service Account (required for API Key)
resource "groundcover_serviceaccount" "example" {
  name = "terraform-provider-apikey-example-sa"
  policy_uuids = [
    groundcover_policy.example.uuid,
  ]
}

# Example API Key
resource "groundcover_apikey" "example" {
  name               = "terraform-provider-example-key"
  description        = "API Key generated via Terraform example."
  service_account_id = groundcover_serviceaccount.example.id

  # Optional: Set an expiration date (RFC3339 format)
  # expiration_date  = "2025-01-01T00:00:00Z"
}

output "apikey_example_id" {
  description = "The ID of the example API Key."
  value       = groundcover_apikey.example.id
}

output "apikey_example_value" {
  description = "The value of the example API Key (sensitive)."
  value       = groundcover_apikey.example.api_key
  sensitive   = true
}

output "apikey_example_creation_date" {
  description = "The creation date of the example API Key."
  value       = groundcover_apikey.example.creation_date
} 