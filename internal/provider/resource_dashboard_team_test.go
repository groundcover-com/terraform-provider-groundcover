package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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