// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLogsPipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccLogsPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "updated_at"),
					// Assert the singleton holds this run's own document, not one that
					// merely has the right shape.
					resource.TestMatchResourceAttr("groundcover_logspipeline.test", "value",
						fixtureTokenRegexp("test-rule-")),
				),
			},
			// Update and Read testing
			{
				Config: testAccLogsPipelineResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "updated_at"),
					resource.TestMatchResourceAttr("groundcover_logspipeline.test", "value",
						fixtureTokenRegexp("test-rule-updated-")),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// The ruleName carries testRunToken so that a value written by another CI run is
// attributable in the plan diff rather than indistinguishable from this run's own.
func testAccLogsPipelineResourceConfig() string {
	return fmt.Sprintf(`
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule-%[1]s
  conditions:
    - container_name == "nginx"
  statements:
    - set(attributes["test.key"], "test-value")
YAML
}
`, testRunToken())
}

func testAccLogsPipelineResourceConfigUpdated() string {
	return fmt.Sprintf(`
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule-updated-%[1]s
  conditions:
    - container_name == "nginx"
  statements:
    - set(attributes["test.key"], "test-value-updated")
YAML
}
`, testRunToken())
}

// A config that is semantically identical to state must not plan any changes, even after a
// refresh returns the backend's own serialization of the document.
func TestAccLogsPipelineResource_noDiffOnReformattedYaml(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLogsPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "updated_at"),
					resource.TestMatchResourceAttr("groundcover_logspipeline.test", "value",
						fixtureTokenRegexp("test-rule-")),
				),
			},
			// Same pipeline, reformatted YAML (indentation, key order, a comment): no
			// diff, so no write and no new timestamp. The framework's post-apply plan on
			// the previous step already covers the unchanged-config case.
			{Config: testAccLogsPipelineResourceConfigReformatted(), PlanOnly: true},
		},
	})
}

// testAccLogsPipelineResourceConfigReformatted is testAccLogsPipelineResourceConfig with
// the same ottlRules written differently: deeper sequence indentation, reordered mapping
// keys, and an added comment.
func testAccLogsPipelineResourceConfigReformatted() string {
	return fmt.Sprintf(`
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
# managed by terraform
ottlRules:
  - conditions:
      - container_name == "nginx"
    statements:
      - set(attributes["test.key"], "test-value")
    ruleName: test-rule-%[1]s
YAML
}
`, testRunToken())
}

// Embedding ApiClient keeps the stub to the one method these tests exercise; any other call
// panics, which is the intent.
type stubLogsPipelineClient struct {
	ApiClient
	config *models.LogsPipelineConfig
	err    error
}

func (c *stubLogsPipelineClient) GetLogsPipeline(context.Context) (*models.LogsPipelineConfig, error) {
	return c.config, c.err
}

func logsPipelineTestSchema(ctx context.Context, t *testing.T) fwschema.Schema {
	t.Helper()

	var r logsPipelineResource
	resp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// logsPipelineTestValue builds a raw object for the logs pipeline schema. nil makes a field
// null; tftypes.UnknownValue makes it unknown.
func logsPipelineTestValue(ctx context.Context, s fwschema.Schema, value, updatedAt interface{}) tftypes.Value {
	objType := s.Type().TerraformType(ctx)
	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"value":      tftypes.NewValue(tftypes.String, value),
		"updated_at": tftypes.NewValue(tftypes.String, updatedAt),
	})
}

const (
	testLogsPipelineStateValue = "ottlRules:\n- ruleName: test-rule\n  conditions:\n    - container_name == \"nginx\"\n"
	// Same document as testLogsPipelineStateValue, serialized differently — what a backend
	// that normalizes on write hands back.
	testLogsPipelineRemoteValue = "ottlRules:\n  - conditions:\n      - container_name == \"nginx\"\n    ruleName: test-rule\n"
	testLogsPipelineOtherValue  = "ottlRules:\n- ruleName: other-rule\n"

	testLogsPipelineStateTimestamp  = "2026-07-01T10:00:00.000Z"
	testLogsPipelineRemoteTimestamp = "2026-07-27T09:00:00.000Z"
)

func testLogsPipelineRemoteConfig(value string) *models.LogsPipelineConfig {
	ts, err := time.Parse(time.RFC3339, testLogsPipelineRemoteTimestamp)
	if err != nil {
		panic(err)
	}
	return &models.LogsPipelineConfig{
		UUID:             "6b09aee0-9001-4095-afb1-152b487e5bdd",
		Value:            value,
		CreatedTimestamp: strfmt.DateTime(ts),
	}
}

// The backend re-serializes the document and mints a fresh timestamp on every write, so a
// refresh must not overwrite state that means the same thing.
func TestLogsPipelineReadKeepsSemanticallyUnchangedState(t *testing.T) {
	ctx := context.Background()
	s := logsPipelineTestSchema(ctx, t)

	tests := []struct {
		name              string
		remote            *models.LogsPipelineConfig
		stateValue        interface{}
		stateUpdatedAt    interface{}
		expectedValue     string
		expectedUpdatedAt string
	}{
		{
			name:              "reformatted remote value keeps state value and timestamp",
			remote:            testLogsPipelineRemoteConfig(testLogsPipelineRemoteValue),
			stateValue:        testLogsPipelineStateValue,
			stateUpdatedAt:    testLogsPipelineStateTimestamp,
			expectedValue:     testLogsPipelineStateValue,
			expectedUpdatedAt: testLogsPipelineStateTimestamp,
		},
		{
			name:              "byte-identical remote value keeps state timestamp",
			remote:            testLogsPipelineRemoteConfig(testLogsPipelineStateValue),
			stateValue:        testLogsPipelineStateValue,
			stateUpdatedAt:    testLogsPipelineStateTimestamp,
			expectedValue:     testLogsPipelineStateValue,
			expectedUpdatedAt: testLogsPipelineStateTimestamp,
		},
		{
			name:              "genuinely changed remote value is adopted with its timestamp",
			remote:            testLogsPipelineRemoteConfig(testLogsPipelineOtherValue),
			stateValue:        testLogsPipelineStateValue,
			stateUpdatedAt:    testLogsPipelineStateTimestamp,
			expectedValue:     testLogsPipelineOtherValue,
			expectedUpdatedAt: testLogsPipelineRemoteTimestamp,
		},
		{
			name:              "missing timestamp in state is repopulated from the API",
			remote:            testLogsPipelineRemoteConfig(testLogsPipelineRemoteValue),
			stateValue:        testLogsPipelineStateValue,
			stateUpdatedAt:    nil,
			expectedValue:     testLogsPipelineStateValue,
			expectedUpdatedAt: testLogsPipelineRemoteTimestamp,
		},
		{
			name:              "empty remote config clears the value",
			remote:            nil,
			stateValue:        testLogsPipelineStateValue,
			stateUpdatedAt:    testLogsPipelineStateTimestamp,
			expectedValue:     "",
			expectedUpdatedAt: "",
		},
		{
			name:              "empty remote config clears the timestamp even when state is already empty",
			remote:            nil,
			stateValue:        "",
			stateUpdatedAt:    testLogsPipelineStateTimestamp,
			expectedValue:     "",
			expectedUpdatedAt: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := logsPipelineTestValue(ctx, s, tc.stateValue, tc.stateUpdatedAt)
			r := &logsPipelineResource{client: &stubLogsPipelineClient{config: tc.remote}}

			resp := &fwresource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw.Copy()}}
			r.Read(ctx, fwresource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			var got logsPipelineResourceModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("could not read refreshed state: %v", diags)
			}

			if got.Value.ValueString() != tc.expectedValue {
				t.Errorf("value = %q, want %q", got.Value.ValueString(), tc.expectedValue)
			}
			if got.UpdatedAt.ValueString() != tc.expectedUpdatedAt {
				t.Errorf("updated_at = %q, want %q", got.UpdatedAt.ValueString(), tc.expectedUpdatedAt)
			}
		})
	}
}

// The refresh reconciliation must not swallow the not-found branch.
func TestLogsPipelineReadRemovesResourceWhenNotFound(t *testing.T) {
	ctx := context.Background()
	s := logsPipelineTestSchema(ctx, t)

	raw := logsPipelineTestValue(ctx, s, testLogsPipelineStateValue, testLogsPipelineStateTimestamp)
	r := &logsPipelineResource{client: &stubLogsPipelineClient{err: ErrNotFound}}

	resp := &fwresource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw.Copy()}}
	r.Read(ctx, fwresource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("expected the resource to be removed from state, got %s", resp.State.Raw)
	}
}

func TestLogsPipelineModifyPlanCollapsesFormattingOnlyDiff(t *testing.T) {
	ctx := context.Background()
	s := logsPipelineTestSchema(ctx, t)

	tests := []struct {
		name string
		// nil means a null object: no prior state (create) or no plan (destroy).
		stateRaw          *tftypes.Value
		planRaw           *tftypes.Value
		expectedValue     string
		expectedUpdatedAt string
		expectUnknown     bool
	}{
		{
			name:              "formatting-only change is planned as the prior value",
			stateRaw:          tfValuePtr(logsPipelineTestValue(ctx, s, testLogsPipelineStateValue, testLogsPipelineStateTimestamp)),
			planRaw:           tfValuePtr(logsPipelineTestValue(ctx, s, testLogsPipelineRemoteValue, tftypes.UnknownValue)),
			expectedValue:     testLogsPipelineStateValue,
			expectedUpdatedAt: testLogsPipelineStateTimestamp,
		},
		{
			name:          "real change keeps the planned value and unknown timestamp",
			stateRaw:      tfValuePtr(logsPipelineTestValue(ctx, s, testLogsPipelineStateValue, testLogsPipelineStateTimestamp)),
			planRaw:       tfValuePtr(logsPipelineTestValue(ctx, s, testLogsPipelineOtherValue, tftypes.UnknownValue)),
			expectedValue: testLogsPipelineOtherValue,
			expectUnknown: true,
		},
		{
			name:              "unchanged plan is left alone",
			stateRaw:          tfValuePtr(logsPipelineTestValue(ctx, s, testLogsPipelineStateValue, testLogsPipelineStateTimestamp)),
			planRaw:           tfValuePtr(logsPipelineTestValue(ctx, s, testLogsPipelineStateValue, testLogsPipelineStateTimestamp)),
			expectedValue:     testLogsPipelineStateValue,
			expectedUpdatedAt: testLogsPipelineStateTimestamp,
		},
		{
			name:          "create plan has no prior state to reconcile against",
			stateRaw:      nil,
			planRaw:       tfValuePtr(logsPipelineTestValue(ctx, s, testLogsPipelineStateValue, tftypes.UnknownValue)),
			expectedValue: testLogsPipelineStateValue,
			expectUnknown: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The client must not be consulted during plan; a nil ApiClient makes any call
			// panic, proving the plan no longer hits the API.
			r := &logsPipelineResource{}

			state := tfsdk.State{Schema: s, Raw: nullLogsPipelineValue(ctx, s)}
			if tc.stateRaw != nil {
				state.Raw = tc.stateRaw.Copy()
			}
			plan := tfsdk.Plan{Schema: s, Raw: nullLogsPipelineValue(ctx, s)}
			if tc.planRaw != nil {
				plan.Raw = tc.planRaw.Copy()
			}

			resp := &fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: s, Raw: plan.Raw.Copy()}}
			r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{State: state, Plan: plan}, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			var got logsPipelineResourceModel
			if diags := resp.Plan.Get(ctx, &got); diags.HasError() {
				t.Fatalf("could not read modified plan: %v", diags)
			}

			if got.Value.ValueString() != tc.expectedValue {
				t.Errorf("value = %q, want %q", got.Value.ValueString(), tc.expectedValue)
			}
			if tc.expectUnknown {
				if !got.UpdatedAt.IsUnknown() {
					t.Errorf("updated_at = %v, want unknown", got.UpdatedAt)
				}
			} else if got.UpdatedAt.ValueString() != tc.expectedUpdatedAt {
				t.Errorf("updated_at = %q, want %q", got.UpdatedAt.ValueString(), tc.expectedUpdatedAt)
			}
		})
	}
}

// A destroy plan must be left untouched rather than erroring on the absent plan data.
func TestLogsPipelineModifyPlanOnDestroy(t *testing.T) {
	ctx := context.Background()
	s := logsPipelineTestSchema(ctx, t)

	r := &logsPipelineResource{}
	state := tfsdk.State{Schema: s, Raw: logsPipelineTestValue(ctx, s, testLogsPipelineStateValue, testLogsPipelineStateTimestamp)}
	plan := tfsdk.Plan{Schema: s, Raw: nullLogsPipelineValue(ctx, s)}

	resp := &fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: s, Raw: plan.Raw.Copy()}}
	r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{State: state, Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.Plan.Raw.IsNull() {
		t.Errorf("expected the destroy plan to stay null, got %s", resp.Plan.Raw)
	}
}

func nullLogsPipelineValue(ctx context.Context, s fwschema.Schema) tftypes.Value {
	return tftypes.NewValue(s.Type().TerraformType(ctx), nil)
}

func tfValuePtr(v tftypes.Value) *tftypes.Value {
	return &v
}
