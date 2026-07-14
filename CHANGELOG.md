## 1.18.0

* Fixed perpetual plan diffs on monitor resources when using `w` (week) duration units like `1w`, which now round-trip cleanly instead of drifting against the backend's canonical hours. Duration normalization for `groundcover_monitor` is now scoped to actual duration values, so free-text fields (e.g. a description containing "1w") are no longer rewritten

## 1.17.1

* Fixed `groundcover_synthetic_test` sending `GET /api/synthetics/v1/rules/{id}` with an empty ID — the provider now treats it as not-found instead of matching `GET /api/synthetics/v1/rules/` (which redirects to the list and returns 200)

## 1.17.0

* Added `groundcover_monitor_v2_json` — a variant of `groundcover_monitor_v2` whose `notification_settings.connected_app_params` is a JSON string (`jsonencode({...})`) instead of an HCL nested map. Schema, behaviour, and the underlying API are otherwise identical; the JSON-string form is for configs generated/consumed by tooling that can't model nested maps (e.g. the Crossplane provider). The existing `groundcover_monitor_v2` is unchanged
* Fixed `groundcover_monitor_v2`/`groundcover_monitor_v2_json` connected-app delivery: `notification_settings.connected_app_params.channels` was sent to the API as bare strings, but the backend expects Slack channel objects, so channel delivery was silently broken. Channels are now objects with a required `id` and optional `name`. **Breaking:** existing configs must change `channels = ["C123"]` to `channels = [{ id = "C123" }]` (typed) / the equivalent JSON `{"id":"C123"}` shape (JSON variant). Updated `github.com/groundcover-com/groundcover-sdk-go` to `v1.320.0` for the corrected `ConnectedAppChannel` model

## 1.16.2

* Deprecated `groundcover_monitor` — the resource now emits a deprecation warning directing users to `groundcover_monitor_v2`, which provides a typed Terraform schema in place of the raw YAML blob. The resource continues to work for backward compatibility
* Refreshed the `groundcover_monitor` example to a K8s Pod Crashed monitor using GCQL
* Removed `entities`, `rum`, and `issues` from the supported `groundcover_monitor_v2` GCQL `data_type` values — only `logs`, `traces`, `events`, and `apm` are supported. These values were not actually backed by the monitors API; configs using them are now rejected at plan time with a clear validation error

## 1.16.1

* Added `pkg/tfprovider`, a public re-export of the plugin-framework provider constructor (`internal/provider.New`), so the groundcover Crossplane provider can hand the live provider to upjet for schema introspection during code generation. No change to provider behaviour.

## 1.16.0

* Added `groundcover_connected_app_json` — a variant of `groundcover_connected_app` whose `data` is a JSON string (`jsonencode({...})`) instead of a dynamic object. Behaviour, drift detection (`data_hash`), and the underlying API are identical; the JSON-string form is for configs generated/consumed by tooling that can't model dynamic objects (e.g. the Crossplane provider). The existing `groundcover_connected_app` is unchanged.

## 1.15.0

* Added `apm` as a supported GCQL `data_type` for `groundcover_monitor_v2`

## 1.14.1

* Fixed `groundcover_monitor_v2` import/update failures caused by sending the reserved `_gc_monitor_v2_query_type` annotation back to the monitors API
* Added computed `data_hash` attribute to `groundcover_connected_app` — exposes the SHA-256 hash groundcover computes over the stored (pre-redaction) data, so Terraform can detect changes to the sensitive, redacted `data` without retrieving the secret
* `groundcover_connected_app` now corrects out-of-band drift: when the API-reported `data_hash` differs from the value recorded in state, the next plan shows a diff and apply restores the configured `data`. Detection is forward-looking — it covers changes made after the baseline hash is first recorded in state (on the first refresh after upgrading to this version); resources created by an older provider adopt the current server hash as their baseline, so pre-upgrade out-of-band changes are not retroactively flagged
* Updated `github.com/groundcover-com/groundcover-sdk-go` to `v1.291.0` (adds `data_hash` to connected-app responses)

## 1.14.0

* Fixed `groundcover_synthetic_test` stability issues when Terraform/OpenTofu produces unknown values for `assertion` blocks, including module patterns using `dynamic`, `for_each`, or `optional()` values
* Fixed perpetual diffs for `groundcover_synthetic_test.http_check.headers` when users explicitly configure an empty map (`headers = {}`)
* Added `groundcover_monitor_v2`, a typed monitor resource that replaces the raw YAML blob with Terraform schema fields and supports GCQL, MetricsQL, raw SQL, custom resolve thresholds, and connected-app delivery params
* Updated `github.com/groundcover-com/groundcover-sdk-go` to `v1.287.0` so monitor resources use the current generated monitor model; `isProvisioned` is now sent on monitor create requests per the current SDK shape

## 1.13.4

* Documented that `groundcover_policy` import takes the policy **UUID** (not the name) — the import example and generated docs now show a UUID placeholder and point at the UI/network tab for finding it
* Documented the allowed keys in the `groundcover_policy` `role` map (`read`, `write`, `admin`) in the schema description so they appear in the generated docs; the value is unused on the backend
* Added schema validation for the `groundcover_policy` `role` map — the provider now rejects invalid keys (`read`, `write`, and `admin` are the only accepted values) at plan time with a clear error, rather than propagating an opaque 400 from the backend

## 1.13.3

* Fixed wrong parsing of time duration in monitors (1d and complex formats like 1h30m)

## 1.13.2

* Documented that managing `groundcover_secret` resources (create, update, delete) requires a service account with the **admin** role — clarifies the permission requirement that previously surfaced only as an API error

## 1.13.1

* Documentation fix for `Monitor` resource examples

## 1.13.0

* Added `groundcover_metricspipeline` resource for configuring metrics relabeling rules (keep/drop metrics by regex, add labels, raw VictoriaMetrics relabel rules)
* Updated groundcover SDK from v1.244.0 to v1.256.0
* Added Terraform provisioning metadata to `groundcover_monitor` create and update requests — monitor resources now always send `isProvisioned: true`, matching dashboard behavior

## 1.12.2

* Updated `groundcover_synthetic_test` assertion `source` field — SSL/TLS properties (`certificateValid`, `certificateExpiresIn`, `tlsVersion`, `chainValid`, `cipherSuite`) and TCP properties (`tcpConnection`, `responseContains`) can now be used directly as `source` values instead of requiring `source = "ssl"/"tcp"` with a separate `property` field
* Added config validation that rejects `property` when using a property-as-source value (e.g., `source = "certificateValid"` with `property` set)
* Updated docs and examples to use the new direct-source syntax; the old `source + property` syntax remains fully backwards compatible

## 1.12.1

* Fixed DNS assertion source — fixed from `dns` to `dnsAnswer` to match the backend API

## 1.12.0

* Improved `property` field documentation for `groundcover_synthetic_test` assertions — added per-source usage details and HCL examples for SSL properties (`certificateValid`, `certificateExpiresIn`, `chainValid`)
* Added DNS check support to `groundcover_synthetic_test` resource — supports `domain`, `record_type`, `port`, `resolver`, `dnssec`, and `timeout` configuration for DNS resolution monitoring
* Added DNS assertion source (`dns`) for DNS check assertions
* Added TCP check support to `groundcover_synthetic_test` resource — supports `host`, `port`, `send`, `expect_response`, and `receive_max_bytes` configuration for TCP connectivity monitoring
* Added TCP assertion source (`tcp`) for TCP check assertions
* Fixed `follow_redirects` and `allow_insecure` state handling in `groundcover_synthetic_test` HTTP checks — the SDK changed these fields from `bool` to `*bool`, and the provider now uses an import/normal-read pattern to avoid perpetual diffs from server-side defaults
* Updated groundcover SDK from v1.235.0 to v1.244.0
* Fixed perpetual plan diffs on `groundcover_monitor` when the API returns boolean label values as quoted strings (e.g., `pagerduty: "true"` instead of `pagerduty: true`) — `deepEqual` now treats bool/string equivalents as semantically identical
* Fixed perpetual plan diffs on `groundcover_monitor` when using human-readable durations (e.g., `instantRollup: 10 minutes`) — time normalization now converts human-readable formats to Go duration format before comparison

## 1.11.0

* Removed `groundcover_recurring_silence` resource — backend API is being reworked

## 1.10.0

* Added notification routing support to `groundcover_synthetic_test` monitor block — supports `notification_method`, `connected_apps`, `status_filters`, and `disable_renotification` for controlling how synthetic monitor alerts are delivered
* Added SSL/TLS check support to `groundcover_synthetic_test` resource — supports `host`, `port`, `verify`, `min_version`, `sni`, and `timeout` configuration for proactive certificate and TLS connection monitoring
* Added SSL assertion source (`ssl`) with properties: `certificateValid`, `certificateExpiresIn`, `tlsVersion`, `chainValid`
* Updated groundcover SDK from v1.225.0 to v1.235.0

## 1.9.1

* Added `content_hash` computed attribute to `groundcover_secret` resource — returns FNV1a hash (hex encoded) of the secret content, enabling drift detection for external changes
* Secret resource now uses the GetSecretHash API endpoint to verify secret existence and detect external modifications during `terraform plan` and `terraform apply`
* Secrets deleted outside of Terraform are now properly detected and removed from state
* Updated groundcover SDK from v1.218.0 to v1.225.0
* Fixed `handleApiError` incorrectly mapping non-404 API errors to "resource not found" when the error message contained the substring "not found" — the status code regex now handles go-swagger's `[STATUS_CODE]` error format, and substring-based fallbacks are only used when no HTTP status code can be extracted
* Same fix applied to the "read-only" error mapping to prevent similar false positives
* Fixed dashboard update failing with `CurrentRevision excluded_if` validation error — use `Override: true` instead of sending `CurrentRevision` in update requests
* Deprecated the `override` attribute on `groundcover_dashboard` — it is now always enabled internally and the attribute will be removed in a future version
* Added `TestAccDashboardResource_Update` acceptance test covering name, description, and preset (spec) updates

## 1.9.0

* Added import documentation for all resources so `terraform import` usage is documented in the registry
* Added monitor and notification routing controls to `groundcover_synthetic_test` resource — supports `monitor_name`, `severity`, `issue_summary`, `issue_description`, `no_data_state`, `execution_error_state`, `lookbehind_window`, `renotification_interval`, `enabled_workflows`, and `evaluation_interval` (interval + pending_for)
* Added recurring_silence resource for managing recurring alert silences with daily, weekly, and monthly schedules
* Extracted shared matcher logic into a dedicated module for reuse across silence and recurring silence resources
* Updated groundcover SDK from v1.192.0 to v1.218.0

## 1.8.2

* Added MS-teams connected app documentation and tests.

## 1.8.1

* Fixed incorrect ingestion key `type` documentation — valid values are `sensor`, `rum`, and `thirdParty` (not `ingestion`)
* Added acceptance tests for all ingestion key type options (`sensor`, `rum`, `thirdParty`)

## 1.8.0

* Added traces pipeline resource for configuring traces processing pipelines (singleton resource)
* Updated groundcover SDK from v1.176.0 to v1.192.0

## 1.7.1

* Added AWS CloudWatch `awsNamespaces` (pull all metrics from a namespace) and `withContextTagsOnInfoMetrics` (enrich with resource labels) options and documentation
* Added Clickhouse (query log & custom metrics, system metrics) and PostgreSQL (slow queries & custom metrics, system metrics) examples to data integration resource documentation
* Updated groundcover SDK from v1.154.0 to v1.176.0
* Fixed connected_app resource to preserve plan/state value for the sensitive `data` attribute when the API omits or returns a different value, resolving "inconsistent values for sensitive attribute" errors and apply failures after create/update

## 1.7.0

* Added synthetic_test resource for proactive HTTP endpoint monitoring with assertions, retries, and authentication (basic/bearer with secret store support)
* Updated groundcover SDK from v1.151.0 to v1.154.0

## 1.6.0

* Added connected_app resource for managing integrations with external services (Slack, PagerDuty)
* Added notification_route resource for routing alerts to connected apps based on issue queries and statuses
* Updated groundcover SDK from v1.139.0 to v1.151.0

## 1.5.5

* Added connected_app resource for managing integrations with external services (Slack, PagerDuty)
* Added notification_route resource for routing alerts to connected apps based on issue queries and statuses
* Updated groundcover SDK from v1.139.0 to v1.151.0

## 1.5.4

* Added silence resource for managing alert silences in groundcover
* Updated groundcover SDK from v1.136.0 to v1.139.0
* Resolved critical bug where dashboards were being updated on every Terraform apply even when no logical changes were made, causing unnecessary revision increments and apply loops.
* Comprehensive debug logging: Added extensive debug logging throughout the dashboard resource.
* Enhanced logic to preserve original preset JSON format when semantically identical, preventing format drift cycles that could cause apply loops.
* Added RabbitMQ and Redis Cloud examples to data integration resource documentation

## 1.5.3

* Fixed apply loop bug in monitor resources where `FilterYamlKeysBasedOnTemplate` only looked at the first item in arrays (like `groupBy`), causing optional fields like `alias` that exist in later items to be filtered out, leading to false drift detection and apply loops. The fix merges keys from all array items to capture all possible fields.
* Added Prometheus static targets, Prometheus target discovery, and MongoDB Atlas examples to data integration resource documentation
* Improved automatic normalization of monitor YAML to convert single-line pipe syntax to simple strings
* Improved semantic comparison to ignore formatting differences (trailing newlines, multiline syntax)
* Added normalization for expression fields to handle multiline formatting differences (e.g., expressions split across lines with trailing spaces vs single-line expressions returned by API)
* Updated groundcover SDK from v1.28.0 to v1.136.0

## 1.5.2

* Added GCP and Azure examples to data integration resource documentation

## 1.5.1

* Fixed monitor drift detection false positives causing apply-loops when server returns numeric values in scientific notation (e.g., `5e+06` vs `5000000`)
* Fixed monitor drift detection to ignore `link` field and empty `description`/`annotations` fields that the server doesn't persist

## 1.5.0

* Added secret resource for securely storing sensitive values (API keys, passwords, credentials) and receiving reference IDs for use in other resources

## 1.4.2

* Added automatic retry with exponential backoff for 429 rate limit errors at the HTTP transport level, improving reliability for users provisioning larger scale infrastructure

## 1.4.1

* Fixed policy resource state upgrade from v0.1.x to properly transform data_scope schema (added disabled field and advanced block support)

## 1.4.0

* Added metrics aggregation resource for configuring metrics aggregation rules (singleton resource)
* Updated groundcover SDK from v1.89.0 to v1.126.0

## 1.3.0

* Added advanced data scope support for policy resource, enabling per-data-type filtering rules (logs, metrics, traces, events, workloads)
* Improved docs and examples for monitors, dashboards and data integrations

## 1.2.0

* Added data integration resource for managing data integrations in groundcover

## 1.1.1

* Fixed dashboard resource team field to properly handle null/empty values without causing drift
* Fixed dashboard updates to work in-place instead of requiring destroy/recreate operations
* Fixed API URL normalization to properly handle malformed URLs and missing https:// prefix

## 1.1.0

* Added dashboard resource for managing groundcover dashboards
* Fixed ingestion key resource compatibility with SDK v1.84.0 (deprecated creation_date field)
* Updated groundcover SDK from v1.41.0 to v1.84.0

## 1.0.0

* Aligned and improved documentation for the first stable release milestone
## 0.8.0

* Added `backend_id` as the new standard configuration option (replaces `org_name`, which remains supported for backwards compatibility)
* Added support for in-cloud backend configuration via `GROUNDCOVER_INCLOUD_BACKEND_ID`
* Updated Go version to 1.24

## 0.7.4

* Fixed drift detection with time normalization and empty field removal
* Added state upgrader for policies to handle schema changes

## 0.7.3

* Made `api_url` optional, defaults to production URL

## 0.7.2

* Security updates for dependencies (cloudflare/circl, cli/go-gh)

## 0.7.1

* Fixed duplicate test runs in CI pipeline

## 0.7.0

* Added ingestion key resource for in-cloud backends
* Enhanced test coverage for all resources

## 0.6.0

* Added logs pipeline resource
* Improved YAML parsing utilities

## 0.5.0

* Upgraded the terraform plugin framework module to support terraform v0.12+

## 0.4.0

* Fixed a critical issue with monitor provisioning where some fields like `contextHeaderLabels` and `evaluationInterval` were not updated
