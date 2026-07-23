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

# Define a groundcover Policy
resource "groundcover_policy" "my_policy" {
  name        = "Example Policy (Terraform)"
  description = "This policy is managed by Terraform."

  # Optional: Specify SSO claim role for mapping
  claim_role = "sso-admin-role"

  # Define policy roles (adjust keys/values as needed for your groundcover setup)
  role = {
    admin = "admin" # key is "read"/"write"/"admin" - value is ignored
  }

  # Optional: Define data scope restrictions
  data_scope = {
    simple = {
      operator = "and" # Can be "and" or "or"
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
        },
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
  }
}

# Define a policy with unrestricted data access.
# An empty data_scope block means no data restrictions — the policy grants access to
# all data (all clusters, namespaces, etc.), exactly like omitting data_scope entirely.
# Useful for tooling that always emits the data_scope block (e.g. the Crossplane
# provider generated from this provider).
resource "groundcover_policy" "admin_all_data" {
  name        = "Admin All Data (Terraform)"
  description = "Full admin access to all data."

  role = {
    admin = "admin"
  }

  data_scope = {}
}

# Output the generated policy UUID
output "policy_uuid" {
  description = "The unique ID (UUID) of the created policy."
  value       = groundcover_policy.my_policy.uuid
}

# Output the policy revision number
output "policy_revision_number" {
  description = "The revision number of the created policy."
  value       = groundcover_policy.my_policy.revision_number
} 