# examples/resources/groundcover_serviceaccount/resource.tf

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
  name = "terraform-provider-sa-example-policy"
  role = {
    read = "read" # Grant read-only role
  }
  description = "Example policy for service account resource example."
}

# Example Service Account
resource "groundcover_serviceaccount" "example" {
  name        = "terraform-provider-example-sa"
  email       = "terraform-sa@example.com" # Optional: specify an email
  description = "Service account created via Terraform example."

  policy_uuids = [
    groundcover_policy.example.uuid, # Link to the example policy
  ]
}

output "serviceaccount_example_id" {
  description = "The ID of the example Service Account."
  value       = groundcover_serviceaccount.example.id
}

output "serviceaccount_example_email" {
  description = "The email of the example Service Account."
  value       = groundcover_serviceaccount.example.email
} 