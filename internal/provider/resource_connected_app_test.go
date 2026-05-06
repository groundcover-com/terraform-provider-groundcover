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
			{
				ResourceName:            "groundcover_connected_app.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"data"},
			},
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
			{
				Config: testAccConnectedAppConfig_basic(initialName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", initialName),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "slack-webhook"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_basic(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "slack-webhook"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

// TestAccConnectedApp_applyLoop tests that applying the same configuration multiple times
// doesn't cause an apply loop due to server-side normalization or formatting differences.
func TestAccConnectedApp_applyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-slack-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create connected app
			{
				Config: testAccConnectedAppConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This is the critical test - if there's an apply loop, this step will show changes
			{
				Config: testAccConnectedAppConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccConnectedAppConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

// TestAccConnectedApp_applyLoopWithSeverityMapping tests that applying the same configuration
// with nested severity_mapping doesn't cause an apply loop due to dynamic attribute handling.
func TestAccConnectedApp_applyLoopWithSeverityMapping(t *testing.T) {
	name := acctest.RandomWithPrefix("test-pagerduty-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create connected app with nested severity_mapping
			{
				Config: testAccConnectedAppConfig_pagerdutyWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "critical"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This is the critical test - if there's an apply loop, this step will show changes
			{
				Config: testAccConnectedAppConfig_pagerdutyWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "critical"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccConnectedAppConfig_pagerdutyWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "critical"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

// TestAccConnectedApp_rootlyApplyLoop tests that applying a rootly config with only api_key
// (no webhook_url) doesn't cause an apply loop due to the server returning an empty webhook_url.
func TestAccConnectedApp_rootlyApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-rootly-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create rootly app with only api_key (no webhook_url)
			{
				Config: testAccConnectedAppConfig_rootlyNoWebhookURL(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "rootly"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.api_key", "test-rootly-api-key-123"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// Also verify webhook_url doesn't leak into state from the server's empty response
			{
				Config: testAccConnectedAppConfig_rootlyNoWebhookURL(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckNoResourceAttr("groundcover_connected_app.test", "data.webhook_url"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccConnectedAppConfig_rootlyNoWebhookURL(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckNoResourceAttr("groundcover_connected_app.test", "data.webhook_url"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

// OpsGenie tests

func TestAccConnectedApp_opsgenieApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-opsgenie-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_opsgenieWithPriorityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.priority_mapping.critical", "P1"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_opsgenieWithPriorityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.priority_mapping.critical", "P1"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_opsgenieWithPriorityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.priority_mapping.critical", "P1"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

// Incident.io tests

func TestAccConnectedApp_incidentioApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-incidentio-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_incidentioWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "SEV0"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_incidentioWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "SEV0"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_incidentioWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "SEV0"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

// MS Teams tests

func TestAccConnectedApp_msTeamsApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-msteams-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_msTeams(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_msTeams(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_msTeams(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

// Webhook tests

func TestAccConnectedApp_webhookApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-webhook-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_webhookWithBearerAuth(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.auth_type", "bearer"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_webhookWithBearerAuth(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.auth_type", "bearer"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
			{
				Config: testAccConnectedAppConfig_webhookWithBearerAuth(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.auth_type", "bearer"),
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

		ctx := context.Background()

		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}

		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, orgName)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		if err := client.DeleteConnectedApp(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete connected app: %v", err)
		}

		return nil
	}
}

func testAccConnectedAppConfig_rootlyNoWebhookURL(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "rootly"
  data = {
    api_key = "test-rootly-api-key-123"
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

func testAccConnectedAppConfig_opsgenieWithPriorityMapping(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "opsgenie"
  data = {
    api_key = "test-opsgenie-api-key-123"
    region  = "eu"
    priority_mapping = {
      critical = "P1"
      error    = "P2"
      warning  = "P3"
      info     = "P4"
    }
  }
}
`, name)
}

func testAccConnectedAppConfig_incidentioWithSeverityMapping(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "incidentio"
  data = {
    url = "https://api.incident.io/v2/alert-events/http/test-source-123"
    severity_mapping = {
      critical = "SEV0"
      error    = "SEV1"
      warning  = "SEV2"
      info     = "SEV3"
    }
  }
}
`, name)
}

func testAccConnectedAppConfig_msTeams(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "ms-teams"
  data = {
    url = "https://prod-00.westus.logic.azure.com:443/workflows/test"
  }
}
`, name)
}

func testAccConnectedAppConfig_webhookWithBearerAuth(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "webhook"
  data = {
    url       = "https://example.com/webhook"
    method    = "POST"
    auth_type = "bearer"
    api_key   = "test-bearer-token-123"
  }
}
`, name)
}
