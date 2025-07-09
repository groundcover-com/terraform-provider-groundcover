// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLogsPipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccLogsPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "updated_at"),
				),
			},
			// Update and Read testing
			{
				Config: testAccLogsPipelineResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "updated_at"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccLogsPipelineResource_complex(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with complex pipeline configuration
			{
				Config: testAccLogsPipelineResourceConfigComplex(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "updated_at"),
					// Check that YAML contains expected elements
					resource.TestMatchResourceAttr("groundcover_logspipeline.test", "value", regexp.MustCompile("ottlRules")),
					resource.TestMatchResourceAttr("groundcover_logspipeline.test", "value", regexp.MustCompile("filter-errors")),
				),
			},
		},
	})
}

func testAccLogsPipelineResourceConfig() string {
	return `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule
  conditions:
    - container_name == "nginx"
  statements:
    - set(attributes["test.key"], "test-value")
YAML
}
`
}

func testAccLogsPipelineResourceConfigUpdated() string {
	return `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule-updated
  conditions:
    - container_name == "nginx"
  statements:
    - set(attributes["test.key"], "test-value-updated")
YAML
}
`
}

func testAccLogsPipelineResourceConfigComplex() string {
	return `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: filter-errors
  conditions:
    - container_name == "nginx"
  statements:
    - set(attributes["error.processed"], true)
- ruleName: enrich-logs
  conditions:
    - container_name == "web"
  statements:
    - set(attributes["service.name"], "web-service")
YAML
}
`
}
