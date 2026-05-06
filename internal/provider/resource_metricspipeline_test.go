// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMetricsPipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMetricsPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.keep_regex.#", "1"),
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.keep_regex.0", "http_requests_total"),
					resource.TestCheckResourceAttrSet("groundcover_metricspipeline.test", "updated_at"),
				),
			},
			{
				Config: testAccMetricsPipelineResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.keep_regex.#", "2"),
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.drop_regex.#", "1"),
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.drop_regex.0", "go_.*"),
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.add_label.team", "platform"),
					resource.TestCheckResourceAttrSet("groundcover_metricspipeline.test", "updated_at"),
				),
			},
		},
	})
}

func TestAccMetricsPipelineResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMetricsPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetricsPipelineResourceExists("groundcover_metricspipeline.test"),
					testAccCheckMetricsPipelineResourceDisappears(),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccMetricsPipelineResourceConfig() string {
	return `
resource "groundcover_metricspipeline" "test" {
  rules = {
    keep_regex = ["http_requests_total"]
  }
}
`
}

func testAccMetricsPipelineResourceConfigUpdated() string {
	return `
resource "groundcover_metricspipeline" "test" {
  rules = {
    keep_regex = ["http_requests_total", "process_cpu_seconds_total"]
    drop_regex = ["go_.*"]
    add_label = {
      team = "platform"
    }
  }
}
`
}

func testAccCheckMetricsPipelineResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		return nil
	}
}

func testAccCheckMetricsPipelineResourceDisappears() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()

		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		backendID := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}

		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, backendID)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		if err := client.DeleteMetricsPipeline(ctx); err != nil {
			return fmt.Errorf("Failed to delete metrics pipeline: %v", err)
		}

		return nil
	}
}
