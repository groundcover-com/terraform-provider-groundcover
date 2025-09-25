// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDataIntegrationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDataIntegrationResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "is_paused", "false"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "config"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "updated_at"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "updated_by"),
					// Check that config contains expected JSON elements
					resource.TestMatchResourceAttr("groundcover_dataintegration.test", "config", regexp.MustCompile("stsRegion")),
					resource.TestMatchResourceAttr("groundcover_dataintegration.test", "config", regexp.MustCompile("us-east-1")),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_dataintegration.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccDataIntegrationImportStateIdFunc,
			},
			// Update and Read testing
			{
				Config: testAccDataIntegrationResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "is_paused", "true"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "updated_at"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "updated_by"),
					// Check that config contains updated elements
					resource.TestMatchResourceAttr("groundcover_dataintegration.test", "config", regexp.MustCompile("us-west-2")),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccDataIntegrationResource_withCluster(t *testing.T) {
	cluster := acctest.RandomWithPrefix("test-cluster")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with cluster
			{
				Config: testAccDataIntegrationResourceConfigWithCluster(cluster),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "cluster", cluster),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "updated_at"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "updated_by"),
				),
			},
		},
	})
}

func TestAccDataIntegrationResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataIntegrationResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDataIntegrationResourceExists("groundcover_dataintegration.test"),
					testAccCheckDataIntegrationResourceDisappears("groundcover_dataintegration.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDataIntegrationResourceConfig() string {
	return `
resource "groundcover_dataintegration" "test" {
  type = "cloudwatch"
  config = jsonencode({
	version = 1
	name = "test-cloudwatch"
	exporters = ["prometheus"]
	scrapeInterval = "5m"
    stsRegion = "us-east-1"
    regions = ["us-east-1"]
    roleArn = "arn:aws:iam::123456789012:role/test-role"
    awsMetrics = [
      {
        namespace = "AWS/EC2"
        metrics = [
          {
            name = "CPUUtilization"
            statistics = ["Average"]
            period = 300
            length = 300
            nullAsZero = false
          }
        ]
      }
    ]
    apiConcurrencyLimits = {
      listMetrics = 3
      getMetricData = 5
      getMetricStatistics = 5
      listInventory = 10
    }
    withContextTagsOnInfoMetrics = false
    withInventoryDiscovery = false
  })
  is_paused = false
}
`
}

func testAccDataIntegrationResourceConfigUpdated() string {
	return `
resource "groundcover_dataintegration" "test" {
  type = "cloudwatch"
  config = jsonencode({
	version = 1
	name = "test-cloudwatch-updated"
	exporters = ["prometheus"]
	scrapeInterval = "5m"
    stsRegion = "us-east-1"
    regions = ["us-east-1", "us-west-2"]
    roleArn = "arn:aws:iam::123456789012:role/test-role-updated"
    awsMetrics = [
      {
        namespace = "AWS/EC2"
        metrics = [
          {
            name = "CPUUtilization"
            statistics = ["Average", "Maximum"]
            period = 300
            length = 300
            nullAsZero = false
          }
        ]
      }
    ]
    apiConcurrencyLimits = {
      listMetrics = 2
      getMetricData = 10
      getMetricStatistics = 10
      listInventory = 20
    }
    withContextTagsOnInfoMetrics = true
    withInventoryDiscovery = true
  })
  is_paused = true
}
`
}

func testAccDataIntegrationResourceConfigWithCluster(cluster string) string {
	return fmt.Sprintf(`
resource "groundcover_dataintegration" "test" {
  type     = "cloudwatch"
  cluster  = %[1]q
  config = jsonencode({
	version = 1
	name = "test-cloudwatch-with-cluster"
	exporters = ["prometheus"]
	scrapeInterval = "5m"
    stsRegion = "us-east-1"
    regions = ["us-east-1"]
    roleArn = "arn:aws:iam::123456789012:role/test-role"
    awsMetrics = [
      {
        namespace = "AWS/EC2"
        metrics = [
          {
            name = "CPUUtilization"
            statistics = ["Average"]
            period = 300
            length = 300
            nullAsZero = false
          }
        ]
      }
    ]
    apiConcurrencyLimits = {
      listMetrics = 3
      getMetricData = 5
      getMetricStatistics = 5
      listInventory = 10
    }
    withContextTagsOnInfoMetrics = false
    withInventoryDiscovery = false
  })
  is_paused = false
}
`, cluster)
}

func testAccCheckDataIntegrationResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DataIntegration ID is set")
		}

		if rs.Primary.Attributes["type"] == "" {
			return fmt.Errorf("No DataIntegration type is set")
		}

		return nil
	}
}

func testAccCheckDataIntegrationResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DataIntegration ID is set")
		}

		// Create a provider client to delete the resource
		ctx := context.Background()

		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.io"
		}

		// Create the client wrapper
		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, orgName)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		// Get the type and ID from state
		integrationType := rs.Primary.Attributes["type"]
		integrationID := rs.Primary.ID

		// Get optional parameters from state
		var cluster *string
		if clusterVal := rs.Primary.Attributes["cluster"]; clusterVal != "" {
			cluster = &clusterVal
		}

		// Delete the resource using the client
		if err := client.DeleteDataIntegration(ctx, integrationType, integrationID, cluster); err != nil {
			return fmt.Errorf("Failed to delete DataIntegration: %v", err)
		}

		return nil
	}
}

func testAccDataIntegrationImportStateIdFunc(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["groundcover_dataintegration.test"]
	if !ok {
		return "", fmt.Errorf("Not found: groundcover_dataintegration.test")
	}

	// Import format: "type:id"
	return fmt.Sprintf("%s:%s", rs.Primary.Attributes["type"], rs.Primary.ID), nil
}
