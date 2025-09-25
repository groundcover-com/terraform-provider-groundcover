package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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