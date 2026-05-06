// SPDX-License-Identifier: MPL-2.0

package provider

import (
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
