package provider

import (
	"context"
	"strings"
	"testing"
)

// TestIssue48_HumanReadableDurationNormalization tests that human-readable durations
// like "10 minutes" are normalized to Go duration format "10m" for semantic comparison.
// This prevents perpetual diffs when the API returns Go-format durations but the user
// writes human-readable durations in their Terraform config.
// See: https://github.com/groundcover-com/terraform-provider-groundcover/issues/48
func TestIssue48_HumanReadableDurationNormalization(t *testing.T) {
	tests := []struct {
		name     string
		userYaml string
		apiYaml  string
	}{
		{
			name:     "10 minutes vs 10m",
			userYaml: "instantRollup: 10 minutes",
			apiYaml:  "instantRollup: 10m",
		},
		{
			name:     "5 minutes vs 5m",
			userYaml: "instantRollup: 5 minutes",
			apiYaml:  "instantRollup: 5m",
		},
		{
			name:     "1 hour vs 1h",
			userYaml: "interval: 1 hour",
			apiYaml:  "interval: 1h",
		},
		{
			name:     "30 seconds vs 30s",
			userYaml: "timeout: 30 seconds",
			apiYaml:  "timeout: 30s",
		},
		{
			name:     "1 minute vs 1m",
			userYaml: "interval: 1 minute",
			apiYaml:  "interval: 1m",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			same, err := CompareYamlSemantically(tc.userYaml, tc.apiYaml)
			if err != nil {
				t.Fatalf("CompareYamlSemantically failed: %v", err)
			}
			if !same {
				t.Errorf("Human-readable duration not treated as semantically equal to Go duration format")
			}
		})
	}
}

// TestIssue48_FullMonitorYamlRoundTrip tests the full round-trip of a monitor YAML
// that uses human-readable durations, simulating the Create -> Read -> Plan cycle.
func TestIssue48_FullMonitorYamlRoundTrip(t *testing.T) {
	ctx := context.Background()

	userYaml := `title: "[Terraform] ArgoCD Secret Key Missing"
display:
  header: "ArgoCD Secret Key Missing on cluster {{labels.cluster}}"
  description: |
    Cluster: {{labels.cluster}}
    This Monitor fires when the ArgoCD server secret key is missing, which may lead to authentication issues.
  resourceHeaderLabels: []
  contextHeaderLabels:
    - cluster
severity: S2
model:
  queries:
    - name: threshold_input_query
      dataType: logs
      expression: '"server.secretkey is missing" | stats by (cluster) count() count_all_result'
      instantRollup: 10 minutes
  thresholds:
    - name: threshold_1
      inputName: threshold_input_query
      operator: gt
      values:
        - 10
labels:
  type: infra
  pagerduty: true
annotations: {}
executionErrorState: OK
noDataState: OK
evaluationInterval:
  interval: 1m
  pendingFor: 0s
notificationSettings:
  renotificationInterval: 4h
  method: notificationRoutes
measurementType: event
isPaused: false`

	// Simulate API response: "10 minutes" -> "10m"
	apiResponseYaml := strings.ReplaceAll(userYaml, "10 minutes", "10m")

	normalized1, err := NormalizeMonitorYaml(ctx, userYaml)
	if err != nil {
		t.Fatalf("NormalizeMonitorYaml(userYaml) failed: %v", err)
	}
	normalized2, err := NormalizeMonitorYaml(ctx, apiResponseYaml)
	if err != nil {
		t.Fatalf("NormalizeMonitorYaml(apiYaml) failed: %v", err)
	}

	same, err := CompareYamlSemantically(normalized1, normalized2)
	if err != nil {
		t.Fatalf("CompareYamlSemantically failed: %v", err)
	}
	if !same {
		t.Errorf("User YAML and API response (with '10m' instead of '10 minutes') are not semantically equal - perpetual diff!")
	}
}

// TestIssue48_BoolStringEquivalence tests that boolean values are treated as
// semantically equal to their string representations, since the API may convert
// boolean label values to quoted strings (e.g., pagerduty: true -> pagerduty: "true").
func TestIssue48_BoolStringEquivalence(t *testing.T) {
	tests := []struct {
		name     string
		userYaml string
		apiYaml  string
	}{
		{
			name:     "bool true vs string true",
			userYaml: "labels:\n  pagerduty: true",
			apiYaml:  "labels:\n  pagerduty: \"true\"",
		},
		{
			name:     "bool false vs string false",
			userYaml: "labels:\n  enabled: false",
			apiYaml:  "labels:\n  enabled: \"false\"",
		},
		{
			name:     "multiple labels with mixed types",
			userYaml: "labels:\n  pagerduty: true\n  type: infra",
			apiYaml:  "labels:\n  pagerduty: \"true\"\n  type: infra",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			same, err := CompareYamlSemantically(tc.userYaml, tc.apiYaml)
			if err != nil {
				t.Fatalf("CompareYamlSemantically failed: %v", err)
			}
			if !same {
				t.Errorf("Bool/string equivalence not handled - causes perpetual diffs when API converts booleans to strings")
			}
		})
	}
}

// TestNormalizeHumanDurations tests the normalizeHumanDurations function directly.
func TestNormalizeHumanDurations(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10 minutes", "10m"},
		{"5 minutes", "5m"},
		{"1 minute", "1m"},
		{"1 hour", "1h"},
		{"2 hours", "2h"},
		{"30 seconds", "30s"},
		{"1 second", "1s"},
		{"no duration here", "no duration here"},
		{"10m", "10m"},
		{"1h30m", "1h30m"},
		{"mixed 5 minutes and 10m", "mixed 5m and 10m"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeHumanDurations(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeHumanDurations(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
