// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestSyntheticTestFromSDKResponsePreservesExplicitEmptyHTTPHeaders(t *testing.T) {
	ctx := context.Background()
	emptyHeaders := types.MapValueMust(types.StringType, map[string]attr.Value{})
	state := &syntheticTestResourceModel{
		HTTPCheck: &syntheticHTTPCheckModel{
			Headers: emptyHeaders,
		},
	}

	fromSDKResponse(ctx, &models.SyntheticTestCreateRequest{
		Name:     "empty-headers",
		Enabled:  true,
		Interval: "1m",
		Version:  1,
		CheckConfig: &models.WorkerRequest{
			Request: &models.Request{
				HTTP: &models.HTTPRequest{
					Kind:    "http",
					URL:     "https://example.com",
					Method:  "GET",
					Timeout: "10s",
					Headers: map[string]string{},
				},
			},
		},
	}, state)

	if state.HTTPCheck == nil {
		t.Fatal("expected http_check to be set")
	}
	if state.HTTPCheck.Headers.IsNull() {
		t.Fatal("expected explicit empty headers map to be preserved, got null")
	}
	if got := len(state.HTTPCheck.Headers.Elements()); got != 0 {
		t.Fatalf("expected empty headers map, got %d elements", got)
	}
}

func TestAssertionModelsFromListHandlesUnknownValues(t *testing.T) {
	ctx := context.Background()

	assertions, diags := assertionModelsFromList(ctx, types.ListUnknown(syntheticAssertionObjectType()), true)
	if diags.HasError() {
		t.Fatalf("expected unknown assertion list to be skipped during validation, got diagnostics: %v", diags)
	}
	if len(assertions) != 0 {
		t.Fatalf("expected no assertion models from unknown list, got %d", len(assertions))
	}

	assertionObj, diags := types.ObjectValue(
		syntheticAssertionAttrTypes(),
		map[string]attr.Value{
			"source":   types.StringUnknown(),
			"operator": types.StringValue("eq"),
			"target":   types.StringNull(),
			"property": types.StringUnknown(),
			"severity": types.StringNull(),
		},
	)
	if diags.HasError() {
		t.Fatalf("failed to build assertion object: %v", diags)
	}

	assertionList := types.ListValueMust(syntheticAssertionObjectType(), []attr.Value{assertionObj})
	assertions, diags = assertionModelsFromList(ctx, assertionList, true)
	if diags.HasError() {
		t.Fatalf("expected unknown assertion attributes to be representable, got diagnostics: %v", diags)
	}
	if len(assertions) != 1 {
		t.Fatalf("expected one assertion model, got %d", len(assertions))
	}
	if !assertions[0].Source.IsUnknown() {
		t.Fatal("expected source to remain unknown")
	}
	if !assertions[0].Property.IsUnknown() {
		t.Fatal("expected property to remain unknown")
	}
}

func TestAccSyntheticTestResource_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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
			{
				ResourceName:            "groundcover_synthetic_test.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"http_check.auth"},
			},
		},
	})
}

func TestAccSyntheticTestResource_update(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth")
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "2"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_updated(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "5m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "statusCode"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_applyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
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

	resource.Test(t, resource.TestCase{
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

func TestAccSyntheticTestResource_withMonitor(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-monitor")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
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
			{
				Config: testAccSyntheticTestResourceConfig_withMonitorUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.severity", "S2"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.no_data_state", "OK"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.renotification_interval", "4h"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.evaluation_interval.pending_for", "5m"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_sslApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "ssl_check.host", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_sslBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
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

func TestAccSyntheticTestResource_notificationMethodApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-notif-loop")

	resource.Test(t, resource.TestCase{
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

func TestAccSyntheticTestResource_followRedirectsApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-bool-loop")

	resource.Test(t, resource.TestCase{
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
			{
				Config: testAccSyntheticTestResourceConfig_withBooleans(name, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.follow_redirects", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.allow_insecure", "false"),
				),
			},
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

func TestAccSyntheticTestResource_tcpApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "tcp_check.host", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_tcpBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
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

func TestAccSyntheticTestResource_sslNewStyleApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-ssl-newsrc-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_sslNewStyleAssertions(name),
			},
			{
				Config:   testAccSyntheticTestResourceConfig_sslNewStyleAssertions(name),
				PlanOnly: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_tcpNewStyleApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-tcp-newsrc-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_tcpNewStyleConnection(name),
			},
			{
				Config:   testAccSyntheticTestResourceConfig_tcpNewStyleConnection(name),
				PlanOnly: true,
			},
		},
	})
}

func TestAccSyntheticTestResource_dnsApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-dns-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "dns_check.domain", "google.com"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config: testAccSyntheticTestResourceConfig_dnsBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
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

func testAccSyntheticTestResourceConfig_sslNewStyleAssertions(name string) string {
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
    source   = "certificateValid"
    operator = "eq"
    target   = "true"
  }

  assertion {
    source   = "certificateExpiresIn"
    operator = "gt"
    target   = "30"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_tcpNewStyleConnection(name string) string {
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
    source   = "tcpConnection"
    operator = "exists"
    target   = "true"
  }
}
`, name)
}

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

func testAccCheckSyntheticTestResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Synthetic Test ID is set")
		}

		return nil
	}
}

func testAccCheckSyntheticTestResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Synthetic Test ID is set")
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
			return fmt.Errorf("failed to create client: %v", err)
		}

		if err := client.DeleteSyntheticTest(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("failed to delete Synthetic Test: %v", err)
		}

		return nil
	}
}
