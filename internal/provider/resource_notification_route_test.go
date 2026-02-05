// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNotificationRoute_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.connected_apps.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.connected_apps.0.type", "slack-webhook"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "routes.0.connected_apps.0.id"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "notification_settings.renotification_interval", "4h"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "created_by"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_notification_route.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccNotificationRoute_update(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-update")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationRouteConfig_update_step1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testAccNotificationRouteConfig_update_step2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:production"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "2"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.1", "Resolved"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccNotificationRoute_multipleRoutes(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-multi")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationRouteConfig_multipleRoutes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "2"),
					// First route
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.connected_apps.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.connected_apps.0.type", "slack-webhook"),
					// Second route
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.1.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.1.status.0", "Resolved"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.1.connected_apps.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.1.connected_apps.0.type", "slack-webhook"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccNotificationRoute_durationNormalization(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-duration")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRouteConfig_durationNormalization(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "notification_settings.renotification_interval", "60m"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
			{
				Config:             testAccNotificationRouteConfig_durationNormalization(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccNotificationRoute_applyLoop tests that applying the same configuration multiple times
// doesn't cause an apply loop due to server-side normalization or formatting differences.
func TestAccNotificationRoute_applyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create notification route
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This is the critical test - if there's an apply loop, this step will show changes
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
		},
	})
}

func testAccNotificationRouteConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "4h"
  }
}
`, name)
}

func testAccNotificationRouteConfig_update_step1(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "1h"
  }
}
`, name)
}

func testAccNotificationRouteConfig_update_step2(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:production"

  routes = [{
    status = ["Alerting", "Resolved"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "1h"
  }
}
`, name)
}

func testAccNotificationRouteConfig_multipleRoutes(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test1" {
  name = "%[1]s-slack-1"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL1"
  }
}

resource "groundcover_connected_app" "test2" {
  name = "%[1]s-slack-2"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL2"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [
    {
      status = ["Alerting"]
      connected_apps = [{
        type = "slack-webhook"
        id   = groundcover_connected_app.test1.id
      }]
    },
    {
      status = ["Resolved"]
      connected_apps = [{
        type = "slack-webhook"
        id   = groundcover_connected_app.test2.id
      }]
    }
  ]

  notification_settings = {
    renotification_interval = "2h"
  }
}
`, name)
}

func testAccNotificationRouteConfig_durationNormalization(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "60m"
  }
}
`, name)
}

// TestAccNotificationRoute_noNotificationSettings tests that notification routes
// can be created without specifying notification_settings (it's Optional+Computed).
func TestAccNotificationRoute_noNotificationSettings(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-no-settings")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without notification_settings
			{
				Config: testAccNotificationRouteConfig_noNotificationSettings(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
					// notification_settings should be computed with default/empty values
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "notification_settings.%"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_notification_route.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Apply again - should not cause changes (no apply loop)
			{
				Config:             testAccNotificationRouteConfig_noNotificationSettings(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccNotificationRouteConfig_noNotificationSettings(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]
}
`, name)
}
