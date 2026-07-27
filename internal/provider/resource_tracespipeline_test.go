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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTracesPipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTracesPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "updated_at"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTracesPipelineResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "updated_at"),
					resource.TestMatchResourceAttr("groundcover_tracespipeline.test", "value", regexp.MustCompile("test-rule-updated")),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTracesPipelineResourceConfig() string {
	return `
resource "groundcover_tracespipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule
  conditions:
    - workload == "nginx"
  statements:
    - set(attributes["test.key"], "test-value")
YAML
}
`
}

func testAccTracesPipelineResourceConfigUpdated() string {
	return `
resource "groundcover_tracespipeline" "test" {
  value = <<-YAML
ottlRules:
- ruleName: test-rule-updated
  conditions:
    - workload == "nginx"
  statements:
    - set(attributes["test.key"], "test-value-updated")
YAML
}
`
}

// Mirrors TestAccLogsPipelineResource_noDiffOnReformattedYaml.
func TestAccTracesPipelineResource_noDiffOnReformattedYaml(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTracesPipelineResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "value"),
					resource.TestCheckResourceAttrSet("groundcover_tracespipeline.test", "updated_at"),
				),
			},
			// Same config, replanned: refreshing must not introduce a diff.
			{
				Config:             testAccTracesPipelineResourceConfig(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Same pipeline, reformatted YAML: still no diff, so no write and no new
			// timestamp.
			{
				Config:             testAccTracesPipelineResourceConfigReformatted(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// testAccTracesPipelineResourceConfigReformatted is testAccTracesPipelineResourceConfig
// with the same ottlRules written differently: deeper sequence indentation, reordered
// mapping keys, and an added comment.
func testAccTracesPipelineResourceConfigReformatted() string {
	return `
resource "groundcover_tracespipeline" "test" {
  value = <<-YAML
# managed by terraform
ottlRules:
  - conditions:
      - workload == "nginx"
    statements:
      - set(attributes["test.key"], "test-value")
    ruleName: test-rule
YAML
}
`
}

// stubTracesPipelineClient serves a canned GetTracesPipeline response.
type stubTracesPipelineClient struct {
	ApiClient
	config *models.TracesPipelineConfig
	err    error
}

func (c *stubTracesPipelineClient) GetTracesPipeline(context.Context) (*models.TracesPipelineConfig, error) {
	return c.config, c.err
}

// Mirrors the logs pipeline refresh test; both resources share the same config store.
func TestTracesPipelineReadKeepsSemanticallyUnchangedState(t *testing.T) {
	ctx := context.Background()

	var r tracesPipelineResource
	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema

	remoteTs, err := time.Parse(time.RFC3339, testLogsPipelineRemoteTimestamp)
	if err != nil {
		t.Fatalf("could not parse test timestamp: %v", err)
	}

	tests := []struct {
		name              string
		remoteAbsent      bool
		remoteValue       string
		expectedValue     string
		expectedUpdatedAt string
	}{
		{
			name:              "absent singleton clears the value and timestamp",
			remoteAbsent:      true,
			expectedValue:     "",
			expectedUpdatedAt: "",
		},
		{
			name:              "reformatted remote value keeps state value and timestamp",
			remoteValue:       testLogsPipelineRemoteValue,
			expectedValue:     testLogsPipelineStateValue,
			expectedUpdatedAt: testLogsPipelineStateTimestamp,
		},
		{
			name:              "genuinely changed remote value is adopted with its timestamp",
			remoteValue:       testLogsPipelineOtherValue,
			expectedValue:     testLogsPipelineOtherValue,
			expectedUpdatedAt: testLogsPipelineRemoteTimestamp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
				"value":      tftypes.NewValue(tftypes.String, testLogsPipelineStateValue),
				"updated_at": tftypes.NewValue(tftypes.String, testLogsPipelineStateTimestamp),
			})

			var remote *models.TracesPipelineConfig
			if !tc.remoteAbsent {
				remote = &models.TracesPipelineConfig{
					UUID:             "d6f4d1a0-0f4c-4b6a-9f5e-3f2b1c0d9e8a",
					Value:            tc.remoteValue,
					CreatedTimestamp: strfmt.DateTime(remoteTs),
				}
			}

			res := &tracesPipelineResource{client: &stubTracesPipelineClient{config: remote}}

			resp := &fwresource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw.Copy()}}
			res.Read(ctx, fwresource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			var got tracesPipelineResourceModel
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

// Mirrors the logs pipeline plan test.
func TestTracesPipelineModifyPlanCollapsesFormattingOnlyDiff(t *testing.T) {
	ctx := context.Background()

	var r tracesPipelineResource
	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	objType := s.Type().TerraformType(ctx)

	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"value":      tftypes.NewValue(tftypes.String, testLogsPipelineStateValue),
		"updated_at": tftypes.NewValue(tftypes.String, testLogsPipelineStateTimestamp),
	})
	planRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"value":      tftypes.NewValue(tftypes.String, testLogsPipelineRemoteValue),
		"updated_at": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	// A nil ApiClient makes any API call panic, proving the plan no longer hits the API.
	res := &tracesPipelineResource{}
	resp := &fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: s, Raw: planRaw.Copy()}}
	res.ModifyPlan(ctx, fwresource.ModifyPlanRequest{
		State: tfsdk.State{Schema: s, Raw: stateRaw},
		Plan:  tfsdk.Plan{Schema: s, Raw: planRaw},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	var got tracesPipelineResourceModel
	if diags := resp.Plan.Get(ctx, &got); diags.HasError() {
		t.Fatalf("could not read modified plan: %v", diags)
	}

	if got.Value.ValueString() != testLogsPipelineStateValue {
		t.Errorf("value = %q, want the prior state value %q", got.Value.ValueString(), testLogsPipelineStateValue)
	}
	if got.UpdatedAt.ValueString() != testLogsPipelineStateTimestamp {
		t.Errorf("updated_at = %q, want the prior state timestamp %q", got.UpdatedAt.ValueString(), testLogsPipelineStateTimestamp)
	}
}
