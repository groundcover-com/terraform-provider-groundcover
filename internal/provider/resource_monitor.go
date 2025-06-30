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
	"gopkg.in/yaml.v3"
)

// Constants for drift detection
const (
	// Length ratio thresholds for distinguishing semantic vs formatting changes
	MinSemanticChangeRatio = 0.7
	MaxSemanticChangeRatio = 1.3
	// Maximum sample length for debug logging to avoid excessive log output
	MaxDebugSampleLength = 500
)

var _ resource.Resource = &monitorResource{}
var _ resource.ResourceWithImportState = &monitorResource{}
var _ resource.ResourceWithConfigure = &monitorResource{}
var _ resource.ResourceWithModifyPlan = &monitorResource{}

func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

type monitorResource struct {
	client ApiClient
}

type monitorResourceModel struct {
	Id          types.String `tfsdk:"id"`
	MonitorYaml types.String `tfsdk:"monitor_yaml"`
}

func (r *monitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *monitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Groundcover Monitor resource managed via raw YAML.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Monitor identifier (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"monitor_yaml": schema.StringAttribute{
				MarkdownDescription: "The monitor definition in YAML format.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{},
			},
		},
	}
}

func (r *monitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "monitor resource configured successfully")
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data monitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating monitor resource from YAML")

	userInputMonitorYaml := data.MonitorYaml.ValueString()
	normalizedApiYaml, err := NormalizeMonitorYaml(ctx, userInputMonitorYaml)
	if err != nil {
		resp.Diagnostics.AddError("YAML Normalization Error", fmt.Sprintf("Unable to normalize monitor YAML during Create: %s", err))
		return
	}

	monitorYamlBytesForApi := []byte(normalizedApiYaml)

	var createReq models.CreateMonitorRequest
	err = yaml.Unmarshal(monitorYamlBytesForApi, &createReq)
	if err != nil {
		resp.Diagnostics.AddError("YAML Unmarshal Error", fmt.Sprintf("Unable to unmarshal monitor config into SDK request model: %s. YAML: %s", err.Error(), normalizedApiYaml))
		return
	}

	tflog.Debug(ctx, "Creating monitor via SDK with unmarshalled request", map[string]any{"title_from_plan": createReq.Title})
	apiResp, err := r.client.CreateMonitor(ctx, &createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create monitor, got error: %s", err.Error()))
		return
	}

	if apiResp == nil || apiResp.MonitorID == "" {
		resp.Diagnostics.AddError("API Error", "Monitor creation response did not contain a MonitorID")
		return
	}

	data.Id = types.StringValue(apiResp.MonitorID)
	data.MonitorYaml = types.StringValue(userInputMonitorYaml)

	tflog.Trace(ctx, "Created monitor resource from YAML", map[string]interface{}{"id": data.Id.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to safely dereference a string pointer for logging
func derefString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// detectAndHandleDrift performs drift detection between user and remote YAML
// and updates the data model if significant semantic changes are detected
func (r *monitorResource) detectAndHandleDrift(ctx context.Context, data *monitorResourceModel, remoteYamlBytes []byte) {
	monitorId := data.Id.ValueString()
	userOriginalYaml := data.MonitorYaml.ValueString()

	if userOriginalYaml == "" || remoteYamlBytes == nil {
		return
	}

	remoteYaml := string(remoteYamlBytes)

	// Filter remote YAML to only include keys that exist in user's original YAML
	filteredRemoteYaml, err := FilterYamlKeysBasedOnTemplate(ctx, remoteYaml, userOriginalYaml)
	if err != nil {
		tflog.Warn(ctx, "Failed to filter remote YAML for drift detection, skipping drift check", map[string]interface{}{
			"id":    monitorId,
			"error": err.Error(),
		})
		return
	}

	// Apply time normalization to both user and remote YAML before comparison
	timeNormalizedUserYaml := NormalizeTimeStringsInYaml(userOriginalYaml)
	timeNormalizedRemoteYaml := NormalizeTimeStringsInYaml(filteredRemoteYaml)

	// Normalize both YAMLs using the same method as ModifyPlan for comparison
	normalizedUserYaml, err := NormalizeMonitorYaml(ctx, timeNormalizedUserYaml)
	if err != nil {
		tflog.Warn(ctx, "Failed to normalize user YAML for drift detection, skipping drift check", map[string]interface{}{
			"id":    monitorId,
			"error": err.Error(),
		})
		return
	}

	normalizedRemoteYaml, err := NormalizeMonitorYaml(ctx, timeNormalizedRemoteYaml)
	if err != nil {
		tflog.Warn(ctx, "Failed to normalize remote YAML for drift detection, skipping drift check", map[string]interface{}{
			"id":    monitorId,
			"error": err.Error(),
		})
		return
	}

	// Compare normalized YAMLs
	if normalizedUserYaml == normalizedRemoteYaml {
		tflog.Debug(ctx, "No drift detected: Normalized YAMLs are identical", map[string]interface{}{
			"id": monitorId,
		})
		return
	}

	tflog.Info(ctx, "Drift detected: Remote monitor state differs from Terraform state", map[string]interface{}{
		"id":                 monitorId,
		"user_yaml_length":   len(normalizedUserYaml),
		"remote_yaml_length": len(normalizedRemoteYaml),
	})

	// For debugging: log the actual differences (truncated to avoid excessive logs)
	userSample := normalizedUserYaml
	remoteSample := normalizedRemoteYaml
	if len(userSample) > MaxDebugSampleLength {
		userSample = userSample[:MaxDebugSampleLength] + "..."
	}
	if len(remoteSample) > MaxDebugSampleLength {
		remoteSample = remoteSample[:MaxDebugSampleLength] + "..."
	}
	tflog.Debug(ctx, "YAML drift details", map[string]interface{}{
		"id":                 monitorId,
		"user_yaml_sample":   userSample,
		"remote_yaml_sample": remoteSample,
	})

	// Check if this is a significant semantic difference or just formatting
	lengthDifference := float64(len(normalizedRemoteYaml)) / float64(len(normalizedUserYaml))
	if lengthDifference < MinSemanticChangeRatio || lengthDifference > MaxSemanticChangeRatio {
		// Significant size difference suggests real semantic changes
		tflog.Info(ctx, "Significant semantic drift detected, updating state", map[string]interface{}{
			"id":           monitorId,
			"length_ratio": lengthDifference,
		})
		data.MonitorYaml = types.StringValue(normalizedRemoteYaml)
	} else {
		// Likely just formatting differences, log but don't update state
		tflog.Info(ctx, "Minor formatting drift detected, not updating state", map[string]interface{}{
			"id":           monitorId,
			"length_ratio": lengthDifference,
		})
	}
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data monitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := data.Id.ValueString()
	tflog.Debug(ctx, "Reading monitor resource YAML", map[string]interface{}{"id": monitorId})

	remoteYamlBytes, err := r.client.GetMonitor(ctx, monitorId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Monitor %s not found (handled via ErrNotFound), removing from state", monitorId))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read monitor %s YAML, got error: %s", monitorId, err))
		return
	}

	tflog.Trace(ctx, "Read monitor resource YAML (confirmed existence)", map[string]interface{}{"id": monitorId})

	// Enhanced drift detection: compare remote state with user's original YAML
	r.detectAndHandleDrift(ctx, &data, remoteYamlBytes)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorResourceModel
	var state monitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := state.Id.ValueString()
	tflog.Debug(ctx, "Updating monitor resource from YAML", map[string]interface{}{"id": monitorId})

	userInputMonitorYaml := plan.MonitorYaml.ValueString()
	normalizedApiYaml, err := NormalizeMonitorYaml(ctx, userInputMonitorYaml)
	if err != nil {
		resp.Diagnostics.AddError("YAML Normalization Error", fmt.Sprintf("Unable to normalize monitor YAML during Update for monitor %s: %s", monitorId, err))
		return
	}

	monitorYamlBytesForApi := []byte(normalizedApiYaml)

	var updateReq models.UpdateMonitorRequest
	err = yaml.Unmarshal(monitorYamlBytesForApi, &updateReq)
	if err != nil {
		resp.Diagnostics.AddError("YAML Unmarshal Error", fmt.Sprintf("Unable to unmarshal monitor config into SDK update request model: %s. YAML: %s", err.Error(), normalizedApiYaml))
		return
	}

	tflog.Debug(ctx, "Updating monitor via SDK with unmarshalled request", map[string]any{"id": monitorId, "title_from_yaml": derefString(updateReq.Title)})
	err = r.client.UpdateMonitor(ctx, monitorId, &updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update monitor %s, got error: %s", monitorId, err.Error()))
		return
	}

	tflog.Trace(ctx, "Updated monitor resource from YAML", map[string]interface{}{"id": monitorId})

	updatedState := plan
	updatedState.Id = state.Id
	updatedState.MonitorYaml = types.StringValue(userInputMonitorYaml)

	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data monitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := data.Id.ValueString()
	tflog.Debug(ctx, "Deleting monitor resource", map[string]interface{}{"id": monitorId})

	err := r.client.DeleteMonitor(ctx, monitorId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("DeleteMonitor returned ErrNotFound for %s, which should have been handled by the wrapper. Removing from state anyway.", monitorId))
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete monitor %s, got error: %s", monitorId, err))
			return
		}
	}

	tflog.Trace(ctx, "Deleted monitor resource", map[string]interface{}{"id": monitorId})
}

func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *monitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "ModifyPlan: Skipping custom YAML diff for new or destroyed resource.")
		return
	}

	var plannedYaml types.String
	diags := req.Plan.GetAttribute(ctx, path.Root("monitor_yaml"), &plannedYaml)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateYaml types.String
	diags = req.State.GetAttribute(ctx, path.Root("monitor_yaml"), &stateYaml)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plannedYaml.IsNull() || plannedYaml.IsUnknown() || stateYaml.IsNull() || stateYaml.IsUnknown() {
		tflog.Debug(ctx, "ModifyPlan: Planned or State YAML is null/unknown, skipping custom diff.")
		return
	}

	plannedYamlString := plannedYaml.ValueString()
	stateYamlString := stateYaml.ValueString()

	if plannedYamlString == stateYamlString {
		tflog.Debug(ctx, "ModifyPlan: Raw YAML strings are identical.")
		return
	}

	// Apply time normalization to both planned and state YAML before comparison
	timeNormalizedPlannedYaml := NormalizeTimeStringsInYaml(plannedYamlString)
	timeNormalizedStateYaml := NormalizeTimeStringsInYaml(stateYamlString)

	normalizedPlannedYaml, err := NormalizeMonitorYaml(ctx, timeNormalizedPlannedYaml)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("monitor_yaml"),
			"Plan YAML Normalization Error",
			fmt.Sprintf("Failed to normalize planned monitor_yaml: %s. Input: %s", err, plannedYamlString),
		)
		return
	}

	normalizedStateYaml, err := NormalizeMonitorYaml(ctx, timeNormalizedStateYaml)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("monitor_yaml"),
			"State YAML Normalization Error",
			fmt.Sprintf("Failed to normalize state monitor_yaml: %s. Input: %s", err, stateYamlString),
		)
		return
	}

	if normalizedPlannedYaml == normalizedStateYaml {
		tflog.Info(ctx, "ModifyPlan: Normalized YAMLs are identical. Setting plan's monitor_yaml to state's monitor_yaml to suppress diff.")
		diags := resp.Plan.SetAttribute(ctx, path.Root("monitor_yaml"), stateYaml)
		resp.Diagnostics.Append(diags...)
	} else {
		tflog.Info(ctx, "ModifyPlan: Normalized YAMLs differ. Plan will proceed with update for monitor_yaml.")
	}
}
