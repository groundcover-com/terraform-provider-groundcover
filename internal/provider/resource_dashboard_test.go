package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDashboardResource(t *testing.T) {
	timestamp := time.Now().Unix()
	dashboardName := fmt.Sprintf("test_dashboard_%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDashboardResourceConfig(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "name", dashboardName),
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "description", "Test dashboard description"),
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "team", "engineering"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "preset"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "owner"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "status"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "revision_number"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_dashboard.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"override",
					"preset",
				},
			},
			{
				Config: testAccDashboardResourceConfigUpdatedDescription(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "name", dashboardName),
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "description", "Updated dashboard description"),
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "team", "engineering"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "preset"),
				),
			},
		},
	})
}

func TestAccDashboardResource_disappears(t *testing.T) {
	timestamp := time.Now().Unix()
	dashboardName := fmt.Sprintf("disappears_dashboard_%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDashboardResourceConfig(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardResourceDisappears("groundcover_dashboard.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckDashboardResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Dashboard ID is set")
		}

		// Create a provider client to delete the resource
		ctx := context.Background()

		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_BACKEND_ID")
		if orgName == "" {
			orgName = os.Getenv("GROUNDCOVER_ORG_NAME")
		}
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}

		// Create the client wrapper
		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, orgName)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		// Delete the resource using the client
		if err := client.DeleteDashboard(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete dashboard: %v", err)
		}

		return nil
	}
}

func testAccDashboardResourceConfig(name string) string {
	preset := `{
  "duration": "Last 1 hour",
  "layout": [
    {
      "id": "A",
      "x": 0,
      "y": 0,
      "w": 6,
      "h": 4,
      "minH": 2
    }
  ],
  "widgets": [
    {
      "id": "A",
      "type": "widget",
      "name": "Test Widget",
      "queries": [
        {
          "id": "A",
          "expr": "avg(groundcover_node_rt_disk_space_used_percent{})",
          "dataType": "metrics",
          "step": null,
          "editorMode": "builder"
        }
      ],
      "visualizationConfig": {
        "type": "time-series"
      }
    }
  ],
  "variables": {},
  "schemaVersion": 3
}`
	return fmt.Sprintf(`
resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Test dashboard description"
  team        = "engineering"
  preset      = jsonencode(%s)
}
`, name, preset)
}

func testAccDashboardResourceConfigUpdatedDescription(name string) string {
	preset := `{
  "duration": "Last 1 hour",
  "layout": [
    {
      "id": "A",
      "x": 0,
      "y": 0,
      "w": 6,
      "h": 4,
      "minH": 2
    }
  ],
  "widgets": [
    {
      "id": "A",
      "type": "widget",
      "name": "Test Widget",
      "queries": [
        {
          "id": "A",
          "expr": "avg(groundcover_node_rt_disk_space_used_percent{})",
          "dataType": "metrics",
          "step": null,
          "editorMode": "builder"
        }
      ],
      "visualizationConfig": {
        "type": "time-series"
      }
    }
  ],
  "variables": {},
  "schemaVersion": 3
}`
	return fmt.Sprintf(`
resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Updated dashboard description"
  team        = "engineering"
  preset      = jsonencode(%s)
}
`, name, preset)
}
