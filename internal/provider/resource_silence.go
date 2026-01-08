// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &silenceResource{}
var _ resource.ResourceWithConfigure = &silenceResource{}
var _ resource.ResourceWithImportState = &silenceResource{}
var _ resource.ResourceWithValidateConfig = &silenceResource{}

func NewSilenceResource() resource.Resource {
	return &silenceResource{}
}

type silenceResource struct {
	client ApiClient
}

type silenceResourceModel struct {
	ID       types.String `tfsdk:"id"`
	StartsAt types.String `tfsdk:"starts_at"`
	EndsAt   types.String `tfsdk:"ends_at"`
	Comment  types.String `tfsdk:"comment"`
	Matchers types.List   `tfsdk:"matchers"`
}

type silenceMatcherModel struct {
	Name       types.String `tfsdk:"name"`
	Value      types.String `tfsdk:"value"`
	IsEqual    types.Bool   `tfsdk:"is_equal"`
	IsContains types.Bool   `tfsdk:"is_contains"`
}

func (r *silenceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_silence"
}

func (r *silenceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a groundcover Silence.

Silences allow you to suppress alerts for a specific time window based on matching criteria. This is useful for planned maintenance, deployments, or other situations where you want to temporarily mute alerts.

A silence is defined by: a time window (starts_at, ends_at), a comment describing the reason for the silence, and one or more matchers that define which alerts to silence.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the silence.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"starts_at": schema.StringAttribute{
				MarkdownDescription: "The start time of the silence in RFC3339 format UTC 0 (e.g., `2024-01-15T10:00:00Z`).",
				Required:            true,
			},
			"ends_at": schema.StringAttribute{
				MarkdownDescription: "The end time of the silence in RFC3339 format UTC 0 (e.g., `2024-01-15T12:00:00Z`).",
				Required:            true,
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "A comment describing the reason for the silence.",
				Optional:            true,
				Computed:            true,
			},
			"matchers": schema.ListNestedAttribute{
				MarkdownDescription: "A list of matchers that define which alerts to silence. Each matcher specifies a label name and value to match against.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the label to match (e.g., `service`, `environment`, `workload`).",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The value to match against. Can be an exact value or a partial match pattern if `is_contains` is true.",
							Required:            true,
						},
						"is_equal": schema.BoolAttribute{
							MarkdownDescription: "If true, the matcher will match when the label value equals the specified value. If false, it matches when the value does NOT equal. Defaults to `true`.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
						},
						"is_contains": schema.BoolAttribute{
							MarkdownDescription: "If true, the value is treated as a contains pattern (partial match). If false, the value must match exactly. Defaults to `false`.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func (r *silenceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "Silence resource configured successfully")
}

func (r *silenceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config silenceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Skip validation if values are unknown (e.g., computed or from other resources)
	if config.StartsAt.IsUnknown() || config.EndsAt.IsUnknown() {
		return
	}

	startsAt, err := time.Parse(time.RFC3339, config.StartsAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("starts_at"),
			"Invalid starts_at format",
			fmt.Sprintf("starts_at must be in RFC3339 format UTC 0 (e.g., 2024-01-15T10:00:00Z): %s", err.Error()),
		)
		return
	}

	endsAt, err := time.Parse(time.RFC3339, config.EndsAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("ends_at"),
			"Invalid ends_at format",
			fmt.Sprintf("ends_at must be in RFC3339 format UTC 0 (e.g., 2024-01-15T12:00:00Z): %s", err.Error()),
		)
		return
	}

	if !endsAt.After(startsAt) {
		resp.Diagnostics.AddAttributeError(
			path.Root("ends_at"),
			"Invalid time range",
			fmt.Sprintf("ends_at (%s) must be after starts_at (%s)", config.EndsAt.ValueString(), config.StartsAt.ValueString()),
		)
	}

	// Validate that comment is not an empty string
	if !config.Comment.IsNull() && !config.Comment.IsUnknown() && config.Comment.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("comment"),
			"Invalid comment",
			"Comment cannot be an empty string. Either omit the comment attribute or provide a non-empty value.",
		)
	}
}

// --- Helper Functions ---

func (r *silenceResource) matchersFromModel(ctx context.Context, matchersList types.List) (models.Matchers, error) {
	if matchersList.IsNull() || matchersList.IsUnknown() {
		return nil, nil
	}

	var matcherModels []silenceMatcherModel
	diags := matchersList.ElementsAs(ctx, &matcherModels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to extract matchers from list")
	}

	matchers := make(models.Matchers, 0, len(matcherModels))
	for _, m := range matcherModels {
		isEqual := m.IsEqual.ValueBool()
		isContains := m.IsContains.ValueBool()

		matcher := &models.SilenceMatcher{
			Name:    m.Name.ValueString(),
			Value:   m.Value.ValueString(),
			IsEqual: &isEqual,
			IsRegex: &isContains, // SDK uses IsRegex, provider exposes as is_contains
		}
		matchers = append(matchers, matcher)
	}

	return matchers, nil
}

func (r *silenceResource) matchersToModel(ctx context.Context, apiMatchers models.Matchers) (types.List, error) {
	matcherAttrType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":        types.StringType,
			"value":       types.StringType,
			"is_equal":    types.BoolType,
			"is_contains": types.BoolType,
		},
	}

	if len(apiMatchers) == 0 {
		return types.ListNull(matcherAttrType), nil
	}

	matcherValues := make([]attr.Value, 0, len(apiMatchers))
	for _, m := range apiMatchers {
		isEqual := true
		if m.IsEqual != nil {
			isEqual = *m.IsEqual
		}
		isContains := false
		if m.IsRegex != nil {
			isContains = *m.IsRegex // SDK uses IsRegex, provider exposes as is_contains
		}

		matcherObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"name":        types.StringType,
				"value":       types.StringType,
				"is_equal":    types.BoolType,
				"is_contains": types.BoolType,
			},
			map[string]attr.Value{
				"name":        types.StringValue(m.Name),
				"value":       types.StringValue(m.Value),
				"is_equal":    types.BoolValue(isEqual),
				"is_contains": types.BoolValue(isContains),
			},
		)
		if diags.HasError() {
			return types.ListNull(matcherAttrType), fmt.Errorf("failed to create matcher object")
		}
		matcherValues = append(matcherValues, matcherObj)
	}

	matcherList, diags := types.ListValue(matcherAttrType, matcherValues)
	if diags.HasError() {
		return types.ListNull(matcherAttrType), fmt.Errorf("failed to create matchers list")
	}

	return matcherList, nil
}

func parseRFC3339(value string) (strfmt.DateTime, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return strfmt.DateTime{}, fmt.Errorf("invalid RFC3339 time format: %s", err)
	}
	return strfmt.DateTime(t), nil
}

// --- CRUD Operations ---

func (r *silenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan silenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse time values
	startsAt, err := parseRFC3339(plan.StartsAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid starts_at", err.Error())
		return
	}

	endsAt, err := parseRFC3339(plan.EndsAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ends_at", err.Error())
		return
	}

	// Validate that ends_at is after starts_at
	if !time.Time(endsAt).After(time.Time(startsAt)) {
		resp.Diagnostics.AddError(
			"Invalid time range",
			"ends_at must be after starts_at",
		)
		return
	}

	// Convert matchers
	matchers, err := r.matchersFromModel(ctx, plan.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Invalid matchers", err.Error())
		return
	}

	// Default comment to "created YYYY-MM-DD HH:MM:SS" if empty or null
	comment := plan.Comment.ValueString()
	if plan.Comment.IsNull() || comment == "" {
		comment = fmt.Sprintf("created %s", time.Now().UTC().Format("2006-01-02 15:04:05"))
	}

	apiRequest := &models.CreateSilenceRequest{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
		Comment:  comment,
		Matchers: matchers,
	}

	tflog.Debug(ctx, "CreateSilence SDK Call Request constructed", map[string]any{"comment": plan.Comment.ValueString()})
	apiResponse, err := r.client.CreateSilence(ctx, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Create Silence Error", fmt.Sprintf("Failed to create silence: %s", err.Error()))
		return
	}

	if apiResponse == nil || apiResponse.UUID.String() == "" {
		resp.Diagnostics.AddError("SDK Client Create Silence Error", "Create response missing silence ID")
		return
	}

	tflog.Info(ctx, "Silence created successfully via SDK", map[string]any{"id": apiResponse.UUID.String()})

	// Map response back to state
	plan.ID = types.StringValue(apiResponse.UUID.String())

	// Update times from API response if available
	if !time.Time(apiResponse.StartsAt).IsZero() {
		plan.StartsAt = types.StringValue(time.Time(apiResponse.StartsAt).Format(time.RFC3339))
	}
	if !time.Time(apiResponse.EndsAt).IsZero() {
		plan.EndsAt = types.StringValue(time.Time(apiResponse.EndsAt).Format(time.RFC3339))
	}
	// Use comment from API response (will contain the default we generated if user didn't provide one)
	if apiResponse.Comment != "" {
		plan.Comment = types.StringValue(apiResponse.Comment)
	}

	// Update matchers from API response
	matchersList, err := r.matchersToModel(ctx, apiResponse.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Error processing response matchers", err.Error())
		return
	}
	plan.Matchers = matchersList

	tflog.Info(ctx, "Saving new silence to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *silenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state silenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	silenceID := state.ID.ValueString()
	tflog.Debug(ctx, "Reading Silence info", map[string]any{"id": silenceID})

	apiResponse, err := r.client.GetSilence(ctx, silenceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Silence %s not found, removing from state", silenceID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("SDK Client Read Silence Error", fmt.Sprintf("Failed to read silence %s: %s", silenceID, err.Error()))
		return
	}

	if apiResponse == nil {
		resp.Diagnostics.AddError("SDK Client Read Silence Error", "Read response was nil")
		return
	}

	tflog.Info(ctx, "Silence read successfully via SDK", map[string]any{"id": silenceID})

	// Unconditionally update state from API response to ensure state reflects the API
	if apiResponse.UUID.String() != "" {
		state.ID = types.StringValue(apiResponse.UUID.String())
	}
	state.StartsAt = types.StringValue(time.Time(apiResponse.StartsAt).Format(time.RFC3339))
	state.EndsAt = types.StringValue(time.Time(apiResponse.EndsAt).Format(time.RFC3339))
	state.Comment = types.StringValue(apiResponse.Comment)

	// Update matchers from API response
	matchersList, err := r.matchersToModel(ctx, apiResponse.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Error processing response matchers", err.Error())
		return
	}
	state.Matchers = matchersList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *silenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan silenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state silenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	silenceID := state.ID.ValueString()
	tflog.Debug(ctx, "Updating Silence", map[string]any{"id": silenceID})

	// Parse time values
	startsAt, err := parseRFC3339(plan.StartsAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid starts_at", err.Error())
		return
	}

	endsAt, err := parseRFC3339(plan.EndsAt.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ends_at", err.Error())
		return
	}

	// Validate that ends_at is after starts_at
	if !time.Time(endsAt).After(time.Time(startsAt)) {
		resp.Diagnostics.AddError(
			"Invalid time range",
			"ends_at must be after starts_at",
		)
		return
	}

	// Convert matchers
	matchers, err := r.matchersFromModel(ctx, plan.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Invalid matchers", err.Error())
		return
	}

	// Use existing comment from state if not specified in plan
	// Note: Empty string is rejected by ValidateConfig, so we only need to check for null/unknown
	comment := state.Comment.ValueString()
	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		comment = plan.Comment.ValueString()
	}

	apiRequest := &models.UpdateSilenceRequest{
		StartsAt: startsAt,
		EndsAt:   endsAt,
		Comment:  comment,
		Matchers: matchers,
	}

	tflog.Debug(ctx, "UpdateSilence SDK Call Request constructed", map[string]any{"id": silenceID})
	apiResponse, err := r.client.UpdateSilence(ctx, silenceID, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Update Silence Error", fmt.Sprintf("Failed to update silence %s: %s", silenceID, err.Error()))
		return
	}

	tflog.Info(ctx, "Silence updated successfully via SDK", map[string]any{"id": silenceID})

	// Update state with response data
	plan.ID = state.ID

	if apiResponse != nil {
		// Unconditionally update from API response to ensure state reflects the API
		plan.StartsAt = types.StringValue(time.Time(apiResponse.StartsAt).Format(time.RFC3339))
		plan.EndsAt = types.StringValue(time.Time(apiResponse.EndsAt).Format(time.RFC3339))
		plan.Comment = types.StringValue(apiResponse.Comment)
		matchersList, err := r.matchersToModel(ctx, apiResponse.Matchers)
		if err != nil {
			resp.Diagnostics.AddError("Error processing response matchers", err.Error())
			return
		}
		plan.Matchers = matchersList
	}

	tflog.Info(ctx, "Saving updated silence to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *silenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state silenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	silenceID := state.ID.ValueString()
	tflog.Debug(ctx, "Deleting Silence", map[string]any{"id": silenceID})

	err := r.client.DeleteSilence(ctx, silenceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Silence already deleted externally, treating as success", map[string]any{"id": silenceID})
			return
		}
		resp.Diagnostics.AddError("SDK Client Delete Silence Error", fmt.Sprintf("Failed to delete silence %s: %s", silenceID, err.Error()))
		return
	}

	tflog.Info(ctx, "Silence deleted successfully via SDK", map[string]any{"id": silenceID})
	// Terraform automatically removes the resource from state when Delete returns no error.
}

func (r *silenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
