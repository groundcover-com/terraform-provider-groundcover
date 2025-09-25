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
