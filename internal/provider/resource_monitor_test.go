// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestMonitorResourceRequestsForceIsProvisioned(t *testing.T) {
	monitorYaml := `title: Test Monitor
isProvisioned: false
display:
  header: Test Monitor
severity: critical
model:
  queries:
    - name: test_query
      dataType: metrics
      pipeline:
        metric: up
  thresholds:
    - name: threshold_1
      inputName: test_query
      operator: gt
      values:
        - 1
measurementType: state`

	ctx := context.Background()

	createReq, _, err := buildCreateMonitorRequest(ctx, monitorYaml)
	if err != nil {
		t.Fatalf("buildCreateMonitorRequest() error = %v", err)
	}
	if createReq.IsProvisioned == nil || !*createReq.IsProvisioned {
		t.Fatalf("buildCreateMonitorRequest().IsProvisioned = false, want true")
	}

	updateReq, _, err := buildUpdateMonitorRequest(ctx, monitorYaml)
	if err != nil {
		t.Fatalf("buildUpdateMonitorRequest() error = %v", err)
	}
	if updateReq.Title == nil || *updateReq.Title != "Test Monitor" {
		t.Fatalf("buildUpdateMonitorRequest().Title = %v, want Test Monitor", updateReq.Title)
	}
}

func TestMonitorV2BuildCreateRequestGCQL(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:          types.StringValue(monitorV2QueryTypeGCQL),
		Expression:    types.StringValue(`level:error | stats count() count_all_result`),
		DataType:      types.StringValue("logs"),
		InstantRollup: types.StringValue("5 minutes"),
	})

	req, diags := buildMonitorV2CreateRequest(ctx, &plan)
	if diags.HasError() {
		t.Fatalf("buildMonitorV2CreateRequest() diagnostics: %v", diags)
	}
	if req.IsProvisioned == nil || !*req.IsProvisioned {
		t.Fatalf("CreateMonitorRequest.IsProvisioned = false, want true")
	}
	if req.Model == nil || len(req.Model.Queries) != 1 {
		t.Fatalf("CreateMonitorRequest.Model.Queries length = %d, want 1", len(req.Model.Queries))
	}
	query := req.Model.Queries[0]
	if query.Name != monitorV2DefaultQueryName {
		t.Fatalf("query.Name = %q, want %q", query.Name, monitorV2DefaultQueryName)
	}
	if query.DataType != "logs" {
		t.Fatalf("query.DataType = %q, want logs", query.DataType)
	}
	if query.Expression != `level:error | stats count() count_all_result` {
		t.Fatalf("query.Expression = %q", query.Expression)
	}
	if query.InstantRollup != "5m" {
		t.Fatalf("query.InstantRollup = %q, want 5m", query.InstantRollup)
	}
	requireNoInternalMonitorV2Annotations(t, req.Annotations)
}

func TestMonitorV2BuildCreateRequestGCQLAPM(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:          types.StringValue(monitorV2QueryTypeGCQL),
		Expression:    types.StringValue(`* | stats count() count_all_result`),
		DataType:      types.StringValue("apm"),
		InstantRollup: types.StringValue("5 minutes"),
	})

	req, diags := buildMonitorV2CreateRequest(ctx, &plan)
	if diags.HasError() {
		t.Fatalf("buildMonitorV2CreateRequest() diagnostics: %v", diags)
	}
	query := req.Model.Queries[0]
	if query.DataType != "apm" {
		t.Fatalf("query.DataType = %q, want apm", query.DataType)
	}
	if query.InstantRollup != "5m" {
		t.Fatalf("query.InstantRollup = %q, want 5m", query.InstantRollup)
	}
	requireNoInternalMonitorV2Annotations(t, req.Annotations)
}

func TestMonitorV2BuildCreateRequestMetricsQL(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:       types.StringValue(monitorV2QueryTypeMetricsQL),
		Expression: types.StringValue(`sum(up) by (cluster)`),
		Rollup: &monitorV2RollupModel{
			Function: types.StringValue("last"),
			Time:     types.StringValue("5m"),
		},
	})

	req, diags := buildMonitorV2CreateRequest(ctx, &plan)
	if diags.HasError() {
		t.Fatalf("buildMonitorV2CreateRequest() diagnostics: %v", diags)
	}
	query := req.Model.Queries[0]
	if query.DatasourceType != monitorV2DatasourcePrometheus {
		t.Fatalf("query.DatasourceType = %q, want %q", query.DatasourceType, monitorV2DatasourcePrometheus)
	}
	if query.QueryType != monitorV2QueryTypeInstant {
		t.Fatalf("query.QueryType = %q, want %q", query.QueryType, monitorV2QueryTypeInstant)
	}
	if query.Rollup == nil {
		t.Fatalf("query.Rollup = nil, want rollup")
	}
	if query.Rollup.Function != "last" {
		t.Fatalf("query.Rollup.Function = %q, want last", query.Rollup.Function)
	}
	if time.Duration(query.Rollup.Time) != 5*time.Minute {
		t.Fatalf("query.Rollup.Time = %s, want 5m", time.Duration(query.Rollup.Time))
	}
	requireNoInternalMonitorV2Annotations(t, req.Annotations)
}

func TestMonitorV2BuildCreateRequestRawSQL(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:       types.StringValue(monitorV2QueryTypeRawSQL),
		Expression: types.StringValue(`SELECT count(*) AS count_all_result FROM logs`),
		QueryType:  types.StringValue("range"),
	})

	req, diags := buildMonitorV2CreateRequest(ctx, &plan)
	if diags.HasError() {
		t.Fatalf("buildMonitorV2CreateRequest() diagnostics: %v", diags)
	}
	query := req.Model.Queries[0]
	if query.DatasourceType != monitorV2DatasourceClickhouse {
		t.Fatalf("query.DatasourceType = %q, want %q", query.DatasourceType, monitorV2DatasourceClickhouse)
	}
	if query.QueryType != "range" {
		t.Fatalf("query.QueryType = %q, want range", query.QueryType)
	}
	if query.Rollup != nil {
		t.Fatalf("query.Rollup = %#v, want nil", query.Rollup)
	}
	requireNoInternalMonitorV2Annotations(t, req.Annotations)
}

func TestMonitorV2BuildUpdateRequestFiltersInternalAnnotations(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:       types.StringValue(monitorV2QueryTypeMetricsQL),
		Expression: types.StringValue(`sum(up) by (cluster)`),
		Rollup: &monitorV2RollupModel{
			Function: types.StringValue("last"),
			Time:     types.StringValue("5m"),
		},
	})
	annotations, mapDiags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		monitorV2QueryTypeAnnotationKey: monitorV2QueryTypeMetricsQL,
		monitorV2DataTypeAnnotationKey:  "metrics",
		"team":                          "platform",
	})
	if mapDiags.HasError() {
		t.Fatalf("types.MapValueFrom() diagnostics: %v", mapDiags)
	}
	plan.Annotations = annotations

	req, diags := buildMonitorV2UpdateRequest(ctx, &plan)
	if diags.HasError() {
		t.Fatalf("buildMonitorV2UpdateRequest() diagnostics: %v", diags)
	}
	requireNoInternalMonitorV2Annotations(t, req.Annotations)
	if req.Annotations["team"] != "platform" {
		t.Fatalf("Annotations[team] = %q, want platform", req.Annotations["team"])
	}
}

func TestMonitorV2BuildCreateRequestCustomResolveThreshold(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:          types.StringValue(monitorV2QueryTypeGCQL),
		Expression:    types.StringValue(`level:error | stats count() count_all_result`),
		DataType:      types.StringValue("logs"),
		InstantRollup: types.StringValue("5m"),
	})

	resolveValues, diags := types.ListValueFrom(ctx, types.Float64Type, []float64{0.5})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() diagnostics: %v", diags)
	}
	plan.Thresholds[0].CustomResolveThreshold = &monitorV2CustomResolveThresholdModel{
		Operator: types.StringValue("lt"),
		Values:   resolveValues,
	}

	req, buildDiags := buildMonitorV2CreateRequest(ctx, &plan)
	if buildDiags.HasError() {
		t.Fatalf("buildMonitorV2CreateRequest() diagnostics: %v", buildDiags)
	}
	threshold := req.Model.Thresholds[0]
	if threshold.CustomResolveThreshold == nil {
		t.Fatalf("threshold.CustomResolveThreshold = nil, want value")
	}
	if threshold.CustomResolveThreshold.Operator == nil || *threshold.CustomResolveThreshold.Operator != "lt" {
		t.Fatalf("threshold.CustomResolveThreshold.Operator = %v, want lt", threshold.CustomResolveThreshold.Operator)
	}
	if len(threshold.CustomResolveThreshold.Values) != 1 || threshold.CustomResolveThreshold.Values[0] != 0.5 {
		t.Fatalf("threshold.CustomResolveThreshold.Values = %#v, want [0.5]", threshold.CustomResolveThreshold.Values)
	}
}

func TestMonitorV2BuildCreateRequestRejectsUnsupportedCustomResolveParentOperator(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:          types.StringValue(monitorV2QueryTypeGCQL),
		Expression:    types.StringValue(`level:error | stats count() count_all_result`),
		DataType:      types.StringValue("logs"),
		InstantRollup: types.StringValue("5m"),
	})

	resolveValues, diags := types.ListValueFrom(ctx, types.Float64Type, []float64{0.5})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() diagnostics: %v", diags)
	}
	plan.Thresholds[0].Operator = types.StringValue("eq")
	plan.Thresholds[0].CustomResolveThreshold = &monitorV2CustomResolveThresholdModel{
		Operator: types.StringValue("lt"),
		Values:   resolveValues,
	}

	_, buildDiags := buildMonitorV2CreateRequest(ctx, &plan)
	if !buildDiags.HasError() {
		t.Fatalf("buildMonitorV2CreateRequest() diagnostics = none, want unsupported custom resolve parent operator error")
	}
	requireDiagnosticSummary(t, buildDiags, "Unsupported threshold operator for custom resolve threshold")
}

func TestMonitorV2ValidateUnsupportedQueryFields(t *testing.T) {
	tests := []struct {
		name      string
		queryType string
		query     *monitorV2QueryModel
		wantPath  string
		wantError bool
	}{
		{
			name:      "gcql rollup",
			queryType: monitorV2QueryTypeGCQL,
			query: &monitorV2QueryModel{
				Rollup: &monitorV2RollupModel{
					Function: types.StringValue("last"),
					Time:     types.StringValue("5m"),
				},
			},
			wantPath:  "query.rollup",
			wantError: true,
		},
		{
			name:      "metricsql instant_rollup",
			queryType: monitorV2QueryTypeMetricsQL,
			query: &monitorV2QueryModel{
				InstantRollup: types.StringValue("5m"),
			},
			wantPath:  "query.instant_rollup",
			wantError: true,
		},
		{
			name:      "metricsql empty instant_rollup",
			queryType: monitorV2QueryTypeMetricsQL,
			query: &monitorV2QueryModel{
				InstantRollup: types.StringValue(""),
			},
		},
		{
			name:      "raw_sql data_type",
			queryType: monitorV2QueryTypeRawSQL,
			query: &monitorV2QueryModel{
				DataType: types.StringValue("logs"),
			},
			wantPath:  "query.data_type",
			wantError: true,
		},
		{
			name:      "raw_sql datasource_type",
			queryType: monitorV2QueryTypeRawSQL,
			query: &monitorV2QueryModel{
				DatasourceType: types.StringValue(monitorV2DatasourcePrometheus),
			},
			wantPath:  "query.datasource_type",
			wantError: true,
		},
		{
			name:      "gcql empty datasource_id",
			queryType: monitorV2QueryTypeGCQL,
			query: &monitorV2QueryModel{
				DatasourceID: types.StringValue(""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			monitorV2ValidateUnsupportedQueryFields(tt.query, tt.queryType, &diags)
			if !tt.wantError {
				if diags.HasError() {
					t.Fatalf("monitorV2ValidateUnsupportedQueryFields() diagnostics = %v, want none", diags)
				}
				return
			}
			requireDiagnosticSummary(t, diags, "Unsupported query")
			diagnosticWithPath, ok := diags[0].(diag.DiagnosticWithPath)
			if !ok {
				t.Fatalf("diagnostic does not expose a path: %#v", diags[0])
			}
			if got := diagnosticWithPath.Path().String(); got != tt.wantPath {
				t.Fatalf("diagnostic path = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestMonitorV2ValidateNotificationSettings(t *testing.T) {
	connectedApps, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"slack-app-id"})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() connected apps diagnostics: %v", diags)
	}
	statusFilters, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"Alerting"})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() status filters diagnostics: %v", diags)
	}

	var missingAppsDiags diag.Diagnostics
	monitorV2ValidateNotificationSettings(&monitorV2NotificationSettingsModel{
		Method:        types.StringValue("connectedApps"),
		ConnectedApps: types.ListNull(types.StringType),
	}, &missingAppsDiags)
	requireDiagnosticSummary(t, missingAppsDiags, "Missing connected apps")

	var invalidCombinationDiags diag.Diagnostics
	monitorV2ValidateNotificationSettings(&monitorV2NotificationSettingsModel{
		Method:        types.StringValue("notificationRoutes"),
		ConnectedApps: connectedApps,
		StatusFilters: statusFilters,
	}, &invalidCombinationDiags)
	requireDiagnosticSummary(t, invalidCombinationDiags, "Invalid notification settings combination")
}

func TestMonitorV2BuildCreateRequestConnectedAppParams(t *testing.T) {
	ctx := context.Background()
	plan := testMonitorV2BasePlan(t, &monitorV2QueryModel{
		Type:          types.StringValue(monitorV2QueryTypeGCQL),
		Expression:    types.StringValue(`level:error | stats count() count_all_result`),
		DataType:      types.StringValue("logs"),
		InstantRollup: types.StringValue("5m"),
	})

	connectedApps, diags := types.ListValueFrom(ctx, types.StringType, []string{"slack-app-id"})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() connected apps diagnostics: %v", diags)
	}
	channels, diags := types.ListValueFrom(ctx, types.StringType, []string{"C0123456789"})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() channels diagnostics: %v", diags)
	}
	appParams, diags := types.ObjectValue(monitorV2ConnectedAppDeliveryOptionsAttrTypes(), map[string]attr.Value{
		"channels": channels,
	})
	if diags.HasError() {
		t.Fatalf("types.ObjectValue() diagnostics: %v", diags)
	}
	connectedAppParams, diags := types.MapValue(
		types.ObjectType{AttrTypes: monitorV2ConnectedAppDeliveryOptionsAttrTypes()},
		map[string]attr.Value{"slack-app-id": appParams},
	)
	if diags.HasError() {
		t.Fatalf("types.MapValue() diagnostics: %v", diags)
	}
	statusFilters, diags := types.ListValueFrom(ctx, types.StringType, []string{"Alerting", "Resolved"})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() status filters diagnostics: %v", diags)
	}

	plan.NotificationSettings = &monitorV2NotificationSettingsModel{
		Method:                 types.StringValue("connectedApps"),
		ConnectedApps:          connectedApps,
		ConnectedAppParams:     connectedAppParams,
		StatusFilters:          statusFilters,
		DisableRenotification:  types.BoolValue(false),
		RenotificationInterval: types.StringValue("4h"),
	}

	req, buildDiags := buildMonitorV2CreateRequest(ctx, &plan)
	if buildDiags.HasError() {
		t.Fatalf("buildMonitorV2CreateRequest() diagnostics: %v", buildDiags)
	}
	if req.NotificationSettings == nil {
		t.Fatalf("NotificationSettings = nil, want value")
	}
	params := req.NotificationSettings.ConnectedAppParams
	if len(params) != 1 {
		t.Fatalf("ConnectedAppParams length = %d, want 1", len(params))
	}
	channelsOut := params["slack-app-id"].Channels
	if len(channelsOut) != 1 || channelsOut[0] != "C0123456789" {
		t.Fatalf("ConnectedAppParams channels = %#v, want [C0123456789]", channelsOut)
	}
}

func TestMonitorV2MapSDKToModelNormalizesDurations(t *testing.T) {
	ctx := context.Background()
	title := "duration monitor"
	thresholdName := "threshold_1"
	thresholdInput := monitorV2DefaultQueryName
	thresholdOperator := "gt"
	pendingFor := models.Duration(5 * time.Minute)
	remote := &models.UpdateMonitorRequest{
		Title:           &title,
		Severity:        "critical",
		MeasurementType: "state",
		EvaluationInterval: &models.EvaluationInterval{
			Interval:   strfmt.Duration(5 * time.Minute),
			PendingFor: &pendingFor,
		},
		Model: &models.Model{
			Queries: []*models.BaseQuery{
				{
					Name:           monitorV2DefaultQueryName,
					Expression:     "up",
					DatasourceType: monitorV2DatasourcePrometheus,
					QueryType:      monitorV2QueryTypeInstant,
					Rollup: &models.Rollup{
						Function: "last",
						Time:     models.Duration(5 * time.Minute),
					},
				},
			},
			Thresholds: []*models.Threshold{
				{
					Name:      &thresholdName,
					InputName: &thresholdInput,
					Operator:  &thresholdOperator,
					Values:    []float64{1},
				},
			},
		},
	}

	var state monitorV2ResourceModel
	var diags diag.Diagnostics
	mapMonitorV2SDKToModel(ctx, "monitor-id", remote, &state, &diags)
	if diags.HasError() {
		t.Fatalf("mapMonitorV2SDKToModel() diagnostics: %v", diags)
	}
	if state.EvaluationInterval.Interval.ValueString() != "5m" {
		t.Fatalf("state.EvaluationInterval.Interval = %q, want 5m", state.EvaluationInterval.Interval.ValueString())
	}
	if state.EvaluationInterval.PendingFor.ValueString() != "5m" {
		t.Fatalf("state.EvaluationInterval.PendingFor = %q, want 5m", state.EvaluationInterval.PendingFor.ValueString())
	}
	if state.Query.Rollup.Time.ValueString() != "5m" {
		t.Fatalf("state.Query.Rollup.Time = %q, want 5m", state.Query.Rollup.Time.ValueString())
	}
}

func TestMonitorV2MapSDKToModelPreservesEquivalentConfiguredDurations(t *testing.T) {
	ctx := context.Background()
	title := "duration monitor"
	thresholdName := "threshold_1"
	thresholdInput := monitorV2DefaultQueryName
	thresholdOperator := "gt"
	pendingFor := models.Duration(1 * time.Minute)
	remote := &models.UpdateMonitorRequest{
		Title:           &title,
		Severity:        "critical",
		MeasurementType: "state",
		EvaluationInterval: &models.EvaluationInterval{
			Interval:   strfmt.Duration(1 * time.Minute),
			PendingFor: &pendingFor,
		},
		Model: &models.Model{
			Queries: []*models.BaseQuery{
				{
					Name:           monitorV2DefaultQueryName,
					Expression:     "up",
					DatasourceType: monitorV2DatasourcePrometheus,
					QueryType:      monitorV2QueryTypeInstant,
					Rollup: &models.Rollup{
						Function: "last",
						Time:     models.Duration(5 * time.Minute),
					},
					RelativeTimerange: &models.RelativeTimerange{
						From: strfmt.Duration(-5 * time.Minute),
						To:   strfmt.Duration(0),
					},
				},
			},
			Thresholds: []*models.Threshold{
				{
					Name:      &thresholdName,
					InputName: &thresholdInput,
					Operator:  &thresholdOperator,
					Values:    []float64{1},
				},
			},
		},
	}

	state := monitorV2ResourceModel{
		EvaluationInterval: &monitorV2EvaluationIntervalModel{
			Interval:   types.StringValue("60 seconds"),
			PendingFor: types.StringValue("1 minute"),
		},
		Query: &monitorV2QueryModel{
			Rollup: &monitorV2RollupModel{
				Time: types.StringValue("5 minutes"),
			},
			RelativeTimerange: &monitorV2RelativeRangeModel{
				From: types.StringValue("-5 minutes"),
				To:   types.StringValue("0m"),
			},
		},
	}
	var diags diag.Diagnostics
	mapMonitorV2SDKToModel(ctx, "monitor-id", remote, &state, &diags)
	if diags.HasError() {
		t.Fatalf("mapMonitorV2SDKToModel() diagnostics: %v", diags)
	}
	if state.EvaluationInterval.Interval.ValueString() != "60 seconds" {
		t.Fatalf("state.EvaluationInterval.Interval = %q, want 60 seconds", state.EvaluationInterval.Interval.ValueString())
	}
	if state.EvaluationInterval.PendingFor.ValueString() != "1 minute" {
		t.Fatalf("state.EvaluationInterval.PendingFor = %q, want 1 minute", state.EvaluationInterval.PendingFor.ValueString())
	}
	if state.Query.Rollup.Time.ValueString() != "5 minutes" {
		t.Fatalf("state.Query.Rollup.Time = %q, want 5 minutes", state.Query.Rollup.Time.ValueString())
	}
	if state.Query.RelativeTimerange.From.ValueString() != "-5 minutes" {
		t.Fatalf("state.Query.RelativeTimerange.From = %q, want -5 minutes", state.Query.RelativeTimerange.From.ValueString())
	}
	if state.Query.RelativeTimerange.To.ValueString() != "0m" {
		t.Fatalf("state.Query.RelativeTimerange.To = %q, want 0m", state.Query.RelativeTimerange.To.ValueString())
	}
}

func TestMonitorV2MapSDKToModelPreservesEquivalentNotificationDuration(t *testing.T) {
	ctx := context.Background()
	title := "duration monitor"
	remote := &models.UpdateMonitorRequest{
		Title:           &title,
		Severity:        "critical",
		MeasurementType: "state",
		NotificationSettings: &models.NotificationSettings{
			Method:                 "connectedApps",
			RenotificationInterval: models.RenotificationDuration("1h"),
		},
	}

	state := monitorV2ResourceModel{
		NotificationSettings: &monitorV2NotificationSettingsModel{
			Method:                 types.StringValue("connectedApps"),
			RenotificationInterval: types.StringValue("60m"),
		},
	}
	var diags diag.Diagnostics
	mapMonitorV2SDKToModel(ctx, "monitor-id", remote, &state, &diags)
	if diags.HasError() {
		t.Fatalf("mapMonitorV2SDKToModel() diagnostics: %v", diags)
	}
	if state.NotificationSettings.RenotificationInterval.ValueString() != "60m" {
		t.Fatalf("state.NotificationSettings.RenotificationInterval = %q, want 60m", state.NotificationSettings.RenotificationInterval.ValueString())
	}
}

func TestMonitorV2DurationNormalizationPreservesZero(t *testing.T) {
	if got := monitorV2DurationToType(0).ValueString(); got != "0m" {
		t.Fatalf("monitorV2DurationToType(0) = %q, want 0m", got)
	}
	if got := monitorV2DurationStringToType("5 minutes").ValueString(); got != "5m" {
		t.Fatalf("monitorV2DurationStringToType(5 minutes) = %q, want 5m", got)
	}
}

func TestMonitorV2MapSDKToModelClassifiesMetricsQLByRollup(t *testing.T) {
	query := monitorV2QueryFromSDK(&models.BaseQuery{
		Expression:     "sum(metric)",
		DatasourceType: monitorV2DatasourceClickhouse,
		Rollup: &models.Rollup{
			Function: "last",
			Time:     models.Duration(5 * time.Minute),
		},
	}, nil)
	if query.Type.ValueString() != monitorV2QueryTypeMetricsQL {
		t.Fatalf("query.Type = %q, want %q", query.Type.ValueString(), monitorV2QueryTypeMetricsQL)
	}

	rawSQLQuery := monitorV2QueryFromSDK(&models.BaseQuery{
		Expression:     "SELECT 0 AS count_all_result",
		DatasourceType: monitorV2DatasourceClickhouse,
		QueryType:      monitorV2QueryTypeInstant,
	}, nil)
	if rawSQLQuery.Type.ValueString() != monitorV2QueryTypeRawSQL {
		t.Fatalf("rawSQLQuery.Type = %q, want %q", rawSQLQuery.Type.ValueString(), monitorV2QueryTypeRawSQL)
	}
}

func TestMonitorV2MapSDKToModelUsesQueryTypeAnnotation(t *testing.T) {
	query := monitorV2QueryFromSDK(&models.BaseQuery{
		Expression:     "SELECT 0 AS count_all_result",
		DatasourceType: monitorV2DatasourcePrometheus,
		QueryType:      monitorV2QueryTypeInstant,
	}, map[string]string{
		monitorV2QueryTypeAnnotationKey: monitorV2QueryTypeRawSQL,
	})
	if query.Type.ValueString() != monitorV2QueryTypeRawSQL {
		t.Fatalf("query.Type = %q, want %q", query.Type.ValueString(), monitorV2QueryTypeRawSQL)
	}
}

func TestMonitorV2FilterAnnotationsRemovesInternalMarkers(t *testing.T) {
	filtered := monitorV2FilterAnnotations(map[string]string{
		monitorV2QueryTypeAnnotationKey: monitorV2QueryTypeMetricsQL,
		monitorV2DataTypeAnnotationKey:  "metrics",
		"team":                          "platform",
	})
	if _, ok := filtered[monitorV2QueryTypeAnnotationKey]; ok {
		t.Fatalf("filtered annotations still include %q", monitorV2QueryTypeAnnotationKey)
	}
	if _, ok := filtered[monitorV2DataTypeAnnotationKey]; ok {
		t.Fatalf("filtered annotations still include %q", monitorV2DataTypeAnnotationKey)
	}
	if filtered["team"] != "platform" {
		t.Fatalf("filtered team annotation = %q, want platform", filtered["team"])
	}
}

func TestMonitorV2ValidateAnnotationsRejectsReservedInternalMarkers(t *testing.T) {
	for _, key := range []string{monitorV2QueryTypeAnnotationKey, monitorV2DataTypeAnnotationKey} {
		t.Run(key, func(t *testing.T) {
			annotations, mapDiags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
				key:    monitorV2QueryTypeGCQL,
				"team": "platform",
			})
			if mapDiags.HasError() {
				t.Fatalf("types.MapValueFrom() diagnostics: %v", mapDiags)
			}

			var diags diag.Diagnostics
			monitorV2ValidateAnnotations(annotations, &diags)
			requireDiagnosticSummary(t, diags, "Reserved monitor annotation")
		})
	}
}

func requireNoInternalMonitorV2Annotations(t *testing.T, annotations map[string]string) {
	t.Helper()

	for key := range annotations {
		if monitorV2IsInternalAnnotation(key) {
			t.Fatalf("annotations unexpectedly include internal key %q", key)
		}
	}
}

func TestMonitorResourceDisappearsDefaultAPIURLMatchesProviderDefault(t *testing.T) {
	t.Setenv("GROUNDCOVER_API_URL", "")

	if got, want := testAccMonitorAPIURL(), "https://api.groundcover.com"; got != want {
		t.Fatalf("testAccMonitorAPIURL() = %q, want %q", got, want)
	}
}

func requireDiagnosticSummary(t *testing.T, diagnostics diag.Diagnostics, summary string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Summary(), summary) {
			return
		}
	}

	t.Fatalf("diagnostics did not include summary %q: %v", summary, diagnostics)
}

func testMonitorV2BasePlan(t *testing.T, query *monitorV2QueryModel) monitorV2ResourceModel {
	t.Helper()

	values, diags := types.ListValueFrom(context.Background(), types.Float64Type, []float64{1})
	if diags.HasError() {
		t.Fatalf("types.ListValueFrom() diagnostics: %v", diags)
	}

	return monitorV2ResourceModel{
		Title:           types.StringValue("test monitor v2"),
		Severity:        types.StringValue("critical"),
		MeasurementType: types.StringValue("event"),
		Query:           query,
		Thresholds: []monitorV2ThresholdModel{
			{
				Name:      types.StringValue("threshold_1"),
				InputName: types.StringValue(monitorV2DefaultQueryName),
				Operator:  types.StringValue("gt"),
				Values:    values,
			},
		},
	}
}

func TestAccMonitorResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor")
	updatedName := acctest.RandomWithPrefix("test-monitor-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMonitorResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					// Check the YAML contains our title
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(name)),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_monitor.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"monitor_yaml"}, // YAML may not be identical after import
			},
			// Update and Read testing
			{
				Config: testAccMonitorResourceConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(updatedName)),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccMonitorV2Resource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-v2")
	updatedName := acctest.RandomWithPrefix("test-monitor-v2-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorV2ResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.test", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "title", name),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.type", monitorV2QueryTypeGCQL),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.data_type", "logs"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.instant_rollup", "5m"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "threshold.#", "1"),
				),
			},
			{
				ResourceName:      "groundcover_monitor_v2.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccMonitorV2ResourceConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "title", updatedName),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "display.header", updatedName),
				),
			},
			{
				Config:             testAccMonitorV2ResourceConfig(updatedName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccMonitorV2ResourceHumanDurationConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.type", monitorV2QueryTypeMetricsQL),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.relative_timerange.from", "-5 minutes"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.relative_timerange.to", "0m"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.rollup.time", "5 minutes"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "evaluation_interval.interval", "60 seconds"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "evaluation_interval.pending_for", "1 minute"),
				),
			},
			{
				Config:             testAccMonitorV2ResourceHumanDurationConfig(updatedName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccMonitorV2Resource_importThenUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-v2-import")
	updatedName := acctest.RandomWithPrefix("test-monitor-v2-import-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorV2MetricsQLImportUpdateConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.test", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "title", name),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "query.type", monitorV2QueryTypeMetricsQL),
					resource.TestCheckNoResourceAttr("groundcover_monitor_v2.test", "annotations."+monitorV2QueryTypeAnnotationKey),
				),
			},
			{
				ResourceName:      "groundcover_monitor_v2.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccMonitorV2MetricsQLImportUpdateConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "title", updatedName),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.test", "display.header", updatedName),
					resource.TestCheckNoResourceAttr("groundcover_monitor_v2.test", "annotations."+monitorV2QueryTypeAnnotationKey),
				),
			},
		},
	})
}

func TestAccMonitorV2Resource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-v2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorV2ResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMonitorResourceExists("groundcover_monitor_v2.test"),
					testAccCheckMonitorResourceDisappears("groundcover_monitor_v2.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccMonitorV2Resource_allSupportedQueryTypes(t *testing.T) {
	if os.Getenv("TF_ACC_MONITOR_V2_ALL_QUERY_TYPES") == "" {
		t.Skip("Set TF_ACC_MONITOR_V2_ALL_QUERY_TYPES=1 to run the Monitor V2 all-supported-query-types acceptance test")
	}

	// This intentionally broad live check is opt-in only. It verifies the backend accepts every supported Monitor V2 query family.
	suffix := fmt.Sprintf("%d-%s", time.Now().UnixNano(), acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorV2AllSupportedQueryTypesConfig(suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.logs", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.logs", "query.type", monitorV2QueryTypeGCQL),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.logs", "query.data_type", "logs"),
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.traces", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.traces", "query.type", monitorV2QueryTypeGCQL),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.traces", "query.data_type", "traces"),
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.events", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.events", "query.type", monitorV2QueryTypeGCQL),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.events", "query.data_type", "events"),
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.apm", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.apm", "query.type", monitorV2QueryTypeGCQL),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.apm", "query.data_type", "apm"),
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.metricsql", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.metricsql", "query.type", monitorV2QueryTypeMetricsQL),
					resource.TestCheckResourceAttrSet("groundcover_monitor_v2.raw_sql", "id"),
					resource.TestCheckResourceAttr("groundcover_monitor_v2.raw_sql", "query.type", monitorV2QueryTypeRawSQL),
				),
			},
		},
	})
}

func testAccMonitorV2ResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor_v2" "test" {
  title            = %[1]q
  severity         = "critical"
  measurement_type = "event"

  display {
    header      = %[1]q
    description = "Test monitor created by acceptance tests"
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

func testAccMonitorV2MetricsQLImportUpdateConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor_v2" "test" {
  title            = %[1]q
  severity         = "warning"
  measurement_type = "state"
  is_paused        = true

  display {
    header      = %[1]q
    description = "Monitor V2 import/update acceptance test"
  }

  query {
    type       = "metricsql"
    expression = "sum(groundcover_kube_pod_container_status_running{})"

    rollup {
      function = "avg"
      time     = "5m"
    }
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [999999]
  }

  evaluation_interval {
    interval    = "5m"
    pending_for = "10m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"

  notification_settings {
    method = "notificationRoutes"
  }
}
`, name)
}

func testAccMonitorV2ResourceHumanDurationConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor_v2" "test" {
  title            = %[1]q
  severity         = "critical"
  measurement_type = "state"

  display {
    header      = %[1]q
    description = "Test monitor created by acceptance tests"
  }

  query {
    type       = "metricsql"
    expression = "sum(groundcover_kube_pod_container_status_running{})"

    relative_timerange {
      from = "-5 minutes"
      to   = "0m"
    }

    rollup {
      function = "last"
      time     = "5 minutes"
    }
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [1]
  }

  evaluation_interval {
    interval    = "60 seconds"
    pending_for = "1 minute"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}
`, name)
}

func testAccMonitorV2AllSupportedQueryTypesConfig(suffix string) string {
	var config strings.Builder

	gcqlCases := []struct {
		resourceName  string
		dataType      string
		query         string
		instantRollup bool
	}{
		{
			resourceName:  "logs",
			dataType:      "logs",
			query:         "level:error | stats count() count_all_result",
			instantRollup: true,
		},
		{
			resourceName:  "traces",
			dataType:      "traces",
			query:         "* | stats count() count_all_result",
			instantRollup: true,
		},
		{
			resourceName:  "events",
			dataType:      "events",
			query:         "* | stats count() count_all_result",
			instantRollup: true,
		},
		{
			resourceName:  "apm",
			dataType:      "apm",
			query:         "* | stats count() count_all_result",
			instantRollup: true,
		},
	}

	for _, tc := range gcqlCases {
		config.WriteString(testAccMonitorV2GCQLVariantConfig(
			tc.resourceName,
			fmt.Sprintf("tf-acc-monitor-v2-%s-%s", tc.resourceName, suffix),
			tc.dataType,
			tc.query,
			tc.instantRollup,
		))
	}

	config.WriteString(testAccMonitorV2MetricsQLVariantConfig(fmt.Sprintf("tf-acc-monitor-v2-metricsql-%s", suffix)))
	config.WriteString(testAccMonitorV2RawSQLVariantConfig(fmt.Sprintf("tf-acc-monitor-v2-raw-sql-%s", suffix)))

	return config.String()
}

func testAccMonitorV2GCQLVariantConfig(resourceName, title, dataType, query string, instantRollup bool) string {
	instantRollupConfig := ""
	if instantRollup {
		instantRollupConfig = `    instant_rollup = "5m"
`
	}

	return fmt.Sprintf(`
resource "groundcover_monitor_v2" "%s" {
  title            = %q
  severity         = "critical"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = %q
    description = "Monitor V2 all-supported-query-types acceptance test"
  }

  query {
    type       = "gcql"
    data_type  = %q
    expression = %q
%s  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [999999]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}
`, resourceName, title, title, dataType, query, instantRollupConfig)
}

func testAccMonitorV2MetricsQLVariantConfig(title string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor_v2" "metricsql" {
  title            = %q
  severity         = "critical"
  measurement_type = "state"
  is_paused        = true

  display {
    header      = %q
    description = "Monitor V2 MetricsQL acceptance test"
  }

  query {
    type       = "metricsql"
    expression = "sum(groundcover_kube_pod_container_status_running{})"

    relative_timerange {
      from = "-5m"
      to   = "0m"
    }

    rollup {
      function = "last"
      time     = "5 minutes"
    }
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [999999]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}
`, title, title)
}

func testAccMonitorV2RawSQLVariantConfig(title string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor_v2" "raw_sql" {
  title            = %q
  severity         = "critical"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = %q
    description = "Monitor V2 raw SQL acceptance test"
  }

  query {
    type       = "raw_sql"
    query_type = "instant"
    expression = "SELECT 0 AS count_all_result"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [999999]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}
`, title, title)
}

func TestAccMonitorResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMonitorResourceExists("groundcover_monitor.test"),
					testAccCheckMonitorResourceDisappears("groundcover_monitor.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccMonitorResource_trailingNewline tests that monitors with trailing newlines
// do not cause apply loops. This simulates the issue where the server returns YAML
// with different formatting (including trailing newlines) and verifies that the
// normalization fixes prevent unnecessary updates.
func TestAccMonitorResource_trailingNewline(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-newline")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with trailing newlines in YAML
			{
				Config: testAccMonitorResourceConfigWithTrailingNewline(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(name)),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This verifies that the normalization fixes prevent false drift detection
			{
				Config: testAccMonitorResourceConfigWithTrailingNewline(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(name)),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccMonitorResourceConfigWithTrailingNewline(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccMonitorResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: %[1]s
display:
  header: %[1]s Test Monitor
  description: Test monitor created by acceptance tests
severity: critical
model:
  queries:
    - name: test_query
      dataType: metrics
      pipeline:
        function:
          name: sum_over_time
          pipelines:
            - metric: up
          args:
          - 5m
  thresholds:
    - name: threshold_1
      inputName: test_query
      operator: gt
      values:
        - 1
evaluationInterval:
  interval: 1m
  pendingFor: 1m
measurementType: state
YAML
}
`, name)
}

func testAccCheckMonitorResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Monitor ID is set")
		}

		return nil
	}
}

// testAccCheckMonitorResourcePrintDetails prints the monitor ID and YAML for verification
func testAccCheckMonitorResourcePrintDetails(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		monitorID := rs.Primary.ID
		monitorYaml := rs.Primary.Attributes["monitor_yaml"]

		fmt.Printf("\n✅ Monitor created successfully!\n")
		fmt.Printf("   Monitor ID: %s\n", monitorID)
		fmt.Printf("   Monitor YAML (first 200 chars): %s\n", func() string {
			if len(monitorYaml) > 200 {
				return monitorYaml[:200] + "..."
			}
			return monitorYaml
		}())
		fmt.Printf("   Full YAML length: %d characters\n\n", len(monitorYaml))

		return nil
	}
}

func testAccCheckMonitorResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Monitor ID is set")
		}

		// Create a provider client to delete the resource
		ctx := context.Background()

		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := testAccMonitorAPIURL()

		// Create the client wrapper
		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, orgName)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		// Delete the resource using the client
		if err := client.DeleteMonitor(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete monitor: %v", err)
		}

		return nil
	}
}

func testAccMonitorAPIURL() string {
	apiURL := os.Getenv("GROUNDCOVER_API_URL")
	if apiURL == "" {
		return "https://api.groundcover.com"
	}
	return apiURL
}

// testAccMonitorResourceConfigWithTrailingNewline creates a monitor config with trailing newlines
// to simulate the issue where YAML formatting differences cause apply loops
func testAccMonitorResourceConfigWithTrailingNewline(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: %[1]s
display:
  header: %[1]s Test Monitor
  description: Test monitor with trailing newlines
severity: critical
model:
  queries:
    - name: test_query
      dataType: metrics
      pipeline:
        function:
          name: sum_over_time
          pipelines:
            - metric: up
          args:
          - 5m
  thresholds:
    - name: threshold_1
      inputName: test_query
      operator: gt
      values:
        - 1
evaluationInterval:
  interval: 1m
  pendingFor: 1m
measurementType: state

YAML
}
`, name)
}

// TestAccMonitorResource_multilinePipeSyntax tests that monitors with multiline pipe syntax (|)
// for title and header fields do not cause apply loops. This simulates the issue shown in the
// image where `title: |` followed by the value on the next line should be normalized to `title: value`.
func TestAccMonitorResource_multilinePipeSyntax(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-pipe")
	titleValue := fmt.Sprintf("CloudSql Connection Count %s", name)
	headerValue := fmt.Sprintf("CloudSql Connection Count %s", name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with multiline pipe syntax (|) for title and header
			{
				Config: testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(titleValue))),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This verifies that semantic comparison treats `title: |\nvalue` and `title: value` as equivalent
			{
				Config: testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(titleValue))),
				),
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
				),
			},
		},
	})
}

// testAccMonitorResourceConfigWithMultilinePipe creates a monitor config using multiline pipe syntax (|)
// for title and header fields. This tests that single-line values using multiline pipe syntax
// (e.g., `title: |\n  value`) are normalized to simple string format (e.g., `title: value`)
// because Grafana/monitor API doesn't accept multiline pipe syntax for single-line values.
func testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: |
  %s
display:
  header: |
    %s
  description: Test monitor with multiline pipe syntax
severity: critical
model:
  queries:
    - name: test_query
      dataType: metrics
      pipeline:
        function:
          name: sum_over_time
          pipelines:
            - metric: up
          args:
          - 5m
  thresholds:
    - name: threshold_1
      inputName: test_query
      operator: gt
      values:
        - 1
evaluationInterval:
  interval: 1m
  pendingFor: 1m
measurementType: state
YAML
}
`, titleValue, headerValue)
}

func TestAccMonitorResource_applyLoopIssue(t *testing.T) {
	title := acctest.RandomWithPrefix("k8s eu-povs node not ready")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with the exact YAML from monitor test.yml
			{
				Config: testAccMonitorResourceConfigApplyLoop(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(title))),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This is the critical test - if there's an apply loop, this step will show changes
			{
				Config: testAccMonitorResourceConfigApplyLoop(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(title))),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccMonitorResourceConfigApplyLoop(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
				),
			},
		},
	})
}

// testAccMonitorResourceConfigApplyLoop creates a monitor config that matches the user's
// monitor test.yml file.
func testAccMonitorResourceConfigApplyLoop(title string) string {
	// This is the exact YAML from monitor test.yml
	yaml := fmt.Sprintf(`title: %s
display:
  header: %s
  contextHeaderLabels:
  - cluster
  - node
  - environment
  - cluster
  - env
  description: is not ready
severity: error
measurementType: state
model:
  queries:
  - name: threshold_input_query
    # trailing space after status="true"}) tests yaml normalization
    expression: sum(kube_node_status_condition{cluster="eu-povs", condition="Ready",status="true"}) 
      by (cluster, node, environment) > 0
    datasourceType: prometheus
    queryType: instant
    rollup:
      function: last
      time: 10m
  thresholds:
  - name: threshold_1
    inputName: threshold_input_query
    operator: gt
    values:
    - 0
annotations:
  Pagerduty_Incidents: enabled
  Slack-Prod-Alerts: enabled
executionErrorState: Error
noDataState: OK
evaluationInterval:
  interval: 5m0s
  pendingFor: 5m0s
notificationSettings:
    renotificationInterval: 2h
isPaused: true`, title, title)

	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
%s
YAML
}
`, yaml)
}

// TestAccMonitorResource_multilineExpression tests that monitors with multiline expressions
// (where the expression spans multiple lines with trailing spaces) do not cause apply loops.
// This tests the specific issue where the API returns expressions on a single line, but the
// input has them split across lines with trailing spaces.
func TestAccMonitorResource_multilineExpression(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-multiline-expr")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with multiline expression (trailing space + continuation)
			{
				Config: testAccMonitorResourceConfigMultilineExpression(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(name))),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Plan with the same config - should show no changes (no apply loop)
			// This explicitly verifies that multiline expressions are normalized correctly
			// and don't cause drift. If there were an apply loop, this would show changes.
			{
				Config:             testAccMonitorResourceConfigMultilineExpression(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // Explicitly expect no changes - this is the key check for apply loop
			},
			// Step 3: Apply the same config again - should not detect changes (no apply loop)
			// This verifies that applying doesn't trigger updates
			{
				Config: testAccMonitorResourceConfigMultilineExpression(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(name))),
				),
				ExpectNonEmptyPlan: false, // Explicitly expect no changes
			},
			// Step 4: Plan one more time to be absolutely sure there's no apply loop
			{
				Config:             testAccMonitorResourceConfigMultilineExpression(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // Explicitly expect no changes
			},
		},
	})
}

// testAccMonitorResourceConfigMultilineExpression creates a monitor config with a multiline expression
// that has a trailing space on the first line and continuation on the next line. This simulates
// the exact issue where the API returns the expression on a single line, but the input has it
// split across lines with trailing spaces.
func testAccMonitorResourceConfigMultilineExpression(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: %s
display:
  header: %s
  description: Test monitor with multiline expression
severity: error
measurementType: state
model:
  queries:
  - name: threshold_input_query
    expression: sum(kube_node_status_condition{cluster="test", condition="Ready",status="true"}) 
      by (cluster, node, environment) > 0
    datasourceType: prometheus
    queryType: instant
    rollup:
      function: last
      time: 10m
  thresholds:
  - name: threshold_1
    inputName: threshold_input_query
    operator: gt
    values:
    - 0
executionErrorState: Error
noDataState: OK
evaluationInterval:
  interval: 5m0s
  pendingFor: 5m0s
isPaused: true
YAML
}
`, name, name)
}

// TestAccMonitorResource_oomKilled tests the K8s Pod OOM Killed Monitor from monitor.yml
// as an integration/regression test. This monitor uses sqlPipeline with complex selectors,
// filters, groupBy, and orderBy configurations.
//
// NOTE: The primary test for the apply loop bug is the unit test:
// TestFilterYamlKeysBasedOnTemplate_EdgeCases/optional_fields_in_arrays_-_groupBy_with_aliases
// which directly tests the filtering logic and reliably catches the bug.
//
// This acceptance test serves as:
// 1. An integration test to verify the fix works end-to-end with real API
// 2. A regression test for the specific monitor.yml configuration
// 3. Verification that the monitor can be created and managed without issues
func TestAccMonitorResource_oomKilled(t *testing.T) {
	title := fmt.Sprintf("K8s Pod OOM Killed Monitor %s", acctest.RandomWithPrefix("test"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with the exact YAML from monitor.yml
			{
				Config: testAccMonitorResourceConfigOomKilled(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(title))),
					// Verify that groupBy entries with aliases are present
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(`(?s)groupBy:.*alias: pod_name`)),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This verifies idempotency after state refresh
			{
				Config: testAccMonitorResourceConfigOomKilled(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
				),
				ExpectNonEmptyPlan: false, // Should be no changes
			},
		},
	})
}

// testAccMonitorResourceConfigOomKilled creates a monitor config that matches the exact
// YAML from monitor.yml. This monitor uses sqlPipeline with complex selectors, filters,
// groupBy, and orderBy configurations.
func testAccMonitorResourceConfigOomKilled(title string) string {
	// This is the exact YAML from monitor.yml with a dynamic title
	yaml := fmt.Sprintf(`title: %s
display:
  header: %s
  resourceHeaderLabels:
  - pod_name
  - container
  contextHeaderLabels:
  - env
  - cluster
  - namespace
  - workload
  description: This Monitor fires when a pod has been OOM Killed, leading to potential
    application instability.
severity: error
measurementType: event
model:
  queries:
  - dataType: events
    name: threshold_input_query
    sqlPipeline:
      selectors:
      - key: _time
        origin: root
        type: string
        processors:
        - op: toStartOfInterval
          args:
          - 5 minutes
        alias: _bucket_timestamp
      - key: podName
        origin: root
        type: string
        alias: pod_name
      - key: container
        origin: root
        type: string
        alias: container
      - key: cluster
        origin: root
        type: string
        alias: cluster
      - key: namespace
        origin: root
        type: string
        alias: namespace
      - key: env
        origin: root
        type: string
        alias: env
      - key: workload
        origin: root
        type: string
        alias: workload
      - key: '*'
        origin: root
        type: string
        processors:
        - op: count
        alias: crashes_count
      filters:
        conditions:
        - key: type
          origin: root
          type: string
          filters:
          - op: match
            value: container_crash
        - key: reason
          origin: root
          type: string
          filters:
          - op: match
            value: OOMKilled
        operator: and
      groupBy:
      - key: _bucket_timestamp
        origin: root
        type: string
      - key: podName
        origin: root
        type: string
        alias: pod_name
      - key: container
        origin: root
        type: string
        alias: container
      - key: cluster
        origin: root
        type: string
        alias: cluster
      - key: namespace
        origin: root
        type: string
        alias: namespace
      - key: env
        origin: root
        type: string
        alias: env
      - key: workload
        origin: root
        type: string
        alias: workload
      orderBy:
      - selector:
          key: _bucket_timestamp
          origin: root
          type: string
        direction: ASC
    instantRollup: 5 minutes
  thresholds:
  - name: threshold_1
    inputName: threshold_input_query
    operator: gt
    values:
    - 0
executionErrorState: OK
noDataState: OK
evaluationInterval:
  interval: 1m0s
  pendingFor: 1m0s`, title, title)

	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
%s
YAML
}
`, yaml)
}
