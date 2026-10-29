// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
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

// An empty ID must be treated as not-found before any HTTP call: GET /rules/{id}
// with an empty id redirects to the collection route and returns the full list, which
// callers (e.g. upjet's pre-create existence check) would misread as an existing
// resource. The guard returns ErrNotFound without touching the SDK client, so a
// nil-client wrapper is sufficient here.
func TestGetSyntheticTestEmptyIDReturnsNotFound(t *testing.T) {
	c := &SdkClientWrapper{}
	resp, err := c.GetSyntheticTest(context.Background(), "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for empty id, got %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response for empty id, got %v", resp)
	}
}

func TestSyntheticTestFromSDKResponsePreservesEmptyHTTPHeaders(t *testing.T) {
	ctx := context.Background()
	emptyHeaders := types.MapValueMust(types.StringType, map[string]attr.Value{})

	tests := []struct {
		name        string
		state       *syntheticTestResourceModel
		sdkHeaders  map[string]string
		wantMapNull bool
	}{
		{
			name: "preserves configured empty headers when API omits headers",
			state: &syntheticTestResourceModel{
				HTTPCheck: &syntheticHTTPCheckModel{
					Headers: emptyHeaders,
				},
			},
			wantMapNull: false,
		},
		{
			name:        "reflects empty API headers during import",
			state:       &syntheticTestResourceModel{},
			sdkHeaders:  map[string]string{},
			wantMapNull: false,
		},
		{
			name:        "keeps omitted API headers null during import",
			state:       &syntheticTestResourceModel{},
			wantMapNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fromSDKResponse(ctx, syntheticTestResponseWithHeaders(tt.sdkHeaders), tt.state)

			if tt.state.HTTPCheck == nil {
				t.Fatal("expected http_check to be set")
			}
			if got := tt.state.HTTPCheck.Headers.IsNull(); got != tt.wantMapNull {
				t.Fatalf("expected headers IsNull=%t, got %t", tt.wantMapNull, got)
			}
			if !tt.wantMapNull {
				if got := len(tt.state.HTTPCheck.Headers.Elements()); got != 0 {
					t.Fatalf("expected empty headers map, got %d elements", got)
				}
			}
		})
	}
}

func syntheticTestResponseWithHeaders(headers map[string]string) *models.SyntheticTestCreateRequest {
	return &models.SyntheticTestCreateRequest{
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
					Headers: headers,
				},
			},
		},
	}
}

func TestSyntheticAssertionsToListUsesNullForAbsentAssertions(t *testing.T) {
	for _, assertions := range [][]*models.Assertion{nil, {}} {
		assertionsList := syntheticAssertionsToList(context.Background(), assertions, types.ListNull(syntheticAssertionObjectType()))
		if !assertionsList.IsNull() {
			t.Fatal("expected absent assertions to be represented as a null list")
		}
	}
}

func TestSyntheticAssertionsToListPreservesConfiguredEmptyTarget(t *testing.T) {
	ctx := context.Background()
	apiAssertions := []*models.Assertion{{Source: "responseHeader", Operator: "eq", Property: "x-test", Target: ""}}

	prior := types.ListValueMust(syntheticAssertionObjectType(), []attr.Value{
		types.ObjectValueMust(syntheticAssertionAttrTypes(), map[string]attr.Value{
			"source":   types.StringValue("responseHeader"),
			"operator": types.StringValue("eq"),
			"target":   types.StringValue(""),
			"property": types.StringValue("x-test"),
			"severity": types.StringNull(),
		}),
	})
	got, diags := assertionModelsFromList(ctx, syntheticAssertionsToList(ctx, apiAssertions, prior), false)
	if diags.HasError() || len(got) != 1 {
		t.Fatalf("unexpected result: %v %v", got, diags)
	}
	if got[0].Target.IsNull() || got[0].Target.ValueString() != "" {
		t.Fatalf("expected configured empty target to be preserved, got %v", got[0].Target)
	}
	if !got[0].Severity.IsNull() {
		t.Fatalf("expected absent severity to stay null, got %v", got[0].Severity)
	}

	// Without a prior "" (import, or target never set) an empty API value is still null.
	got, diags = assertionModelsFromList(ctx, syntheticAssertionsToList(ctx, apiAssertions, types.ListNull(syntheticAssertionObjectType())), false)
	if diags.HasError() || len(got) != 1 || !got[0].Target.IsNull() {
		t.Fatalf("expected empty API target with no prior state to be null, got %v %v", got, diags)
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

func TestAccSyntheticTestResource_emptyAssertionTargetApplyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-empty-target")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_emptyAssertionTarget(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.1.target", ""),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			{
				Config:   testAccSyntheticTestResourceConfig_emptyAssertionTarget(name),
				PlanOnly: true,
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

func TestAccSyntheticTestResource_withoutMonitor(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-nomon")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_createMonitor(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "create_monitor", "false"),
					resource.TestCheckNoResourceAttr("groundcover_synthetic_test.test", "monitor.%"),
					testAccCheckSyntheticTestMonitorExists("groundcover_synthetic_test.test", false),
				),
			},
			{
				ResourceName:            "groundcover_synthetic_test.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"http_check.auth"},
			},
			{
				// Flipping to true on update must create the missing monitor.
				Config: testAccSyntheticTestResourceConfig_createMonitor(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "create_monitor", "true"),
					testAccCheckSyntheticTestMonitorExists("groundcover_synthetic_test.test", true),
				),
			},
			{
				// Flipping back to false must delete the monitor.
				Config: testAccSyntheticTestResourceConfig_createMonitor(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "create_monitor", "false"),
					testAccCheckSyntheticTestMonitorExists("groundcover_synthetic_test.test", false),
				),
			},
			{
				// Dropping create_monitor and adding a monitor block must create the monitor.
				Config: testAccSyntheticTestResourceConfig_withMonitor(name),
				Check:  resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "monitor.severity", "S1"),
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

func testAccSyntheticTestResourceConfig_emptyAssertionTarget(name string) string {
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
    source   = "responseHeader"
    operator = "exists"
    property = "content-type"
    target   = ""
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_createMonitor(name string, createMonitor bool) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name           = %[1]q
  interval       = "1m"
  create_monitor = %[2]t

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
`, name, createMonitor)
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

// testAccCheckSyntheticTestMonitorExists asserts via the API whether the test has a bundled monitor.
func testAccCheckSyntheticTestMonitorExists(n string, want bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		ctx := context.Background()
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}
		client, err := NewSdkClientWrapper(ctx, apiURL, os.Getenv("GROUNDCOVER_API_KEY"), os.Getenv("GROUNDCOVER_BACKEND_ID"))
		if err != nil {
			return fmt.Errorf("failed to create client: %v", err)
		}
		got, err := client.GetSyntheticTest(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to get Synthetic Test: %v", err)
		}
		if (got.Monitor != nil) != want {
			return fmt.Errorf("synthetic test %s: monitor exists=%v, want %v", rs.Primary.ID, got.Monitor != nil, want)
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

// Configured labels = {}, an omitted body.type that the provider sends as "raw",
// and monitor_name / issue_summary = "" that the API replaces with its default
// must all survive a read unchanged (BE-3828).
func TestSyntheticTestFromSDKResponsePreservesConfiguredEmpties(t *testing.T) {
	ctx := context.Background()
	resp := syntheticTestResponseWithHeaders(nil)
	resp.CheckConfig.Request.HTTP.Body = &models.Body{Type: "raw", Content: ""}
	resp.Monitor = &models.SyntheticMonitorConfig{MonitorName: "[synthetic] - x", IssueSummary: "[synthetic] - x"}

	state := &syntheticTestResourceModel{
		Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		HTTPCheck: &syntheticHTTPCheckModel{Body: &syntheticHTTPBodyModel{Type: types.StringNull(), Content: types.StringValue("")}},
		Monitor:   &syntheticMonitorModel{MonitorName: types.StringValue(""), IssueSummary: types.StringValue("")},
	}
	fromSDKResponse(ctx, resp, state)

	if state.Labels.IsNull() {
		t.Error("configured labels = {} became null")
	}
	if state.HTTPCheck.Body == nil || !state.HTTPCheck.Body.Type.IsNull() {
		t.Errorf("omitted body.type should stay null, got %v", state.HTTPCheck.Body)
	}
	if state.Monitor.MonitorName.ValueString() != "" || state.Monitor.IssueSummary.ValueString() != "" {
		t.Errorf("configured \"\" replaced by API default: %v / %v", state.Monitor.MonitorName, state.Monitor.IssueSummary)
	}

	imported := &syntheticTestResourceModel{Monitor: &syntheticMonitorModel{MonitorName: types.StringValue("old")}}
	fromSDKResponse(ctx, resp, imported)
	if !imported.Labels.IsNull() || imported.HTTPCheck.Body.Type.ValueString() != "raw" || imported.Monitor.MonitorName.ValueString() != "[synthetic] - x" {
		t.Errorf("unconfigured values should reflect the API: labels=%v type=%v name=%v", imported.Labels, imported.HTTPCheck.Body.Type, imported.Monitor.MonitorName)
	}
}
