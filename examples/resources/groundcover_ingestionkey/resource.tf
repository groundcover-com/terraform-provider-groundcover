# examples/resources/groundcover_ingestionkey/resource.tf

terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover-com/groundcover"
      version = ">= 0.6.0" # Replace with actual version constraint
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

# Example Ingestion Key
resource "groundcover_ingestionkey" "example" {
  name = "terraform-provider-example-ingestion-key"
  type = "ingestion"
  
  # Optional: Enable remote configuration
  remote_config = true
  
  # Optional: Add tags
  tags = ["terraform", "example", "ingestion"]
}

# Example Ingestion Key with minimal configuration
resource "groundcover_ingestionkey" "minimal" {
  name = "terraform-provider-minimal-key"
  type = "ingestion"
}

output "ingestionkey_example_key" {
  description = "The key value of the example ingestion key (sensitive)."
  value       = groundcover_ingestionkey.example.key
  sensitive   = true
}

output "ingestionkey_example_creation_date" {
  description = "The creation date of the example ingestion key."
  value       = groundcover_ingestionkey.example.creation_date
}

output "ingestionkey_example_created_by" {
  description = "The user who created the example ingestion key."
  value       = groundcover_ingestionkey.example.created_by
}

output "ingestionkey_minimal_key" {
  description = "The key value of the minimal ingestion key (sensitive)."
  value       = groundcover_ingestionkey.minimal.key
  sensitive   = true
} 