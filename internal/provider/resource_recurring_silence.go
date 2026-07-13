// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &recurringSilenceResource{}
var _ resource.ResourceWithConfigure = &recurringSilenceResource{}
var _ resource.ResourceWithImportState = &recurringSilenceResource{}
var _ resource.ResourceWithValidateConfig = &recurringSilenceResource{}

const (
	recurrenceTypeDaily   = "daily"
	recurrenceTypeWeekly  = "weekly"
	recurrenceTypeMonthly = "monthly"

	dailyTimeframeKey = "every_day"
)

// weekdays are the valid timeframe day keys for weekly recurrence (backend uses lowercase names).
var weekdays = map[string]bool{
	"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
	"friday": true, "saturday": true, "sunday": true,
}

func NewRecurringSilenceResource() resource.Resource {
	return &recurringSilenceResource{}
}

type recurringSilenceResource struct {
	client ApiClient
}

type recurringSilenceResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RecurrenceType types.String `tfsdk:"recurrence_type"`
	Timezone       types.String `tfsdk:"timezone"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Comment        types.String `tfsdk:"comment"`
	Timeframes     types.Set    `tfsdk:"timeframes"`
	Matchers       types.List   `tfsdk:"matchers"`
}

// timeframeModel is one flattened window. The backend groups these into a
// map[day][]TimeRange; we expose a flat set so the schema stays Crossplane-friendly
// (no MapNestedAttribute — see dynamic_attribute_guardrail_test.go).
type timeframeModel struct {
	Day       types.String `tfsdk:"day"`
	StartTime types.String `tfsdk:"start_time"`
	EndTime   types.String `tfsdk:"end_time"`
}

var timeframeObjectAttrTypes = map[string]attr.Type{
	"day":        types.StringType,
	"start_time": types.StringType,
	"end_time":   types.StringType,
}

var timeframeObjectType = types.ObjectType{AttrTypes: timeframeObjectAttrTypes}

func (r *recurringSilenceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recurring_silence"
}

func (r *recurringSilenceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a groundcover Recurring Silence.

A recurring silence repeatedly suppresses alerts on a schedule (daily, weekly, or monthly) rather than for a single fixed window. Each occurrence is generated from the ` + "`recurrence_type`" + `, ` + "`timeframes`" + `, and ` + "`timezone`" + `. For a one-off suppression window, use ` + "`groundcover_silence`" + ` instead.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the recurring silence.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"recurrence_type": schema.StringAttribute{
				MarkdownDescription: "How the silence recurs: `daily`, `weekly`, or `monthly`. This determines the valid `day` values in `timeframes`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(recurrenceTypeDaily, recurrenceTypeWeekly, recurrenceTypeMonthly),
				},
			},
			"timezone": schema.StringAttribute{
				MarkdownDescription: "IANA timezone name the timeframes are evaluated in (e.g. `UTC`, `America/New_York`, `Asia/Jerusalem`).",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the recurring silence is active. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "A comment describing the reason for the recurring silence.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeframes": schema.SetNestedAttribute{
				MarkdownDescription: "The set of time windows to silence. Each entry maps a `day` to a `start_time`/`end_time` (24-hour `HH:MM`). Valid `day` values depend on `recurrence_type`: `daily` uses `every_day`; `weekly` uses lowercase weekday names (`monday`..`sunday`); `monthly` uses day-of-month numbers as strings (`1`..`31`).",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"day": schema.StringAttribute{
							MarkdownDescription: "The day key. `every_day` (daily), a lowercase weekday name (weekly), or a day-of-month string (monthly).",
							Required:            true,
						},
						"start_time": schema.StringAttribute{
							MarkdownDescription: "Window start time, 24-hour `HH:MM` (e.g. `09:00`).",
							Required:            true,
						},
						"end_time": schema.StringAttribute{
							MarkdownDescription: "Window end time, 24-hour `HH:MM` (e.g. `17:30`).",
							Required:            true,
						},
					},
				},
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
							MarkdownDescription: "The value to match against. It can be an exact value or a partial match pattern if `is_contains` is true.",
							Required:            true,
						},
						"is_equal": schema.BoolAttribute{
							MarkdownDescription: "If true, the matcher matches when the label value equals the specified value. If false, it matches when the value does NOT equal. Defaults to `true`.",
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
	tflog.Info(ctx, "Recurring silence resource configured successfully")
}

func (r *recurringSilenceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config recurringSilenceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate timezone is a real IANA name.
	if !config.Timezone.IsNull() && !config.Timezone.IsUnknown() {
		if _, err := time.LoadLocation(config.Timezone.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("timezone"),
				"Invalid timezone",
				fmt.Sprintf("timezone must be a valid IANA timezone name (e.g. UTC, America/New_York): %s", err.Error()),
			)
		}
	}

	if config.Comment.IsNull() || config.Comment.IsUnknown() {
		// nothing to validate
	} else if config.Comment.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("comment"),
			"Invalid comment",
			"Comment cannot be an empty string. Either omit the comment attribute or provide a non-empty value.",
		)
	}

	if !config.Matchers.IsNull() && !config.Matchers.IsUnknown() && len(config.Matchers.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("matchers"),
			"Empty matchers",
			"matchers must contain at least one matcher.",
		)
	}

	if config.Timeframes.IsNull() || config.Timeframes.IsUnknown() {
		return
	}
	if len(config.Timeframes.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("timeframes"),
			"Empty timeframes",
			"timeframes must contain at least one entry.",
		)
		return
	}

	var timeframes []timeframeModel
	if diags := config.Timeframes.ElementsAs(ctx, &timeframes, false); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	recurrenceType := config.RecurrenceType.ValueString()
	for _, tf := range timeframes {
		if !tf.Day.IsUnknown() && !config.RecurrenceType.IsUnknown() {
			if err := validateTimeframeDay(recurrenceType, tf.Day.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("timeframes"), "Invalid timeframe day", err.Error())
			}
		}
		for attrName, v := range map[string]types.String{"start_time": tf.StartTime, "end_time": tf.EndTime} {
			if v.IsUnknown() {
				continue
			}
			if _, err := time.Parse("15:04", v.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(
					path.Root("timeframes"),
					"Invalid timeframe time",
					fmt.Sprintf("%s %q must be in 24-hour HH:MM format (e.g. 09:00)", attrName, v.ValueString()),
				)
			}
		}
	}
}

// validateTimeframeDay checks a day key against the recurrence type's allowed vocabulary.
func validateTimeframeDay(recurrenceType, day string) error {
	switch recurrenceType {
	case recurrenceTypeDaily:
		if day != dailyTimeframeKey {
			return fmt.Errorf("for recurrence_type %q the only valid day is %q, got %q", recurrenceType, dailyTimeframeKey, day)
		}
	case recurrenceTypeWeekly:
		if !weekdays[day] {
			return fmt.Errorf("for recurrence_type %q, day must be a lowercase weekday name (monday..sunday), got %q", recurrenceType, day)
		}
	case recurrenceTypeMonthly:
		n, err := strconv.Atoi(day)
		if err != nil || n < 1 || n > 31 {
			return fmt.Errorf("for recurrence_type %q, day must be a day-of-month string (\"1\"..\"31\"), got %q", recurrenceType, day)
		}
	}
	return nil
}

// --- Timeframe conversion helpers ---

// timeframesFromModel flattens the timeframe set into the backend's map[day][]TimeRange.
func timeframesFromModel(ctx context.Context, set types.Set) (map[string][]models.TimeRange, error) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var timeframes []timeframeModel
	if diags := set.ElementsAs(ctx, &timeframes, false); diags.HasError() {
		return nil, fmt.Errorf("failed to extract timeframes from set")
	}

	result := make(map[string][]models.TimeRange, len(timeframes))
	for _, tf := range timeframes {
		start := tf.StartTime.ValueString()
		end := tf.EndTime.ValueString()
		day := tf.Day.ValueString()
		result[day] = append(result[day], models.TimeRange{StartTime: &start, EndTime: &end})
	}
	return result, nil
}

// timeframesToModel expands the backend's map[day][]TimeRange into a flat timeframe set.
func timeframesToModel(timeframes map[string][]models.TimeRange) (types.Set, error) {
	values := make([]attr.Value, 0, len(timeframes))
	for day, ranges := range timeframes {
		for _, tr := range ranges {
			start := ""
			if tr.StartTime != nil {
				start = *tr.StartTime
			}
			end := ""
			if tr.EndTime != nil {
				end = *tr.EndTime
			}
			obj, diags := types.ObjectValue(timeframeObjectAttrTypes, map[string]attr.Value{
				"day":        types.StringValue(day),
				"start_time": types.StringValue(start),
				"end_time":   types.StringValue(end),
			})
			if diags.HasError() {
				return types.SetNull(timeframeObjectType), fmt.Errorf("failed to build timeframe object")
			}
			values = append(values, obj)
		}
	}

	set, diags := types.SetValue(timeframeObjectType, values)
	if diags.HasError() {
		return types.SetNull(timeframeObjectType), fmt.Errorf("failed to build timeframes set")
	}
	return set, nil
}

// --- CRUD Operations ---

func (r *recurringSilenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recurringSilenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeframes, err := timeframesFromModel(ctx, plan.Timeframes)
	if err != nil {
		resp.Diagnostics.AddError("Invalid timeframes", err.Error())
		return
	}

	matchers, err := silenceMatchersFromModel(ctx, plan.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Invalid matchers", err.Error())
		return
	}

	comment := plan.Comment.ValueString()
	if plan.Comment.IsNull() || comment == "" {
		comment = fmt.Sprintf("created %s", time.Now().UTC().Format("2006-01-02 15:04:05"))
	}

	silenceType := models.V2CreateSilenceRequestTypeRecurring
	apiRequest := &models.V2CreateSilenceRequest{
		Type:           &silenceType,
		RecurrenceType: plan.RecurrenceType.ValueString(),
		Timezone:       plan.Timezone.ValueString(),
		Enabled:        plan.Enabled.ValueBool(),
		Comment:        comment,
		Timeframes:     timeframes,
		Matchers:       matchers,
	}

	apiResponse, err := r.client.CreateRecurringSilence(ctx, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Create Recurring Silence Error", fmt.Sprintf("Failed to create recurring silence: %s", err.Error()))
		return
	}

	if apiResponse == nil || apiResponse.UUID.String() == "" {
		resp.Diagnostics.AddError("SDK Client Create Recurring Silence Error", "Create response missing recurring silence ID")
		return
	}

	tflog.Info(ctx, "Recurring silence created successfully via SDK", map[string]any{"id": apiResponse.UUID.String()})

	resp.Diagnostics.Append(r.updateModelFromResponse(&plan, apiResponse)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recurringSilenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state recurringSilenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	apiResponse, err := r.client.GetRecurringSilence(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Recurring silence %s not found, removing from state", id))
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

	resp.Diagnostics.Append(r.updateModelFromResponse(&state, apiResponse)...)
	if resp.Diagnostics.HasError() {
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

	timeframes, err := timeframesFromModel(ctx, plan.Timeframes)
	if err != nil {
		resp.Diagnostics.AddError("Invalid timeframes", err.Error())
		return
	}

	matchers, err := silenceMatchersFromModel(ctx, plan.Matchers)
	if err != nil {
		resp.Diagnostics.AddError("Invalid matchers", err.Error())
		return
	}

	comment := state.Comment.ValueString()
	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		comment = plan.Comment.ValueString()
	}

	apiRequest := &models.V2UpdateSilenceRequest{
		Type:           models.V2CreateSilenceRequestTypeRecurring,
		RecurrenceType: plan.RecurrenceType.ValueString(),
		Timezone:       plan.Timezone.ValueString(),
		Enabled:        plan.Enabled.ValueBool(),
		Comment:        comment,
		Timeframes:     timeframes,
		Matchers:       matchers,
	}

	apiResponse, err := r.client.UpdateRecurringSilence(ctx, id, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Update Recurring Silence Error", fmt.Sprintf("Failed to update recurring silence %s: %s", id, err.Error()))
		return
	}

	tflog.Info(ctx, "Recurring silence updated successfully via SDK", map[string]any{"id": id})

	plan.ID = state.ID
	if apiResponse != nil {
		resp.Diagnostics.Append(r.updateModelFromResponse(&plan, apiResponse)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recurringSilenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state recurringSilenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeleteRecurringSilence(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Recurring silence already deleted externally, treating as success", map[string]any{"id": id})
			return
		}
		resp.Diagnostics.AddError("SDK Client Delete Recurring Silence Error", fmt.Sprintf("Failed to delete recurring silence %s: %s", id, err.Error()))
		return
	}
	tflog.Info(ctx, "Recurring silence deleted successfully via SDK", map[string]any{"id": id})
}

func (r *recurringSilenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// updateModelFromResponse maps an API response back onto the Terraform model.
func (r *recurringSilenceResource) updateModelFromResponse(m *recurringSilenceResourceModel, apiResponse *models.V2SilenceResponse) (diags diag.Diagnostics) {
	if apiResponse.UUID.String() != "" {
		m.ID = types.StringValue(apiResponse.UUID.String())
	}
	m.RecurrenceType = types.StringValue(apiResponse.RecurrenceType)
	m.Timezone = types.StringValue(apiResponse.Timezone)
	m.Enabled = types.BoolValue(apiResponse.Enabled)
	m.Comment = types.StringValue(apiResponse.Comment)

	timeframesSet, err := timeframesToModel(apiResponse.Timeframes)
	if err != nil {
		diags.AddError("Error processing response timeframes", err.Error())
		return diags
	}
	m.Timeframes = timeframesSet

	matchersList, err := silenceMatchersToModel(apiResponse.Matchers)
	if err != nil {
		diags.AddError("Error processing response matchers", err.Error())
		return diags
	}
	m.Matchers = matchersList
	return diags
}
