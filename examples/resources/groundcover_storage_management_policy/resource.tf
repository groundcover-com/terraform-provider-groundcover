terraform {
  required_providers {
    groundcover = {
      source = "groundcover-com/groundcover"
    }
  }
}

# Storage management policies are seeded by groundcover per data type and can
# only be updated — never created or deleted. This resource adopts the existing
# policy for a data type and manages its retention configuration.
resource "groundcover_storage_management_policy" "logs" {
  data_type = "logs"
  retention = "30d"

  cold_move_duration = "7d"

  custom_rules = [
    {
      name      = "debug-logs"
      retention = "3d"
      filters   = "level = 'debug'"
    },
  ]
}

# The API returns the full policy. `version`, `uuid`, and `created_timestamp`
# are computed (managed by groundcover) and available as read-only outputs.
# Example values after apply:
#   version           = 4
#   uuid              = "3f1c8a2e-9b7d-4e6a-8c11-0a2b3c4d5e6f"
#   created_timestamp = "2026-07-22T09:14:03.000Z"
output "logs_policy" {
  value = {
    version           = groundcover_storage_management_policy.logs.version
    uuid              = groundcover_storage_management_policy.logs.uuid
    created_timestamp = groundcover_storage_management_policy.logs.created_timestamp
  }
}
