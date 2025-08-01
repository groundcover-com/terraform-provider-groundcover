# groundcover Terraform Provider

Terraform provider for managing groundcover resources.

## Usage Examples

Basic usage examples can be found in the `examples/` directory:

*   **Policy Resource:** [`examples/resources/groundcover_policy/resource.tf`](./examples/resources/groundcover_policy/resource.tf)
    *   Demonstrates how to define a `groundcover_policy` resource, including roles and data scope.
*   **ServiceAccount Resource:** [`examples/resources/groundcover_serviceaccount/resource.tf`](./examples/resources/groundcover_serviceaccount/resource.tf)
    *   Shows how to create and manage service accounts with associated policies.
*   **API Key Resource:** [`examples/resources/groundcover_apikey/resource.tf`](./examples/resources/groundcover_apikey/resource.tf)
    *   Illustrates API key creation and management for service accounts.
*   **Monitor Resource:** [`examples/resources/groundcover_monitor/resource.tf`](./examples/resources/groundcover_monitor/resource.tf)
    *   Provides examples of configuring monitoring rules and alerts.
*   **Ingestion Key Resource:** [`examples/resources/groundcover_ingestionkey/resource.tf`](./examples/resources/groundcover_ingestionkey/resource.tf)
    *   Demonstrates how to create and manage ingestion keys for data ingestion.
*   **Logs Pipeline Resource:** [`examples/resources/groundcover_logspipeline/resource.tf`](./examples/resources/groundcover_logspipeline/resource.tf)
    *   Shows how to configure logs processing pipelines.

## Local Development and Testing

To use this provider locally before it is published to the Terraform Registry, follow these steps:

1.  **Build the Provider:**
    Compile the provider binary using the Makefile:
    ```bash
    make build
    ```
    This command typically places the compiled `terraform-provider-groundcover` executable into the `./dist` directory within your project.

2.  **Configure Terraform CLI for Local Override:**
    Terraform needs to know where to find your locally built provider instead of trying to download it from a registry. Create or edit the Terraform CLI configuration file (`~/.terraformrc` on macOS/Linux, `%APPDATA%\terraform.rc` on Windows) and add the following `provider_installation` block:

    ```hcl
    # ~/.terraformrc or %APPDATA%\terraform.rc

    provider_installation {
      # Replace "groundcover-com/groundcover" if you used a different source 
      # address in main.go. Replace the path with the actual absolute path
      # to the directory containing the built provider binary (step 1).
      dev_overrides {
        "registry.terraform.io/groundcover-com/groundcover" = "/Users/<YOUR_HOME_FOLDER>/projects/terraform-provider-groundcover/dist"
        # Example for Windows:
        # "registry.terraform.io/groundcover-com/groundcover" = "C:/Users/<YourUser>/path/to/terraform-provider-groundcover/dist"
      }

      # For all other providers, install them directly from their origin registries.
      direct {}
    }
    ```

    *   **Important:** Replace `registry.terraform.io/groundcover-com/groundcover` if your provider address in `main.go` is different.
    *   **Important:** Replace `/Users/<YOUR_HOME_FOLDER>/projects/terraform-provider-groundcover/dist` with the **absolute path** to the directory containing the `terraform-provider-groundcover` binary built by `make build`.

3.  **Use in a Terraform Project:**
    In a separate directory for your Terraform configuration:
    *   Create a `.tf` file (e.g., `main.tf`).
    *   Declare the provider requirement, ensuring the `source` matches the one used in `dev_overrides`:
        ```hcl
        # main.tf

        terraform {
          required_providers {
            # This 'source' value MUST exactly match:
            # 1. The key used in your ~/.terraformrc dev_overrides block
            # 2. The 'Address' set in your provider's main.go
            groundcover = {
              source = "registry.terraform.io/groundcover-com/groundcover"
              # Version constraint is still good practice, but less critical
              # for local dev as dev_overrides takes precedence.
              # Use ">= 0.1.0" or similar if you haven't tagged releases yet.
              version = ">= 0.0.0"
            }
          }
        }

        # Configure the provider instance
        provider "groundcover" {
          # It's STRONGLY recommended to provide the API key via an environment variable
          # export TF_VAR_groundcover_api_key="YOUR_API_KEY_HERE"
          api_key = var.groundcover_api_key # Use this if defining a variable below

          # Base URL is optional, defaults to api.groundcover.com in the provider code
          api_url = "https://api.main.groundcover.com" # defaults to https://api.groundcover.com
          org_name = "groundcover"                     # your organization ID as provided in the installation
        }

        # (Optional but recommended) Define input variables
        variable "groundcover_api_key" {
          type        = string
          description = "groundcover API Key"
          sensitive   = true
        }

        # Define a policy resource using your provider
        resource "groundcover_policy" "test_policy" {
          name        = "My Terraform Test Policy"
          description = "Policy managed via local Terraform provider build"
          claim_role  = "tf-test-claim"
          role = {
            admin  = "admin" # key is "read"/"write"/"admin" - value is ignored
          }

          # Example data_scope (adjust based on actual API needs)
          data_scope = {
            simple = {
              operator = "and"
              conditions = [
                {
                  key    = "cluster"
                  origin = "root"
                  type   = "string"
                  filters = [
                    {
                      op    = "match"
                      value = "my-prod-cluster"
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

        output "policy_id" {
          value = groundcover_policy.test_policy.uuid
        }

        output "policy_revision" {
          value = groundcover_policy.test_policy.revision_number
        }
        ```
    *   Run `terraform init`. Terraform will detect the `dev_overrides` and use your local build.

## Requirements

*   [Terraform](https://www.terraform.io/downloads.html) >= 1.0 (Check `required_version` if specified in `main.tf`)
*   [Go](https://golang.org/doc/install) >= 1.21 (to build the provider plugin)
*   groundcover Account and API Key.

## Provider Reference

Configure the groundcover provider in your Terraform configuration:

```hcl
provider "groundcover" {
  # api_key  = "YOUR_API_KEY" # Required
  # base_url = "https://api.your-instance.groundcover.com" # Optional
}
```

### Arguments

*   `api_key` (String, Required, Sensitive): Your groundcover API key. It is strongly recommended to configure this using the `TF_VAR_groundcover_api_key` environment variable rather than hardcoding it.
*   `base_url` (String, Optional): The base URL for the groundcover API. Defaults to `api.groundcover.com` if not specified.

## Testing

The provider includes comprehensive acceptance tests for all resources. To run the tests:

### Prerequisites

Set the required environment variables:

```bash
export GROUNDCOVER_API_KEY="your-api-key-here"
export GROUNDCOVER_API_URL="https://api.main.groundcover.com/"
export GROUNDCOVER_ORG_NAME="your-org-name"

# For ingestion key tests (requires cloud backend):
export GROUNDCOVER_CLOUD_ORG_NAME="your-cloud-org-name"
```

### Running Tests

```bash
# Run all tests
TF_ACC=1 go test ./internal/provider -v

# Run specific resource tests
TF_ACC=1 go test ./internal/provider -v -run TestAccPolicyResource
TF_ACC=1 go test ./internal/provider -v -run TestAccServiceAccountResource
TF_ACC=1 go test ./internal/provider -v -run TestAccIngestionKeyResource

# Run unit tests only (no API calls)
go test ./internal/provider -v
```

### Test Features

- **Create, Read, Update, Delete (CRUD)** testing for all resources
- **Import functionality** testing to ensure resources can be imported into Terraform state
- **Disappears testing** to verify proper handling when resources are deleted outside Terraform
- **Error handling** validation for various API error conditions
- **Cloud backend support** for ingestion key operations

### Test Architecture

The test suite includes:
- **Unit tests** for utility functions and error handling
- **Acceptance tests** that interact with real API endpoints
- **Retry logic** to handle eventual consistency in cloud APIs
- **Environment-specific configurations** for different backend types

## Resource Reference

See the [REFERENCE.md](./REFERENCE.md) file for detailed documentation of each resource.
For detailed examples of how to use each resource, see the [examples](./examples) directory.
