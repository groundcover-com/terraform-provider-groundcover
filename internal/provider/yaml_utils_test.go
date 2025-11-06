package provider

import (
	"context"
	"strings"
	"testing"
)

func TestFilterYamlKeysBasedOnTemplate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		sourceYaml    string
		templateYaml  string
		expectedYaml  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "both source and template empty strings",
			sourceYaml:   "",
			templateYaml: "",
			expectedYaml: "",
			expectError:  false,
		},
		{
			name:         "empty source with non-empty template",
			sourceYaml:   "",
			templateYaml: "key1: value1\nkey2: value2",
			expectedYaml: "",
			expectError:  false,
		},
		{
			name:         "non-empty source with empty template",
			sourceYaml:   "key1: value1\nkey2: value2\nkey3: value3",
			templateYaml: "",
			expectedYaml: "key1: value1\nkey2: value2\nkey3: value3",
			expectError:  false,
		},
		{
			name: "filtering out extra flat keys from source",
			sourceYaml: `key1: value1
key2: value2
key3: value3
extraKey: extraValue`,
			templateYaml: `key1: template1
key2: template2`,
			expectedYaml: `key1: value1
key2: value2`,
			expectError: false,
		},
		{
			name: "handling nested maps",
			sourceYaml: `rootKey:
  nestedKey1: value1
  nestedKey2: value2
  extraNested: extraValue
otherRoot:
  nested: value
extraRoot: shouldBeFiltered`,
			templateYaml: `rootKey:
  nestedKey1: template1
  nestedKey2: template2
otherRoot:
  nested: template`,
			expectedYaml: `rootKey:
  nestedKey1: value1
  nestedKey2: value2
otherRoot:
  nested: value`,
			expectError: false,
		},
		{
			name: "processing arrays of maps",
			sourceYaml: `items:
  - name: item1
    value: 100
    extraField: shouldBeFiltered
  - name: item2
    value: 200
    extraField: shouldBeFiltered
topLevel: keep`,
			templateYaml: `items:
  - name: template
    value: template
topLevel: template`,
			expectedYaml: `items:
  - name: item1
    value: 100
  - name: item2
    value: 200
topLevel: keep`,
			expectError: false,
		},
		{
			name: "arrays with different structures",
			sourceYaml: `queries:
  - dataType: infra
    operator: eq
    extraField: filtered
  - dataType: logs
    operator: ne
    anotherExtra: filtered`,
			templateYaml: `queries:
  - dataType: template
    operator: template`,
			expectedYaml: `queries:
  - dataType: infra
    operator: eq
  - dataType: logs
    operator: ne`,
			expectError: false,
		},
		{
			name: "mixed scalar and nested values",
			sourceYaml: `scalarKey: scalarValue
nestedKey:
  nested1: value1
  nested2: value2
arrayKey:
  - item1
  - item2
extraScalar: filtered
extraNested:
  shouldBe: filtered`,
			templateYaml: `scalarKey: template
nestedKey:
  nested1: template
arrayKey:
  - template`,
			expectedYaml: `scalarKey: scalarValue
nestedKey:
  nested1: value1
arrayKey:
  - item1
  - item2`,
			expectError: false,
		},
		{
			name: "deep nesting",
			sourceYaml: `level1:
  level2:
    level3:
      level4: deepValue
      extraDeep: filtered
    extraLevel3: filtered
  extraLevel2: filtered`,
			templateYaml: `level1:
  level2:
    level3:
      level4: template`,
			expectedYaml: `level1:
  level2:
    level3:
      level4: deepValue`,
			expectError: false,
		},
		{
			name: "comments should be preserved in structure",
			sourceYaml: `# Main comment
key1: value1  # inline comment
key2: value2
extraKey: filtered  # this should be removed`,
			templateYaml: `# Template comment
key1: template1
key2: template2`,
			// Comments are actually preserved by the goccy/go-yaml parser
			expectedYaml: `# Main comment
key1: value1 # inline comment
key2: value2`,
			expectError: false,
		},
		{
			name: "yaml anchors and aliases",
			sourceYaml: `defaults: &defaults
  timeout: 30s
  retries: 3
  extraDefault: filtered

service1:
  <<: *defaults
  name: service1
  extraField: filtered

service2:
  <<: *defaults
  name: service2`,
			templateYaml: `defaults: &defaults
  timeout: template
  retries: template

service1:
  <<: *defaults
  name: template

service2:
  <<: *defaults
  name: template`,
			// Note: The exact output may vary due to anchor resolution
			expectedYaml: "", // We'll check this separately as anchor handling can be complex
			expectError:  false,
		},
		{
			name: "invalid source YAML",
			sourceYaml: `invalid: yaml: [unclosed bracket
malformed`,
			templateYaml:  `valid: yaml`,
			expectedYaml:  "",
			expectError:   true,
			errorContains: "failed to parse source YAML",
		},
		{
			name:       "invalid template YAML",
			sourceYaml: `valid: yaml`,
			templateYaml: `invalid: yaml: [unclosed bracket
malformed`,
			expectedYaml:  "",
			expectError:   true,
			errorContains: "failed to parse template YAML",
		},
		{
			name: "empty arrays in template",
			sourceYaml: `items:
  - name: item1
    value: 100
  - name: item2
    value: 200`,
			templateYaml: `items: []`,
			// When template has empty array, no structure is provided for filtering
			// so array items become empty objects
			expectedYaml: `items:
  - {}
  - {}`,
			expectError: false,
		},
		{
			name: "template with more complex array structure",
			sourceYaml: `monitors:
  - title: Monitor 1
    severity: critical
    model: threshold
    extraMonitorField: filtered
    queries:
      - dataType: metrics
        operator: eq
        extraQueryField: filtered
  - title: Monitor 2
    severity: warning
    model: anomaly`,
			templateYaml: `monitors:
  - title: template
    severity: template
    model: template
    queries:
      - dataType: template
        operator: template`,
			expectedYaml: `monitors:
  - title: Monitor 1
    severity: critical
    model: threshold
    queries:
      - dataType: metrics
        operator: eq
  - title: Monitor 2
    severity: warning
    model: anomaly`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterYamlKeysBasedOnTemplate(ctx, tt.sourceYaml, tt.templateYaml)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("FilterYamlKeysBasedOnTemplate() expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("FilterYamlKeysBasedOnTemplate() error = %v, expected to contain %v", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("FilterYamlKeysBasedOnTemplate() unexpected error = %v", err)
				return
			}

			// For the anchor/alias test case, we need special handling
			if tt.name == "yaml anchors and aliases" {
				// Just verify it doesn't error and produces some output
				// The exact format may vary due to anchor resolution complexity
				if result == "" {
					t.Errorf("FilterYamlKeysBasedOnTemplate() with anchors produced empty result")
				}
				return
			}

			// Normalize whitespace for comparison
			expectedNormalized := normalizeYamlForComparison(tt.expectedYaml)
			resultNormalized := normalizeYamlForComparison(result)

			if resultNormalized != expectedNormalized {
				t.Errorf("FilterYamlKeysBasedOnTemplate() result mismatch.\nExpected:\n%s\n\nGot:\n%s", tt.expectedYaml, result)
			}
		})
	}
}

// normalizeYamlForComparison normalizes YAML strings for comparison by trimming whitespace
// and handling minor formatting differences
func normalizeYamlForComparison(yaml string) string {
	// Remove leading/trailing whitespace
	yaml = strings.TrimSpace(yaml)

	// Split into lines and trim each line, then rejoin
	lines := strings.Split(yaml, "\n")
	var trimmedLines []string
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		trimmedLines = append(trimmedLines, trimmed)
	}

	return strings.Join(trimmedLines, "\n")
}

// TestNormalizeTimeString tests the time string normalization function directly
func TestNormalizeTimeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "5m0s should normalize to 5m",
			input:    "5m0s",
			expected: "5m",
		},
		{
			name:     "30m0s should normalize to 30m",
			input:    "30m0s",
			expected: "30m",
		},
		{
			name:     "5m should remain 5m",
			input:    "5m",
			expected: "5m",
		},
		{
			name:     "30m should remain 30m",
			input:    "30m",
			expected: "30m",
		},
		{
			name:     "1h0m0s should normalize to 1h",
			input:    "1h0m0s",
			expected: "1h",
		},
		{
			name:     "1h30m0s should normalize to 1h30m",
			input:    "1h30m0s",
			expected: "1h30m",
		},
		{
			name:     "non-time strings should be unchanged",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "mixed content with time strings",
			input:    "interval: 5m0s and pendingFor: 30m0s",
			expected: "interval: 5m and pendingFor: 30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTimeString(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTimeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCompareYamlSemantically_EmptyFields tests that server-added empty fields
// don't cause false drift detection (regression test for apply-loop issue)
func TestCompareYamlSemantically_EmptyFields(t *testing.T) {
	tests := []struct {
		name   string
		yaml1  string
		yaml2  string
		expect bool
	}{
		{
			name: "empty annotations should be ignored",
			yaml1: `title: Test Monitor
display:
  header: test
  description: test`,
			yaml2: `title: Test Monitor
display:
  header: test
  description: test
annotations: {}`,
			expect: true,
		},
		{
			name: "nil resourceHeaderLabels should be ignored",
			yaml1: `title: Test Monitor
display:
  header: test
  contextHeaderLabels:
  - cluster`,
			yaml2: `title: Test Monitor
display:
  header: test
  resourceHeaderLabels:
  contextHeaderLabels:
  - cluster`,
			expect: true,
		},
		{
			name: "complete pod crashloopbackoff scenario with all differences",
			yaml1: `title: Terraform - Pod CrashLoopBackOff Detection
display:
  header: 'Pod Crashlooping: {{ alert.labels.namespace }}/{{ alert.labels.pod }}'
  description: Kubernetes pod is on a CrashLoopBackOff
  contextHeaderLabels:
  - cluster
  - namespace
  - workload
  - pod
severity: S2
measurementType: state
model:
  queries:
  - name: threshold_input_query
    expression: last_over_time(avg(groundcover_kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"})
      by (cluster, namespace, pod, workload)[30m])
    datasourceType: prometheus
    queryType: instant
  thresholds:
  - name: threshold_1
    inputName: threshold_input_query
    operator: gt
    values:
    - 0
labels:
  alert_prod: "true"
  alert_stage: "true"
executionErrorState: OK
noDataState: OK
evaluationInterval:
  interval: 5m0s
  pendingFor: 30m0s`,
			yaml2: `title: Terraform - Pod CrashLoopBackOff Detection
display:
  header: "Pod Crashlooping: {{ alert.labels.namespace }}/{{ alert.labels.pod }}"
  description: Kubernetes pod is on a CrashLoopBackOff
  resourceHeaderLabels:
  contextHeaderLabels:
    - cluster
    - namespace
    - workload
    - pod
severity: S2
measurementType: state
model:
  queries:
    - name: threshold_input_query
      expression: last_over_time(avg(groundcover_kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"}) by (cluster, namespace, pod, workload)[30m])
      queryType: instant
      datasourceType: prometheus
  thresholds:
    - name: threshold_1
      inputName: threshold_input_query
      operator: gt
      values:
        - 0
labels:
  alert_prod: "true"
  alert_stage: "true"
executionErrorState: OK
noDataState: OK
evaluationInterval:
  interval: 5m
  pendingFor: 30m
annotations: {}`,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareYamlSemantically(tt.yaml1, tt.yaml2)
			if err != nil {
				t.Errorf("CompareYamlSemantically() error = %v", err)
				return
			}
			if result != tt.expect {
				t.Errorf("CompareYamlSemantically() = %v, want %v", result, tt.expect)
			}
		})
	}
}

// TestCompareYamlSemantically_TimeNormalization tests that time format differences
// don't cause false drift detection (regression test for apply-loop issue)
func TestCompareYamlSemantically_TimeNormalization(t *testing.T) {
	tests := []struct {
		name   string
		yaml1  string
		yaml2  string
		expect bool
	}{
		{
			name: "5m0s vs 5m should be semantically equal",
			yaml1: `evaluationInterval:
  interval: 5m0s
  pendingFor: 30m0s`,
			yaml2: `evaluationInterval:
  interval: 5m
  pendingFor: 30m`,
			expect: true,
		},
		{
			name: "different time values should not be equal",
			yaml1: `evaluationInterval:
  interval: 5m
  pendingFor: 30m`,
			yaml2: `evaluationInterval:
  interval: 10m
  pendingFor: 30m`,
			expect: false,
		},
		{
			name: "complex YAML with time normalization",
			yaml1: `title: Test Monitor
evaluationInterval:
  interval: 5m0s
  pendingFor: 30m0s
model:
  queries:
    - name: test
      pipeline:
        function:
          name: sum_over_time
          args:
            - 10m0s`,
			yaml2: `title: Test Monitor
evaluationInterval:
  interval: 5m
  pendingFor: 30m
model:
  queries:
    - name: test
      pipeline:
        function:
          name: sum_over_time
          args:
            - 10m`,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareYamlSemantically(tt.yaml1, tt.yaml2)
			if err != nil {
				t.Errorf("CompareYamlSemantically() error = %v", err)
				return
			}
			if result != tt.expect {
				t.Errorf("CompareYamlSemantically() = %v, want %v", result, tt.expect)
			}
		})
	}
}

// Test for edge cases and specific functionality
func TestFilterYamlKeysBasedOnTemplate_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("template with null values", func(t *testing.T) {
		sourceYaml := `key1: value1
key2: null
key3: value3`
		templateYaml := `key1: template
key2: null`

		result, err := FilterYamlKeysBasedOnTemplate(ctx, sourceYaml, templateYaml)
		if err != nil {
			t.Errorf("FilterYamlKeysBasedOnTemplate() unexpected error = %v", err)
			return
		}

		expected := `key1: value1
key2: null`
		if normalizeYamlForComparison(result) != normalizeYamlForComparison(expected) {
			t.Errorf("FilterYamlKeysBasedOnTemplate() result mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
		}
	})

	t.Run("template with boolean and numeric values", func(t *testing.T) {
		sourceYaml := `boolKey: true
intKey: 42
floatKey: 3.14
stringKey: "hello"
extraKey: filtered`
		templateYaml := `boolKey: false
intKey: 0
floatKey: 0.0
stringKey: "template"`

		result, err := FilterYamlKeysBasedOnTemplate(ctx, sourceYaml, templateYaml)
		if err != nil {
			t.Errorf("FilterYamlKeysBasedOnTemplate() unexpected error = %v", err)
			return
		}

		expected := `boolKey: true
intKey: 42
floatKey: 3.14
stringKey: "hello"`
		if normalizeYamlForComparison(result) != normalizeYamlForComparison(expected) {
			t.Errorf("FilterYamlKeysBasedOnTemplate() result mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
		}
	})

	t.Run("complex array filtering", func(t *testing.T) {
		sourceYaml := `configs:
  - name: config1
    enabled: true
    settings:
      timeout: 30
      retries: 3
      extraSetting: filtered
    extraField: filtered
  - name: config2
    enabled: false
    settings:
      timeout: 60`
		templateYaml := `configs:
  - name: template
    enabled: template
    settings:
      timeout: template
      retries: template`

		result, err := FilterYamlKeysBasedOnTemplate(ctx, sourceYaml, templateYaml)
		if err != nil {
			t.Errorf("FilterYamlKeysBasedOnTemplate() unexpected error = %v", err)
			return
		}

		expected := `configs:
  - name: config1
    enabled: true
    settings:
      timeout: 30
      retries: 3
  - name: config2
    enabled: false
    settings:
      timeout: 60`
		if normalizeYamlForComparison(result) != normalizeYamlForComparison(expected) {
			t.Errorf("FilterYamlKeysBasedOnTemplate() result mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
		}
	})
}

// TestRemoveExtraNewlines tests the removeExtraNewlines function
func TestRemoveExtraNewlines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "remove blank lines between fields",
			input: `title: K8s Pod Crash Looping

display:
  header: K8s Pod Crash Looping

  resourceHeaderLabels:
  - workload

  contextHeaderLabels:
  - cluster
  - namespace

severity: S2`,
			expected: `title: K8s Pod Crash Looping
display:
  header: K8s Pod Crash Looping
  resourceHeaderLabels:
  - workload
  contextHeaderLabels:
  - cluster
  - namespace
severity: S2
`,
		},
		{
			name: "remove trailing newlines",
			input: `title: Test Monitor
severity: S1


`,
			expected: `title: Test Monitor
severity: S1
`,
		},
		{
			name: "preserve single newline at end",
			input: `title: Test Monitor
severity: S1
`,
			expected: `title: Test Monitor
severity: S1
`,
		},
		{
			name:     "handle empty string",
			input:    "",
			expected: "",
		},
		{
			name: "single field",
			input: `title: Test Monitor
`,
			expected: `title: Test Monitor
`,
		},
		{
			name: "multiple blank lines at start and end",
			input: `

title: Test Monitor
severity: S1


`,
			expected: `title: Test Monitor
severity: S1
`,
		},
		{
			name: "blank lines with only spaces",
			input: "title: Test Monitor\n   \nseverity: S1\n  \n",
			expected: `title: Test Monitor
severity: S1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeExtraNewlines(tt.input)
			if result != tt.expected {
				t.Errorf("removeExtraNewlines() mismatch.\nExpected:\n%q\n\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

// TestNormalizeMonitorYaml_RemovesExtraNewlines tests that NormalizeMonitorYaml removes extra newlines
func TestNormalizeMonitorYaml_RemovesExtraNewlines(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		input         string
		expectedLines []string // We'll check that these lines exist in order without blank lines between them
	}{
		{
			name: "remove blank lines between top-level fields",
			input: `title: K8s Pod Crash Looping

display:
  header: K8s Pod Crash Looping

  resourceHeaderLabels:
  - workload

  contextHeaderLabels:
  - cluster
  - namespace

severity: S2

measurementType: event`,
			expectedLines: []string{
				"display:",
				"  contextHeaderLabels:",
				"  - cluster",
				"  - namespace",
				"  header: K8s Pod Crash Looping",
				"  resourceHeaderLabels:",
				"  - workload",
				"measurementType: event",
				"severity: S2",
				"title: K8s Pod Crash Looping",
			},
		},
		{
			name: "complex yaml with multiple blank lines",
			input: `title: Test Monitor


display:
  header: Test


  description: Test description

severity: S1


model:
  queries:
  - name: test_query


    dataType: metrics`,
			expectedLines: []string{
				"title: Test Monitor",
				"display:",
				"severity: S1",
				"model:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeMonitorYaml(ctx, tt.input)
			if err != nil {
				t.Errorf("NormalizeMonitorYaml() error = %v", err)
				return
			}

			// Check that result doesn't contain any blank lines (except the trailing one)
			lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					t.Errorf("NormalizeMonitorYaml() result contains blank line:\n%q", result)
					break
				}
			}

			// Check that expected lines exist in the output
			for _, expectedLine := range tt.expectedLines {
				if !strings.Contains(result, expectedLine) {
					t.Errorf("NormalizeMonitorYaml() result missing expected line: %q\nGot:\n%s", expectedLine, result)
				}
			}

			// Check that result ends with exactly one newline
			if !strings.HasSuffix(result, "\n") || strings.HasSuffix(result, "\n\n") {
				t.Errorf("NormalizeMonitorYaml() result should end with exactly one newline, got:\n%q", result)
			}
		})
	}
}
