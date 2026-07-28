// Copyright groundcover 2026
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &storageManagementPolicyResource{}
var _ resource.ResourceWithConfigure = &storageManagementPolicyResource{}
var _ resource.ResourceWithImportState = &storageManagementPolicyResource{}
var _ resource.ResourceWithModifyPlan = &storageManagementPolicyResource{}

func NewStorageManagementPolicyResource() resource.Resource {
	return &storageManagementPolicyResource{}
}

type storageManagementPolicyResource struct{ client ApiClient }

// coldVolume is fixed for all policies and is not user-configurable.
const coldVolume = "cold"

// dataTypeImmutable rejects data_type changes at plan time. A replace would
// silently abandon the old policy (destroy is a state-only no-op) while its
// applied configuration stays active, so require an explicit new resource instead.
type dataTypeImmutable struct{}

func (dataTypeImmutable) Description(ctx context.Context) string {
	return "data_type cannot be changed after the policy is managed"
}

func (m dataTypeImmutable) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (dataTypeImmutable) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Data Type Cannot Be Changed",
			fmt.Sprintf("Changing data_type (from %q to %q) would replace the resource, leaving the %q policy active but unmanaged. "+
				"Add a separate resource for the new data type and remove this one instead.",
				req.StateValue.ValueString(), req.PlanValue.ValueString(), req.StateValue.ValueString()),
		)
	}
}

type storageManagementPolicyResourceModel struct {
	DataType         types.String             `tfsdk:"data_type"`
	Retention        types.String             `tfsdk:"retention"`
	ColdMoveDuration types.String             `tfsdk:"cold_move_duration"`
	CustomRules      []storageCustomRuleModel `tfsdk:"custom_rules"`
	Version          types.Int64              `tfsdk:"version"`
	UUID             types.String             `tfsdk:"uuid"`
	CreatedTimestamp types.String             `tfsdk:"created_timestamp"`
}

type storageCustomRuleModel struct {
	Name      types.String `tfsdk:"name"`
	Retention types.String `tfsdk:"retention"`
	Filters   types.String `tfsdk:"filters"`
}

func (r *storageManagementPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_management_policy"
}

func (r *storageManagementPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the groundcover storage management (retention) policy for a single data type. " +
			"Policies are seeded by groundcover and can only be updated, never created or deleted — destroying this resource only removes it from Terraform state and leaves the policy active with its current configuration. " +
			"Applying replaces the entire policy with the configured values: custom rules that are not declared are archived and their names cannot be reused. " +
			"Adopting a policy whose existing custom rules are not all declared in the configuration fails — `terraform import` it first and align the configuration.",
		Attributes: map[string]schema.Attribute{
			"data_type": schema.StringAttribute{
				MarkdownDescription: "Data type the policy applies to (e.g. `logs`, `traces`, `events`). Identifies the policy and cannot be changed. Validated by the API on apply.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{dataTypeImmutable{}},
			},
			"retention": schema.StringAttribute{
				MarkdownDescription: "Default retention duration for this data type (e.g. `30d`).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"cold_move_duration": schema.StringAttribute{
				MarkdownDescription: "Optional duration after which data is moved to cold storage (e.g. `7d`).",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"custom_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Ordered custom retention rules; supported only for the `logs`, `traces`, and `events` data types. " +
					"The full list is replaced on every update. Removing a rule archives it in groundcover and its name cannot be reused.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":      schema.StringAttribute{MarkdownDescription: "Rule name.", Required: true},
						"retention": schema.StringAttribute{MarkdownDescription: "Retention duration for data matched by this rule.", Required: true},
						"filters":   schema.StringAttribute{MarkdownDescription: "Filter expression selecting the data this rule applies to.", Required: true},
					},
				},
			},
			"version":           schema.Int64Attribute{MarkdownDescription: "Policy version, managed by groundcover.", Computed: true},
			"uuid":              schema.StringAttribute{MarkdownDescription: "Policy UUID returned by the API.", Computed: true},
			"created_timestamp": schema.StringAttribute{MarkdownDescription: "Timestamp of the current policy version.", Computed: true},
		},
	}
}

func (r *storageManagementPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(ApiClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected provider.ApiClient, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *storageManagementPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageManagementPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dataType := plan.DataType.ValueString()

	// Adopt the existing (seeded) policy to obtain its current version, which the update requires.
	existing, err := r.client.GetStorageManagementPolicy(ctx, dataType)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.Diagnostics.AddError(
				"Storage Management Policy Not Found",
				fmt.Sprintf("No storage management policy exists for data type %q. Policies are seeded by groundcover and cannot be created via Terraform.", dataType),
			)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Storage Management Policy", fmt.Sprintf("Failed to read policy for %q: %s", dataType, err))
		return
	}

	// Refuse to adopt over existing rules the config doesn't declare: applying would
	// archive them irreversibly (archived rule names can never be reused).
	if undeclared := undeclaredCustomRules(plan.CustomRules, existing.CustomRules); len(undeclared) > 0 {
		resp.Diagnostics.AddError(
			"Existing Custom Rules Not Declared",
			fmt.Sprintf("The %q policy has custom rules not present in the configuration: %s. "+
				"Applying would archive them permanently and their names could never be reused. "+
				"Import the policy and align the configuration first, e.g. by adding:\n\n"+
				"import {\n  to = groundcover_storage_management_policy.<name>\n  id = %q\n}",
				dataType, strings.Join(undeclared, ", "), dataType),
		)
		return
	}

	updated, err := r.client.UpdateStorageManagementPolicy(ctx, dataType, storagePolicyRequestFromModel(plan, existing.Version))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Storage Management Policy", fmt.Sprintf("Failed to update policy for %q: %s", dataType, err))
		return
	}
	model := storagePolicyModelFromAPI(dataType, updated)
	preserveExplicitEmptyCustomRules(plan.CustomRules, &model)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *storageManagementPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageManagementPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dataType := state.DataType.ValueString()
	policy, err := r.client.GetStorageManagementPolicy(ctx, dataType)
	if errors.Is(err, ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Storage Management Policy", fmt.Sprintf("Failed to read policy for %q: %s", dataType, err))
		return
	}
	model := storagePolicyModelFromAPI(dataType, policy)
	preserveExplicitEmptyCustomRules(state.CustomRules, &model)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *storageManagementPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state storageManagementPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dataType := plan.DataType.ValueString()

	// Pass the current version through; groundcover bumps it and the response carries the new value.
	updated, err := r.client.UpdateStorageManagementPolicy(ctx, dataType, storagePolicyRequestFromModel(plan, state.Version.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Storage Management Policy", fmt.Sprintf("Failed to update policy for %q: %s", dataType, err))
		return
	}
	model := storagePolicyModelFromAPI(dataType, updated)
	preserveExplicitEmptyCustomRules(plan.CustomRules, &model)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

const (
	storagePolicyDeleteWarningSummary = "Storage Management Policy Not Deleted"
	storagePolicyDeleteWarningDetail  = "groundcover has no delete API for storage management policies: a data type must always have a retention policy. " +
		"Destroying this resource only removes it from Terraform state; the policy remains active with its current configuration."
)

// ModifyPlan surfaces the destroy semantics at plan time, before the practitioner confirms.
func (r *storageManagementPolicyResource) ModifyPlan(_ context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning(storagePolicyDeleteWarningSummary, storagePolicyDeleteWarningDetail)
	}
}

// Delete never calls the API; returning only a warning lets the framework remove the resource from state.
func (r *storageManagementPolicyResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(storagePolicyDeleteWarningSummary, storagePolicyDeleteWarningDetail)
}

func (r *storageManagementPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("data_type"), req, resp)
}

func storagePolicyRequestFromModel(model storageManagementPolicyResourceModel, version int64) *models.StorageManagementPolicyRequest {
	retention := model.Retention.ValueString()
	coldMoveDuration := model.ColdMoveDuration.ValueString()
	req := &models.StorageManagementPolicyRequest{
		Retention:        &retention,
		Version:          &version,
		ColdMoveDuration: coldMoveDuration,
		CustomRules:      make([]*models.CustomRule, 0, len(model.CustomRules)),
	}
	// cold_volume and cold_move_duration must be set together, or the API rejects the request.
	if coldMoveDuration != "" {
		req.ColdVolume = coldVolume
	}
	for _, rule := range model.CustomRules {
		name, ruleRetention, filters := rule.Name.ValueString(), rule.Retention.ValueString(), rule.Filters.ValueString()
		req.CustomRules = append(req.CustomRules, &models.CustomRule{Name: &name, Retention: &ruleRetention, Filters: &filters})
	}
	return req
}

func storagePolicyModelFromAPI(dataType string, policy *models.StorageManagementPolicyResponse) storageManagementPolicyResourceModel {
	model := storageManagementPolicyResourceModel{
		DataType:         types.StringValue(dataType),
		Retention:        types.StringValue(policy.Retention),
		ColdMoveDuration: optionalStorageString(policy.ColdMoveDuration),
		Version:          types.Int64Value(policy.Version),
		UUID:             types.StringValue(policy.UUID),
		CreatedTimestamp: types.StringValue(policy.CreatedTimestamp.String()),
	}
	for _, rule := range policy.CustomRules {
		if rule == nil {
			continue
		}
		model.CustomRules = append(model.CustomRules, storageCustomRuleModel{
			Name:      types.StringValue(stringValue(rule.Name)),
			Retention: types.StringValue(stringValue(rule.Retention)),
			Filters:   types.StringValue(stringValue(rule.Filters)),
		})
	}
	return model
}

// preserveExplicitEmptyCustomRules keeps an explicitly empty custom_rules list ([]) from
// collapsing to null, which the API response cannot distinguish from an omitted list and
// which Terraform would reject as an inconsistent result after apply.
func preserveExplicitEmptyCustomRules(prior []storageCustomRuleModel, model *storageManagementPolicyResourceModel) {
	if prior != nil && len(prior) == 0 && model.CustomRules == nil {
		model.CustomRules = []storageCustomRuleModel{}
	}
}

// optionalStorageString maps an empty API string to null so an omitted optional attribute does not drift against "".
func optionalStorageString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func undeclaredCustomRules(declared []storageCustomRuleModel, existing []*models.CustomRule) []string {
	names := make(map[string]struct{}, len(declared))
	for _, rule := range declared {
		names[rule.Name.ValueString()] = struct{}{}
	}
	var undeclared []string
	for _, rule := range existing {
		if rule == nil {
			continue
		}
		if _, ok := names[stringValue(rule.Name)]; !ok {
			undeclared = append(undeclared, stringValue(rule.Name))
		}
	}
	return undeclared
}
