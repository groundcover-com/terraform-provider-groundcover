// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccConnectedApp_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-slack-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConnectedAppConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "slack-webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.url", "https://hooks.slack.com/services/TEST/WEBHOOK/URL"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "created_by"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_connected_app.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"data"}, // data is sensitive and not returned on import
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccConnectedApp_update(t *testing.T) {
	initialName := acctest.RandomWithPrefix("test-slack-app-initial")
	updatedName := acctest.RandomWithPrefix("test-slack-app-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConnectedAppConfig_basic(initialName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", initialName),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "slack-webhook"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testAccConnectedAppConfig_basic(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "slack-webhook"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccConnectedApp_pagerduty(t *testing.T) {
	name := acctest.RandomWithPrefix("test-pagerduty-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConnectedAppConfig_pagerduty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "pagerduty"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.routing_key", "a1234567890123456789012345678901"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "created_by"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "created_at"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccConnectedApp_pagerdutyWithSeverityMapping(t *testing.T) {
	name := acctest.RandomWithPrefix("test-pagerduty-severity")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_pagerdutyWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "pagerduty"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.routing_key", "a1234567890123456789012345678901"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "critical"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.error", "error"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.warning", "warning"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.info", "info"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

func TestAccConnectedApp_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-slack-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConnectedAppResourceExists("groundcover_connected_app.test"),
					testAccCheckConnectedAppResourceDisappears("groundcover_connected_app.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccConnectedAppConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}
`, name)
}

func testAccConnectedAppConfig_pagerduty(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "pagerduty"
  data = {
    routing_key = "a1234567890123456789012345678901"
  }
}
`, name)
}

func testAccConnectedAppConfig_pagerdutyWithSeverityMapping(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "pagerduty"
  data = {
    routing_key = "a1234567890123456789012345678901"
    severity_mapping = {
      critical = "critical"
      error    = "error"
      warning  = "warning"
      info     = "info"
    }
  }
}
`, name)
}

func testAccCheckConnectedAppResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Connected App ID is set")
		}

		return nil
	}
}

func testAccCheckConnectedAppResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Connected App ID is set")
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

		// Delete the resource using the client
		if err := client.DeleteConnectedApp(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete connected app: %v", err)
		}

		return nil
	}
}
