# Configure the groundcover Provider
provider "groundcover" {
  # Configure API key using environment variable TF_VAR_groundcover_api_key is recommended
  # api_key = var.groundcover_api_key
  # Or uncomment and set directly (less secure):
  # api_key = "YOUR_API_KEY_HERE"

  backend_id = "groundcover" # your backend ID as provided in the installation
  # Optionally override the base URL (defaults to api.groundcover.com)
  # api_url = "https://api.your-instance.groundcover.com"
}

# Optional: Define variable for API key (recommended)
# variable "groundcover_api_key" {
#   type        = string
#   description = "groundcover API Key"
#   sensitive   = true
#   nullable    = false # Ensure it's provided (e.g., via TF_VAR_...)
# }

# Define a groundcover Policy with Advanced Data Scope
# Advanced data scope allows per-data-type filtering for fine-grained access control
resource "groundcover_policy" "advanced_policy" {
  name        = "Advanced Policy Example (Terraform)"
  description = "This policy demonstrates advanced data scope with per-data-type filtering."

  # Optional: Specify SSO claim role for mapping
  claim_role = "sso-developer-role"

  # Define policy roles (adjust keys/values as needed for your groundcover setup)
  role = {
    read = "read" # key is "read"/"write"/"admin" - value is ignored
  }

  # Advanced data scope: Define different filtering rules for each data type
  data_scope = {
    advanced = {
      # Logs: Restrict to specific namespaces
      logs = {
        operator = "or"
        conditions = [
          {
            key    = "namespace"
            origin = "root"
            type   = "string"
            filters = [
              {
                op    = "match"
                value = "app-backend"
              }
            ]
          },
          {
            key    = "namespace"
            origin = "root"
            type   = "string"
            filters = [
              {
                op    = "match"
                value = "app-frontend"
              }
            ]
          }
        ]
      }

      # Metrics: Restrict to production cluster
      metrics = {
        operator = "and"
        conditions = [
          {
            key    = "cluster"
            origin = "root"
            type   = "string"
            filters = [
              {
                op    = "match"
                value = "production-cluster"
              }
            ]
          }
        ]
      }

      # Traces: Restrict to specific service
      traces = {
        operator = "and"
        conditions = [
          {
            key    = "service"
            origin = "root"
            type   = "string"
            filters = [
              {
                op    = "match"
                value = "payment-service"
              }
            ]
          }
        ]
      }

      # Events: Restrict to non-system namespaces
      events = {
        operator = "and"
        conditions = [
          {
            key    = "namespace"
            origin = "root"
            type   = "string"
            filters = [
              {
                op    = "not_match"
                value = "kube-system"
              }
            ]
          }
        ]
      }

      # Workloads: Allow all workloads in development environments
      workloads = {
        operator = "or"
        conditions = [
          {
            key    = "environment"
            origin = "root"
            type   = "string"
            filters = [
              {
                op    = "match"
                value = "dev"
              }
            ]
          },
          {
            key    = "environment"
            origin = "root"
            type   = "string"
            filters = [
              {
                op    = "match"
                value = "staging"
              }
            ]
          }
        ]
      }
    }
  }
}

# Output the generated policy UUID
output "advanced_policy_uuid" {
  description = "The unique ID (UUID) of the created advanced policy."
  value       = groundcover_policy.advanced_policy.uuid
}

# Output the policy revision number
output "advanced_policy_revision_number" {
  description = "The revision number of the created advanced policy."
  value       = groundcover_policy.advanced_policy.revision_number
}
