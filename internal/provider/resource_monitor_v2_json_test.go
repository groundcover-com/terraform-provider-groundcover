// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestMonitorV2JsonModelParity guards the typed-clone against drift: groundcover_monitor_v2_json
// reuses monitor_v2's schema and build/read logic, but duplicates the model structs. This test
// fails if monitor_v2 gains/changes a field that the JSON twin doesn't mirror, so a developer
// editing monitor_v2 is told to update the JSON twin too. Combined with copyMonitorV2SharedFields
// (which copies shared fields by reflection), keeping the structs in parity is all that's needed.
func TestMonitorV2JsonModelParity(t *testing.T) {
	// Top-level models match except notification_settings (string-vs-map params is the whole point).
	assertTfsdkFieldParity(t,
		reflect.TypeOf(monitorV2ResourceModel{}),
		reflect.TypeOf(monitorV2JsonResourceModel{}),
		map[string]bool{"notification_settings": true},
	)
	// Notification settings match except connected_app_params (JSON string vs nested map).
	assertTfsdkFieldParity(t,
		reflect.TypeOf(monitorV2NotificationSettingsModel{}),
		reflect.TypeOf(monitorV2JsonNotificationSettingsModel{}),
		map[string]bool{"connected_app_params": true},
	)
}

// TestAccMonitorV2JsonResource_basic is a lean live CRUD check matching the connected_app_json
// precedent. CRUD fully delegates to monitor_v2's tested logic (covered by TestAccMonitorV2Resource);
// this just proves the JSON sibling registers and round-trips end-to-end. The connected_app_params
// JSON path is covered by the unit tests above (a real connected app would be needed here to use it).
func TestAccMonitorV2JsonResource_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-v2-json")
	updatedName := acctest.RandomWithPrefix("test-monitor-v2-json-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorV2JsonResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2_json.test", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2_json.test", "title", name),
					resource.TestCheckResourceAttr("groundcover_monitor_v2_json.test", "display.header", name),
					resource.TestCheckResourceAttr("groundcover_monitor_v2_json.test", "query.type", monitorV2QueryTypeGCQL),
					resource.TestCheckResourceAttr("groundcover_monitor_v2_json.test", "query.data_type", "logs"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2_json.test", "threshold.#", "1"),
				),
			},
			{
				ResourceName:      "groundcover_monitor_v2_json.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccMonitorV2JsonResourceConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_monitor_v2_json.test", "title", updatedName),
				),
			},
			{
				Config:             testAccMonitorV2JsonResourceConfig(updatedName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccMonitorV2JsonResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor_v2_json" "test" {
  title            = %[1]q
  severity         = "critical"
  measurement_type = "event"

  display {
    header = %[1]q
  }

  query {
    type           = "gcql"
    data_type      = "logs"
    expression     = "level:error | stats count() count_all_result"
    instant_rollup = "5m"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [1]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}
`, name)
}

func assertTfsdkFieldParity(t *testing.T, typed, jsonModel reflect.Type, except map[string]bool) {
	t.Helper()
	ft := tfsdkFields(typed)
	fj := tfsdkFields(jsonModel)
	for tag, tt := range ft {
		tj, ok := fj[tag]
		if !ok {
			t.Errorf("%s has tfsdk field %q but %s does not — add it to both models so the JSON twin tracks the typed resource", typed.Name(), tag, jsonModel.Name())
			continue
		}
		// Go field name must match too: copyMonitorV2SharedFields bridges by FieldByName, so a
		// renamed field with the same tfsdk tag would be silently dropped at runtime.
		if tt.name != tj.name {
			t.Errorf("tfsdk field %q Go-name mismatch: %s has %q, %s has %q — copyMonitorV2SharedFields copies by field name, so these must match", tag, typed.Name(), tt.name, jsonModel.Name(), tj.name)
		}
		if !except[tag] && tt.typ != tj.typ {
			t.Errorf("tfsdk field %q type mismatch: %s has %s, %s has %s", tag, typed.Name(), tt.typ, jsonModel.Name(), tj.typ)
		}
	}
	for tag := range fj {
		if _, ok := ft[tag]; !ok {
			t.Errorf("%s has tfsdk field %q but %s does not", jsonModel.Name(), tag, typed.Name())
		}
	}
}

type tfsdkField struct {
	name string
	typ  reflect.Type
}

func tfsdkFields(t reflect.Type) map[string]tfsdkField {
	out := map[string]tfsdkField{}
	for i := 0; i < t.NumField(); i++ {
		if tag, ok := t.Field(i).Tag.Lookup("tfsdk"); ok {
			out[tag] = tfsdkField{name: t.Field(i).Name, typ: t.Field(i).Type}
		}
	}
	return out
}

// TestConnectedAppParamJSONTracksSDK guards the hidden drift seam: connectedAppParamJSON is a
// hand-written mirror of the SDK's ConnectedAppDeliveryOptions, used only to (de)serialize the
// connected_app_params JSON string. If the SDK gains a delivery-option field, the typed resource
// carries it but this struct would silently drop it. Fails when the SDK's json-tag set isn't
// covered here, prompting an update to connectedAppParamJSON and the conversion helpers.
func TestConnectedAppParamJSONTracksSDK(t *testing.T) {
	mirror := jsonTags(reflect.TypeOf(connectedAppParamJSON{}))
	for tag := range jsonTags(reflect.TypeOf(models.ConnectedAppDeliveryOptions{})) {
		if !mirror[tag] {
			t.Errorf("models.ConnectedAppDeliveryOptions has json field %q that connectedAppParamJSON does not mirror — add it here and to the conversion helpers", tag)
		}
	}
}

func jsonTags(t reflect.Type) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		if tag, ok := t.Field(i).Tag.Lookup("json"); ok {
			name, _, _ := strings.Cut(tag, ",")
			if name != "" && name != "-" {
				out[name] = true
			}
		}
	}
	return out
}

// TestUnitConnectedAppParamsJSONRoundTrip verifies the only non-trivial logic in the JSON
// sibling: parsing the connected_app_params JSON string into the nested map and serializing it
// back must preserve the data semantically.
func TestUnitConnectedAppParamsJSONRoundTrip(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	const input = `{"app-1":{"channels":[{"id":"C123","name":"#alerts"},{"id":"C456"}]},"app-2":{"channels":[{"id":"C789"}]}}`

	m := connectedAppParamsJSONToMap(ctx, types.StringValue(input), &diags)
	if diags.HasError() {
		t.Fatalf("JSONToMap returned errors: %v", diags.Errors())
	}
	if m.IsNull() {
		t.Fatal("expected a non-null map for valid params JSON")
	}

	out := connectedAppParamsMapToJSON(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("MapToJSON returned errors: %v", diags.Errors())
	}
	if out.IsNull() {
		t.Fatal("expected a non-null JSON string after round-trip")
	}
	if !connectedAppParamsJSONEqual(input, out.ValueString()) {
		t.Errorf("round-trip changed params: got %q, want semantically %q", out.ValueString(), input)
	}
}

func TestUnitConnectedAppParamsJSONInvalid(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	connectedAppParamsJSONToMap(ctx, types.StringValue(`not json`), &diags)
	if !diags.HasError() {
		t.Fatal("expected an error for invalid params JSON")
	}
}

func TestUnitConnectedAppParamsJSONRejectsNull(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	// A JSON-string literal "null" is not the same as an unset attribute; reject it.
	connectedAppParamsJSONToMap(ctx, types.StringValue(`null`), &diags)
	if !diags.HasError() {
		t.Fatal("expected an error for a literal null payload")
	}
}

func TestUnitConnectedAppParamsJSONNullPassthrough(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	m := connectedAppParamsJSONToMap(ctx, types.StringNull(), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if !m.IsNull() {
		t.Error("null params string should yield a null map")
	}
}

func TestUnitConnectedAppParamsJSONUnknownPassthrough(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	m := connectedAppParamsJSONToMap(ctx, types.StringUnknown(), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if !m.IsUnknown() {
		t.Error("unknown params string should yield an unknown map")
	}
}

func TestUnitConnectedAppParamsJSONRejectsUnknownKeys(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	// "chanels" is a typo for "channels"; strict decoding must reject it rather than drop it.
	connectedAppParamsJSONToMap(ctx, types.StringValue(`{"app-1":{"chanels":["C1"]}}`), &diags)
	if !diags.HasError() {
		t.Fatal("expected an error for an unknown nested key")
	}
}

func TestUnitConnectedAppParamsJSONRejectsChannelWithoutID(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	// The backend requires each channel to carry an `id`; a channel object missing it must error.
	connectedAppParamsJSONToMap(ctx, types.StringValue(`{"app-1":{"channels":[{"name":"#alerts"}]}}`), &diags)
	if !diags.HasError() {
		t.Fatal("expected an error for a channel missing id")
	}
}

func TestUnitConnectedAppParamsEmptyPreserved(t *testing.T) {
	// An authored empty object must round-trip as itself, not flip to null (perpetual diff).
	prior := monitorV2JsonResourceModel{NotificationSettings: &monitorV2JsonNotificationSettingsModel{
		ConnectedAppParams: types.StringValue(`{}`),
	}}
	fresh := monitorV2JsonResourceModel{NotificationSettings: &monitorV2JsonNotificationSettingsModel{
		ConnectedAppParams: types.StringNull(),
	}}
	fresh.preserveParamsIfUnchanged(prior)
	got := fresh.NotificationSettings.ConnectedAppParams
	if got.IsNull() || got.ValueString() != `{}` {
		t.Errorf("authored empty params should be preserved, got %v", got)
	}
}
