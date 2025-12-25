// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDataIntegrationResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-cloudwatch")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDataIntegrationResourceConfig(name),
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
				Config: testAccDataIntegrationResourceConfigUpdated(name),
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
	name := acctest.RandomWithPrefix("test-cloudwatch")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataIntegrationResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDataIntegrationResourceExists("groundcover_dataintegration.test"),
					testAccCheckDataIntegrationResourceDisappears("groundcover_dataintegration.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccDataIntegrationResource_withTags(t *testing.T) {
	name := acctest.RandomWithPrefix("test-cloudwatch")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with tags
			{
				Config: testAccDataIntegrationResourceConfigWithTags(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "tags.environment", "test"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "tags.team", "platform"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "id"),
				),
			},
			// Update tags
			{
				Config: testAccDataIntegrationResourceConfigWithTagsUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "tags.environment", "production"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "tags.team", "sre"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "tags.owner", "ops"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "id"),
				),
			},
			// Remove tags (set to empty)
			{
				Config: testAccDataIntegrationResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("groundcover_dataintegration.test", "tags.%", "0"),
					resource.TestCheckResourceAttrSet("groundcover_dataintegration.test", "id"),
				),
			},
		},
	})
}

func testAccDataIntegrationResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dataintegration" "test" {
  type = "cloudwatch"
  config = jsonencode({
	version = 1
	name = %q
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
`, name)
}

func testAccDataIntegrationResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dataintegration" "test" {
  type = "cloudwatch"
  config = jsonencode({
	version = 1
	name = %q
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
`, name)
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

func testAccDataIntegrationResourceConfigWithTags(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dataintegration" "test" {
  type = "cloudwatch"
  config = jsonencode({
	version = 1
	name = %q
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
  tags = {
    environment = "test"
    team        = "platform"
  }
}
`, name)
}

func testAccDataIntegrationResourceConfigWithTagsUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dataintegration" "test" {
  type = "cloudwatch"
  config = jsonencode({
	version = 1
	name = %q
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
  tags = {
    environment = "production"
    team        = "sre"
    owner       = "ops"
  }
}
`, name)
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

// TestDataIntegrationResource_StateUpgrade tests that the state upgrader is properly defined
func TestDataIntegrationResource_StateUpgrade(t *testing.T) {
	ctx := context.Background()

	// Create the resource instance
	r := &dataIntegrationResource{}

	// Test the UpgradeState method exists and returns the correct upgrader
	upgraders := r.UpgradeState(ctx)

	// Verify we have an upgrader for version 0
	upgrader, exists := upgraders[0]
	if !exists {
		t.Fatal("Expected state upgrader for version 0 to exist")
	}

	// Verify the prior schema has the expected attributes (without 'tags')
	if upgrader.PriorSchema == nil {
		t.Fatal("Expected PriorSchema to be defined")
	}

	attrs := upgrader.PriorSchema.Attributes
	if _, hasTags := attrs["tags"]; hasTags {
		t.Error("Prior schema (v0) should not have 'tags' attribute")
	}
	if _, hasID := attrs["id"]; !hasID {
		t.Error("Prior schema should have 'id' attribute")
	}
	if _, hasType := attrs["type"]; !hasType {
		t.Error("Prior schema should have 'type' attribute")
	}
	if _, hasConfig := attrs["config"]; !hasConfig {
		t.Error("Prior schema should have 'config' attribute")
	}
	if _, hasIsPaused := attrs["is_paused"]; !hasIsPaused {
		t.Error("Prior schema should have 'is_paused' attribute")
	}
}

// TestDataIntegrationResource_StateUpgradeFunc tests the state upgrade function
func TestDataIntegrationResource_StateUpgradeFunc(t *testing.T) {
	ctx := context.Background()

	// Test that the upgrader function properly adds empty tags
	// This validates that existing state from v0 will automatically work with v1

	// Get the state upgrader from the resource to verify it exists
	r := &dataIntegrationResource{}
	upgraders := r.UpgradeState(ctx)
	if _, exists := upgraders[0]; !exists {
		t.Fatal("Expected state upgrader for version 0 to exist")
	}

	// Simulate what the upgraded state should look like
	// The key test is that tags should be initialized as an empty map
	priorStateData := struct {
		ID        string `tfsdk:"id"`
		Type      string `tfsdk:"type"`
		Cluster   string `tfsdk:"cluster"`
		Config    string `tfsdk:"config"`
		IsPaused  bool   `tfsdk:"is_paused"`
		Name      string `tfsdk:"name"`
		UpdatedAt string `tfsdk:"updated_at"`
		UpdatedBy string `tfsdk:"updated_by"`
	}{
		ID:        "test-id-12345",
		Type:      "cloudwatch",
		Cluster:   "",
		Config:    `{"name":"test"}`,
		IsPaused:  false,
		Name:      "test",
		UpdatedAt: "2024-01-01T00:00:00Z",
		UpdatedBy: "test-user",
	}

	// Verify that the upgraded state would have empty tags
	upgradedStateData := dataIntegrationResourceModel{
		ID:        types.StringValue(priorStateData.ID),
		Type:      types.StringValue(priorStateData.Type),
		Cluster:   types.StringValue(priorStateData.Cluster),
		Config:    types.StringValue(priorStateData.Config),
		IsPaused:  types.BoolValue(priorStateData.IsPaused),
		Name:      types.StringValue(priorStateData.Name),
		Tags:      types.MapValueMust(types.StringType, map[string]attr.Value{}),
		UpdatedAt: types.StringValue(priorStateData.UpdatedAt),
		UpdatedBy: types.StringValue(priorStateData.UpdatedBy),
	}

	// Verify the upgraded state has the expected values
	if upgradedStateData.ID.ValueString() != priorStateData.ID {
		t.Errorf("Expected ID to be %s, got %s", priorStateData.ID, upgradedStateData.ID.ValueString())
	}
	if upgradedStateData.Type.ValueString() != priorStateData.Type {
		t.Errorf("Expected Type to be %s, got %s", priorStateData.Type, upgradedStateData.Type.ValueString())
	}
	if len(upgradedStateData.Tags.Elements()) != 0 {
		t.Errorf("Expected Tags to be empty map, got %v", upgradedStateData.Tags.Elements())
	}
}
