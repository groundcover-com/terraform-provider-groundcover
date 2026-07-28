// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// storageMockClient stubs only the storage management API; the embedded
// ApiClient interface satisfies the rest and panics if anything else is called.
type storageMockClient struct {
	ApiClient
	getResp      *models.StorageManagementPolicyResponse
	getErr       error
	updateResp   *models.StorageManagementPolicyResponse
	updateErr    error
	gotDataType  string
	updateReq    *models.StorageManagementPolicyRequest
	updateCalled bool
}

func (m *storageMockClient) GetStorageManagementPolicy(_ context.Context, dataType string) (*models.StorageManagementPolicyResponse, error) {
	m.gotDataType = dataType
	return m.getResp, m.getErr
}

func (m *storageMockClient) UpdateStorageManagementPolicy(_ context.Context, dataType string, req *models.StorageManagementPolicyRequest) (*models.StorageManagementPolicyResponse, error) {
	m.updateCalled = true
	m.gotDataType = dataType
	m.updateReq = req
	return m.updateResp, m.updateErr
}

func storagePolicyTestResource(client ApiClient) *storageManagementPolicyResource {
	return &storageManagementPolicyResource{client: client}
}

func storagePolicyTestState(t *testing.T, model storageManagementPolicyResourceModel) tfsdk.State {
	t.Helper()
	schemaResp := &resource.SchemaResponse{}
	(&storageManagementPolicyResource{}).Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to build state: %v", diags)
	}
	return state
}

func storagePolicyStateModel(t *testing.T, state tfsdk.State) storageManagementPolicyResourceModel {
	t.Helper()
	var model storageManagementPolicyResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to read state: %v", diags)
	}
	return model
}

func TestStoragePolicyCreateAdoptsExistingPolicy(t *testing.T) {
	ruleName, ruleRetention, ruleFilters := "debug", "3d", "level = 'debug'"
	mock := &storageMockClient{
		getResp: &models.StorageManagementPolicyResponse{DataType: "logs", Retention: "90d", Version: 3},
		updateResp: &models.StorageManagementPolicyResponse{
			DataType: "logs", Retention: "30d", ColdVolume: "cold", ColdMoveDuration: "7d",
			Version: 4, UUID: "policy-uuid",
			CustomRules: []*models.CustomRule{{Name: &ruleName, Retention: &ruleRetention, Filters: &ruleFilters}},
		},
	}
	plan := storageManagementPolicyResourceModel{
		DataType:         types.StringValue("logs"),
		Retention:        types.StringValue("30d"),
		ColdMoveDuration: types.StringValue("7d"),
		CustomRules: []storageCustomRuleModel{
			{Name: types.StringValue("debug"), Retention: types.StringValue("3d"), Filters: types.StringValue("level = 'debug'")},
		},
	}

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: storagePolicyTestState(t, plan).Schema}}
	storagePolicyTestResource(mock).Create(context.Background(),
		resource.CreateRequest{Plan: tfsdk.Plan(storagePolicyTestState(t, plan))}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	req := mock.updateReq
	if req == nil || req.Version == nil || *req.Version != 3 {
		t.Fatalf("PUT must pass through the GET version (3) unchanged, got %+v", req)
	}
	if req.Retention == nil || *req.Retention != "30d" {
		t.Fatalf("Retention = %v, want 30d", req.Retention)
	}
	if req.ColdVolume != coldVolume || req.ColdMoveDuration != "7d" {
		t.Fatalf("cold fields not mapped: cold_volume=%q cold_move_duration=%q", req.ColdVolume, req.ColdMoveDuration)
	}
	if req.CreatedBy != "" {
		t.Fatalf("CreatedBy = %q, want empty (backend sets it from the caller)", req.CreatedBy)
	}
	if len(req.CustomRules) != 1 || *req.CustomRules[0].Name != "debug" ||
		*req.CustomRules[0].Retention != "3d" || *req.CustomRules[0].Filters != "level = 'debug'" {
		t.Fatalf("custom rules not mapped: %#v", req.CustomRules)
	}

	state := storagePolicyStateModel(t, resp.State)
	if state.Version.ValueInt64() != 4 || state.UUID.ValueString() != "policy-uuid" {
		t.Fatalf("state must reflect the PUT response (version 4, uuid), got %#v", state)
	}
	if state.ColdMoveDuration.ValueString() != "7d" || len(state.CustomRules) != 1 {
		t.Fatalf("state not mapped from response: %#v", state)
	}
}

func TestStoragePolicyCreateFailsWhenPolicyNotSeeded(t *testing.T) {
	mock := &storageMockClient{getErr: ErrNotFound}
	plan := storageManagementPolicyResourceModel{
		DataType:  types.StringValue("logs"),
		Retention: types.StringValue("30d"),
	}

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: storagePolicyTestState(t, plan).Schema}}
	storagePolicyTestResource(mock).Create(context.Background(),
		resource.CreateRequest{Plan: tfsdk.Plan(storagePolicyTestState(t, plan))}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when no seeded policy exists")
	}
	if mock.updateCalled {
		t.Fatal("PUT must not be attempted when the policy does not exist")
	}
}

func TestStoragePolicyCreateBlocksUndeclaredExistingRules(t *testing.T) {
	// Adopting must not silently archive existing rules the config doesn't declare —
	// archived rule names can never be reused, so Create errors before the PUT.
	existingName, ret, filters := "existing-rule", "3d", "level = 'debug'"
	cases := []struct {
		name    string
		rules   []storageCustomRuleModel
		wantErr bool
	}{
		{"omitted", nil, true},
		{"different rule declared", []storageCustomRuleModel{
			{Name: types.StringValue("other"), Retention: types.StringValue("5d"), Filters: types.StringValue("level = 'info'")},
		}, true},
		{"existing rule declared", []storageCustomRuleModel{
			{Name: types.StringValue("existing-rule"), Retention: types.StringValue("5d"), Filters: types.StringValue("level = 'info'")},
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &storageMockClient{
				getResp: &models.StorageManagementPolicyResponse{
					DataType: "logs", Retention: "90d", Version: 3,
					CustomRules: []*models.CustomRule{{Name: &existingName, Retention: &ret, Filters: &filters}},
				},
				updateResp: &models.StorageManagementPolicyResponse{DataType: "logs", Retention: "30d", Version: 4},
			}
			plan := storageManagementPolicyResourceModel{
				DataType:    types.StringValue("logs"),
				Retention:   types.StringValue("30d"),
				CustomRules: tc.rules,
			}
			resp := &resource.CreateResponse{State: tfsdk.State{Schema: storagePolicyTestState(t, plan).Schema}}
			storagePolicyTestResource(mock).Create(context.Background(),
				resource.CreateRequest{Plan: tfsdk.Plan(storagePolicyTestState(t, plan))}, resp)

			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("HasError() = %v, want %v (%v)", resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
			if mock.updateCalled == tc.wantErr {
				t.Fatalf("updateCalled = %v, want %v — the PUT must only run when adoption is allowed", mock.updateCalled, !tc.wantErr)
			}
		})
	}
}

func TestStoragePolicyCreateWarnsWhenColdTierNotDeclared(t *testing.T) {
	// Adopting without cold_move_duration replaces a policy that had a cold tier with
	// one that doesn't; that is reversible, so it warns instead of erroring.
	cases := []struct {
		name     string
		plan     types.String
		wantWarn bool
	}{
		{"omitted", types.StringNull(), true},
		{"declared", types.StringValue("7d"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &storageMockClient{
				getResp:    &models.StorageManagementPolicyResponse{DataType: "logs", Retention: "90d", ColdVolume: "cold", ColdMoveDuration: "7d", Version: 3},
				updateResp: &models.StorageManagementPolicyResponse{DataType: "logs", Retention: "30d", Version: 4},
			}
			plan := storageManagementPolicyResourceModel{
				DataType:         types.StringValue("logs"),
				Retention:        types.StringValue("30d"),
				ColdMoveDuration: tc.plan,
			}
			resp := &resource.CreateResponse{State: tfsdk.State{Schema: storagePolicyTestState(t, plan).Schema}}
			storagePolicyTestResource(mock).Create(context.Background(),
				resource.CreateRequest{Plan: tfsdk.Plan(storagePolicyTestState(t, plan))}, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected errors: %v", resp.Diagnostics)
			}
			if !mock.updateCalled {
				t.Fatal("the PUT must still run — the cold-tier warning is not blocking")
			}
			warned := false
			for _, w := range resp.Diagnostics.Warnings() {
				if w.Summary() == "Existing Cold Storage Configuration Not Declared" {
					warned = true
				}
			}
			if warned != tc.wantWarn {
				t.Fatalf("cold-tier warning = %v, want %v (%v)", warned, tc.wantWarn, resp.Diagnostics)
			}
		})
	}
}

func TestStoragePolicyCreatePreservesExplicitEmptyCustomRules(t *testing.T) {
	// custom_rules = [] (e.g. an empty for-expression) must round-trip as an empty
	// list, not null, or Terraform fails with an inconsistent-result error.
	mock := &storageMockClient{
		getResp:    &models.StorageManagementPolicyResponse{Retention: "90d", Version: 1},
		updateResp: &models.StorageManagementPolicyResponse{Retention: "30d", Version: 2},
	}
	plan := storageManagementPolicyResourceModel{
		DataType:    types.StringValue("logs"),
		Retention:   types.StringValue("30d"),
		CustomRules: []storageCustomRuleModel{},
	}

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: storagePolicyTestState(t, plan).Schema}}
	storagePolicyTestResource(mock).Create(context.Background(),
		resource.CreateRequest{Plan: tfsdk.Plan(storagePolicyTestState(t, plan))}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	state := storagePolicyStateModel(t, resp.State)
	if state.CustomRules == nil || len(state.CustomRules) != 0 {
		t.Fatalf("explicitly empty custom_rules must stay an empty list, got %#v", state.CustomRules)
	}
}

func TestStoragePolicyReadRemovesResourceWhenNotFound(t *testing.T) {
	mock := &storageMockClient{getErr: ErrNotFound}
	state := storagePolicyTestState(t, storageManagementPolicyResourceModel{
		DataType:  types.StringValue("logs"),
		Retention: types.StringValue("30d"),
	})

	resp := &resource.ReadResponse{State: state}
	storagePolicyTestResource(mock).Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("resource must be removed from state when the policy is not found")
	}
}

func TestStoragePolicyReadRefreshesFromAPI(t *testing.T) {
	mock := &storageMockClient{
		getResp: &models.StorageManagementPolicyResponse{DataType: "logs", Retention: "60d", Version: 8, UUID: "policy-uuid"},
	}
	state := storagePolicyTestState(t, storageManagementPolicyResourceModel{
		DataType:  types.StringValue("logs"),
		Retention: types.StringValue("30d"),
		Version:   types.Int64Value(7),
	})

	resp := &resource.ReadResponse{State: state}
	storagePolicyTestResource(mock).Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	got := storagePolicyStateModel(t, resp.State)
	if mock.gotDataType != "logs" || got.Retention.ValueString() != "60d" || got.Version.ValueInt64() != 8 {
		t.Fatalf("state not refreshed from API: %#v", got)
	}
	// An omitted optional field in the response must map to null, not "".
	if !got.ColdMoveDuration.IsNull() {
		t.Fatalf("empty cold_move_duration must be null, got %v", got.ColdMoveDuration)
	}
}

func TestStoragePolicyUpdatePassesCurrentVersionThrough(t *testing.T) {
	mock := &storageMockClient{
		updateResp: &models.StorageManagementPolicyResponse{DataType: "logs", Retention: "14d", Version: 8},
	}
	plan := storageManagementPolicyResourceModel{
		DataType:  types.StringValue("logs"),
		Retention: types.StringValue("14d"),
	}
	state := storageManagementPolicyResourceModel{
		DataType:  types.StringValue("logs"),
		Retention: types.StringValue("30d"),
		Version:   types.Int64Value(7),
	}

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: storagePolicyTestState(t, plan).Schema}}
	storagePolicyTestResource(mock).Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan(storagePolicyTestState(t, plan)),
		State: storagePolicyTestState(t, state),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	req := mock.updateReq
	if req == nil || req.Version == nil || *req.Version != 7 {
		t.Fatalf("PUT must pass through the state version (7) unchanged, got %+v", req)
	}
	// cold_volume and cold_move_duration must be set together or the API rejects
	// the request; with no cold_move_duration both must be omitted.
	if req.ColdVolume != "" || req.ColdMoveDuration != "" {
		t.Fatalf("cold fields must be omitted without cold_move_duration, got cold_volume=%q cold_move_duration=%q", req.ColdVolume, req.ColdMoveDuration)
	}
	if got := storagePolicyStateModel(t, resp.State); got.Version.ValueInt64() != 8 {
		t.Fatalf("state must carry the backend-bumped version (8), got %d", got.Version.ValueInt64())
	}
}

func TestStoragePolicyDataTypeChangeIsBlockedAtPlanTime(t *testing.T) {
	// data_type must fail at plan time: a replace would need a delete, which the
	// backend does not support, so the apply could never converge.
	cases := []struct {
		name        string
		state, plan types.String
		wantErr     bool
	}{
		{"create", types.StringNull(), types.StringValue("logs"), false},
		{"unchanged", types.StringValue("logs"), types.StringValue("logs"), false},
		{"changed", types.StringValue("logs"), types.StringValue("traces"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}
			dataTypeImmutable{}.PlanModifyString(context.Background(),
				planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan}, resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("HasError() = %v, want %v (%v)", resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestStoragePolicyDeleteWarnsWithoutCallingAPI(t *testing.T) {
	// Policies have no delete API; Delete must succeed with only a warning so the
	// framework removes the resource from state, and must never reach the client.
	r := &storageManagementPolicyResource{client: &storageMockClient{}}
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected Delete to succeed, got errors: %v", resp.Diagnostics.Errors())
	}
	warnings := resp.Diagnostics.Warnings()
	if len(warnings) != 1 || warnings[0].Summary() != "Storage Management Policy Not Deleted" {
		t.Fatalf("expected a single not-deleted warning, got: %v", warnings)
	}

	// The same warning must surface at plan time on a destroy plan (null plan).
	planResp := &resource.ModifyPlanResponse{}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{}, planResp)
	if got := planResp.Diagnostics.Warnings(); len(got) != 1 || got[0].Summary() != "Storage Management Policy Not Deleted" {
		t.Fatalf("expected destroy-plan warning, got: %v", got)
	}

	// A non-destroy plan must not warn.
	normalPlan := tfsdk.Plan(storagePolicyTestState(t, storageManagementPolicyResourceModel{
		DataType:  types.StringValue("logs"),
		Retention: types.StringValue("30d"),
	}))
	noWarnResp := &resource.ModifyPlanResponse{}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: normalPlan}, noWarnResp)
	if got := noWarnResp.Diagnostics.Warnings(); len(got) != 0 {
		t.Fatalf("non-destroy plan must not warn, got: %v", got)
	}
}
