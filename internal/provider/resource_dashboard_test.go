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

func TestAccDashboardResource_Simple(t *testing.T) {
	timestamp := time.Now().Unix()
	dashboardName := fmt.Sprintf("simple_dashboard_%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDashboardResourceConfigSimple(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "name", dashboardName),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "revision_number"),
				),
			},
		},
	})
}

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

func TestAccDashboardResource_EmptyTeam(t *testing.T) {
	timestamp := time.Now().Unix()
	dashboardName := fmt.Sprintf("empty_team_dashboard_%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without team field
			{
				Config: testAccDashboardResourceConfigNoTeam(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "name", dashboardName),
					resource.TestCheckNoResourceAttr("groundcover_dashboard.test", "team"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "id"),
				),
			},
			// Update description, team should remain empty
			{
				Config: testAccDashboardResourceConfigNoTeamUpdated(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "name", dashboardName),
					resource.TestCheckResourceAttr("groundcover_dashboard.test", "description", "Updated empty team dashboard"),
					resource.TestCheckNoResourceAttr("groundcover_dashboard.test", "team"),
				),
			},
		},
	})
}

// TestAccDashboardResource_ApplyLoopIssue tests that applying the same configuration multiple times
// doesn't cause an apply loop, even when using jsonencode which can produce different formatting
func TestAccDashboardResource_ApplyLoopIssue(t *testing.T) {
	timestamp := time.Now().Unix()
	dashboardName := fmt.Sprintf("apply_loop_test_%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create dashboard with jsonencode (simulating for_each usage)
			{
				Config: testAccDashboardResourceConfigWithJsonEncode(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "preset"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "revision_number"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This is the critical test - if there's an apply loop, this step will show changes
			{
				Config: testAccDashboardResourceConfigWithJsonEncode(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "preset"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "revision_number"),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccDashboardResourceConfigWithJsonEncode(dashboardName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "preset"),
					resource.TestCheckResourceAttrSet("groundcover_dashboard.test", "revision_number"),
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
	return fmt.Sprintf(`
resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Test dashboard description"
  team        = "engineering"
  preset      = jsonencode({
    duration = "Last 1 hour"
    layout = [
      {
        id   = "A"
        x    = 0
        y    = 0
        w    = 6
        h    = 4
        minH = 2
      }
    ]
    widgets = [
      {
        id   = "A"
        type = "widget"
        name = "Test Widget"
        queries = [
          {
            id         = "A"
            expr       = "avg(groundcover_node_rt_disk_space_used_percent{})"
            dataType   = "metrics"
            step       = null
            editorMode = "builder"
          }
        ]
        visualizationConfig = {
          type = "time-series"
        }
      }
    ]
    variables     = {}
    schemaVersion = 3
  })
}
`, name)
}

func testAccDashboardResourceConfigUpdatedDescription(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Updated dashboard description"
  team        = "engineering"
  preset      = jsonencode({
    duration = "Last 1 hour"
    layout = [
      {
        id   = "A"
        x    = 0
        y    = 0
        w    = 6
        h    = 4
        minH = 2
      }
    ]
    widgets = [
      {
        id   = "A"
        type = "widget"
        name = "Test Widget"
        queries = [
          {
            id         = "A"
            expr       = "avg(groundcover_node_rt_disk_space_used_percent{})"
            dataType   = "metrics"
            step       = null
            editorMode = "builder"
          }
        ]
        visualizationConfig = {
          type = "time-series"
        }
      }
    ]
    variables     = {}
    schemaVersion = 3
  })
}
`, name)
}

func testAccDashboardResourceConfigSimple(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Simple test dashboard"
  preset      = jsonencode({
    duration      = "Last 1 hour"
    widgets       = []
    layout        = []
    variables     = {}
    schemaVersion = 3
  })
}
`, name)
}

func testAccDashboardResourceConfigNoTeam(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Dashboard without team field"
  preset      = jsonencode({
    duration      = "Last 1 hour"
    widgets       = []
    layout        = []
    variables     = {}
    schemaVersion = 3
  })
}
`, name)
}

func testAccDashboardResourceConfigNoTeamUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Updated empty team dashboard"
  preset      = jsonencode({
    duration      = "Last 1 hour"
    widgets       = []
    layout        = []
    variables     = {}
    schemaVersion = 3
  })
}
`, name)
}

// testAccDashboardResourceConfigWithJsonEncode simulates the for_each pattern where
// jsonencode is used, which can produce different JSON formatting on each run
func testAccDashboardResourceConfigWithJsonEncode(name string) string {
	return fmt.Sprintf(`
locals {
  preset_data = {
    duration = "Last 1 hour"
    layout = [
      {
        id   = "A"
        x    = 0
        y    = 0
        w    = 6
        h    = 4
        minH = 2
      }
    ]
    widgets = [
      {
        id   = "A"
        type = "widget"
        name = "Test Widget"
        queries = [
          {
            id         = "A"
            expr       = "avg(groundcover_node_rt_disk_space_used_percent{})"
            dataType   = "metrics"
            step       = null
            editorMode = "builder"
          }
        ]
        visualizationConfig = {
          type = "time-series"
        }
      }
    ]
    variables     = {}
    schemaVersion = 3
  }
}

resource "groundcover_dashboard" "test" {
  name        = "%s"
  description = "Test dashboard for apply loop detection"
  team        = "engineering"
  preset      = jsonencode(local.preset_data)
}
`, name)
}
