package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dashboardResource{}
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
	_ resource.ResourceWithModifyPlan  = &dashboardResource{}
)

func NewDashboardResource() resource.Resource {
	return &dashboardResource{}
}

type dashboardResource struct {
	client ApiClient
}

type dashboardResourceModel struct {
	UUID           types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Team           types.String `tfsdk:"team"`
	Preset         types.String `tfsdk:"preset"`
	RevisionNumber types.Int32  `tfsdk:"revision_number"`
	Override       types.Bool   `tfsdk:"override"`
	Owner          types.String `tfsdk:"owner"`
	Status         types.String `tfsdk:"status"`
}

func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Dashboard resource for managing groundcover dashboards.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The UUID of the dashboard.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the dashboard.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the dashboard.",
				Optional:    true,
			},
			"team": schema.StringAttribute{
				Description: "The team that owns the dashboard.",
				Optional:    true,
			},
			"preset": schema.StringAttribute{
				Description: "The preset configuration for the dashboard.",
				Required:    true,
			},
			"revision_number": schema.Int32Attribute{
				Description: "The revision number of the dashboard.",
				Computed:    true,
			},
			"override": schema.BoolAttribute{
				Description: "Whether to override the dashboard on update.",
				Optional:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The owner of the dashboard.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the dashboard.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *dashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
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
}

func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dashboardResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planPresetStr := plan.Preset.ValueString()
	tflog.Debug(ctx, "Create: Creating Dashboard", map[string]interface{}{
		"plan_name":           plan.Name.ValueString(),
		"plan_description":    plan.Description.ValueString(),
		"plan_team":           plan.Team.ValueString(),
		"plan_preset_len":     len(planPresetStr),
		"plan_preset_preview": getPreview(planPresetStr, 200),
	})

	createReq := &models.CreateDashboardRequest{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Team:          plan.Team.ValueString(),
		Preset:        planPresetStr,
		IsProvisioned: true,
	}

	dashboard, err := r.client.CreateDashboard(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Dashboard",
			fmt.Sprintf("Could not create dashboard: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Create: API response", map[string]interface{}{
		"uuid":                    dashboard.UUID,
		"response_name":           dashboard.Name,
		"response_description":    dashboard.Description,
		"response_team":           dashboard.Team,
		"response_revision":       dashboard.RevisionNumber,
		"response_preset_len":     len(dashboard.Preset),
		"response_preset_preview": getPreview(dashboard.Preset, 200),
	})

	plan.UUID = types.StringValue(dashboard.UUID)
	plan.Name = types.StringValue(dashboard.Name)
	plan.Description = types.StringValue(dashboard.Description)
	if plan.Team.IsNull() && dashboard.Team == "" {
		plan.Team = types.StringNull()
	} else {
		plan.Team = types.StringValue(dashboard.Team)
	}
	// Keep the user's original preset format if semantically the same
	apiPresetStr := dashboard.Preset
	areSemanticallySame, err := CompareJSONSemantically(planPresetStr, apiPresetStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Dashboard Preset",
			fmt.Sprintf("Failed to parse dashboard preset JSON: %s", err.Error()),
		)
		return
	}

	// Normalize for logging
	normalizedPlan, errPlanNorm := NormalizeJSON(ctx, planPresetStr)
	normalizedApi, errApiNorm := NormalizeJSON(ctx, apiPresetStr)
	if errPlanNorm == nil && errApiNorm == nil {
		tflog.Debug(ctx, "Create: Normalized preset comparison", map[string]interface{}{
			"uuid":                dashboard.UUID,
			"normalized_plan_len": len(normalizedPlan),
			"normalized_api_len":  len(normalizedApi),
			"normalized_equal":    normalizedPlan == normalizedApi,
		})
	}

	if !areSemanticallySame {
		tflog.Info(ctx, "Create: Preset JSON is semantically different, using API response", map[string]interface{}{
			"uuid":            dashboard.UUID,
			"plan_preset_len": len(planPresetStr),
			"api_preset_len":  len(apiPresetStr),
		})
		plan.Preset = types.StringValue(apiPresetStr)
	} else {
		tflog.Debug(ctx, "Create: Preset JSON is semantically same, keeping plan format", map[string]interface{}{
			"uuid":            dashboard.UUID,
			"plan_preset_len": len(planPresetStr),
			"api_preset_len":  len(apiPresetStr),
		})
	}
	plan.Owner = types.StringValue(dashboard.Owner)
	plan.Status = types.StringValue(dashboard.Status)
	plan.RevisionNumber = types.Int32Value(dashboard.RevisionNumber)

	tflog.Debug(ctx, "Dashboard created - setting state", map[string]interface{}{
		"uuid":            dashboard.UUID,
		"revision_number": dashboard.RevisionNumber,
		"preset_len":      len(dashboard.Preset),
		"plan_preset_len": len(plan.Preset.ValueString()),
	})

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Dashboard created successfully", map[string]interface{}{
		"uuid":            dashboard.UUID,
		"revision_number": dashboard.RevisionNumber,
	})
}

func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dashboardResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Dashboard", map[string]interface{}{
		"uuid":                  state.UUID.ValueString(),
		"state_revision_number": state.RevisionNumber.ValueInt32(),
	})

	dashboard, err := r.client.GetDashboard(ctx, state.UUID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Dashboard",
			fmt.Sprintf("Could not read dashboard %s: %s", state.UUID.ValueString(), err.Error()),
		)
		return
	}

	// Store the original state preset for comparison
	originalStatePreset := state.Preset.ValueString()
	apiPreset := dashboard.Preset

	// Log preset comparison details for debugging apply loop issues
	tflog.Debug(ctx, "Read: Comparing state preset with API preset", map[string]interface{}{
		"uuid":                 state.UUID.ValueString(),
		"state_preset_len":     len(originalStatePreset),
		"api_preset_len":       len(apiPreset),
		"presets_string_equal": originalStatePreset == apiPreset,
		"state_revision":       state.RevisionNumber.ValueInt32(),
		"api_revision":         dashboard.RevisionNumber,
	})

	state.UUID = types.StringValue(dashboard.UUID)
	state.Name = types.StringValue(dashboard.Name)
	state.Description = types.StringValue(dashboard.Description)
	if state.Team.IsNull() && dashboard.Team == "" {
		state.Team = types.StringNull()
	} else {
		state.Team = types.StringValue(dashboard.Team)
	}

	// Normalize both presets for detailed comparison logging
	normalizedStatePreset, errStateNorm := NormalizeJSON(ctx, originalStatePreset)
	normalizedApiPreset, errApiNorm := NormalizeJSON(ctx, apiPreset)

	if errStateNorm == nil && errApiNorm == nil {
		tflog.Debug(ctx, "Read: Normalized preset comparison", map[string]interface{}{
			"uuid":                     state.UUID.ValueString(),
			"normalized_state_len":     len(normalizedStatePreset),
			"normalized_api_len":       len(normalizedApiPreset),
			"normalized_equal":         normalizedStatePreset == normalizedApiPreset,
			"normalized_state_preview": getPreview(normalizedStatePreset, 200),
			"normalized_api_preview":   getPreview(normalizedApiPreset, 200),
		})
	} else {
		tflog.Warn(ctx, "Read: Failed to normalize presets for comparison", map[string]interface{}{
			"uuid":           state.UUID.ValueString(),
			"state_norm_err": errStateNorm,
			"api_norm_err":   errApiNorm,
		})
	}

	// Keep the user's original preset format if semantically the same
	areSemanticallySame, err := CompareJSONSemantically(originalStatePreset, apiPreset)
	if err != nil {
		// If we can't parse the JSON, use the API response
		// This can happen if the state has invalid JSON from an older version
		tflog.Warn(ctx, "Read: Failed to compare preset JSON semantically, using API response", map[string]interface{}{
			"uuid":  state.UUID.ValueString(),
			"error": err.Error(),
		})
		state.Preset = types.StringValue(apiPreset)
	} else if !areSemanticallySame {
		tflog.Info(ctx, "Read: Preset JSON is semantically different, using API response", map[string]interface{}{
			"uuid":                 state.UUID.ValueString(),
			"state_preset_len":     len(originalStatePreset),
			"api_preset_len":       len(apiPreset),
			"state_preset_preview": getPreview(originalStatePreset, 200),
			"api_preset_preview":   getPreview(apiPreset, 200),
		})
		state.Preset = types.StringValue(apiPreset)
	} else {
		tflog.Debug(ctx, "Read: Preset JSON is semantically same, keeping state format", map[string]interface{}{
			"uuid":             state.UUID.ValueString(),
			"state_preset_len": len(originalStatePreset),
			"api_preset_len":   len(apiPreset),
		})
		// Keep the existing state preset format
		// state.Preset stays as is
	}

	state.RevisionNumber = types.Int32Value(dashboard.RevisionNumber)
	state.Owner = types.StringValue(dashboard.Owner)
	state.Status = types.StringValue(dashboard.Status)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dashboardResourceModel
	var state dashboardResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log all field comparisons before update
	tflog.Debug(ctx, "Update: Field comparison (plan vs state vs API)", map[string]interface{}{
		"uuid":              state.UUID.ValueString(),
		"plan_name":         plan.Name.ValueString(),
		"state_name":        state.Name.ValueString(),
		"plan_description":  plan.Description.ValueString(),
		"state_description": state.Description.ValueString(),
		"plan_team":         plan.Team.ValueString(),
		"state_team":        state.Team.ValueString(),
		"plan_revision":     plan.RevisionNumber.ValueInt32(),
		"state_revision":    state.RevisionNumber.ValueInt32(),
		"plan_preset_len":   len(plan.Preset.ValueString()),
		"state_preset_len":  len(state.Preset.ValueString()),
	})

	// First, read the current state from the API to get the latest revision number
	// This prevents concurrency conflicts when the resource was modified externally
	// and ensures we use the most up-to-date revision number
	tflog.Debug(ctx, "Update: Reading current dashboard state before update to get latest revision", map[string]interface{}{
		"uuid": state.UUID.ValueString(),
	})
	apiCurrentState, err := r.client.GetDashboard(ctx, state.UUID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.Diagnostics.AddError(
				"Error Reading Dashboard",
				fmt.Sprintf("Failed to read dashboard %s before update because it was not found. It may have been deleted externally.", state.UUID.ValueString()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Error Reading Dashboard",
				fmt.Sprintf("Failed to read current dashboard state %s before update: %s", state.UUID.ValueString(), err.Error()),
			)
		}
		return
	}

	// Log API current state for comparison
	tflog.Debug(ctx, "Update: API current state before update", map[string]interface{}{
		"uuid":            state.UUID.ValueString(),
		"api_name":        apiCurrentState.Name,
		"api_description": apiCurrentState.Description,
		"api_team":        apiCurrentState.Team,
		"api_revision":    apiCurrentState.RevisionNumber,
		"api_preset_len":  len(apiCurrentState.Preset),
		"api_owner":       apiCurrentState.Owner,
		"api_status":      apiCurrentState.Status,
	})

	// Use the current revision number from the API for the update request
	currentRevision := apiCurrentState.RevisionNumber
	tflog.Debug(ctx, "Update: Using current revision from API for update", map[string]interface{}{
		"uuid":             state.UUID.ValueString(),
		"state_revision":   state.RevisionNumber.ValueInt32(),
		"api_revision":     currentRevision,
		"revision_changed": state.RevisionNumber.ValueInt32() != int32(currentRevision),
	})

	// Build update request - always use plan values for all fields
	// The API requires all fields to be present in the update request
	updateReq := &models.UpdateDashboardRequest{
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		Team:            plan.Team.ValueString(),
		Preset:          plan.Preset.ValueString(),
		IsProvisioned:   true, // Always set to true as requested
		CurrentRevision: currentRevision,
		Override:        false, // Use false by default to avoid conflicts
	}

	// Log what we're sending in the update request
	tflog.Debug(ctx, "Update: Sending update request", map[string]interface{}{
		"uuid":                     state.UUID.ValueString(),
		"request_name":             updateReq.Name,
		"request_description":      updateReq.Description,
		"request_team":             updateReq.Team,
		"request_preset_len":       len(updateReq.Preset),
		"request_is_provisioned":   updateReq.IsProvisioned,
		"request_current_revision": updateReq.CurrentRevision,
		"request_override":         updateReq.Override,
		"request_preset_preview":   getPreview(updateReq.Preset, 200),
	})

	// Only set override to true if explicitly set in the plan
	if !plan.Override.IsNull() && !plan.Override.IsUnknown() && plan.Override.ValueBool() {
		updateReq.Override = true
	}

	dashboard, err := r.client.UpdateDashboard(ctx, state.UUID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Dashboard",
			fmt.Sprintf("Could not update dashboard %s: %s", state.UUID.ValueString(), err.Error()),
		)
		return
	}

	// Log API response after update
	tflog.Debug(ctx, "Update: API response after update", map[string]interface{}{
		"uuid":                    state.UUID.ValueString(),
		"response_name":           dashboard.Name,
		"response_description":    dashboard.Description,
		"response_team":           dashboard.Team,
		"response_revision":       dashboard.RevisionNumber,
		"response_preset_len":     len(dashboard.Preset),
		"response_owner":          dashboard.Owner,
		"response_status":         dashboard.Status,
		"response_preset_preview": getPreview(dashboard.Preset, 200),
	})

	// Log field-by-field comparison: request vs response
	tflog.Debug(ctx, "Update: Request vs Response comparison", map[string]interface{}{
		"uuid":                 state.UUID.ValueString(),
		"name_match":           updateReq.Name == dashboard.Name,
		"description_match":    updateReq.Description == dashboard.Description,
		"team_match":           updateReq.Team == dashboard.Team,
		"preset_string_match":  updateReq.Preset == dashboard.Preset,
		"revision_incremented": dashboard.RevisionNumber > currentRevision,
		"revision_before":      currentRevision,
		"revision_after":       dashboard.RevisionNumber,
	})

	plan.UUID = types.StringValue(dashboard.UUID)
	plan.Name = types.StringValue(dashboard.Name)
	plan.Description = types.StringValue(dashboard.Description)
	if plan.Team.IsNull() && dashboard.Team == "" {
		plan.Team = types.StringNull()
	} else {
		plan.Team = types.StringValue(dashboard.Team)
	}
	// Keep the user's original preset format if semantically the same
	// This prevents format drift when the API returns semantically identical but differently formatted JSON
	// The ModifyPlan method normalizes and compares during plan phase, so plan.Preset should already
	// match state.Preset format if they're semantically the same. Keeping plan format here ensures
	// consistency and prevents format drift cycles that could cause apply loops.
	planPresetStr := plan.Preset.ValueString()
	apiPresetStr := dashboard.Preset
	statePresetStr := state.Preset.ValueString()

	tflog.Debug(ctx, "Update: Comparing preset JSONs", map[string]interface{}{
		"uuid":             state.UUID.ValueString(),
		"plan_preset_len":  len(planPresetStr),
		"state_preset_len": len(statePresetStr),
		"api_preset_len":   len(apiPresetStr),
		"plan_eq_state":    planPresetStr == statePresetStr,
		"plan_eq_api":      planPresetStr == apiPresetStr,
		"state_eq_api":     statePresetStr == apiPresetStr,
		"revision_before":  currentRevision,
		"revision_after":   dashboard.RevisionNumber,
	})

	// Normalize all three presets for detailed comparison
	normalizedPlan, errPlanNorm := NormalizeJSON(ctx, planPresetStr)
	normalizedState, errStateNorm := NormalizeJSON(ctx, statePresetStr)
	normalizedApi, errApiNorm := NormalizeJSON(ctx, apiPresetStr)

	if errPlanNorm == nil && errStateNorm == nil && errApiNorm == nil {
		tflog.Debug(ctx, "Update: Normalized preset comparison", map[string]interface{}{
			"uuid":                     state.UUID.ValueString(),
			"normalized_plan_len":      len(normalizedPlan),
			"normalized_state_len":     len(normalizedState),
			"normalized_api_len":       len(normalizedApi),
			"norm_plan_eq_norm_state":  normalizedPlan == normalizedState,
			"norm_plan_eq_norm_api":    normalizedPlan == normalizedApi,
			"norm_state_eq_norm_api":   normalizedState == normalizedApi,
			"normalized_plan_preview":  getPreview(normalizedPlan, 200),
			"normalized_state_preview": getPreview(normalizedState, 200),
			"normalized_api_preview":   getPreview(normalizedApi, 200),
		})
	} else {
		tflog.Warn(ctx, "Update: Failed to normalize some presets", map[string]interface{}{
			"uuid":           state.UUID.ValueString(),
			"plan_norm_err":  errPlanNorm,
			"state_norm_err": errStateNorm,
			"api_norm_err":   errApiNorm,
		})
	}

	areSemanticallySame, err := CompareJSONSemantically(planPresetStr, apiPresetStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Dashboard Preset",
			fmt.Sprintf("Failed to parse dashboard preset JSON: %s", err.Error()),
		)
		return
	}
	if !areSemanticallySame {
		tflog.Info(ctx, "Update: Preset JSON is semantically different, using API response", map[string]interface{}{
			"uuid":                state.UUID.ValueString(),
			"plan_preset_len":     len(planPresetStr),
			"api_preset_len":      len(apiPresetStr),
			"plan_preset_preview": getPreview(planPresetStr, 200),
			"api_preset_preview":  getPreview(apiPresetStr, 200),
		})
		plan.Preset = types.StringValue(apiPresetStr)
	} else {
		tflog.Debug(ctx, "Update: Preset JSON is semantically same, keeping plan format to prevent format drift", map[string]interface{}{
			"uuid":            state.UUID.ValueString(),
			"plan_preset_len": len(planPresetStr),
			"api_preset_len":  len(apiPresetStr),
			"plan_eq_state":   planPresetStr == statePresetStr,
		})
		// Keep the plan format (which should match state format after ModifyPlan)
		// This ensures consistency and prevents format drift cycles
	}
	plan.Owner = types.StringValue(dashboard.Owner)
	plan.Status = types.StringValue(dashboard.Status)
	plan.RevisionNumber = types.Int32Value(dashboard.RevisionNumber)

	// Log final state that will be saved
	tflog.Debug(ctx, "Update: Final state being saved", map[string]interface{}{
		"uuid":                 state.UUID.ValueString(),
		"final_name":           plan.Name.ValueString(),
		"final_description":    plan.Description.ValueString(),
		"final_team":           plan.Team.ValueString(),
		"final_revision":       plan.RevisionNumber.ValueInt32(),
		"final_preset_len":     len(plan.Preset.ValueString()),
		"final_owner":          plan.Owner.ValueString(),
		"final_status":         plan.Status.ValueString(),
		"final_preset_preview": getPreview(plan.Preset.ValueString(), 200),
	})

	// Log summary comparison: state before vs state after
	tflog.Info(ctx, "Update: Summary - State before vs after", map[string]interface{}{
		"uuid":                state.UUID.ValueString(),
		"name_changed":        state.Name.ValueString() != plan.Name.ValueString(),
		"description_changed": state.Description.ValueString() != plan.Description.ValueString(),
		"team_changed":        state.Team.ValueString() != plan.Team.ValueString(),
		"preset_changed":      state.Preset.ValueString() != plan.Preset.ValueString(),
		"preset_len_changed":  len(state.Preset.ValueString()) != len(plan.Preset.ValueString()),
		"revision_changed":    state.RevisionNumber.ValueInt32() != plan.RevisionNumber.ValueInt32(),
		"revision_before":     state.RevisionNumber.ValueInt32(),
		"revision_after":      plan.RevisionNumber.ValueInt32(),
	})

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Dashboard updated successfully", map[string]interface{}{
		"uuid": dashboard.UUID,
	})
}

func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dashboardResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Dashboard", map[string]interface{}{
		"uuid": state.UUID.ValueString(),
	})

	err := r.client.DeleteDashboard(ctx, state.UUID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Debug(ctx, "Dashboard already deleted", map[string]interface{}{
				"uuid": state.UUID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Dashboard",
			fmt.Sprintf("Could not delete dashboard %s: %s", state.UUID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Trace(ctx, "Dashboard deleted successfully", map[string]interface{}{
		"uuid": state.UUID.ValueString(),
	})
}

func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *dashboardResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Info(ctx, "ModifyPlan called for dashboard resource")
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "ModifyPlan: Skipping for new or destroyed dashboard resource")
		return
	}

	var plan dashboardResourceModel
	var state dashboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if there are any actual changes to the dashboard (excluding preset)
	hasChanges := !plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!plan.Team.Equal(state.Team)

	plannedPreset := plan.Preset.ValueString()
	statePreset := state.Preset.ValueString()

	tflog.Debug(ctx, "ModifyPlan: Checking for changes", map[string]interface{}{
		"uuid":                 plan.UUID.ValueString(),
		"plan_preset_len":      len(plannedPreset),
		"state_preset_len":     len(statePreset),
		"presets_equal":        plan.Preset.Equal(state.Preset),
		"presets_string_equal": plannedPreset == statePreset,
		"plan_revision":        plan.RevisionNumber.ValueInt32(),
		"state_revision":       state.RevisionNumber.ValueInt32(),
		"has_other_changes":    hasChanges,
	})

	// Handle preset comparison
	if !plan.Preset.IsNull() && !plan.Preset.IsUnknown() && !state.Preset.IsNull() && !state.Preset.IsUnknown() {
		// Skip if presets are already identical
		if plannedPreset == statePreset {
			tflog.Debug(ctx, "ModifyPlan: Presets are already identical, no normalization needed", map[string]interface{}{
				"uuid":       plan.UUID.ValueString(),
				"preset_len": len(plannedPreset),
			})
		} else {
			// Log raw presets before normalization for debugging
			tflog.Debug(ctx, "ModifyPlan: Raw preset comparison (before normalization)", map[string]interface{}{
				"uuid":                 plan.UUID.ValueString(),
				"plan_preset_len":      len(plannedPreset),
				"state_preset_len":     len(statePreset),
				"plan_preset_preview":  getPreview(plannedPreset, 300),
				"state_preset_preview": getPreview(statePreset, 300),
			})

			// Try to normalize and compare the presets
			normalizedPlanned, err := NormalizeJSON(ctx, plannedPreset)
			if err != nil {
				// If we can't normalize the planned preset, it's likely invalid JSON
				// Allow Terraform to proceed with the update so the user can fix it
				tflog.Warn(ctx, "ModifyPlan: Failed to normalize planned preset JSON", map[string]interface{}{
					"uuid":                plan.UUID.ValueString(),
					"plan_preset_len":     len(plannedPreset),
					"plan_preset_preview": getPreview(plannedPreset, 200),
					"error":               err.Error(),
				})
				hasChanges = true
			} else {
				normalizedState, err := NormalizeJSON(ctx, statePreset)
				if err != nil {
					// If we can't normalize the state preset, allow the update to proceed
					// This could happen if the state has corrupted data from a previous version
					tflog.Warn(ctx, "ModifyPlan: Failed to normalize state preset JSON, allowing update", map[string]interface{}{
						"uuid":                 plan.UUID.ValueString(),
						"state_preset_len":     len(statePreset),
						"state_preset_preview": getPreview(statePreset, 200),
						"error":                err.Error(),
					})
					hasChanges = true
				} else {
					// Log normalized presets for detailed debugging
					tflog.Debug(ctx, "ModifyPlan: Normalized preset comparison (after normalization)", map[string]interface{}{
						"uuid":                     plan.UUID.ValueString(),
						"normalized_plan_len":      len(normalizedPlanned),
						"normalized_state_len":     len(normalizedState),
						"normalized_equal":         normalizedPlanned == normalizedState,
						"normalized_plan_preview":  getPreview(normalizedPlanned, 300),
						"normalized_state_preview": getPreview(normalizedState, 300),
					})

					areSemanticallySame, err := CompareJSONSemantically(normalizedPlanned, normalizedState)
					if err != nil {
						// If we can't compare after successful normalization, something is wrong
						// Allow the update to proceed rather than blocking it
						tflog.Warn(ctx, "ModifyPlan: Failed to perform semantic JSON comparison, allowing update", map[string]interface{}{
							"uuid":                 plan.UUID.ValueString(),
							"normalized_plan_len":  len(normalizedPlanned),
							"normalized_state_len": len(normalizedState),
							"error":                err.Error(),
						})
						hasChanges = true
					} else if !areSemanticallySame {
						hasChanges = true
						tflog.Info(ctx, "ModifyPlan: Preset JSONs have semantic differences.", map[string]interface{}{
							"uuid":                     plan.UUID.ValueString(),
							"plan_preset_len":          len(plannedPreset),
							"state_preset_len":         len(statePreset),
							"normalized_plan_len":      len(normalizedPlanned),
							"normalized_state_len":     len(normalizedState),
							"normalized_plan_preview":  getPreview(normalizedPlanned, 300),
							"normalized_state_preview": getPreview(normalizedState, 300),
						})
					} else {
						// Presets are semantically the same, suppress the diff
						tflog.Info(ctx, "ModifyPlan: Preset JSONs are semantically identical. Suppressing diff.", map[string]interface{}{
							"uuid":                 plan.UUID.ValueString(),
							"plan_preset_len":      len(plannedPreset),
							"state_preset_len":     len(statePreset),
							"normalized_plan_len":  len(normalizedPlanned),
							"normalized_state_len": len(normalizedState),
							"normalized_match":     normalizedPlanned == normalizedState,
							"raw_match":            plannedPreset == statePreset,
						})
						plan.Preset = state.Preset
					}
				}
			}
		}
	} else {
		tflog.Debug(ctx, "ModifyPlan: Skipping preset comparison - preset is null or unknown", map[string]interface{}{
			"uuid":             plan.UUID.ValueString(),
			"plan_is_null":     plan.Preset.IsNull(),
			"plan_is_unknown":  plan.Preset.IsUnknown(),
			"state_is_null":    state.Preset.IsNull(),
			"state_is_unknown": state.Preset.IsUnknown(),
		})
	}

	// Handle revision_number: if there are no changes, use state value to prevent false positives
	// If there are changes, allow it to be unknown so it can increment during the update
	if !hasChanges && !state.RevisionNumber.IsNull() && !state.RevisionNumber.IsUnknown() {
		tflog.Debug(ctx, "ModifyPlan: No changes detected, using state revision_number to prevent false positive", map[string]interface{}{
			"uuid":           plan.UUID.ValueString(),
			"state_revision": state.RevisionNumber.ValueInt32(),
		})
		plan.RevisionNumber = state.RevisionNumber
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// getPreview returns a preview of a string, useful for logging long JSON strings
func getPreview(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
