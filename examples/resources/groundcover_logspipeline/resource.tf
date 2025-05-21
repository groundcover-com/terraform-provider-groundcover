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

# Example Logs Pipeline
resource "groundcover_logspipeline" "example" {
  value = <<-EOT
ottlRules:
  - ruleName: extract-kubernetes-metadata
    conditions:
      - resource.attributes["k8s.pod.name"] != nil
    conditionLogicOperator: and
    statements:
      - set(resource.attributes["service.name"], resource.attributes["k8s.pod.name"])
      - set(resource.attributes["service.namespace"], resource.attributes["k8s.namespace.name"])
    statementsErrorMode: ignore

  - ruleName: extract-container-name
    conditions:
      - resource.attributes["container.name"] != nil
    conditionLogicOperator: and
    statements:
      - set(resource.attributes["service.instance.id"], resource.attributes["container.name"]) 
    statementsErrorMode: ignore
EOT
}

output "logs_pipeline_updated_at" {
  description = "The timestamp when the logs pipeline was last updated."
  value       = groundcover_logspipeline.example.updated_at
} 