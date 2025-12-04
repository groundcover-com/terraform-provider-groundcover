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

func TestAccMetricsAggregationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMetricsAggregationResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregation.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregation.test", "updated_at"),
				),
			},
			// Update and Read testing
			{
				Config: testAccMetricsAggregationResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregation.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregation.test", "updated_at"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccMetricsAggregationResource_complex(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with complex aggregation configuration
			{
				Config: testAccMetricsAggregationResourceConfigComplex(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregation.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_metricsaggregation.test", "updated_at"),
					// Check that YAML contains expected elements
					resource.TestMatchResourceAttr("groundcover_metricsaggregation.test", "value", regexp.MustCompile("ignore_old_samples")),
					resource.TestMatchResourceAttr("groundcover_metricsaggregation.test", "value", regexp.MustCompile("total_prometheus")),
				),
			},
		},
	})
}

func TestAccMetricsAggregationResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMetricsAggregationResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetricsAggregationResourceExists("groundcover_metricsaggregation.test"),
					testAccCheckMetricsAggregationResourceDisappears(),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccMetricsAggregationResourceConfig() string {
	return `
resource "groundcover_metricsaggregation" "test" {
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

func testAccMetricsAggregationResourceConfigUpdated() string {
	return `
resource "groundcover_metricsaggregation" "test" {
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

func testAccMetricsAggregationResourceConfigComplex() string {
	return `
resource "groundcover_metricsaggregation" "test" {
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

func testAccCheckMetricsAggregationResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		return nil
	}
}

func testAccCheckMetricsAggregationResourceDisappears() resource.TestCheckFunc {
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
		if err := client.DeleteMetricsAggregation(ctx); err != nil {
			return fmt.Errorf("Failed to delete metrics aggregation: %v", err)
		}

		return nil
	}
}
