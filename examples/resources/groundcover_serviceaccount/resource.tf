# examples/resources/groundcover_serviceaccount/resource.tf

terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover/groundcover"
      version = ">= 0.0.0" # Replace with actual version constraint
    }
  }
}

provider "groundcover" {
  # Configure API key and Org Name via environment variables
  # export TF_VAR_groundcover_api_key="YOUR_API_KEY"
  # export TF_VAR_groundcover_org_name="YOUR_ORG_NAME"
  api_key  = var.groundcover_api_key
  org_name = var.groundcover_org_name
  # api_url = "..." # Optional: Override default API URL
}

variable "groundcover_api_key" {
  type        = string
  description = "Groundcover API Key"
  sensitive   = true
}

variable "groundcover_org_name" {
  type        = string
  description = "Groundcover Organization Name"
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