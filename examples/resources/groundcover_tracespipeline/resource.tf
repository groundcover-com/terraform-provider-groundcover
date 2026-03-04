# examples/resources/groundcover_tracespipeline/resource.tf

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

# Example Traces Pipeline
resource "groundcover_tracespipeline" "tracespipeline" {
  value = <<-EOT
ottlRules:
  - ruleName: example-rule
    conditions:
      - workload == "nginx"
    statements:
      - set(attributes["test.key"], "test-value")
EOT
}

output "traces_pipeline_updated_at" {
  description = "The timestamp when the traces pipeline was last updated."
  value       = groundcover_tracespipeline.tracespipeline.updated_at
}
