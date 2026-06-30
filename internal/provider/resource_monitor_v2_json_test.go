// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// TestUnitConnectedAppParamsJSONRoundTrip verifies the only non-trivial logic in the JSON
// sibling: parsing the connected_app_params JSON string into the nested map and serializing it
// back must preserve the data semantically.
func TestUnitConnectedAppParamsJSONRoundTrip(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	const input = `{"app-1":{"channels":["C123","C456"]},"app-2":{"channels":["C789"]}}`

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
