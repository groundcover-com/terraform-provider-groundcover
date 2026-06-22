// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

// connectedAppJsonResource is a string-based sibling of connectedAppResource: identical
// API behavior, but `data` is a JSON string instead of a dynamic object. It exists because
// terraform-plugin-framework dynamic attributes cannot be represented by code generators
// (e.g. upjet/Crossplane) — mirroring the datadog_dashboard vs datadog_dashboard_json split.
// The dynamic resource stays the default for HCL users; this variant is the codegen-friendly
// path. ponytail: shares the data_hash drift logic and SDK models with the dynamic resource.
var (
	_ resource.Resource                = &connectedAppJsonResource{}
	_ resource.ResourceWithConfigure   = &connectedAppJsonResource{}
	_ resource.ResourceWithImportState = &connectedAppJsonResource{}
)

func NewConnectedAppJsonResource() resource.Resource {
	return &connectedAppJsonResource{}
}

type connectedAppJsonResource struct {
	client ApiClient
}

type connectedAppJsonResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Data      types.String `tfsdk:"data"`
	DataHash  types.String `tfsdk:"data_hash"`
	CreatedBy types.String `tfsdk:"created_by"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedBy types.String `tfsdk:"updated_by"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *connectedAppJsonResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app_json"
}

func (r *connectedAppJsonResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Connected App resource whose type-specific configuration is supplied as a JSON string. Functionally identical to groundcover_connected_app, but the JSON-string `data` makes it usable by schema code generators (Crossplane/upjet) that cannot represent dynamic attributes.",
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
				Description: "Type of connected app (slack-webhook, pagerduty, opsgenie, incidentio, webhook, rootly, or ms-teams).",
				Required:    true,
			},
			"data": schema.StringAttribute{
				Description: "JSON-encoded type-specific configuration. Same shapes as groundcover_connected_app.data, supplied as a JSON object string, e.g. jsonencode({ url = \"https://...\" }) for slack-webhook.",
				Required:    true,
				Sensitive:   true,
			},
			"data_hash": schema.StringAttribute{
				Description: "SHA-256 hash of the stored connected app data, computed by groundcover. Because `data` is sensitive and redacted on read, this hash is how drift in the stored data is detected. Drift detection is forward-looking (see groundcover_connected_app.data_hash).",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{Description: "The user who created the connected app.", Computed: true},
			"created_at": schema.StringAttribute{Description: "The date the connected app was created (RFC3339 format).", Computed: true},
			"updated_by": schema.StringAttribute{Description: "The user who last updated the connected app.", Computed: true},
			"updated_at": schema.StringAttribute{Description: "The date the connected app was last updated (RFC3339 format).", Computed: true},
		},
	}
}

func (r *connectedAppJsonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *connectedAppJsonResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan connectedAppJsonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataAny, diags := jsonStringToMap(plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()
	createResp, err := r.client.CreateConnectedApp(ctx, &models.CreateConnectedAppRequest{Name: &nameStr, Type: &typeStr, Data: dataAny})
	if err != nil {
		resp.Diagnostics.AddError("Error creating connected app", err.Error())
		return
	}

	connectedApp, err := r.client.GetConnectedApp(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created connected app", err.Error())
		return
	}

	mapConnectedAppJsonResponseToModel(connectedApp, &plan, plan.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *connectedAppJsonResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state connectedAppJsonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	// data is redacted on read; preserve the authored JSON string to avoid perpetual diffs,
	// unless data_hash shows the stored data changed out-of-band (same contract as the dynamic resource).
	preserveData := state.Data
	if connectedAppDataDrifted(state.DataHash, connectedApp.DataHash) {
		tflog.Info(ctx, "Connected app data changed outside Terraform; surfacing drift via data_hash", map[string]any{
			"id": state.Id.ValueString(), "state_hash": state.DataHash.ValueString(), "remote_hash": connectedApp.DataHash,
		})
		preserveData = types.StringNull()
	}

	mapConnectedAppJsonResponseToModel(connectedApp, &state, preserveData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *connectedAppJsonResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan connectedAppJsonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataAny, diags := jsonStringToMap(plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()
	if err := r.client.UpdateConnectedApp(ctx, plan.Id.ValueString(), &models.UpdateConnectedAppRequest{Name: &nameStr, Type: &typeStr, Data: dataAny}); err != nil {
		resp.Diagnostics.AddError("Error updating connected app", err.Error())
		return
	}

	connectedApp, err := r.client.GetConnectedApp(ctx, plan.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated connected app", err.Error())
		return
	}

	mapConnectedAppJsonResponseToModel(connectedApp, &plan, plan.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *connectedAppJsonResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state connectedAppJsonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteConnectedApp(ctx, state.Id.ValueString()); err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Connected app already deleted externally, treating as success", map[string]any{"id": state.Id.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting connected app", err.Error())
	}
}

func (r *connectedAppJsonResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapConnectedAppJsonResponseToModel(app *models.ConnectedAppResponse, model *connectedAppJsonResourceModel, preserveData types.String) {
	model.Id = types.StringValue(app.ID)
	model.Name = types.StringValue(app.Name)
	model.Type = types.StringValue(app.Type)

	// data is redacted on read; keep the authored JSON string. (On import there is no prior
	// value, so it stays null until the next apply supplies it.)
	model.Data = preserveData

	if app.DataHash != "" {
		model.DataHash = types.StringValue(app.DataHash)
	} else {
		model.DataHash = types.StringNull()
	}

	model.CreatedBy = types.StringValue(app.CreatedBy)
	if !time.Time(app.CreatedAt).IsZero() {
		model.CreatedAt = types.StringValue(time.Time(app.CreatedAt).Format(time.RFC3339))
	} else {
		model.CreatedAt = types.StringNull()
	}
	model.UpdatedBy = types.StringValue(app.UpdatedBy)
	if !time.Time(app.UpdatedAt).IsZero() {
		model.UpdatedAt = types.StringValue(time.Time(app.UpdatedAt).Format(time.RFC3339))
	} else {
		model.UpdatedAt = types.StringNull()
	}
}

// jsonStringToMap parses the JSON-string `data` attribute into the map the SDK expects.
func jsonStringToMap(data types.String) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if data.IsNull() || data.IsUnknown() {
		diags.AddError("Missing required attribute", "The 'data' attribute is required and cannot be null or unknown")
		return nil, diags
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(data.ValueString()), &m); err != nil {
		diags.AddError("Invalid data JSON", fmt.Sprintf("The 'data' attribute must be a JSON object string: %v", err))
		return nil, diags
	}
	return m, diags
}
