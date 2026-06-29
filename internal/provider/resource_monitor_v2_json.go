// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// monitorV2JsonResource is a codegen-friendly sibling of monitorV2Resource: identical schema
// and API behavior, except notification_settings.connected_app_params is a JSON string instead
// of a map-of-objects (schema.MapNestedAttribute). upjet — which generates the standalone
// Crossplane provider from this provider's schema (BE-2207) — cannot represent a nested map,
// so this variant exposes that one field as a JSON string (mirroring groundcover_connected_app
// vs groundcover_connected_app_json). The typed resource stays the default for HCL users.
//
// Everything else is delegated to the typed resource's logic via a typed bridge model, so the
// two resources never drift in behavior.
var (
	_ resource.Resource                   = &monitorV2JsonResource{}
	_ resource.ResourceWithConfigure      = &monitorV2JsonResource{}
	_ resource.ResourceWithImportState    = &monitorV2JsonResource{}
	_ resource.ResourceWithValidateConfig = &monitorV2JsonResource{}
)

func NewMonitorV2JsonResource() resource.Resource {
	return &monitorV2JsonResource{}
}

type monitorV2JsonResource struct {
	client ApiClient
}

// monitorV2JsonResourceModel mirrors monitorV2ResourceModel exactly except NotificationSettings,
// whose connected_app_params is a JSON string. Keep the non-notification fields in sync with
// monitorV2ResourceModel.
type monitorV2JsonResourceModel struct {
	ID                   types.String                            `tfsdk:"id"`
	Title                types.String                            `tfsdk:"title"`
	Severity             types.String                            `tfsdk:"severity"`
	MeasurementType      types.String                            `tfsdk:"measurement_type"`
	ExecutionErrorState  types.String                            `tfsdk:"execution_error_state"`
	NoDataState          types.String                            `tfsdk:"no_data_state"`
	IsPaused             types.Bool                              `tfsdk:"is_paused"`
	AutoResolve          types.Bool                              `tfsdk:"auto_resolve"`
	Category             types.String                            `tfsdk:"category"`
	Team                 types.String                            `tfsdk:"team"`
	Labels               types.Map                               `tfsdk:"labels"`
	Annotations          types.Map                               `tfsdk:"annotations"`
	Routing              types.List                              `tfsdk:"routing"`
	Query                *monitorV2QueryModel                    `tfsdk:"query"`
	Reducers             []monitorV2ReducerModel                 `tfsdk:"reducer"`
	Thresholds           []monitorV2ThresholdModel               `tfsdk:"threshold"`
	EvaluationInterval   *monitorV2EvaluationIntervalModel       `tfsdk:"evaluation_interval"`
	Display              *monitorV2DisplayModel                  `tfsdk:"display"`
	NotificationSettings *monitorV2JsonNotificationSettingsModel `tfsdk:"notification_settings"`
}

type monitorV2JsonNotificationSettingsModel struct {
	Method                 types.String `tfsdk:"method"`
	ConnectedApps          types.List   `tfsdk:"connected_apps"`
	ConnectedAppParams     types.String `tfsdk:"connected_app_params"`
	StatusFilters          types.List   `tfsdk:"status_filters"`
	DisableRenotification  types.Bool   `tfsdk:"disable_renotification"`
	RenotificationInterval types.String `tfsdk:"renotification_interval"`
}

// connectedAppParamJSON is the JSON shape of a single connected_app_params entry.
type connectedAppParamJSON struct {
	Channels []string `json:"channels"`
}

func (r *monitorV2JsonResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor_v2_json"
}

func (r *monitorV2JsonResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Reuse the typed monitor_v2 schema verbatim, then swap the one un-codegenable attribute
	// (notification_settings.connected_app_params) from a nested map to a JSON string. This keeps
	// the two schemas in lockstep automatically.
	var base resource.SchemaResponse
	(&monitorV2Resource{}).Schema(ctx, req, &base)
	s := base.Schema

	ns := s.Blocks["notification_settings"].(schema.SingleNestedBlock)
	ns.Attributes["connected_app_params"] = schema.StringAttribute{
		MarkdownDescription: "JSON-encoded per-connected-app delivery options keyed by connected app ID. String form of `groundcover_monitor_v2.connected_app_params`, for schema code generators (Crossplane/upjet) that cannot represent nested maps. Example: `jsonencode({ \"app-id\" = { channels = [\"C123\"] } })`.",
		Optional:            true,
	}
	s.Blocks["notification_settings"] = ns

	s.MarkdownDescription = "Manages a groundcover Monitor with a typed Terraform schema. Functionally identical to `groundcover_monitor_v2`, but `notification_settings.connected_app_params` is a JSON string, making it usable by schema code generators (Crossplane/upjet) that cannot represent nested maps."
	resp.Schema = s
}

func (r *monitorV2JsonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(ApiClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected provider.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *monitorV2JsonResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config monitorV2JsonResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	typed := config.toTyped(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	validateMonitorV2Config(ctx, typed, &resp.Diagnostics)
}

func (r *monitorV2JsonResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorV2JsonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	typed := plan.toTyped(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := buildMonitorV2CreateRequest(ctx, typed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.CreateMonitorV2(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create monitor, got error: %s", err.Error()))
		return
	}
	if apiResp == nil || apiResp.MonitorID == "" {
		resp.Diagnostics.AddError("API Error", "Monitor creation response did not contain a MonitorID")
		return
	}

	typed.ID = types.StringValue(apiResp.MonitorID)
	if err := r.readTyped(ctx, apiResp.MonitorID, typed, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read created monitor %s, got error: %s", apiResp.MonitorID, err))
		return
	}

	result := monitorV2JsonModelFromTyped(ctx, typed, &resp.Diagnostics)
	// connected_app_params is Optional (not Computed): the authored JSON string must round-trip
	// unchanged, so preserve it rather than the re-serialized remote form.
	result.preserveAuthoredParams(plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *monitorV2JsonResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorV2JsonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	typed := state.toTyped(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.readTyped(ctx, id, typed, &resp.Diagnostics); err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Monitor %s not found, removing from state", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read monitor %s, got error: %s", id, err))
		return
	}

	result := monitorV2JsonModelFromTyped(ctx, typed, &resp.Diagnostics)
	// Keep the authored JSON string when it is semantically equal to the remote params, so JSON
	// formatting differences don't surface as perpetual drift.
	result.preserveParamsIfUnchanged(state)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *monitorV2JsonResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorV2JsonResourceModel
	var state monitorV2JsonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	typed := plan.toTyped(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildMonitorV2UpdateRequest(ctx, typed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.UpdateMonitorV2(ctx, id, updateReq); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update monitor %s, got error: %s", id, err.Error()))
		return
	}

	typed.ID = types.StringValue(id)
	if err := r.readTyped(ctx, id, typed, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read updated monitor %s, got error: %s", id, err))
		return
	}

	result := monitorV2JsonModelFromTyped(ctx, typed, &resp.Diagnostics)
	result.preserveAuthoredParams(plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *monitorV2JsonResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorV2JsonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.client.DeleteMonitorV2(ctx, id); err != nil && !errors.Is(err, ErrNotFound) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete monitor %s, got error: %s", id, err))
	}
}

func (r *monitorV2JsonResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readTyped reuses the typed resource's SDK->model mapping (including duration preservation).
func (r *monitorV2JsonResource) readTyped(ctx context.Context, id string, typed *monitorV2ResourceModel, diags *diag.Diagnostics) error {
	tr := &monitorV2Resource{client: r.client}
	return tr.readMonitorV2IntoState(ctx, id, typed, diags)
}

// toTyped converts the JSON model into the typed model, parsing the connected_app_params JSON
// string into the nested map the shared logic expects.
func (m *monitorV2JsonResourceModel) toTyped(ctx context.Context, diags *diag.Diagnostics) *monitorV2ResourceModel {
	typed := &monitorV2ResourceModel{
		ID:                  m.ID,
		Title:               m.Title,
		Severity:            m.Severity,
		MeasurementType:     m.MeasurementType,
		ExecutionErrorState: m.ExecutionErrorState,
		NoDataState:         m.NoDataState,
		IsPaused:            m.IsPaused,
		AutoResolve:         m.AutoResolve,
		Category:            m.Category,
		Team:                m.Team,
		Labels:              m.Labels,
		Annotations:         m.Annotations,
		Routing:             m.Routing,
		Query:               m.Query,
		Reducers:            m.Reducers,
		Thresholds:          m.Thresholds,
		EvaluationInterval:  m.EvaluationInterval,
		Display:             m.Display,
	}
	if m.NotificationSettings != nil {
		ns := m.NotificationSettings
		typed.NotificationSettings = &monitorV2NotificationSettingsModel{
			Method:                 ns.Method,
			ConnectedApps:          ns.ConnectedApps,
			ConnectedAppParams:     connectedAppParamsJSONToMap(ctx, ns.ConnectedAppParams, diags),
			StatusFilters:          ns.StatusFilters,
			DisableRenotification:  ns.DisableRenotification,
			RenotificationInterval: ns.RenotificationInterval,
		}
	}
	return typed
}

// monitorV2JsonModelFromTyped converts the typed model (post-read) back into the JSON model,
// serializing the nested params map into a JSON string.
func monitorV2JsonModelFromTyped(ctx context.Context, typed *monitorV2ResourceModel, diags *diag.Diagnostics) *monitorV2JsonResourceModel {
	m := &monitorV2JsonResourceModel{
		ID:                  typed.ID,
		Title:               typed.Title,
		Severity:            typed.Severity,
		MeasurementType:     typed.MeasurementType,
		ExecutionErrorState: typed.ExecutionErrorState,
		NoDataState:         typed.NoDataState,
		IsPaused:            typed.IsPaused,
		AutoResolve:         typed.AutoResolve,
		Category:            typed.Category,
		Team:                typed.Team,
		Labels:              typed.Labels,
		Annotations:         typed.Annotations,
		Routing:             typed.Routing,
		Query:               typed.Query,
		Reducers:            typed.Reducers,
		Thresholds:          typed.Thresholds,
		EvaluationInterval:  typed.EvaluationInterval,
		Display:             typed.Display,
	}
	if typed.NotificationSettings != nil {
		ns := typed.NotificationSettings
		m.NotificationSettings = &monitorV2JsonNotificationSettingsModel{
			Method:                 ns.Method,
			ConnectedApps:          ns.ConnectedApps,
			ConnectedAppParams:     connectedAppParamsMapToJSON(ctx, ns.ConnectedAppParams, diags),
			StatusFilters:          ns.StatusFilters,
			DisableRenotification:  ns.DisableRenotification,
			RenotificationInterval: ns.RenotificationInterval,
		}
	}
	return m
}

// preserveAuthoredParams keeps the params JSON string exactly as the user authored it (required
// for Optional attributes to round-trip on Create/Update).
func (m *monitorV2JsonResourceModel) preserveAuthoredParams(plan monitorV2JsonResourceModel) {
	if m.NotificationSettings != nil && plan.NotificationSettings != nil {
		m.NotificationSettings.ConnectedAppParams = plan.NotificationSettings.ConnectedAppParams
	}
}

// preserveParamsIfUnchanged keeps the prior state's params JSON string when it is semantically
// equal to the freshly-read remote value, avoiding diffs from JSON formatting alone.
func (m *monitorV2JsonResourceModel) preserveParamsIfUnchanged(prior monitorV2JsonResourceModel) {
	if m.NotificationSettings == nil || prior.NotificationSettings == nil {
		return
	}
	priorStr := prior.NotificationSettings.ConnectedAppParams
	fresh := m.NotificationSettings.ConnectedAppParams
	if priorStr.IsNull() || priorStr.IsUnknown() || fresh.IsNull() || fresh.IsUnknown() {
		return
	}
	if connectedAppParamsJSONEqual(priorStr.ValueString(), fresh.ValueString()) {
		m.NotificationSettings.ConnectedAppParams = priorStr
	}
}

func connectedAppParamsJSONEqual(a, b string) bool {
	var pa, pb map[string]connectedAppParamJSON
	if json.Unmarshal([]byte(a), &pa) != nil || json.Unmarshal([]byte(b), &pb) != nil {
		return false
	}
	return reflect.DeepEqual(pa, pb)
}

// connectedAppParamsJSONToMap parses the JSON-string params into the nested types.Map that the
// shared monitor_v2 logic consumes. Null/unknown pass through unchanged.
func connectedAppParamsJSONToMap(ctx context.Context, value types.String, diags *diag.Diagnostics) types.Map {
	objectType := types.ObjectType{AttrTypes: monitorV2ConnectedAppDeliveryOptionsAttrTypes()}
	if value.IsNull() {
		return types.MapNull(objectType)
	}
	if value.IsUnknown() {
		return types.MapUnknown(objectType)
	}

	var parsed map[string]connectedAppParamJSON
	if err := json.Unmarshal([]byte(value.ValueString()), &parsed); err != nil {
		diags.AddAttributeError(
			path.Root("notification_settings").AtName("connected_app_params"),
			"Invalid connected_app_params JSON",
			fmt.Sprintf("`connected_app_params` must be a JSON object keyed by connected app ID, e.g. {\"app-id\":{\"channels\":[\"C123\"]}}: %v", err),
		)
		return types.MapNull(objectType)
	}
	if len(parsed) == 0 {
		return types.MapNull(objectType)
	}

	sdkParams := make(models.ConnectedAppParams, len(parsed))
	for appID, p := range parsed {
		sdkParams[appID] = models.ConnectedAppDeliveryOptions{Channels: p.Channels}
	}
	return monitorV2ConnectedAppParamsType(ctx, sdkParams, diags)
}

// connectedAppParamsMapToJSON serializes the nested params types.Map into a JSON string.
func connectedAppParamsMapToJSON(ctx context.Context, value types.Map, diags *diag.Diagnostics) types.String {
	if value.IsNull() || value.IsUnknown() {
		return types.StringNull()
	}

	var elements map[string]monitorV2ConnectedAppDeliveryOptionsModel
	diags.Append(value.ElementsAs(ctx, &elements, false)...)
	if len(elements) == 0 {
		return types.StringNull()
	}

	out := make(map[string]connectedAppParamJSON, len(elements))
	for appID, opts := range elements {
		out[appID] = connectedAppParamJSON{Channels: monitorV2StringList(ctx, opts.Channels, diags)}
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		diags.AddError("Error encoding connected_app_params", err.Error())
		return types.StringNull()
	}
	return types.StringValue(string(encoded))
}
