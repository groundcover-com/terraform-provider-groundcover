// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConnectedAppDynamicValueToMapSupportsNestedData(t *testing.T) {
	ctx := context.Background()

	tests := map[string]map[string]any{
		"slack": {
			"url": "https://hooks.slack.com/services/TEST/WEBHOOK/URL",
		},
		"pagerduty severity mapping": {
			"routing_key": "a1234567890123456789012345678901",
			"severity_mapping": map[string]any{
				"critical": "critical",
				"error":    "error",
				"warning":  "warning",
				"info":     "info",
			},
		},
		"opsgenie priority mapping": {
			"api_key": "test-opsgenie-api-key-123",
			"region":  "eu",
			"priority_mapping": map[string]any{
				"critical": "P1",
				"error":    "P2",
				"warning":  "P3",
				"info":     "P4",
			},
		},
		"webhook auth and headers": {
			"url":       "https://example.com/webhook",
			"method":    "POST",
			"auth_type": "bearer",
			"api_key":   "test-bearer-token-123",
			"headers": map[string]any{
				"Content-Type":    "application/json",
				"X-Custom-Header": "custom-value",
			},
			"custom_payload": `{"alert": "test-alert", "severity": "critical"}`,
		},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			dynamicValue, err := mapToDynamicValue(ctx, input)
			if err != nil {
				t.Fatalf("mapToDynamicValue() error = %v", err)
			}

			got, diags := dynamicValueToMap(ctx, dynamicValue)
			if diags.HasError() {
				t.Fatalf("dynamicValueToMap() diagnostics = %v", diags)
			}

			if !reflect.DeepEqual(got, input) {
				t.Fatalf("dynamicValueToMap() = %#v, want %#v", got, input)
			}
		})
	}
}

func TestMapConnectedAppResponseToModelPreservesSensitivePlanData(t *testing.T) {
	ctx := context.Background()
	preservedData := map[string]any{
		"routing_key": "a1234567890123456789012345678901",
		"severity_mapping": map[string]any{
			"critical": "critical",
			"error":    "error",
		},
	}

	dynamicValue, err := mapToDynamicValue(ctx, preservedData)
	if err != nil {
		t.Fatalf("mapToDynamicValue() error = %v", err)
	}

	model := connectedAppResourceModel{}
	mapConnectedAppResponseToModel(ctx, &models.ConnectedAppResponse{
		ID:        "app-id",
		Name:      "pagerduty-app",
		Type:      "pagerduty",
		Data:      map[string]any{"redacted": true},
		DataHash:  "20b3664454f5b36a20da19805802a369a9f30793fb646a1de9e39b21a004df4e",
		CreatedBy: "creator@example.com",
		UpdatedBy: "updater@example.com",
	}, &model, dynamicValue)

	got, diags := dynamicValueToMap(ctx, model.Data)
	if diags.HasError() {
		t.Fatalf("dynamicValueToMap() diagnostics = %v", diags)
	}

	if !reflect.DeepEqual(got, preservedData) {
		t.Fatalf("preserved data = %#v, want %#v", got, preservedData)
	}

	// data_hash comes from the API response, not the preserved sensitive data.
	if model.DataHash.ValueString() != "20b3664454f5b36a20da19805802a369a9f30793fb646a1de9e39b21a004df4e" {
		t.Fatalf("data_hash = %q, want the API-provided hash", model.DataHash.ValueString())
	}
}

func TestMapConnectedAppResponseToModelNullsEmptyDataHash(t *testing.T) {
	ctx := context.Background()

	model := connectedAppResourceModel{}
	mapConnectedAppResponseToModel(ctx, &models.ConnectedAppResponse{
		ID:   "app-id",
		Name: "slack-app",
		Type: "slack-webhook",
	}, &model, types.DynamicNull())

	if !model.DataHash.IsNull() {
		t.Fatalf("data_hash = %q, want null when the API returns no hash", model.DataHash.ValueString())
	}
}

func TestConnectedAppDataDrifted(t *testing.T) {
	const hash = "20b3664454f5b36a20da19805802a369a9f30793fb646a1de9e39b21a004df4e"

	tests := map[string]struct {
		stateHash  types.String
		remoteHash string
		want       bool
	}{
		"matching hashes are not drift":   {types.StringValue(hash), hash, false},
		"differing hashes are drift":      {types.StringValue(hash), "deadbeef", true},
		"null state hash is not drift":    {types.StringNull(), hash, false},
		"unknown state hash is not drift": {types.StringUnknown(), hash, false},
		"empty state hash is not drift":   {types.StringValue(""), hash, false},
		"empty remote hash is not drift":  {types.StringValue(hash), "", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := connectedAppDataDrifted(tc.stateHash, tc.remoteHash); got != tc.want {
				t.Fatalf("connectedAppDataDrifted() = %v, want %v", got, tc.want)
			}
		})
	}
}

// When drift is detected the Read flow passes a null preserve value so the redacted remote
// data lands in state, producing a plan diff that restores the configured value on apply.
func TestMapConnectedAppResponseToModelUsesRemoteDataWhenNotPreserving(t *testing.T) {
	ctx := context.Background()

	model := connectedAppResourceModel{}
	mapConnectedAppResponseToModel(ctx, &models.ConnectedAppResponse{
		ID:       "app-id",
		Name:     "pagerduty-app",
		Type:     "pagerduty",
		Data:     map[string]any{"routing_key": "redacted"},
		DataHash: "newhash",
	}, &model, types.DynamicNull())

	got, diags := dynamicValueToMap(ctx, model.Data)
	if diags.HasError() {
		t.Fatalf("dynamicValueToMap() diagnostics = %v", diags)
	}
	want := map[string]any{"routing_key": "redacted"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("data = %#v, want remote value %#v", got, want)
	}
	if model.DataHash.ValueString() != "newhash" {
		t.Fatalf("data_hash = %q, want %q", model.DataHash.ValueString(), "newhash")
	}
}
