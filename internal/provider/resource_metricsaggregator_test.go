// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMetricsAggregatorResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMetricsAggregatorResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregator.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregator.test", "updated_at"),
				),
			},
			// Update and Read testing
			{
				Config: testAccMetricsAggregatorResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregator.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregator.test", "updated_at"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccMetricsAggregatorResource_complex(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with complex aggregation configuration
			{
				Config: testAccMetricsAggregatorResourceConfigComplex(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregator.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregator.test", "updated_at"),
					// Check that YAML contains expected elements
					resource.TestMatchResourceAttr("groundcover_metricsaggregator.test", "value", regexp.MustCompile("ignore_old_samples")),
					resource.TestMatchResourceAttr("groundcover_metricsaggregator.test", "value", regexp.MustCompile("total_prometheus")),
				),
			},
		},
	})
}

func TestAccMetricsAggregatorResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMetricsAggregatorResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetricsAggregatorResourceExists("groundcover_metricsaggregator.test"),
					testAccCheckMetricsAggregatorResourceDisappears(),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccMetricsAggregatorResourceConfig() string {
	return `
resource "groundcover_metricsaggregator" "test" {
  value = <<-YAML
content: |
  - ignore_old_samples: true
    match: '{__name__=~"test_metric_counter"}'
    without: [instance]
    interval: 30s
    outputs: [total_prometheus]
YAML
}
`
}

func testAccMetricsAggregatorResourceConfigUpdated() string {
	return `
resource "groundcover_metricsaggregator" "test" {
  value = <<-YAML
content: |
  - ignore_old_samples: true
    match: '{__name__=~"test_metric_counter_updated"}'
    without: [instance, pod]
    interval: 60s
    outputs: [total_prometheus]
YAML
}
`
}

func testAccMetricsAggregatorResourceConfigComplex() string {
	return `
resource "groundcover_metricsaggregator" "test" {
  value = <<-YAML
content: |
  - ignore_old_samples: true
    match: '{__name__=~"test_metric_counter"}'
    without: [instance]
    interval: 30s
    outputs: [total_prometheus]
  - ignore_old_samples: false
    match: '{__name__=~"http_requests_total"}'
    without: [pod, instance]
    interval: 60s
    outputs: [total_prometheus]
YAML
}
`
}

func testAccCheckMetricsAggregatorResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		return nil
	}
}

func testAccCheckMetricsAggregatorResourceDisappears() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Create a provider client to delete the resource
		ctx := context.Background()

		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}

		// Create the client wrapper
		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, orgName)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		// For singleton resources, no ID is needed - just delete directly
		if err := client.DeleteMetricsAggregator(ctx); err != nil {
			return fmt.Errorf("Failed to delete metrics aggregator: %v", err)
		}

		return nil
	}
}
