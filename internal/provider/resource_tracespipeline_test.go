// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTracesPipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTracesPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "updated_at"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTracesPipelineResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "updated_at"),
					resource.TestMatchResourceAttr("groundcover_tracespipeline.test", "value", regexp.MustCompile("test-rule-updated")),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTracesPipelineResource_complex(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with complex pipeline configuration
			{
				Config: testAccTracesPipelineResourceConfigComplex(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "updated_at"),
					// Check that YAML contains expected elements
					resource.TestMatchResourceAttr("groundcover_tracespipeline.test", "value", regexp.MustCompile("ottlRules")),
					resource.TestMatchResourceAttr("groundcover_tracespipeline.test", "value", regexp.MustCompile("filter-errors")),
				),
			},
		},
	})
}

func testAccTracesPipelineResourceConfig() string {
	return `
resource "groundcover_tracespipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule
  conditions:
    - name == "http.request"
  statements:
    - set(attributes["test.key"], "test-value")
YAML
}
`
}

func testAccTracesPipelineResourceConfigUpdated() string {
	return `
resource "groundcover_tracespipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule-updated
  conditions:
    - name == "http.request"
  statements:
    - set(attributes["test.key"], "test-value-updated")
YAML
}
`
}

func testAccTracesPipelineResourceConfigComplex() string {
	return `
resource "groundcover_tracespipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: filter-errors
  conditions:
    - name == "http.request"
  statements:
    - set(attributes["error.processed"], true)
- ruleName: enrich-traces
  conditions:
    - name == "grpc.request"
  statements:
    - set(attributes["service.name"], "grpc-service")
YAML
}
`
}
