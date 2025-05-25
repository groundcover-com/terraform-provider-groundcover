---
page_title: "groundcover_logspipeline Resource - terraform-provider-groundcover"
subcategory: ""
description: |-
  Manages logs pipeline configuration for groundcover.
---

# groundcover_logspipeline

The logs pipeline resource allows you to define and manage OTTL (Observability Transformation and Templating Language) rules for processing logs in groundcover. This is a singleton resource - only one logs pipeline can exist per groundcover installation.

OTTL rules allow you to transform, enrich, and route logs based on conditions. For example, you can extract Kubernetes metadata and add it to your logs for better observability.

## Singleton Resource Behavior

This resource is implemented as a true singleton resource, meaning:

1. Only one logs pipeline configuration can exist in your groundcover backend at any time.
2. When you run `terraform plan` and a logs pipeline already exists, Terraform will show that it will update the existing resource rather than creating a new one.
3. If you have multiple Terraform configurations with this resource, they will all manage the same underlying pipeline configuration.

## Example Usage

```terraform
resource "groundcover_logspipeline" "logspipeline" {
  value = <<-EOT
ottlRules:
  - ruleName: example-rule
    conditions:
      - container_name == "nginx"
    statements:
      - set(attributes["test.key"], "test-value")
EOT
}
```

## Argument Reference

* `value` - (Required) YAML-formatted string containing the OTTL rules configuration for log processing.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `updated_at` - Timestamp of when the logs pipeline was last updated.

## OTTL Rule Configuration Structure

Each OTTL rule in the configuration requires the following components:

* `ruleName` - A unique name for the rule
* `conditions` - List of conditions that must be met for the rule to be applied
* `statements` - List of transformations to apply when conditions are met

Optional fields:
* `ruleDisabled` - Boolean to disable/enable the rule (default: false)
* `conditionLogicOperator` - Logic operator to combine conditions: `and` or `or` (default `or`)
* `statementsErrorMode` - How to handle errors in statements: `ignore`, `silent`, or `propagate`

## Import

Since this is a singleton resource, you can import the existing logs pipeline configuration with:

```
$ terraform import groundcover_logspipeline.example dummy
```

The import ID is arbitrary for this singleton resource, as the provider will automatically detect and manage the existing resource. 