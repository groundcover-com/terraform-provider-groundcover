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
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.keep_regex.0", testAccMetricsPipelineKeepRegex()),
					resource.TestCheckResourceAttrSet("groundcover_metricspipeline.test", "updated_at"),
				),
			},
			{
				Config: testAccMetricsPipelineResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.keep_regex.#", "2"),
					// Assert the values, not just the shape, so a singleton written by
					// someone else cannot satisfy this step.
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.keep_regex.0", testAccMetricsPipelineKeepRegex()),
					resource.TestCheckResourceAttr("groundcover_metricspipeline.test", "rules.keep_regex.1", "process_cpu_seconds_total"),
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

// testAccMetricsPipelineKeepRegex is the first keep_regex entry, carrying testRunToken
// so that a value written by another CI run is attributable in the plan diff rather
// than indistinguishable from this run's own.
func testAccMetricsPipelineKeepRegex() string {
	return fmt.Sprintf("http_requests_total_%s", testRunToken())
}

func testAccMetricsPipelineResourceConfig() string {
	return fmt.Sprintf(`
resource "groundcover_metricspipeline" "test" {
  rules = {
    keep_regex = [%[1]q]
  }
}
`, testAccMetricsPipelineKeepRegex())
}

func testAccMetricsPipelineResourceConfigUpdated() string {
	return fmt.Sprintf(`
resource "groundcover_metricspipeline" "test" {
  rules = {
    keep_regex = [%[1]q, "process_cpu_seconds_total"]
    drop_regex = ["go_.*"]
    add_label = {
      team = "platform"
    }
  }
}
`, testAccMetricsPipelineKeepRegex())
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
