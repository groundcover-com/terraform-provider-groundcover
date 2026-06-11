// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"
	"time"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

// The SDK's nested models only carry json tags (camelCase). The user's monitor
// YAML must be parsed honoring those tags — yaml.v3 ignores them and silently
// drops camelCase keys like autoComplete, so the values never reach the API.
func TestParseMonitorYAMLPopulatesCamelCaseConditionFields(t *testing.T) {
	monitorYaml := `title: parse-test
severity: error
measurementType: event
model:
  queries:
  - dataType: events
    name: threshold_input_query
    sqlPipeline:
      filters:
        conditions:
        - key: reason
          origin: root
          type: string
          autoComplete: true
          additionalFilter: test-filter
          isNullable: true
          filterKeys:
          - reason
          filters:
          - op: match
            value: OOMKilled
        operator: and
  thresholds:
  - name: threshold_1
    inputName: threshold_input_query
    operator: gt
    values:
    - 0
evaluationInterval:
  interval: 5m0s
  pendingFor: 1m0s
`

	var req models.CreateMonitorRequest
	if err := parseMonitorYAML([]byte(monitorYaml), &req); err != nil {
		t.Fatalf("parseMonitorYAML returned error: %v", err)
	}

	if req.Model == nil || len(req.Model.Queries) != 1 || req.Model.Queries[0].SQLPipeline == nil {
		t.Fatalf("unexpected request shape: %+v", req)
	}
	conditions := req.Model.Queries[0].SQLPipeline.Filters.Conditions
	if len(conditions) != 1 {
		t.Fatalf("conditions length = %d, want 1", len(conditions))
	}
	cond := conditions[0]
	if !cond.AutoComplete {
		t.Errorf("AutoComplete = false, want true")
	}
	if cond.AdditionalFilter != "test-filter" {
		t.Errorf("AdditionalFilter = %q, want test-filter", cond.AdditionalFilter)
	}
	if !cond.IsNullable {
		t.Errorf("IsNullable = false, want true")
	}
	if len(cond.FilterKeys) != 1 || cond.FilterKeys[0] != "reason" {
		t.Errorf("FilterKeys = %v, want [reason]", cond.FilterKeys)
	}

	if req.EvaluationInterval == nil {
		t.Fatalf("EvaluationInterval = nil")
	}
	if time.Duration(req.EvaluationInterval.Interval) != 5*time.Minute {
		t.Errorf("Interval = %s, want 5m", time.Duration(req.EvaluationInterval.Interval))
	}
	if req.EvaluationInterval.PendingFor == nil || time.Duration(*req.EvaluationInterval.PendingFor) != time.Minute {
		t.Errorf("PendingFor = %v, want 1m", req.EvaluationInterval.PendingFor)
	}
}
