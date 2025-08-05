# examples/resources/groundcover_logspipeline/resource.tf

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
  api_key  = var.groundcover_api_key
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

# Example Logs Pipeline
resource "groundcover_logspipeline" "logspipeline" {
  value = <<-EOT
ottlRules:
  - ruleName: example-rule
    conditions:
      - container_name == "nginx"
    statements:
      - set(attributes["test.key"], "test-value")
EOT
}

output "logs_pipeline_updated_at" {
  description = "The timestamp when the logs pipeline was last updated."
  value       = groundcover_logspipeline.logspipeline.updated_at
} 