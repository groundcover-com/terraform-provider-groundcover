// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
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
}
