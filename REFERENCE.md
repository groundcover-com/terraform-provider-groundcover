## Resource Reference

### `groundcover_policy`

Manages an RBAC policy within groundcover.

#### Example Usage

```hcl
resource "groundcover_policy" "my_policy" {
  name        = "Example Policy (Terraform)"
  description = "This policy is managed by Terraform."
  claim_role  = "sso-admin-role"

  role = {
    admin = "admin"
  }

  data_scope = {
    simple = {
      operator = "and"
      conditions = [
        {
          key    = "k8s.cluster.name"
          origin = "deployment"
          type   = "string"
          filters = [
            {
              op    = "match"
              value = "production-cluster"
            }
          ]
        },
      ]
    }
  }
}
```

#### Arguments

*   `name` (String, Required): The name of the policy.
*   `role` (Map of String, Required): Role definitions associated with the policy. Currently, only a single key/value pair is supported, where the key must be one of `read`, `write`, or `admin`, and the value is ignored.
*   `description` (String, Optional): A description for the policy.
*   `claim_role` (String, Optional): SSO Role claim name used for mapping.
*   `data_scope` (Block, Optional): Defines the data scope restrictions for the policy. Currently supports a `simple` block with the following nested arguments:
    *   `simple` (Block, Required if `data_scope` is present):
        *   `operator` (String, Required): Logical operator (`and` or `or`).
        *   `conditions` (List of Blocks, Required): List of conditions for the data scope.
            *   `key` (String, Required): The key for the condition (e.g., `k8s.cluster.name`).
            *   `origin` (String, Required): The origin of the key.
            *   `type` (String, Required): The type of the key.
            *   `filters` (List of Blocks, Required): Filter criteria for the condition.
                *   `op` (String, Required): The filter operation (e.g., `match`).
                *   `value` (String, Required): The value to filter on.

#### Attributes

*   `uuid` (String): The unique identifier (UUID) of the policy.
*   `revision_number` (Number): Revision number of the policy, used for concurrency control.
*   `read_only` (Boolean): Indicates if the policy is read-only (managed internally).

### `groundcover_serviceaccount`

Manages a Groundcover Service Account.

#### Example Usage

```hcl
resource "groundcover_serviceaccount" "my_sa" {
  name        = "My Terraform Service Account"
  email       = "tf-sa@example.com"
  description = "Service account managed by Terraform"
  policy_uuids = [
    groundcover_policy.my_policy.uuid,
    # Add other policy UUIDs here
  ]
}
```

#### Arguments

*   `name` (String, Required): The name of the service account.
*   `email` (String, Required): The email associated with the service account.
*   `policy_uuids` (List of String, Required): List of policy UUIDs to assign to the service account.
*   `description` (String, Optional): An optional description for the service account.

#### Attributes

*   `id` (String): The unique identifier for the service account.

### `groundcover_apikey`

Manages an API Key for a Groundcover Service Account.

#### Example Usage

```hcl
resource "groundcover_apikey" "my_key" {
  name               = "My Terraform API Key"
  service_account_id = groundcover_serviceaccount.my_sa.id
  description        = "API Key for Terraform automation"
  // expiration_date = "2024-12-31T23:59:59Z" # Optional: RFC3339 format
}

output "api_key_secret" {
  value     = groundcover_apikey.my_key.api_key
  sensitive = true
}
```

#### Arguments

*   `name` (String, Required): The name of the API key.
*   `service_account_id` (String, Required): The ID of the service account associated with the API key.
*   `description` (String, Optional): A description for the API key.
*   `expiration_date` (String, Optional): The expiration date for the API key (RFC3339 format). If not set, the key never expires.

#### Attributes

*   `id` (String): The unique identifier for the API key.
*   `api_key` (String, Sensitive): The generated API key. This value is only available upon creation.
*   `created_by` (String): The user who created the API key.
*   `creation_date` (String): The date the API key was created (RFC3339 format).
*   `last_active` (String): The last time the API key was active (RFC3339 format).
*   `revoked_at` (String): The date the API key was revoked (RFC3339 format), if applicable.
*   `expired_at` (String): The date the API key expired (RFC3339 format), based on the `expiration_date` set.
*   `policies` (List of Objects): Policies associated with the service account linked to this API key.
    *   `uuid` (String): Policy UUID.
    *   `name` (String): Policy name.

### `groundcover_monitor`

Manages a Groundcover Monitor resource using raw YAML.

#### Example Usage

```hcl
resource "groundcover_monitor" "my_monitor" {
  monitor_yaml = <<-EOT
    apiVersion: groundcover.com/v1
    kind: Monitor
    metadata:
      name: my-custom-monitor
    spec:
      type: log
      query: "level:error"
      conditions: # Example: Trigger if more than 5 errors in 5 minutes
        - type: threshold
          threshold: 5
          window: 5m
      alert:
        title: "High Error Rate Detected"
        description: "More than 5 errors detected in the last 5 minutes."
        severity: critical
        # notificationTarget: my-slack-channel # Optional
  EOT
}
```

#### Arguments

*   `monitor_yaml` (String, Required): The monitor definition in YAML format.

#### Attributes

*   `id` (String): Monitor identifier (UUID). 