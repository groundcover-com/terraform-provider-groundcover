// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

// The backend decodes monitor YAML strictly with camelCase field names, so the
// producer must emit keys from the SDK models' json tags rather than the
// lowercased Go field names that yaml.v3 produces by default.
func TestJSONTagYAMLProducer_UsesJSONTagCasing(t *testing.T) {
	producer := newJSONTagYAMLProducer()

	condition := &models.Condition{
		AdditionalFilter: "some-filter",
		AutoComplete:     true,
		FilterKeys:       []string{"key1"},
		IsNullable:       true,
		Key:              "container_name",
		Origin:           "root",
		Type:             "string",
	}

	var buf bytes.Buffer
	if err := producer.Produce(&buf, condition); err != nil {
		t.Fatalf("Produce returned error: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"additionalFilter:", "autoComplete:", "filterKeys:", "isNullable:"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
	for _, reject := range []string{"additionalfilter:", "autocomplete:", "filterkeys:", "isnullable:"} {
		if strings.Contains(out, reject) {
			t.Errorf("output contains lowercased key %q, got:\n%s", reject, out)
		}
	}
}

// Numbers must survive the JSON round-trip as numeric YAML scalars,
// not quoted strings.
func TestJSONTagYAMLProducer_PreservesNumbers(t *testing.T) {
	producer := newJSONTagYAMLProducer()

	data := struct {
		IntValue   int     `json:"intValue"`
		FloatValue float64 `json:"floatValue"`
	}{IntValue: 5, FloatValue: 2.5}

	var buf bytes.Buffer
	if err := producer.Produce(&buf, data); err != nil {
		t.Fatalf("Produce returned error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "intValue: 5") {
		t.Errorf("expected unquoted integer scalar 'intValue: 5', got:\n%s", out)
	}
	if !strings.Contains(out, "floatValue: 2.5") {
		t.Errorf("expected unquoted float scalar 'floatValue: 2.5', got:\n%s", out)
	}
}

// End-to-end through the real SDK client stack: the YAML body that hits the
// wire for CreateMonitor must use the json-tag (camelCase) key casing for
// nested models like Condition, which only carry json tags.
func TestCreateMonitor_WireBodyUsesJSONTagCasing(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/monitors") {
			capturedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"monitorId":"test-monitor-id"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewSdkClientWrapper(context.Background(), server.URL, "test-key", "test-backend")
	if err != nil {
		t.Fatalf("NewSdkClientWrapper returned error: %v", err)
	}

	title := "wire-casing-test"
	req := &models.CreateMonitorRequest{
		Title:    &title,
		Severity: "critical",
		Model: &models.Model{
			Queries: []*models.BaseQuery{{
				Name:     "query_a",
				DataType: "events",
				Conditions: []*models.Condition{{
					AdditionalFilter: "some-filter",
					AutoComplete:     true,
					FilterKeys:       []string{"key1"},
					IsNullable:       true,
					Key:              "container_name",
					Origin:           "root",
					Type:             "string",
				}},
			}},
		},
	}

	resp, err := client.CreateMonitor(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateMonitor returned error: %v", err)
	}
	if resp.MonitorID != "test-monitor-id" {
		t.Errorf("unexpected monitor ID: %s", resp.MonitorID)
	}

	body := string(capturedBody)
	for _, want := range []string{"additionalFilter:", "autoComplete:", "filterKeys:", "isNullable:"} {
		if !strings.Contains(body, want) {
			t.Errorf("wire body missing camelCase key %q; body:\n%s", want, body)
		}
	}
	for _, reject := range []string{"additionalfilter:", "autocomplete:", "filterkeys:", "isnullable:"} {
		if strings.Contains(body, reject) {
			t.Errorf("wire body contains lowercased key %q; body:\n%s", reject, body)
		}
	}
}

// omitempty json tags must be honored so we don't send zero-value fields
// the user never specified.
func TestJSONTagYAMLProducer_HonorsOmitEmpty(t *testing.T) {
	producer := newJSONTagYAMLProducer()

	condition := &models.Condition{Key: "container_name"}

	var buf bytes.Buffer
	if err := producer.Produce(&buf, condition); err != nil {
		t.Fatalf("Produce returned error: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "additionalFilter") {
		t.Errorf("expected empty additionalFilter to be omitted, got:\n%s", out)
	}
	if !strings.Contains(out, "key: container_name") {
		t.Errorf("expected 'key: container_name' to be present, got:\n%s", out)
	}
}
