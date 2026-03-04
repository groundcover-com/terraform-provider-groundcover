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

func TestAccConnectedApp_rootly(t *testing.T) {
	name := acctest.RandomWithPrefix("test-rootly-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConnectedAppConfig_rootly(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "rootly"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.api_key", "test-rootly-api-key-123"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.webhook_url", "https://rootly.com/webhook"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "created_by"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "created_at"),
				),
			},
			// Delete testing automatically occurs in TestCase
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

func TestAccConnectedApp_opsgenie(t *testing.T) {
	name := acctest.RandomWithPrefix("test-opsgenie-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_opsgenie(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "opsgenie"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.api_key", "test-opsgenie-api-key-123"),
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

func TestAccConnectedApp_opsgenieWithPriorityMapping(t *testing.T) {
	name := acctest.RandomWithPrefix("test-opsgenie-priority")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_opsgenieWithPriorityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "opsgenie"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.api_key", "test-opsgenie-api-key-123"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.region", "eu"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.priority_mapping.critical", "P1"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.priority_mapping.error", "P2"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.priority_mapping.warning", "P3"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.priority_mapping.info", "P4"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

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

func TestAccConnectedApp_incidentio(t *testing.T) {
	name := acctest.RandomWithPrefix("test-incidentio-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_incidentio(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "incidentio"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.url", "https://api.incident.io/v2/alert-events/http/test-source-123"),
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

func TestAccConnectedApp_incidentioWithSeverityMapping(t *testing.T) {
	name := acctest.RandomWithPrefix("test-incidentio-severity")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_incidentioWithSeverityMapping(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "incidentio"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.url", "https://api.incident.io/v2/alert-events/http/test-source-123"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.critical", "SEV0"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.error", "SEV1"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.warning", "SEV2"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.severity_mapping.info", "SEV3"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

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

// Webhook tests

func TestAccConnectedApp_webhook(t *testing.T) {
	name := acctest.RandomWithPrefix("test-webhook-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_webhook(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.url", "https://example.com/webhook"),
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

func TestAccConnectedApp_webhookWithBearerAuth(t *testing.T) {
	name := acctest.RandomWithPrefix("test-webhook-bearer")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_webhookWithBearerAuth(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.url", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.method", "POST"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.auth_type", "bearer"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.api_key", "test-bearer-token-123"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

func TestAccConnectedApp_webhookWithBasicAuth(t *testing.T) {
	name := acctest.RandomWithPrefix("test-webhook-basic")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_webhookWithBasicAuth(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.url", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.auth_type", "basic"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.username", "testuser"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.password", "testpass123"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

func TestAccConnectedApp_webhookWithCustomPayload(t *testing.T) {
	name := acctest.RandomWithPrefix("test-webhook-payload")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppConfig_webhookWithCustomPayload(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "type", "webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.url", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.headers.Content-Type", "application/json"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.headers.X-Custom-Header", "custom-value"),
					resource.TestCheckResourceAttr("groundcover_connected_app.test", "data.custom_payload", "{\"alert\": \"test-alert\", \"severity\": \"critical\"}"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app.test", "id"),
				),
			},
		},
	})
}

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

func testAccConnectedAppConfig_rootly(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "rootly"
  data = {
    api_key     = "test-rootly-api-key-123"
    webhook_url = "https://rootly.com/webhook"
  }
}
`, name)
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

func testAccConnectedAppConfig_opsgenie(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "opsgenie"
  data = {
    api_key = "test-opsgenie-api-key-123"
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

func testAccConnectedAppConfig_incidentio(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "incidentio"
  data = {
    url = "https://api.incident.io/v2/alert-events/http/test-source-123"
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

func testAccConnectedAppConfig_webhook(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "webhook"
  data = {
    url = "https://example.com/webhook"
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

func testAccConnectedAppConfig_webhookWithBasicAuth(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "webhook"
  data = {
    url       = "https://example.com/webhook"
    auth_type = "basic"
    username  = "testuser"
    password  = "testpass123"
  }
}
`, name)
}

func testAccConnectedAppConfig_webhookWithCustomPayload(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = %[1]q
  type = "webhook"
  data = {
    url = "https://example.com/webhook"
    headers = {
      Content-Type    = "application/json"
      X-Custom-Header = "custom-value"
    }
    custom_payload = "{\"alert\": \"test-alert\", \"severity\": \"critical\"}"
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
			apiURL = "https://api.groundcover.com"
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
