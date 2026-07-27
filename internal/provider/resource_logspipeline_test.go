// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gopkg.in/yaml.v3"
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
			// Same config, replanned: refreshing must not introduce a diff.
			{
				Config:             testAccLogsPipelineResourceConfig(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Same pipeline, reformatted YAML (indentation, key order, a comment): still
			// no diff, so no write and no new timestamp.
			{
				Config:             testAccLogsPipelineResourceConfigReformatted(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
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

const (
	testLogsPipelineStateValue = "ottlRules:\n- ruleName: test-rule\n  conditions:\n    - container_name == \"nginx\"\n"
	// Same document as testLogsPipelineStateValue, serialized differently.
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

//
// These drive the provider through the real Terraform CLI against an in-process stand-in
// for the pipeline config API, so the perpetual-diff regression is covered without
// credentials. They exercise what the resource-level tests above cannot: whether Terraform
// itself considers the resulting plan empty.
//
// They need the `terraform` binary and skip cleanly when it is missing.

// fakeLogsPipelineBackend stands in for the append-only pipeline config store.
type fakeLogsPipelineBackend struct {
	t *testing.T

	// mu guards the fields below, written by handler goroutines and read by the test.
	mu      sync.Mutex
	stored  string
	present bool
	writes  int

	// remoteFn transforms the stored value before returning it, standing in for the
	// backend's own serialization. Nil means "return it verbatim".
	remoteFn func(string) string

	// bumpTimestampOnRead reports a later timestamp on every read.
	bumpTimestampOnRead bool
	readBumps           int
}

func (f *fakeLogsPipelineBackend) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(logsPipelineEndpoint, func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			if !f.present {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if f.bumpTimestampOnRead {
				f.readBumps++
			}
			f.respond(w, http.StatusOK)
		case http.MethodPost, http.MethodPut:
			var body struct{ Value string }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				f.t.Errorf("could not decode %s body: %v", r.Method, err)
			}
			f.stored = body.Value
			f.present = true
			f.writes++
			status := http.StatusOK
			if r.Method == http.MethodPost {
				status = http.StatusCreated
			}
			f.respond(w, status)
		case http.MethodDelete:
			f.present = false
			f.stored = ""
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"deleted"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func (f *fakeLogsPipelineBackend) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes
}

// respond must be called with f.mu held.
func (f *fakeLogsPipelineBackend) respond(w http.ResponseWriter, status int) {
	value := f.stored
	if f.remoteFn != nil {
		value = f.remoteFn(f.stored)
	}

	// Writes advance the timestamp; reads advance it too when bumpTimestampOnRead is set.
	ts := fmt.Sprintf("2026-07-27T09:%02d:00.000Z", f.writes)
	if f.bumpTimestampOnRead && f.readBumps > 0 {
		ts = fmt.Sprintf("2026-07-27T10:%02d:00.000Z", f.readBumps)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"uuid":              "6b09aee0-9001-4095-afb1-152b487e5bdd",
		"value":             value,
		"created_timestamp": ts,
		"created_by":        "terraform",
	})
}

// reserializeYaml round-trips a document through a YAML library, the way the config store
// does: same meaning, different bytes.
func reserializeYaml(stored string) string {
	if stored == "" {
		return ""
	}
	var data interface{}
	if err := yaml.Unmarshal([]byte(stored), &data); err != nil {
		return stored
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return stored
	}
	return string(out)
}

// startFakeLogsPipelineBackend returns the backend and a provider block pointing at it.
func startFakeLogsPipelineBackend(t *testing.T, backend *fakeLogsPipelineBackend) (*fakeLogsPipelineBackend, string) {
	t.Helper()

	if _, err := exec.LookPath("terraform"); err != nil {
		if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
			t.Skip("terraform CLI not found in PATH; skipping fake-backend regression test")
		}
	}

	backend.t = t
	srv := httptest.NewServer(backend.handler())
	t.Cleanup(srv.Close)

	return backend, `
provider "groundcover" {
  api_key    = "fake-api-key"
  backend_id = "fake-backend"
  api_url    = "` + srv.URL + `"
}
`
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

// The backend hands back its own serialization of an unchanged pipeline; the next plan must
// still be empty.
func TestLogsPipelineNoPerpetualDiffAgainstReserializingBackend(t *testing.T) {
	backend, providerBlock := startFakeLogsPipelineBackend(t, &fakeLogsPipelineBackend{remoteFn: reserializeYaml})

	cfg := fakeLogsPipelineConfig(providerBlock, "test-rule")
	const configValue = "ottlRules:\n- ruleName: test-rule\n  conditions:\n    - container_name == \"nginx\"\n"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				// State keeps the practitioner's YAML, not the backend's re-serialization.
				Check: resource.TestCheckResourceAttr("groundcover_logspipeline.test", "value", configValue),
			},
			{
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})

	if backend.writeCount() != 1 {
		t.Errorf("expected exactly 1 write (the create), got %d", backend.writeCount())
	}
}

// A refresh that sees a newer backend timestamp for an unchanged pipeline must not move
// updated_at, which is what drift detection reports on.
func TestLogsPipelineUpdatedAtStableWhenUnchanged(t *testing.T) {
	backend, providerBlock := startFakeLogsPipelineBackend(t, &fakeLogsPipelineBackend{
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
				Check:  resource.TestCheckResourceAttr("groundcover_logspipeline.test", "updated_at", createTimestamp),
			},
			{
				Config: cfg,
				Check:  resource.TestCheckResourceAttr("groundcover_logspipeline.test", "updated_at", createTimestamp),
			},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})

	if backend.writeCount() != 1 {
		t.Errorf("expected exactly 1 write (the create), got %d", backend.writeCount())
	}
}

// Reindenting or reordering the YAML in the .tf file must not schedule a write.
func TestLogsPipelineReformattedConfigPlansNoChange(t *testing.T) {
	_, providerBlock := startFakeLogsPipelineBackend(t, &fakeLogsPipelineBackend{})

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
			{
				Config:             reformatted,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// The reconciliation must not swallow an actual content change, and the resource must
// settle after one apply.
func TestLogsPipelineRealChangeStillApplies(t *testing.T) {
	backend, providerBlock := startFakeLogsPipelineBackend(t, &fakeLogsPipelineBackend{remoteFn: reserializeYaml})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: fakeLogsPipelineConfig(providerBlock, "first")},
			{Config: fakeLogsPipelineConfig(providerBlock, "first"), PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				Config: fakeLogsPipelineConfig(providerBlock, "second"),
				Check: resource.TestMatchResourceAttr("groundcover_logspipeline.test", "value",
					regexp.MustCompile(`ruleName: second`)),
			},
			{Config: fakeLogsPipelineConfig(providerBlock, "second"), PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})

	if backend.writeCount() != 2 {
		t.Errorf("expected exactly 2 writes (create plus one real update), got %d", backend.writeCount())
	}
}
