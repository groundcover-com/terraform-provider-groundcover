// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	// SDK Imports
	sdkreq "github.com/groundcover-com/groundcover-sdk-go/sdk/api/rbac/policies" // Policy request types (CreatePolicyRequest, etc.)
	sdkmodels "github.com/groundcover-com/groundcover-sdk-go/sdk/models"         // Policy model types (Policy, DataScope, RoleMap)
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &policyResource{}
var _ resource.ResourceWithConfigure = &policyResource{}

// var _ resource.ResourceWithImportState = &policyResource{} // TODO: Implement ImportState

// policyResource defines the resource implementation.
type policyResource struct {
	client ApiClient // Removed unused 'version' field
}

// policyResourceModel describes the resource data model.
type policyResourceModel struct {
	UUID           types.String `tfsdk:"uuid"`
	Name           types.String `tfsdk:"name"`
	Role           types.Map    `tfsdk:"role"`
	Description    types.String `tfsdk:"description"`
	ClaimRole      types.String `tfsdk:"claim_role"`
	DataScope      types.Object `tfsdk:"data_scope"`
	RevisionNumber types.Int64  `tfsdk:"revision_number"`
	ReadOnly       types.Bool   `tfsdk:"read_only"`
}

// dataScopeModel maps the data_scope block schema.
// Matches sdkpolicies.DataScope
type dataScopeModel struct {
	Simple types.Object `tfsdk:"simple"`
	// Advanced types.Object `tfsdk:"advanced"` // Assuming SDK might add advanced later
}

// simpleDataScopeModel maps the simple block schema within data_scope.
// Matches sdkpolicies.SimpleDataScope
type simpleDataScopeModel struct {
	Operator   types.String `tfsdk:"operator"`
	Conditions types.List   `tfsdk:"conditions"`
}

// conditionModel maps the conditions block schema.
// Matches sdkmodels.Condition
type conditionModel struct {
	Key     types.String `tfsdk:"key"`
	Origin  types.String `tfsdk:"origin"`
	Type    types.String `tfsdk:"type"`
	Filters types.List   `tfsdk:"filters"`
}

// filtersModel maps the filters block schema.
// Matches sdkmodels.Filter
type filtersModel struct {
	Op    types.String `tfsdk:"op"`
	Value types.String `tfsdk:"value"`
}

// Define nested schema for the 'filters' block
var filtersNestedSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"op": schema.StringAttribute{
			MarkdownDescription: "The filter operation (e.g., 'match').",
			Required:            true,
		},
		"value": schema.StringAttribute{
			MarkdownDescription: "The value to filter on.",
			Required:            true,
		},
	},
}

// Define nested schema for the 'conditions' block (used in ListNestedAttribute)
var conditionNestedSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"key": schema.StringAttribute{
			MarkdownDescription: "The key for the condition (e.g., 'environment').",
			Required:            true,
		},
		"origin": schema.StringAttribute{
			MarkdownDescription: "The origin of the key.",
			Required:            true,
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "The type of the key.",
			Required:            true,
		},
		"filters": schema.ListNestedAttribute{
			MarkdownDescription: "List of filter criteria for the condition.",
			Required:            true,
			NestedObject:        filtersNestedSchema,
		},
	},
}

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

func (r *policyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a groundcover RBAC policy.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the policy.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the policy.",
				Required:            true,
			},
			"role": schema.MapAttribute{
				MarkdownDescription: "Role definitions associated with the policy. Maps role identifiers to specific permissions or access levels.",
				ElementType:         types.StringType,
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description for the policy.",
				Optional:            true,
			},
			"claim_role": schema.StringAttribute{
				MarkdownDescription: "SSO Role claim name used for mapping.",
				Optional:            true,
			},
			"data_scope": schema.SingleNestedAttribute{
				MarkdownDescription: "Defines the data scope restrictions for the policy.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"simple": schema.SingleNestedAttribute{
						MarkdownDescription: "Simple data scope configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"operator": schema.StringAttribute{
								MarkdownDescription: "Logical operator (e.g., 'and', 'or').",
								Required:            true,
							},
							"conditions": schema.ListNestedAttribute{
								MarkdownDescription: "List of conditions for the data scope.",
								Required:            true,
								NestedObject:        conditionNestedSchema,
							},
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"revision_number": schema.Int64Attribute{
				MarkdownDescription: "Revision number of the policy, used for concurrency control.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"read_only": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the policy is read-only (managed internally).",
				Computed:            true,
			},
		},
	}
}

// Configure retrieves the provider configuration and sets up the resource client.
func (r *policyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		// Provider specific data logic
		return
	}

	client, ok := req.ProviderData.(ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected provider.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
	tflog.Info(ctx, "Policy resource configured successfully")
}

// --- CRUD Operations ---

// Create creates the policy resource.
func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRequest, diags := mapPolicyModelToApiCreateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "CreatePolicy SDK Call Request constructed", map[string]any{"name": apiRequest.Name})
	apiResponse, err := r.client.CreatePolicy(ctx, apiRequest)
	if err != nil {
		// Use the specific name from the request in the error message
		resp.Diagnostics.AddError("SDK Client Create Error", fmt.Sprintf("Failed to create policy '%s': %s", apiRequest.Name, err.Error()))
		return
	}
	tflog.Info(ctx, "Policy created successfully via SDK", map[string]any{"uuid": apiResponse.UUID})

	// Update state with computed values from SDK response
	plan.UUID = types.StringValue(apiResponse.UUID)
	plan.RevisionNumber = types.Int64Value(int64(apiResponse.RevisionNumber)) // Cast int32 to int64
	plan.ReadOnly = types.BoolValue(apiResponse.ReadOnly)

	// Update other fields from the API response to reflect the actual state
	// (Name, Role, Description, ClaimRole, DataScope)
	diags = mapPolicyApiResponseToModel(ctx, *apiResponse, &plan) // Map full response back to plan
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		// If mapping fails, still set the essential computed fields, but log error
		tflog.Error(ctx, "Failed to map full API response back to state after create", map[string]any{"uuid": apiResponse.UUID})
		// Re-set essential computed fields just in case mapping overwrote them
		plan.UUID = types.StringValue(apiResponse.UUID)
		plan.RevisionNumber = types.Int64Value(int64(apiResponse.RevisionNumber))
		plan.ReadOnly = types.BoolValue(apiResponse.ReadOnly)
		// Keep plan values for user-settable fields as a fallback
	}

	tflog.Info(ctx, "Saving new policy to state", map[string]any{"uuid": plan.UUID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read reads the policy resource configuration.
func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID := state.UUID.ValueString()
	tflog.Debug(ctx, "GetPolicy SDK Call Request", map[string]any{"uuid": policyUUID})
	apiResponse, err := r.client.GetPolicy(ctx, policyUUID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Policy not found via SDK, removing from state", map[string]any{"uuid": policyUUID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("SDK Client Read Error", fmt.Sprintf("Failed to read policy %s: %s", policyUUID, err.Error()))
		return
	}
	tflog.Debug(ctx, "Policy read successfully via SDK", map[string]any{"uuid": apiResponse.UUID})

	// Update state from SDK Response
	diags := mapPolicyApiResponseToModel(ctx, *apiResponse, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return // Stop if mapping fails
	}

	tflog.Info(ctx, "Updating policy state after read", map[string]any{"uuid": state.UUID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the policy resource.
func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID := state.UUID.ValueString()
	// Use state's revision number for the update request
	apiRequest, diags := mapPolicyModelToApiUpdateRequest(ctx, plan, state.RevisionNumber.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "UpdatePolicy SDK Call Request constructed", map[string]any{"uuid": policyUUID, "revision": apiRequest.CurrentRevision})
	apiResponse, err := r.client.UpdatePolicy(ctx, policyUUID, apiRequest)
	if err != nil {
		// Add specific check for ReadOnly error if the wrapper returns it
		if errors.Is(err, ErrReadOnly) {
			resp.Diagnostics.AddError("SDK Policy ReadOnly Error", fmt.Sprintf("Failed to update policy %s because it is read-only.", policyUUID))
		} else if errors.Is(err, ErrConcurrency) {
			resp.Diagnostics.AddError("SDK Concurrency Error", fmt.Sprintf("Failed to update policy %s due to revision mismatch. Please refresh state and try again.", policyUUID))
		} else if errors.Is(err, ErrNotFound) {
			resp.Diagnostics.AddError("SDK Not Found Error", fmt.Sprintf("Failed to update policy %s because it was not found. It may have been deleted externally.", policyUUID))
		} else {
			resp.Diagnostics.AddError("SDK Client Update Error", fmt.Sprintf("Failed to update policy %s: %s", policyUUID, err.Error()))
		}
		return
	}
	tflog.Info(ctx, "Policy updated successfully via SDK", map[string]any{"uuid": apiResponse.UUID, "new_revision": apiResponse.RevisionNumber})

	// Update state (plan) from SDK Response
	// Preserve UUID from state as SDK response might not always contain it (though it should for update)
	plan.UUID = state.UUID
	diags = mapPolicyApiResponseToModel(ctx, *apiResponse, &plan) // Map full response back to plan
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		// If mapping fails, still set the essential computed fields, but log error
		tflog.Error(ctx, "Failed to map full API response back to state after update", map[string]any{"uuid": apiResponse.UUID})
		// Re-set essential computed fields just in case mapping overwrote them
		plan.UUID = state.UUID // Ensure UUID is preserved
		plan.RevisionNumber = types.Int64Value(int64(apiResponse.RevisionNumber))
		plan.ReadOnly = types.BoolValue(apiResponse.ReadOnly)
		// Keep plan values for user-settable fields as a fallback? Or error out? Erroring out seems safer.
		return // Stop if mapping fails after update
	}

	tflog.Info(ctx, "Saving updated policy to state", map[string]any{"uuid": plan.UUID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the policy resource.
func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID := state.UUID.ValueString()
	tflog.Debug(ctx, "DeletePolicy SDK Call Request", map[string]any{"uuid": policyUUID})
	err := r.client.DeletePolicy(ctx, policyUUID)
	if err != nil {
		// ErrNotFound is handled by the wrapper for idempotency (returns nil)
		// Check for other specific errors like ReadOnly
		if errors.Is(err, ErrReadOnly) {
			resp.Diagnostics.AddError("SDK Policy ReadOnly Error", fmt.Sprintf("Failed to delete policy %s because it is read-only.", policyUUID))
			return // Keep in state if delete fails because it's read-only
		} else {
			// Use the error returned by the wrapper (which could be nil for NotFound)
			resp.Diagnostics.AddError("SDK Client Delete Error", fmt.Sprintf("Failed to delete policy %s: %s", policyUUID, err.Error()))
			return // Keep in state if delete fails unexpectedly
		}
	}

	tflog.Info(ctx, "Policy deleted successfully via SDK (or was already gone)", map[string]any{"uuid": policyUUID})
	// Terraform automatically removes the resource from state when Delete returns no error.
}

// --- Helper Functions for Mapping ---

// mapPolicyModelToApiCreateRequest converts Terraform model to SDK request struct for Create.
func mapPolicyModelToApiCreateRequest(ctx context.Context, plan policyResourceModel) (sdkreq.CreatePolicyRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiRequest := sdkreq.CreatePolicyRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		ClaimRole:   plan.ClaimRole.ValueStringPointer(),
	}

	// Map 'role' (sdkmodels.RoleMap is map[string]string)
	apiRequest.Role = make(sdkreq.RoleMap) // Initialize map
	if !plan.Role.IsNull() {
		// Let ElementsAs populate the map directly from the plan
		diags.Append(plan.Role.ElementsAs(ctx, &apiRequest.Role, false)...)
		if diags.HasError() {
			return apiRequest, diags
		}

		// Check if the required role map is empty after reading from plan
		if len(apiRequest.Role) == 0 {
			diags.AddError("Validation Error", "The 'role' attribute is required and cannot be empty.")
			return apiRequest, diags
		}
		// No need to manually populate keys with empty strings
	} else {
		// This case should ideally not be reached due to TF schema Required=true
		diags.AddError("Validation Error", "The 'role' attribute is required but was found to be null in the plan.")
		return apiRequest, diags
	}

	// Map 'data_scope'
	apiDataScope, dsDiags := mapModelDataScopeToApiDataScope(ctx, plan.DataScope)
	diags.Append(dsDiags...)
	if diags.HasError() {
		return apiRequest, diags
	}
	// If apiDataScope is not nil, dereference it and assign the struct value.
	if apiDataScope != nil {
		apiRequest.DataScope = *apiDataScope // Restore dereference
	}
	// If apiDataScope is nil, apiRequest.DataScope retains its zero-value struct

	return apiRequest, diags
}

// mapPolicyModelToApiUpdateRequest converts Terraform model to SDK request struct for Update.
func mapPolicyModelToApiUpdateRequest(ctx context.Context, plan policyResourceModel, revision int64) (sdkreq.UpdatePolicyRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiRequest := sdkreq.UpdatePolicyRequest{
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueStringPointer(),
		ClaimRole:       plan.ClaimRole.ValueStringPointer(),
		CurrentRevision: int32(revision), // Cast int64 to int32 for SDK
	}

	// Map 'role' (sdkmodels.RoleMap is map[string]string)
	roleMap := make(sdkreq.RoleMap) // Use SDK type directly
	if !plan.Role.IsNull() {
		diags.Append(plan.Role.ElementsAs(ctx, &roleMap, false)...)
		if diags.HasError() {
			return apiRequest, diags
		}
		apiRequest.Role = roleMap // Assign directly
	} else {
		apiRequest.Role = make(sdkreq.RoleMap) // Ensure non-nil map
	}

	// Map 'data_scope'
	apiDataScope, dsDiags := mapModelDataScopeToApiDataScope(ctx, plan.DataScope)
	diags.Append(dsDiags...)
	if diags.HasError() {
		return apiRequest, diags
	}
	// If apiDataScope is not nil, dereference it and assign the struct value.
	if apiDataScope != nil {
		apiRequest.DataScope = *apiDataScope // Restore dereference
	}
	// If apiDataScope is nil, apiRequest.DataScope retains its zero-value struct

	tflog.Debug(ctx, "Mapped update request from plan", map[string]any{"revision": apiRequest.CurrentRevision})
	return apiRequest, diags
}

// mapModelDataScopeToApiDataScope converts the Terraform data_scope object to the SDK *sdkmodels.DataScope struct.
func mapModelDataScopeToApiDataScope(ctx context.Context, modelDataScope types.Object) (*sdkreq.DataScope, diag.Diagnostics) {
	var diags diag.Diagnostics

	if modelDataScope.IsNull() || modelDataScope.IsUnknown() {
		return nil, diags // No data scope configured or known yet
	}

	var dataScopePlan dataScopeModel
	conversionDiags := modelDataScope.As(ctx, &dataScopePlan, basetypes.ObjectAsOptions{})
	diags.Append(conversionDiags...)
	if diags.HasError() {
		return nil, diags
	}

	apiDataScope := &sdkreq.DataScope{} // Initialize with SDK type

	// Map 'simple' scope
	if !dataScopePlan.Simple.IsNull() && !dataScopePlan.Simple.IsUnknown() {
		var simplePlan simpleDataScopeModel
		conversionDiags = dataScopePlan.Simple.As(ctx, &simplePlan, basetypes.ObjectAsOptions{})
		diags.Append(conversionDiags...)
		if diags.HasError() {
			return nil, diags
		}

		apiSimpleScope := &sdkmodels.Group{ // Use SDK type
			Operator: simplePlan.Operator.ValueString(),
		}

		// Map 'conditions'
		if !simplePlan.Conditions.IsNull() && !simplePlan.Conditions.IsUnknown() {
			conditionsPlan := make([]conditionModel, 0, len(simplePlan.Conditions.Elements()))
			diags.Append(simplePlan.Conditions.ElementsAs(ctx, &conditionsPlan, false)...)
			if diags.HasError() {
				return nil, diags
			}

			apiSimpleScope.Conditions = make([]sdkmodels.Condition, len(conditionsPlan)) // Use []*sdkmodels.Condition
			for i, condPlan := range conditionsPlan {
				apiCondition := &sdkmodels.Condition{ // Use sdkmodels.Condition
					Column: sdkmodels.Column{
						Key:    condPlan.Key.ValueString(),
						Origin: condPlan.Origin.ValueString(),
						Type:   condPlan.Type.ValueString(),
					},
				}

				// Map 'filters'
				if !condPlan.Filters.IsNull() && !condPlan.Filters.IsUnknown() && len(condPlan.Filters.Elements()) > 0 {
					filtersPlan := make([]filtersModel, 0, len(condPlan.Filters.Elements()))
					conversionDiags = condPlan.Filters.ElementsAs(ctx, &filtersPlan, false)
					diags.Append(conversionDiags...)
					if diags.HasError() {
						return nil, diags
					}

					apiCondition.Filters = make([]sdkmodels.Filter, len(filtersPlan)) // Use []*sdkmodels.Filter
					for j, filterPlan := range filtersPlan {
						apiCondition.Filters[j] = sdkmodels.Filter{ // Use sdkmodels.Filter
							Op:    filterPlan.Op.ValueString(),
							Value: filterPlan.Value.ValueString(),
						}
					}
				} else {
					apiCondition.Filters = make([]sdkmodels.Filter, 0) // Use []*sdkmodels.Filter
				}
				apiSimpleScope.Conditions[i] = *apiCondition
			}
		} else {
			apiSimpleScope.Conditions = make([]sdkmodels.Condition, 0) // Use []*sdkmodels.Condition
		}
		apiDataScope.Simple = apiSimpleScope // Assign pointer
	}
	// TODO: Add mapping for Advanced scope if/when SDK supports it

	// If no scopes were actually mapped (e.g., simple was null), return nil
	if apiDataScope.Simple == nil /* && apiDataScope.Advanced == nil */ {
		return nil, diags
	}

	return apiDataScope, diags
}

// mapPolicyApiResponseToModel maps the *sdkmodels.Policy from an API response
// back into the Terraform state model (*policyResourceModel).
func mapPolicyApiResponseToModel(ctx context.Context, apiResponse sdkreq.Policy, state *policyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	tflog.Debug(ctx, "Mapping SDK Policy response to Terraform model", map[string]any{"uuid": apiResponse.UUID})

	state.UUID = types.StringValue(apiResponse.UUID)
	state.RevisionNumber = types.Int64Value(int64(apiResponse.RevisionNumber)) // Cast int32 to int64
	state.ReadOnly = types.BoolValue(apiResponse.ReadOnly)
	state.Name = types.StringValue(apiResponse.Name)
	state.Description = types.StringPointerValue(apiResponse.Description) // Works for nil string pointers
	state.ClaimRole = types.StringPointerValue(apiResponse.ClaimRole)     // Works for nil string pointers

	// Map 'role' (sdkmodels.RoleMap is map[string]string)
	roleMapValue, roleDiags := types.MapValueFrom(ctx, types.StringType, apiResponse.Role)
	diags.Append(roleDiags...)
	if diags.HasError() {
		tflog.Error(ctx, "Failed to map Role from SDK response", map[string]any{"uuid": apiResponse.UUID})
		return diags // Stop if role mapping fails
	}
	state.Role = roleMapValue

	// Map 'data_scope' from SDK response back to state model
	dataScopeValue, dsDiags := mapApiResponseDataScopeToState(ctx, &apiResponse.DataScope) // Pass pointer to *sdkmodels.DataScope
	diags.Append(dsDiags...)
	if diags.HasError() {
		tflog.Error(ctx, "Failed to map DataScope from SDK response", map[string]any{"uuid": apiResponse.UUID})
		return diags // Stop if data scope mapping fails
	}
	state.DataScope = dataScopeValue

	tflog.Debug(ctx, "Successfully mapped SDK response to Terraform model", map[string]any{"uuid": apiResponse.UUID})
	return diags
}

// mapApiResponseDataScopeToState converts the *sdkmodels.DataScope from an API response
// back into the Terraform state object type (types.Object).
func mapApiResponseDataScopeToState(ctx context.Context, apiDataScope *sdkreq.DataScope) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Get the target attribute types for the top-level data_scope block
	dataScopeAttrTypes := dataScopeModel{}.attrTypes()
	tflog.Debug(ctx, "Mapping DataScope from SDK response to state object")

	// If apiDataScope is nil or has no actual scope defined (e.g., Simple is nil), return null object.
	if apiDataScope == nil || apiDataScope.Simple == nil /* && apiDataScope.Advanced == nil */ {
		tflog.Debug(ctx, "SDK DataScope is nil or has no defined scope (Simple is nil), returning null object")
		return types.ObjectNull(dataScopeAttrTypes), diags
	}

	// --- Map Simple Scope ---
	var simpleScopeObj attr.Value = types.ObjectNull(dataScopeAttrTypes["simple"].(types.ObjectType).AttrTypes) // Default to null
	var simpleScopeDiags diag.Diagnostics

	if apiDataScope.Simple != nil {
		tflog.Debug(ctx, "Mapping Simple scope from SDK response")
		apiSimpleScope := apiDataScope.Simple // apiSimpleScope is *sdkmodels.SimpleDataScope
		// Get target attribute types for the simple scope block
		simpleAttrTypes := simpleDataScopeModel{}.attrTypes()
		// Get target attribute types for nested lists/objects
		conditionAttrTypes := conditionModel{}.attrTypes() // Corrected: get types from model instance
		filtersAttrTypes := filtersModel{}.attrTypes()     // Corrected: get types from model instance

		// Build the conditions list first
		conditionsList, condDiags := buildConditionsListFromApi(ctx, apiSimpleScope.Conditions, filtersAttrTypes, conditionAttrTypes)
		diags.Append(condDiags...)
		if diags.HasError() {
			tflog.Error(ctx, "Failed to build conditions list from SDK response")
			// Return null for the entire data_scope if conditions fail
			return types.ObjectNull(dataScopeAttrTypes), diags
		}

		// Build the simple scope object itself
		simpleScopeObj, simpleScopeDiags = types.ObjectValue(simpleAttrTypes, map[string]attr.Value{
			"operator":   types.StringValue(apiSimpleScope.Operator),
			"conditions": conditionsList,
		})
		diags.Append(simpleScopeDiags...)
		if diags.HasError() {
			tflog.Error(ctx, "Failed to build simple scope object from SDK response")
			// Return null for the entire data_scope if simple scope fails
			return types.ObjectNull(dataScopeAttrTypes), diags
		}
		tflog.Debug(ctx, "Successfully built simple scope object")
	}

	// --- Map Advanced Scope (Placeholder) ---
	// var advancedScopeObj attr.Value = types.ObjectNull(...) // Assuming schema exists
	// if apiDataScope.Advanced != nil { ... }

	// --- Build final data_scope object ---
	dataScopeAttrMap := map[string]attr.Value{
		"simple": simpleScopeObj,
		// "advanced": advancedScopeObj,
	}

	dataScopeObj, objDiags := types.ObjectValue(dataScopeAttrTypes, dataScopeAttrMap)
	diags.Append(objDiags...)
	if diags.HasError() {
		tflog.Error(ctx, "Failed to build final data scope object from SDK response")
		return types.ObjectNull(dataScopeAttrTypes), diags
	}

	tflog.Debug(ctx, "Successfully mapped DataScope from SDK response to state object")
	return dataScopeObj, diags
}

func buildConditionsListFromApi(ctx context.Context, apiConditions []sdkmodels.Condition, filtersAttrTypes map[string]attr.Type, conditionAttrTypes map[string]attr.Type) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	var conditionsObjList []attr.Value

	// Define element type for the conditions list
	conditionsElemType := types.ObjectType{AttrTypes: conditionAttrTypes}

	// Get the element type for the nested filters list
	filtersListType := conditionAttrTypes["filters"].(types.ListType) // Assume type assertion is safe based on schema
	filtersElemType := filtersListType.ElemType.(types.ObjectType)    // Assume type assertion is safe

	tflog.Debug(ctx, "Building conditions list from SDK response", map[string]any{"count": len(apiConditions)})

	if len(apiConditions) == 0 {
		tflog.Debug(ctx, "No conditions in SDK response, returning empty list")
		// Use the determined element type to create an empty list
		return types.ListValueMust(conditionsElemType, []attr.Value{}), diags
	}

	conditionsObjList = make([]attr.Value, 0, len(apiConditions))
	for i, apiCondition := range apiConditions {
		tflog.Debug(ctx, "Processing SDK condition", map[string]any{
			"index":        i,
			"key":          apiCondition.Key,
			"origin":       apiCondition.Origin,
			"type":         apiCondition.Type,
			"filtersCount": len(apiCondition.Filters),
		})

		// Build the filters list for this condition
		var filtersList types.List
		var filtersListDiags diag.Diagnostics
		if len(apiCondition.Filters) > 0 {
			filtersObjList := make([]attr.Value, 0, len(apiCondition.Filters))
			for j, apiFilter := range apiCondition.Filters {
				tflog.Debug(ctx, "Processing SDK filter", map[string]any{
					"conditionIndex": i,
					"filterIndex":    j,
					"op":             apiFilter.Op,
					"value":          fmt.Sprintf("%v (%T)", apiFilter.Value, apiFilter.Value), // Log value and type
				})

				// Assert apiFilter.Value to string
				filterValueStr, ok := apiFilter.Value.(string)
				if !ok {
					diags.AddError("Filter Value Type Error", fmt.Sprintf("Filter value at condition index %d, filter index %d is not a string (type: %T)", i, j, apiFilter.Value))
					return types.ListNull(conditionsElemType), diags // Stop early
				}

				// Create filter object using the specific filtersAttrTypes
				filterObj, fDiags := types.ObjectValue(filtersAttrTypes, map[string]attr.Value{
					"op":    types.StringValue(apiFilter.Op),
					"value": types.StringValue(filterValueStr),
				})
				diags.Append(fDiags...)
				if diags.HasError() {
					tflog.Error(ctx, "Failed to convert SDK filter to object", map[string]any{"conditionIndex": i, "filterIndex": j})
					return types.ListNull(conditionsElemType), diags // Stop early
				}
				filtersObjList = append(filtersObjList, filterObj)
			}

			// Create the filters list using the determined element type
			filtersList, filtersListDiags = types.ListValue(filtersElemType, filtersObjList)
			diags.Append(filtersListDiags...)
			if diags.HasError() {
				tflog.Error(ctx, "Failed to create filters list from SDK response", map[string]any{"conditionIndex": i})
				return types.ListNull(conditionsElemType), diags // Stop early
			}
		} else {
			// Create an empty list with the correct element type
			filtersList = types.ListValueMust(filtersElemType, []attr.Value{})
		}

		// Build the condition object using the specific conditionAttrTypes
		conditionAttrMap := map[string]attr.Value{
			"key":     types.StringValue(apiCondition.Key),
			"origin":  types.StringValue(apiCondition.Origin),
			"type":    types.StringValue(apiCondition.Type),
			"filters": filtersList, // Assign the created list
		}

		conditionObj, cDiags := types.ObjectValue(conditionAttrTypes, conditionAttrMap)
		diags.Append(cDiags...)
		if diags.HasError() {
			tflog.Error(ctx, "Failed to convert SDK condition to object", map[string]any{"conditionIndex": i})
			return types.ListNull(conditionsElemType), diags // Stop early
		}

		tflog.Debug(ctx, "Successfully created condition object from SDK", map[string]any{"index": i})
		conditionsObjList = append(conditionsObjList, conditionObj)
	}

	// Create the final conditions list
	tflog.Debug(ctx, "Creating final conditions list from SDK response", map[string]any{"count": len(conditionsObjList)})
	finalConditionsList, clistDiags := types.ListValue(conditionsElemType, conditionsObjList)
	diags.Append(clistDiags...)

	if diags.HasError() {
		tflog.Error(ctx, "Failed to create final conditions list from SDK response")
		return types.ListNull(conditionsElemType), diags
	}

	tflog.Debug(ctx, "Successfully created conditions list from SDK response")
	return finalConditionsList, diags
}

// attrTypes returns attribute types for filtersModel
func (m filtersModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"op":    types.StringType,
		"value": types.StringType,
	}
}

// attrTypes returns attribute types for conditionModel
func (m conditionModel) attrTypes() map[string]attr.Type {
	filterType := types.ObjectType{
		AttrTypes: filtersModel{}.attrTypes(),
	}

	return map[string]attr.Type{
		"key":    types.StringType,
		"origin": types.StringType,
		"type":   types.StringType,
		"filters": types.ListType{
			ElemType: filterType,
		},
	}
}

// attrTypes returns attribute types for simpleDataScopeModel
func (m simpleDataScopeModel) attrTypes() map[string]attr.Type {
	conditionType := types.ObjectType{
		AttrTypes: conditionModel{}.attrTypes(),
	}

	return map[string]attr.Type{
		"operator": types.StringType,
		"conditions": types.ListType{
			ElemType: conditionType,
		},
	}
}

// attrTypes returns attribute types for dataScopeModel
func (m dataScopeModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"simple": types.ObjectType{
			AttrTypes: simpleDataScopeModel{}.attrTypes(),
		},
	}
}

// TODO: Implement ImportState function using the ApiClient interface
// func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	 policyUUID := req.ID
// 	 apiResponse, err := r.client.GetPolicy(ctx, policyUUID)
// 	 if err != nil {
// 		 // Handle error (e.g., Not Found)
// 		 return
// 	 }
// 	 var state policyResourceModel
// 	 diags := mapPolicyApiResponseToModel(ctx, *apiResponse, &state)
// 	 resp.Diagnostics.Append(diags...)
// 	 if resp.Diagnostics.HasError() {
// 		 return
// 	 }
// 	 resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
// }
