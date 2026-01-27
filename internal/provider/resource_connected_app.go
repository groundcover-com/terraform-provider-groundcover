// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

var (
	_ resource.Resource                = &connectedAppResource{}
	_ resource.ResourceWithConfigure   = &connectedAppResource{}
	_ resource.ResourceWithImportState = &connectedAppResource{}
)

func NewConnectedAppResource() resource.Resource {
	return &connectedAppResource{}
}

type connectedAppResource struct {
	client ApiClient
}

type connectedAppResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Data      types.Map    `tfsdk:"data"`
	CreatedBy types.String `tfsdk:"created_by"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedBy types.String `tfsdk:"updated_by"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *connectedAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app"
}

func (r *connectedAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Connected App resource for managing integrations with external services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the connected app.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the connected app.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of connected app (slack-webhook or pagerduty).",
				Required:    true,
			},
			"data": schema.MapAttribute{
				Description: "Type-specific configuration. For slack-webhook: {url: string}. For pagerduty: {routing_key: string, severity_mapping?: map[string]string}.",
				ElementType: types.StringType,
				Required:    true,
				Sensitive:   true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the connected app.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date the connected app was created (RFC3339 format).",
				Computed:    true,
			},
			"updated_by": schema.StringAttribute{
				Description: "The user who last updated the connected app.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The date the connected app was last updated (RFC3339 format).",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *connectedAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *connectedAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan connectedAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating connected app: %s", plan.Name.ValueString()))

	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()

	dataMap := make(map[string]string)
	if !plan.Data.IsNull() && !plan.Data.IsUnknown() {
		diags := plan.Data.ElementsAs(ctx, &dataMap, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	dataAny := make(map[string]any)
	for k, v := range dataMap {
		dataAny[k] = v
	}

	createReq := &models.CreateConnectedAppRequest{
		Name: &nameStr,
		Type: &typeStr,
		Data: dataAny,
	}

	createResp, err := r.client.CreateConnectedApp(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating connected app", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Connected app created with ID: %s", createResp.ID))

	connectedApp, err := r.client.GetConnectedApp(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created connected app", err.Error())
		return
	}

	mapConnectedAppResponseToModel(ctx, connectedApp, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated connected app resource: %s", plan.Id.ValueString()))
}

// Read refreshes the Terraform state with the latest data.
func (r *connectedAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state connectedAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading connected app resource: %s", state.Id.ValueString()))

	connectedApp, err := r.client.GetConnectedApp(ctx, state.Id.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Connected app %s not found, removing from state.", state.Id.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading connected app", err.Error())
		return
	}

	mapConnectedAppResponseToModel(ctx, connectedApp, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully read connected app resource: %s", state.Id.ValueString()))
}

// Update updates the resource.
func (r *connectedAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan connectedAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Updating connected app: %s", plan.Id.ValueString()))

	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()

	dataMap := make(map[string]string)
	if !plan.Data.IsNull() && !plan.Data.IsUnknown() {
		diags := plan.Data.ElementsAs(ctx, &dataMap, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	dataAny := make(map[string]any)
	for k, v := range dataMap {
		dataAny[k] = v
	}

	updateReq := &models.UpdateConnectedAppRequest{
		Name: &nameStr,
		Type: &typeStr,
		Data: dataAny,
	}

	err := r.client.UpdateConnectedApp(ctx, plan.Id.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating connected app", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Connected app updated: %s", plan.Id.ValueString()))

	connectedApp, err := r.client.GetConnectedApp(ctx, plan.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated connected app", err.Error())
		return
	}

	mapConnectedAppResponseToModel(ctx, connectedApp, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully updated connected app resource: %s", plan.Id.ValueString()))
}

// Delete deletes the resource from Terraform state.
func (r *connectedAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state connectedAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectedAppId := state.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Deleting connected app resource: %s", connectedAppId))

	err := r.client.DeleteConnectedApp(ctx, connectedAppId)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			resp.Diagnostics.AddError("Error deleting connected app", err.Error())
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted connected app resource: %s", connectedAppId))
}

func (r *connectedAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapConnectedAppResponseToModel(ctx context.Context, app *models.ConnectedAppResponse, model *connectedAppResourceModel) {
	model.Id = types.StringValue(app.ID)
	model.Name = types.StringValue(app.Name)
	model.Type = types.StringValue(app.Type)

	if app.Data != nil {
		dataValue, _ := types.MapValueFrom(ctx, types.StringType, app.Data)
		model.Data = dataValue
	} else {
		model.Data = types.MapNull(types.StringType)
	}

	model.CreatedBy = types.StringValue(app.CreatedBy)
	if !app.CreatedAt.IsZero() {
		model.CreatedAt = types.StringValue(app.CreatedAt.String())
	} else {
		model.CreatedAt = types.StringNull()
	}

	model.UpdatedBy = types.StringValue(app.UpdatedBy)
	if !app.UpdatedAt.IsZero() {
		model.UpdatedAt = types.StringValue(app.UpdatedAt.String())
	} else {
		model.UpdatedAt = types.StringNull()
	}
}
