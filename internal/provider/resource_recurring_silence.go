// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &recurringSilenceResource{}
var _ resource.ResourceWithConfigure = &recurringSilenceResource{}
var _ resource.ResourceWithImportState = &recurringSilenceResource{}
var _ resource.ResourceWithValidateConfig = &recurringSilenceResource{}

func NewRecurringSilenceResource() resource.Resource {
	return &recurringSilenceResource{}
}

type recurringSilenceResource struct {
	client ApiClient
}

type recurringSilenceResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Comment        types.String `tfsdk:"comment"`
	Matchers       types.List   `tfsdk:"matchers"`
	RecurrenceType types.String `tfsdk:"recurrence_type"`
	StartTime      types.String `tfsdk:"start_time"`
	EndTime        types.String `tfsdk:"end_time"`
	Timezone       types.String `tfsdk:"timezone"`
	DaysOfWeek     types.List   `tfsdk:"days_of_week"`
	DaysOfMonth    types.List   `tfsdk:"days_of_month"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

func (r *recurringSilenceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recurring_silence"
}

func (r *recurringSilenceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a groundcover Recurring Silence.

Recurring silences allow you to automatically suppress alerts on a repeating schedule based on matching criteria. This is useful for planned maintenance windows, recurring deployments, or other predictable situations where you want to regularly mute alerts.

A recurring silence is defined by: a recurrence schedule (daily, weekly, or monthly), a time window (start_time, end_time) in a specific timezone, an optional comment, and one or more matchers that define which alerts to silence.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the recurring silence.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "A comment describing the reason for the recurring silence.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"recurrence_type": schema.StringAttribute{
				MarkdownDescription: "The recurrence type. Must be one of: `daily`, `weekly`, `monthly`.",
				Required:            true,
			},
			"start_time": schema.StringAttribute{
				MarkdownDescription: "The start time of the silence window in `HH:MM` format (e.g., `09:00`).",
				Required:            true,
			},
			"end_time": schema.StringAttribute{
				MarkdownDescription: "The end time of the silence window in `HH:MM` format (e.g., `17:00`). Can be before start_time to indicate an overnight window (e.g., `22:00` to `06:00`).",
				Required:            true,
			},
			"timezone": schema.StringAttribute{
				MarkdownDescription: "The IANA timezone name for the silence window (e.g., `America/New_York`, `UTC`).",
				Required:            true,
			},
			"days_of_week": schema.ListAttribute{
				MarkdownDescription: "Days of the week when the silence should be active (0=Sunday, 1=Monday, ..., 6=Saturday). Required when `recurrence_type` is `weekly`.",
				Optional:            true,
				ElementType:         types.Int64Type,
			},
			"days_of_month": schema.ListAttribute{
				MarkdownDescription: "Days of the month when the silence should be active (1-31). Required when `recurrence_type` is `monthly`.",
				Optional:            true,
				ElementType:         types.Int64Type,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the recurring silence is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
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

func (r *recurringSilenceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "Recurring Silence resource configured successfully")
}

func (r *recurringSilenceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config recurringSilenceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate start_time format (HH:MM)
	if !config.StartTime.IsUnknown() && !config.StartTime.IsNull() {
		if err := validateHHMM(config.StartTime.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("start_time"),
				"Invalid start_time format",
				fmt.Sprintf("start_time must be in HH:MM format (e.g., 09:00): %s", err.Error()),
			)
		}
	}

	// Validate end_time format (HH:MM)
	if !config.EndTime.IsUnknown() && !config.EndTime.IsNull() {
		if err := validateHHMM(config.EndTime.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("end_time"),
				"Invalid end_time format",
				fmt.Sprintf("end_time must be in HH:MM format (e.g., 17:00): %s", err.Error()),
			)
		}
	}

	// Validate timezone
	if !config.Timezone.IsUnknown() && !config.Timezone.IsNull() {
		if _, err := time.LoadLocation(config.Timezone.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("timezone"),
				"Invalid timezone",
				fmt.Sprintf("timezone must be a valid IANA timezone name (e.g., America/New_York, UTC): %s", err.Error()),
			)
		}
	}

	// Validate recurrence_type and conditional requirements
	if !config.RecurrenceType.IsUnknown() && !config.RecurrenceType.IsNull() {
		recurrenceType := config.RecurrenceType.ValueString()

		switch recurrenceType {
		case "daily":
			// No additional fields required
		case "weekly":
			if config.DaysOfWeek.IsNull() || config.DaysOfWeek.IsUnknown() {
				resp.Diagnostics.AddAttributeError(
					path.Root("days_of_week"),
					"Missing days_of_week",
					"days_of_week is required when recurrence_type is \"weekly\".",
				)
			} else {
				var daysOfWeek []int64
				resp.Diagnostics.Append(config.DaysOfWeek.ElementsAs(ctx, &daysOfWeek, false)...)
				if !resp.Diagnostics.HasError() {
					if len(daysOfWeek) == 0 {
						resp.Diagnostics.AddAttributeError(
							path.Root("days_of_week"),
							"Empty days_of_week",
							"days_of_week must contain at least one day when recurrence_type is \"weekly\".",
						)
					}
					for _, d := range daysOfWeek {
						if d < 0 || d > 6 {
							resp.Diagnostics.AddAttributeError(
								path.Root("days_of_week"),
								"Invalid days_of_week value",
								fmt.Sprintf("days_of_week values must be between 0 (Sunday) and 6 (Saturday), got %d.", d),
							)
							break
						}
					}
				}
			}
		case "monthly":
			if config.DaysOfMonth.IsNull() || config.DaysOfMonth.IsUnknown() {
				resp.Diagnostics.AddAttributeError(
					path.Root("days_of_month"),
					"Missing days_of_month",
					"days_of_month is required when recurrence_type is \"monthly\".",
				)
			} else {
				var daysOfMonth []int64
				resp.Diagnostics.Append(config.DaysOfMonth.ElementsAs(ctx, &daysOfMonth, false)...)
				if !resp.Diagnostics.HasError() {
					if len(daysOfMonth) == 0 {
						resp.Diagnostics.AddAttributeError(
							path.Root("days_of_month"),
							"Empty days_of_month",
							"days_of_month must contain at least one day when recurrence_type is \"monthly\".",
						)
					}
					for _, d := range daysOfMonth {
						if d < 1 || d > 31 {
							resp.Diagnostics.AddAttributeError(
								path.Root("days_of_month"),
								"Invalid days_of_month value",
								fmt.Sprintf("days_of_month values must be between 1 and 31, got %d.", d),
							)
							break
						}
					}
				}
			}
		default:
			resp.Diagnostics.AddAttributeError(
				path.Root("recurrence_type"),
				"Invalid recurrence_type",
				fmt.Sprintf("recurrence_type must be one of: daily, weekly, monthly. Got: %s", recurrenceType),
			)
		}
	}

	// Validate that matchers is not empty
	if !config.Matchers.IsNull() && !config.Matchers.IsUnknown() && len(config.Matchers.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("matchers"),
			"Empty matchers",
			"matchers must contain at least one matcher.",
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

func validateHHMM(t string) error {
	if len(t) != 5 || t[2] != ':' {
		return fmt.Errorf("expected HH:MM format, got %q", t)
	}
	for i, c := range t {
		if i == 2 {
			continue
		}
		if c < '0' || c > '9' {
			return fmt.Errorf("expected HH:MM format, got %q", t)
		}
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	if h > 23 || m > 59 {
		return fmt.Errorf("hours must be 0-23, minutes 0-59, got %q", t)
	}
	return nil
}

// --- Helper Functions ---

func int64ListFromModel(ctx context.Context, list types.List, diags *diag.Diagnostics) []int64 {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var values []int64
	diags.Append(list.ElementsAs(ctx, &values, false)...)
	return values
}

func int64ListToModel(values []int64) types.List {
	if len(values) == 0 {
		return types.ListNull(types.Int64Type)
	}
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.Int64Value(v)
	}
	list, diags := types.ListValue(types.Int64Type, elems)
	if diags.HasError() {
		return types.ListNull(types.Int64Type)
	}
	return list
}

// --- CRUD Operations ---

func (r *recurringSilenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recurringSilenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert matchers
	matchers, err := silenceMatchersFromModel(ctx, plan.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Invalid matchers", err.Error())
		return
	}

	// Default comment to "created YYYY-MM-DD HH:MM:SS" if empty or null
	comment := plan.Comment.ValueString()
	if plan.Comment.IsNull() || comment == "" {
		comment = fmt.Sprintf("created %s", time.Now().UTC().Format("2006-01-02 15:04:05"))
	}

	startTime := plan.StartTime.ValueString()
	endTime := plan.EndTime.ValueString()
	recurrenceType := plan.RecurrenceType.ValueString()
	timezone := plan.Timezone.ValueString()
	enabled := plan.Enabled.ValueBool()

	daysOfWeek := int64ListFromModel(ctx, plan.DaysOfWeek, &resp.Diagnostics)
	daysOfMonth := int64ListFromModel(ctx, plan.DaysOfMonth, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRequest := &models.CreateRecurringSilenceRequest{
		Comment:        comment,
		Matchers:       matchers,
		RecurrenceType: &recurrenceType,
		StartTime:      &startTime,
		EndTime:        &endTime,
		Timezone:       &timezone,
		DaysOfWeek:     daysOfWeek,
		DaysOfMonth:    daysOfMonth,
		Enabled:        &enabled,
	}

	tflog.Debug(ctx, "CreateRecurringSilence SDK Call Request constructed", map[string]any{"comment": comment})
	apiResponse, err := r.client.CreateRecurringSilence(ctx, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Create Recurring Silence Error", fmt.Sprintf("Failed to create recurring silence: %s", err.Error()))
		return
	}

	responseID := ""
	if apiResponse != nil {
		responseID = apiResponse.UUID.String()
	}
	if responseID == "" {
		resp.Diagnostics.AddError("SDK Client Create Recurring Silence Error", "Create response missing recurring silence ID")
		return
	}

	tflog.Info(ctx, "Recurring Silence created successfully via SDK", map[string]any{"id": responseID})

	if err := r.mapResponseToState(ctx, apiResponse, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing response", err.Error())
		return
	}

	tflog.Info(ctx, "Saving new recurring silence to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recurringSilenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state recurringSilenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, "Reading Recurring Silence info", map[string]any{"id": id})

	apiResponse, err := r.client.GetRecurringSilence(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Recurring Silence %s not found, removing from state", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("SDK Client Read Recurring Silence Error", fmt.Sprintf("Failed to read recurring silence %s: %s", id, err.Error()))
		return
	}

	if apiResponse == nil {
		resp.Diagnostics.AddError("SDK Client Read Recurring Silence Error", "Read response was nil")
		return
	}

	tflog.Info(ctx, "Recurring Silence read successfully via SDK", map[string]any{"id": id})

	if err := r.mapResponseToState(ctx, apiResponse, &state); err != nil {
		resp.Diagnostics.AddError("Error processing response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *recurringSilenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan recurringSilenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state recurringSilenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, "Updating Recurring Silence", map[string]any{"id": id})

	// Convert matchers
	matchers, err := silenceMatchersFromModel(ctx, plan.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Invalid matchers", err.Error())
		return
	}

	// Use existing comment from state if not specified in plan
	comment := state.Comment.ValueString()
	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		comment = plan.Comment.ValueString()
	}

	daysOfWeek := int64ListFromModel(ctx, plan.DaysOfWeek, &resp.Diagnostics)
	daysOfMonth := int64ListFromModel(ctx, plan.DaysOfMonth, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRequest := &models.UpdateRecurringSilenceRequest{
		Comment:        comment,
		Matchers:       matchers,
		RecurrenceType: plan.RecurrenceType.ValueString(),
		StartTime:      plan.StartTime.ValueString(),
		EndTime:        plan.EndTime.ValueString(),
		Timezone:       plan.Timezone.ValueString(),
		DaysOfWeek:     daysOfWeek,
		DaysOfMonth:    daysOfMonth,
		Enabled:        plan.Enabled.ValueBoolPointer(),
	}

	tflog.Debug(ctx, "UpdateRecurringSilence SDK Call Request constructed", map[string]any{"id": id})
	apiResponse, err := r.client.UpdateRecurringSilence(ctx, id, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Update Recurring Silence Error", fmt.Sprintf("Failed to update recurring silence %s: %s", id, err.Error()))
		return
	}

	tflog.Info(ctx, "Recurring Silence updated successfully via SDK", map[string]any{"id": id})

	// Keep existing ID
	plan.ID = state.ID

	if apiResponse != nil {
		if err := r.mapResponseToState(ctx, apiResponse, &plan); err != nil {
			resp.Diagnostics.AddError("Error processing response", err.Error())
			return
		}
	}

	tflog.Info(ctx, "Saving updated recurring silence to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recurringSilenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state recurringSilenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, "Deleting Recurring Silence", map[string]any{"id": id})

	err := r.client.DeleteRecurringSilence(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Delete Recurring Silence Error", fmt.Sprintf("Failed to delete recurring silence %s: %s", id, err.Error()))
		return
	}

	tflog.Info(ctx, "Recurring Silence deleted successfully via SDK", map[string]any{"id": id})
}

func (r *recurringSilenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToState maps the API response to the Terraform state model.
func (r *recurringSilenceResource) mapResponseToState(ctx context.Context, apiResponse *models.RecurringSilenceResponse, state *recurringSilenceResourceModel) error {
	if id := apiResponse.UUID.String(); id != "" {
		state.ID = types.StringValue(id)
	}
	state.Comment = types.StringValue(apiResponse.Comment)
	state.RecurrenceType = types.StringValue(apiResponse.RecurrenceType)
	state.StartTime = types.StringValue(apiResponse.StartTime)
	state.EndTime = types.StringValue(apiResponse.EndTime)
	state.Timezone = types.StringValue(apiResponse.Timezone)
	state.Enabled = types.BoolValue(apiResponse.Enabled)

	state.DaysOfWeek = int64ListToModel(apiResponse.DaysOfWeek)
	state.DaysOfMonth = int64ListToModel(apiResponse.DaysOfMonth)

	matchersList, err := silenceMatchersToModel(apiResponse.Matchers)
	if err != nil {
		return fmt.Errorf("error processing response matchers: %w", err)
	}
	state.Matchers = matchersList
	return nil
}
