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

func TestAccSyntheticTestResource_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "enabled", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "version", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.url", "https://httpbin.org/status/200"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.method", "GET"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.timeout", "10s"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "2"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "statusCode"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.operator", "eq"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.target", "200"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
				// auth token/password are sensitive and not returned from API
				ImportStateVerifyIgnore: []string{"http_check.auth"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSyntheticTestResource_update(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth")
	updatedName := name + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "2"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Update name and interval
			{
				Config: testAccSyntheticTestResourceConfig_updated(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "5m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "statusCode"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSyntheticTestResource_withRetry(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-retry")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_withRetry(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "retry.count", "3"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "retry.interval", "500ms"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_applyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-loop")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 2: Apply same config - should detect no changes
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 3: One more time to be sure
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-disappear")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSyntheticTestResourceExists("groundcover_synthetic_test.test"),
					testAccCheckSyntheticTestResourceDisappears("groundcover_synthetic_test.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_tcpDisappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp-disappear")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSyntheticTestResourceExists("groundcover_synthetic_test.test"),
					testAccCheckSyntheticTestResourceDisappears("groundcover_synthetic_test.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_withMonitor(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-monitor")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with monitor block
			{
				Config: testAccSyntheticTestResourceConfig_withMonitor(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.monitor_name", "Monitor for "+name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.severity", "S1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.no_data_state", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.execution_error_state", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.renotification_interval", "1h"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.evaluation_interval.interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.evaluation_interval.pending_for", "0s"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
				// http_check.auth: sensitive fields (password/token) are never read back from the API.
				// monitor.*: the monitor block is only tracked in state when explicitly configured;
				// on import the prior state is empty so fromSDKResponse skips the block to avoid
				// server-default values (issue_summary, lookbehind_window) leaking into state and
				// causing perpetual plan diffs. Users add the monitor block to their config post-import.
				ImportStateVerifyIgnore: []string{
					"http_check.auth",
					"monitor.%",
					"monitor.monitor_name",
					"monitor.severity",
					"monitor.issue_summary",
					"monitor.issue_description",
					"monitor.no_data_state",
					"monitor.execution_error_state",
					"monitor.lookbehind_window",
					"monitor.renotification_interval",
					"monitor.enabled_workflows",
					"monitor.evaluation_interval.%",
					"monitor.evaluation_interval.interval",
					"monitor.evaluation_interval.pending_for",
				},
			},
			// Update monitor settings
			{
				Config: testAccSyntheticTestResourceConfig_withMonitorUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.severity", "S2"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.no_data_state", "OK"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.renotification_interval", "4h"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.evaluation_interval.pending_for", "5m"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSyntheticTestResource_withMonitorMinimal(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-mon-min")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with minimal monitor (just severity)
			{
				Config: testAccSyntheticTestResourceConfig_withMonitorMinimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.severity", "S2"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

// --- Config helpers ---

func testAccSyntheticTestResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  http_check {
    url     = "https://httpbin.org/status/200"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  assertion {
    source   = "responseTime"
    operator = "lt"
    target   = "5000"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "5m"

  http_check {
    url     = "https://httpbin.org/status/200"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_withRetry(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"

  http_check {
    url     = "https://httpbin.org/status/200"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  retry {
    count    = 3
    interval = "500ms"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_withMonitor(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
	name     = %[1]q
	enabled  = true
	interval = "1m"

	http_check {
		url     = "https://httpbin.org/status/200"
		method  = "GET"
		timeout = "10s"
	}

	assertion {
		source   = "statusCode"
		operator = "eq"
		target   = "200"
	}

	monitor {
		monitor_name            = "Monitor for %[1]s"
		severity                = "S1"
		no_data_state           = "Alerting"
		execution_error_state   = "Alerting"
		renotification_interval = "1h"

		evaluation_interval {
			interval    = "1m"
			pending_for = "0s"
		}
	}
}
`, name)
}

func testAccSyntheticTestResourceConfig_withMonitorUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
	name     = %[1]q
	enabled  = true
	interval = "1m"

	http_check {
		url     = "https://httpbin.org/status/200"
		method  = "GET"
		timeout = "10s"
	}

	assertion {
		source   = "statusCode"
		operator = "eq"
		target   = "200"
	}

	monitor {
		monitor_name            = "Monitor for %[1]s"
		severity                = "S2"
		no_data_state           = "OK"
		execution_error_state   = "Alerting"
		renotification_interval = "4h"

		evaluation_interval {
			interval    = "1m"
			pending_for = "5m"
		}
	}
}
`, name)
}

func testAccSyntheticTestResourceConfig_withMonitorMinimal(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
	name     = %[1]q
	interval = "1m"

	http_check {
		url     = "https://httpbin.org/status/200"
		method  = "GET"
		timeout = "10s"
	}

	assertion {
		source   = "statusCode"
		operator = "eq"
		target   = "200"
	}

	monitor {
		severity = "S2"
	}
}
`, name)
}

// --- Test check helpers ---

func testAccCheckSyntheticTestResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Synthetic Test ID is set")
		}

		return nil
	}
}

// --- SSL check tests ---

func TestAccSyntheticTestResource_sslBasic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "enabled", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.port", "443"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "ssl"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.operator", "exists"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.target", "true"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_synthetic_test.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ssl_check.sni"},
			},
		},
	})
}

func TestAccSyntheticTestResource_sslFull(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl-full")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_sslFull(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.port", "443"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.verify", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.min_version", "1.2"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.sni", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.timeout", "10s"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_sslUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl")
	updatedName := name + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_sslUpdated(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "5m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "github.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.port", "443"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_sslTimeoutUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl-timeout")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.port", "443"),
					resource.TestCheckNoResourceAttr("groundcover_synthetic_test.test", "ssl_check.timeout"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_sslWithTimeout(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.port", "443"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.timeout", "5s"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_sslApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl-loop")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with sni omitted (server defaults it to host)
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 2: Re-apply same config — should detect no changes
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 3: One more time to be sure
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_conflictingChecks(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-conflict")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSyntheticTestResourceConfig_conflicting(name),
				ExpectError: regexp.MustCompile(`Conflicting check configuration`),
			},
		},
	})
}

func testAccCheckSyntheticTestResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Synthetic Test ID is set")
		}

		ctx := context.Background()

		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		backendID := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}

		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, backendID)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		if err := client.DeleteSyntheticTest(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete Synthetic Test: %v", err)
		}

		return nil
	}
}

// --- SSL config helpers ---

func testAccSyntheticTestResourceConfig_sslBasic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  ssl_check {
    host = "google.com"
    port = 443
  }

  assertion {
    source   = "ssl"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_sslFull(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"

  ssl_check {
    host        = "google.com"
    port        = 443
    verify      = true
    min_version = "1.2"
    sni         = "google.com"
    timeout     = "10s"
  }

  assertion {
    source   = "ssl"
    operator = "exists"
    target   = "true"
    property = "certificateValid"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_sslUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "5m"

  ssl_check {
    host = "github.com"
    port = 443
  }

  assertion {
    source   = "ssl"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_sslWithTimeout(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  ssl_check {
    host    = "google.com"
    port    = 443
    timeout = "5s"
  }

  assertion {
    source   = "ssl"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func TestAccSyntheticTestResource_notificationMethod(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-notif")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with notificationRoutes method
			{
				Config: testAccSyntheticTestResourceConfig_notificationRoutes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.notification_method", "notificationRoutes"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.disable_renotification", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_synthetic_test.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"monitor"},
			},
		},
	})
}

func testAccSyntheticTestResourceConfig_notificationRoutes(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  http_check {
    url    = "https://httpbin.org/status/200"
    method = "GET"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  monitor {
    severity                = "S3"
    notification_method     = "notificationRoutes"
    disable_renotification  = true
  }
}
`, name)
}

func TestAccSyntheticTestResource_notificationValidation_connectedAppsWithoutApps(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-val")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"
  http_check {
    url    = "https://httpbin.org/status/200"
    method = "GET"
  }
  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
  monitor {
    notification_method = "connectedApps"
  }
}
`, name),
				ExpectError: regexp.MustCompile(`"connected_apps" must be set and non-empty`),
			},
		},
	})
}

func TestAccSyntheticTestResource_notificationValidation_appsWithoutMethod(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-val")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"
  http_check {
    url    = "https://httpbin.org/status/200"
    method = "GET"
  }
  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
  monitor {
    notification_method = "notificationRoutes"
    connected_apps      = ["app-1"]
  }
}
`, name),
				ExpectError: regexp.MustCompile(`(?s)connected_apps.*can only be set`),
			},
		},
	})
}

func TestAccSyntheticTestResource_notificationValidation_invalidStatusFilter(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-val")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"
  http_check {
    url    = "https://httpbin.org/status/200"
    method = "GET"
  }
  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
  monitor {
    notification_method = "connectedApps"
    connected_apps      = ["app-1"]
    status_filters      = ["Alerting", "InvalidStatus"]
  }
}
`, name),
				ExpectError: regexp.MustCompile(`Invalid status_filter "InvalidStatus"`),
			},
		},
	})
}

func TestAccSyntheticTestResource_notificationMethodConnectedApps(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ca")

	// Kept sequential (resource.Test, not resource.ParallelTest) because the
	// synthetic API in `connectedApps` mode auto-creates an internal
	// notification_route that references the connected_app. Cleanup of that
	// internal route on synthetic destroy is eventually consistent on the dev
	// backend — under parallel load, the connected_app delete races ahead of
	// the route purge and fails with 409 "referenced by notification routes".
	// Running this one test sequentially gives the backend the breathing room
	// to settle, with negligible wall-time impact (single ~30s test).
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with connectedApps
			{
				Config: testAccSyntheticTestResourceConfig_connectedApps(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.notification_method", "connectedApps"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.connected_apps.#", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.status_filters.#", "2"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.status_filters.0", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.status_filters.1", "Resolved"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.disable_renotification", "true"),
				),
			},
			// Step 2: Switch to notificationRoutes — clears connected app reference so destroy succeeds
			{
				Config: testAccSyntheticTestResourceConfig_connectedAppsToRoutes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.notification_method", "notificationRoutes"),
				),
			},
		},
	})
}

func testAccSyntheticTestResourceConfig_connectedApps(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-app"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  http_check {
    url    = "https://httpbin.org/status/200"
    method = "GET"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  monitor {
    severity                = "S3"
    notification_method     = "connectedApps"
    connected_apps          = [groundcover_connected_app.test.id]
    status_filters          = ["Alerting", "Resolved"]
    disable_renotification  = true
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_connectedAppsToRoutes(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-app"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  http_check {
    url    = "https://httpbin.org/status/200"
    method = "GET"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  monitor {
    severity            = "S3"
    notification_method = "notificationRoutes"
  }
}
`, name)
}

func TestAccSyntheticTestResource_notificationMethodApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-notif-loop")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_notificationRoutes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.notification_method", "notificationRoutes"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.disable_renotification", "true"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_notificationRoutes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.notification_method", "notificationRoutes"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_notificationRoutes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.notification_method", "notificationRoutes"),
				),
			},
		},
	})
}

// --- FollowRedirects / AllowInsecure tests ---

func TestAccSyntheticTestResource_followRedirectsAndAllowInsecure(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-bools")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with both booleans set to true
			{
				Config: testAccSyntheticTestResourceConfig_withBooleans(name, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.follow_redirects", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.allow_insecure", "true"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to false
			{
				Config: testAccSyntheticTestResourceConfig_withBooleans(name, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.follow_redirects", "false"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.allow_insecure", "false"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_followRedirectsApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-bool-loop")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_withBooleans(name, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.follow_redirects", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.allow_insecure", "false"),
				),
			},
			// Re-apply same config — should detect no changes
			{
				Config: testAccSyntheticTestResourceConfig_withBooleans(name, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.follow_redirects", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.allow_insecure", "false"),
				),
			},
			// One more time
			{
				Config: testAccSyntheticTestResourceConfig_withBooleans(name, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.follow_redirects", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.allow_insecure", "false"),
				),
			},
		},
	})
}

func testAccSyntheticTestResourceConfig_withBooleans(name string, followRedirects, allowInsecure bool) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  http_check {
    url              = "https://httpbin.org/status/200"
    method           = "GET"
    timeout          = "10s"
    follow_redirects = %[2]t
    allow_insecure   = %[3]t
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
}
`, name, followRedirects, allowInsecure)
}

func testAccSyntheticTestResourceConfig_conflicting(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"

  http_check {
    url    = "https://example.com"
    method = "GET"
  }

  ssl_check {
    host = "example.com"
    port = 443
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
}
`, name)
}

// --- TCP check tests ---

func TestAccSyntheticTestResource_tcpBasic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "enabled", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.port", "443"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "tcp"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.operator", "exists"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.target", "true"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_tcpFull(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp-full")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpFull(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.port", "443"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.send", "PING"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.expect_response", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.receive_max_bytes", "1024"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_tcpUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp")
	updatedName := name + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_tcpUpdated(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "5m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "github.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.port", "443"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_tcpApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp-loop")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 2: Re-apply same config — should detect no changes
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 3: One more time to be sure
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_tcpTimeoutUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp-timeout")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.port", "443"),
					resource.TestCheckNoResourceAttr("groundcover_synthetic_test.test", "tcp_check.timeout"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_tcpWithTimeout(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.port", "443"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.timeout", "5s"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_tcpSendUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp-send")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.port", "443"),
					resource.TestCheckNoResourceAttr("groundcover_synthetic_test.test", "tcp_check.send"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_tcpWithSend(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.port", "443"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.send", "PING"),
				),
			},
		},
	})
}

// --- TCP config helpers ---

func testAccSyntheticTestResourceConfig_tcpBasic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  tcp_check {
    host = "google.com"
    port = 443
  }

  assertion {
    source   = "tcp"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_tcpFull(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"

  tcp_check {
    host              = "google.com"
    port              = 443
    send              = "PING"
    expect_response   = true
    receive_max_bytes = 1024
  }

  assertion {
    source   = "tcp"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_tcpWithSend(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  tcp_check {
    host = "google.com"
    port = 443
    send = "PING"
  }

  assertion {
    source   = "tcp"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_tcpWithTimeout(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  tcp_check {
    host    = "google.com"
    port    = 443
    timeout = "5s"
  }

  assertion {
    source   = "tcp"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func TestAccSyntheticTestResource_sslPropertyAssertions(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl-prop")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_sslPropertyAssertions(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "2"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "ssl"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.property", "certificateValid"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.operator", "eq"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.target", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.1.source", "ssl"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.1.property", "certificateExpiresIn"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.1.operator", "gt"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.1.target", "30"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				ResourceName:            "groundcover_synthetic_test.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ssl_check.sni"},
			},
		},
	})
}

func testAccSyntheticTestResourceConfig_sslPropertyAssertions(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  ssl_check {
    host = "google.com"
    port = 443
  }

  assertion {
    source   = "ssl"
    property = "certificateValid"
    operator = "eq"
    target   = "true"
  }

  assertion {
    source   = "ssl"
    property = "certificateExpiresIn"
    operator = "gt"
    target   = "30"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_tcpUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "5m"

  tcp_check {
    host = "github.com"
    port = 443
  }

  assertion {
    source   = "tcp"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

// --- DNS check tests ---

func TestAccSyntheticTestResource_dnsBasic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-dns")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "enabled", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.record_type", "A"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "dnsAnswer"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.operator", "exists"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.target", "true"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_dnsFull(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-dns-full")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_dnsFull(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.port", "53"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.resolver", "8.8.8.8"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.record_type", "A"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.dnssec", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.timeout", "10s"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_dnsUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-dns")
	updatedName := name + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_dnsUpdated(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "5m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "github.com"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_dnsApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-dns-loop")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 2: Re-apply same config — should detect no changes
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 3: One more time to be sure
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_dnsTimeoutUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-dns-timeout")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "google.com"),
					resource.TestCheckNoResourceAttr("groundcover_synthetic_test.test", "dns_check.timeout"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_dnsWithTimeout(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "google.com"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.timeout", "5s"),
				),
			},
		},
	})
}

// --- DNS config helpers ---

func testAccSyntheticTestResourceConfig_dnsBasic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  dns_check {
    domain      = "google.com"
    record_type = "A"
  }

  assertion {
    source   = "dnsAnswer"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_dnsFull(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"

  dns_check {
    domain      = "google.com"
    port        = 53
    resolver    = "8.8.8.8"
    record_type = "A"
    dnssec      = true
    timeout     = "10s"
  }

  assertion {
    source   = "dnsAnswer"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_dnsUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "5m"

  dns_check {
    domain      = "github.com"
    record_type = "A"
  }

  assertion {
    source   = "dnsAnswer"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_dnsWithTimeout(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  dns_check {
    domain      = "google.com"
    record_type = "A"
    timeout     = "5s"
  }

  assertion {
    source   = "dnsAnswer"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}
