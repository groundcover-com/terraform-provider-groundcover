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

> **Deprecated:** use [`groundcover_monitor_v2`](#groundcover_monitor_v2) instead, which provides a typed schema in place of the raw YAML blob. This resource continues to work for backward compatibility.

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

### `groundcover_monitor_v2`

Manages a groundcover Monitor resource with a typed schema instead of raw YAML. Supports GCQL, MetricsQL, and raw SQL query definitions.

#### Example Usage

The full example file includes GCQL examples for logs, traces, events, and APM, plus MetricsQL and raw SQL examples.

```hcl
resource "groundcover_monitor_v2" "gcql_logs" {
  title            = "GCQL Logs Error Count"
  severity         = "critical"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = "GCQL Logs Error Count"
    description = "Fires when error logs are observed in the evaluation window."
  }

  query {
    type           = "gcql"
    data_type      = "logs"
    expression     = "level:error | stats count() count_all_result"
    instant_rollup = "5m"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [10]
  }

  notification_settings {
    method         = "connectedApps"
    connected_apps = ["slack-connected-app-id"]

    connected_app_params = {
      "slack-connected-app-id" = {
        channels = [{ id = "C0123456789", name = "#alerts" }]
      }
    }
  }
}
```

#### Arguments

*   `title` (String, Required): Monitor title.
*   `severity` (String, Required): Monitor severity.
*   `measurement_type` (String, Required): `state` or `event`.
*   `query` (Block, Required): Typed query definition. Supports `type = "gcql"`, `type = "metricsql"`, and `type = "raw_sql"`. GCQL supports `data_type` values `logs`, `traces`, `events`, and `apm`.
*   `threshold` (Block, Required): One or more threshold definitions. Supports optional `custom_resolve_threshold`.
*   `notification_settings` (Block, Optional): Notification behavior, including `connected_app_params` for per-app Slack channels.
*   `display`, `evaluation_interval`, `reducer`, `labels`, `annotations`, and `routing` are optional monitor settings.

#### Attributes

*   `id` (String): Monitor identifier (UUID).

### `groundcover_skill`

Manages an organizational groundcover Agent Skill. Managing organizational Skills requires an admin service account.

#### Example Usage

```hcl
resource "groundcover_skill" "incident_response" {
  name         = "incident-response"
  description  = "A repeatable workflow for investigating production incidents."
  when_to_use  = "Use when investigating an active production incident or responding to an alert."
  instructions = "Review alerts, correlate telemetry and deployments, then summarize the evidence and next actions."
}
```

#### Arguments

*   `name` (String, Required): Skill name, unique case-insensitively within the organization.
*   `when_to_use` (String, Required): Guidance that tells the Agent when to use the Skill.
*   `instructions` (String, Required): Instructions the Agent follows when using the Skill.
*   `description` (String, Optional): Human-readable Skill description.

#### Attributes

*   `id` (String): Skill UUID, used for import.
*   `identifier` (String): Stable display identifier returned by the API.
*   `revision` (Number): Current Skill revision.
*   `is_organizational` (Boolean): Whether the Skill is available to the organization.
*   `is_provisioned` (Boolean): Whether the Skill is managed by an external provisioner.
*   `created_at`, `created_by`, `updated_at`, and `updated_by`: Audit metadata returned by the API.
