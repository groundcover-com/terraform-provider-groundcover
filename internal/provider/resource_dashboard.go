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
		Description: "Dashboard resource for managing Groundcover dashboards.",
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
			},
			"status": schema.StringAttribute{
				Description: "The status of the dashboard.",
				Computed:    true,
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

	tflog.Debug(ctx, "Creating Dashboard")

	createReq := &models.CreateDashboardRequest{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Team:          plan.Team.ValueString(),
		Preset:        plan.Preset.ValueString(),
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

	plan.UUID = types.StringValue(dashboard.UUID)
	plan.Name = types.StringValue(dashboard.Name)
	plan.Description = types.StringValue(dashboard.Description)
	plan.Team = types.StringValue(dashboard.Team)
	plan.Owner = types.StringValue(dashboard.Owner)
	plan.Status = types.StringValue(dashboard.Status)
	plan.RevisionNumber = types.Int32Value(dashboard.RevisionNumber)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Dashboard created successfully", map[string]interface{}{
		"uuid": dashboard.UUID,
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
		"uuid": state.UUID.ValueString(),
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

	state.UUID = types.StringValue(dashboard.UUID)
	state.Name = types.StringValue(dashboard.Name)
	state.Description = types.StringValue(dashboard.Description)
	state.Team = types.StringValue(dashboard.Team)
	if state.Preset.IsNull() || state.Preset.IsUnknown() {
		state.Preset = types.StringValue(dashboard.Preset)
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

	tflog.Debug(ctx, "Updating Dashboard", map[string]interface{}{
		"uuid":              state.UUID.ValueString(),
		"plan_description":  plan.Description.ValueString(),
		"state_description": state.Description.ValueString(),
	})

	// Build update request - always use plan values for all fields
	// The API requires all fields to be present in the update request
	updateReq := &models.UpdateDashboardRequest{
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		Team:            plan.Team.ValueString(),
		Preset:          plan.Preset.ValueString(),
		IsProvisioned:   true, // Always set to true as requested
		CurrentRevision: state.RevisionNumber.ValueInt32(),
		Override:        false, // Use false by default to avoid conflicts
	}

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

	plan.UUID = types.StringValue(dashboard.UUID)
	plan.Name = types.StringValue(dashboard.Name)
	plan.Description = types.StringValue(dashboard.Description)
	plan.Team = types.StringValue(dashboard.Team)
	plan.Owner = types.StringValue(dashboard.Owner)
	plan.Status = types.StringValue(dashboard.Status)
	plan.RevisionNumber = types.Int32Value(dashboard.RevisionNumber)

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
