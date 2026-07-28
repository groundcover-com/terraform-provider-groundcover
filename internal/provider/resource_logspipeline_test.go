// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"regexp"
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
				),
			},
			// Update and Read testing
			{
				Config: testAccLogsPipelineResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_logspipeline.test", "updated_at"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccLogsPipelineResourceConfig() string {
	return `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule
  conditions:
    - container_name == "nginx"
  statements:
    - set(attributes["test.key"], "test-value")
YAML
}
`
}

func testAccLogsPipelineResourceConfigUpdated() string {
	return `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule-updated
  conditions:
    - container_name == "nginx"
  statements:
    - set(attributes["test.key"], "test-value-updated")
YAML
}
`
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
	return `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
# managed by terraform
ottlRules:
  - conditions:
      - container_name == "nginx"
    statements:
      - set(attributes["test.key"], "test-value")
    ruleName: test-rule
YAML
}
`
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

func testLogsPipelineRemoteConfig(value string) *models.LogsPipelineConfig {
	ts, err := time.Parse(time.RFC3339, testPipelineRemoteTimestamp)
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
			remote:            testLogsPipelineRemoteConfig(testPipelineRemoteValue),
			stateValue:        testPipelineStateValue,
			stateUpdatedAt:    testPipelineStateTimestamp,
			expectedValue:     testPipelineStateValue,
			expectedUpdatedAt: testPipelineStateTimestamp,
		},
		{
			name:              "byte-identical remote value keeps state timestamp",
			remote:            testLogsPipelineRemoteConfig(testPipelineStateValue),
			stateValue:        testPipelineStateValue,
			stateUpdatedAt:    testPipelineStateTimestamp,
			expectedValue:     testPipelineStateValue,
			expectedUpdatedAt: testPipelineStateTimestamp,
		},
		{
			name:              "genuinely changed remote value is adopted with its timestamp",
			remote:            testLogsPipelineRemoteConfig(testPipelineOtherValue),
			stateValue:        testPipelineStateValue,
			stateUpdatedAt:    testPipelineStateTimestamp,
			expectedValue:     testPipelineOtherValue,
			expectedUpdatedAt: testPipelineRemoteTimestamp,
		},
		{
			name:              "missing timestamp in state is repopulated from the API",
			remote:            testLogsPipelineRemoteConfig(testPipelineRemoteValue),
			stateValue:        testPipelineStateValue,
			stateUpdatedAt:    nil,
			expectedValue:     testPipelineStateValue,
			expectedUpdatedAt: testPipelineRemoteTimestamp,
		},
		{
			name:              "empty remote config clears the value",
			remote:            nil,
			stateValue:        testPipelineStateValue,
			stateUpdatedAt:    testPipelineStateTimestamp,
			expectedValue:     "",
			expectedUpdatedAt: "",
		},
		{
			name:              "empty remote config clears the timestamp even when state is already empty",
			remote:            nil,
			stateValue:        "",
			stateUpdatedAt:    testPipelineStateTimestamp,
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

	raw := logsPipelineTestValue(ctx, s, testPipelineStateValue, testPipelineStateTimestamp)
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
			stateRaw:          tfValuePtr(logsPipelineTestValue(ctx, s, testPipelineStateValue, testPipelineStateTimestamp)),
			planRaw:           tfValuePtr(logsPipelineTestValue(ctx, s, testPipelineRemoteValue, tftypes.UnknownValue)),
			expectedValue:     testPipelineStateValue,
			expectedUpdatedAt: testPipelineStateTimestamp,
		},
		{
			name:          "real change keeps the planned value and unknown timestamp",
			stateRaw:      tfValuePtr(logsPipelineTestValue(ctx, s, testPipelineStateValue, testPipelineStateTimestamp)),
			planRaw:       tfValuePtr(logsPipelineTestValue(ctx, s, testPipelineOtherValue, tftypes.UnknownValue)),
			expectedValue: testPipelineOtherValue,
			expectUnknown: true,
		},
		{
			name:              "unchanged plan is left alone",
			stateRaw:          tfValuePtr(logsPipelineTestValue(ctx, s, testPipelineStateValue, testPipelineStateTimestamp)),
			planRaw:           tfValuePtr(logsPipelineTestValue(ctx, s, testPipelineStateValue, testPipelineStateTimestamp)),
			expectedValue:     testPipelineStateValue,
			expectedUpdatedAt: testPipelineStateTimestamp,
		},
		{
			name:          "create plan has no prior state to reconcile against",
			stateRaw:      nil,
			planRaw:       tfValuePtr(logsPipelineTestValue(ctx, s, testPipelineStateValue, tftypes.UnknownValue)),
			expectedValue: testPipelineStateValue,
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
	state := tfsdk.State{Schema: s, Raw: logsPipelineTestValue(ctx, s, testPipelineStateValue, testPipelineStateTimestamp)}
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

// An unchanged pipeline must not drift: a backend that re-serializes the stored document and
// reports a newer timestamp on every read must move neither value nor updated_at, and must
// not cause a write. updated_at is the attribute drift detection reports on.
func TestLogsPipelineUnchangedPipelineDoesNotDrift(t *testing.T) {
	backend, providerBlock := startFakePipelineBackend(t, &fakePipelineBackend{
		endpoint:            logsPipelineEndpoint,
		remoteFn:            reserializeYaml,
		bumpTimestampOnRead: true,
	})

	cfg := fakeLogsPipelineConfig(providerBlock, "test-rule")
	const createTimestamp = "2026-07-27T09:01:00.000Z"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					// State keeps the practitioner's YAML, not the backend's re-serialization.
					resource.TestCheckResourceAttr("groundcover_logspipeline.test", "value", testPipelineStateValue),
					resource.TestCheckResourceAttr("groundcover_logspipeline.test", "updated_at", createTimestamp),
				),
			},
			// Re-applying refreshes first, so this asserts the refresh left both alone. The
			// framework plans after every apply step and fails on a non-empty plan, which
			// covers the no-diff half without a separate PlanOnly step.
			{
				Config: cfg,
				Check:  resource.TestCheckResourceAttr("groundcover_logspipeline.test", "updated_at", createTimestamp),
			},
		},
	})

	if backend.writeCount() != 1 {
		t.Errorf("expected exactly 1 write (the create), got %d", backend.writeCount())
	}
}

// Reindenting or reordering the YAML in the .tf file must not schedule a write.
func TestLogsPipelineReformattedConfigPlansNoChange(t *testing.T) {
	_, providerBlock := startFakePipelineBackend(t, &fakePipelineBackend{endpoint: logsPipelineEndpoint})

	reformatted := providerBlock + `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
# managed by terraform
ottlRules:
  - conditions:
      - container_name == "nginx"
    ruleName: test-rule
YAML
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: fakeLogsPipelineConfig(providerBlock, "test-rule")},
			{Config: reformatted, PlanOnly: true},
		},
	})
}

// The reconciliation must not swallow an actual content change, and the resource must
// settle after one apply.
func TestLogsPipelineRealChangeStillApplies(t *testing.T) {
	backend, providerBlock := startFakePipelineBackend(t, &fakePipelineBackend{endpoint: logsPipelineEndpoint, remoteFn: reserializeYaml})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: fakeLogsPipelineConfig(providerBlock, "first")},
			{
				Config: fakeLogsPipelineConfig(providerBlock, "second"),
				Check: resource.TestMatchResourceAttr("groundcover_logspipeline.test", "value",
					regexp.MustCompile(`ruleName: second`)),
			},
		},
	})

	if backend.writeCount() != 2 {
		t.Errorf("expected exactly 2 writes (create plus one real update), got %d", backend.writeCount())
	}
}

func fakeLogsPipelineConfig(providerBlock, rule string) string {
	return providerBlock + `
resource "groundcover_logspipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: ` + rule + `
  conditions:
    - container_name == "nginx"
YAML
}
`
}
