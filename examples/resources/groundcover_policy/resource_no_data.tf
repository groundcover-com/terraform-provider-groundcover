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

# Define a groundcover Policy with No Data Access
# This corresponds to the "No data" option in the UI
resource "groundcover_policy" "no_data_policy" {
  name        = "No Data Policy Example (Terraform)"
  description = "This policy grants no data access by setting disabled to true."

  # Optional: Specify SSO claim role for mapping
  claim_role = "sso-no-data-role"

  # Define policy roles (adjust keys/values as needed for your groundcover setup)
  role = {
    read = "read" # key is "read"/"write"/"admin" - value is ignored
  }

  # No data access: Use simple data scope with disabled = true
  data_scope = {
    simple = {
      operator   = "and"
      disabled   = true
      conditions = []
    }
  }
}

# Output the generated policy UUID
output "no_data_policy_uuid" {
  description = "The unique ID (UUID) of the created no-data policy."
  value       = groundcover_policy.no_data_policy.uuid
}

# Output the policy revision number
output "no_data_policy_revision_number" {
  description = "The revision number of the created no-data policy."
  value       = groundcover_policy.no_data_policy.revision_number
}
