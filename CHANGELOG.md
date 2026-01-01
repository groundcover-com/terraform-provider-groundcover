## 1.5.3

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

* Added data integration resource for managing data integrations in Groundcover

## 1.1.1

* Fixed dashboard resource team field to properly handle null/empty values without causing drift
* Fixed dashboard updates to work in-place instead of requiring destroy/recreate operations
* Fixed API URL normalization to properly handle malformed URLs and missing https:// prefix

## 1.1.0

* Added dashboard resource for managing Groundcover dashboards
* Fixed ingestion key resource compatibility with SDK v1.84.0 (deprecated creation_date field)
* Updated Groundcover SDK from v1.41.0 to v1.84.0

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
