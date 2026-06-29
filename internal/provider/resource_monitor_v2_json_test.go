// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
