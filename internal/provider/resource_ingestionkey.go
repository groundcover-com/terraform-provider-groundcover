package provider

import (
	"context"
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
	_ resource.Resource                = &ingestionKeyResource{}
	_ resource.ResourceWithConfigure   = &ingestionKeyResource{}
	_ resource.ResourceWithImportState = &ingestionKeyResource{}
)

func NewIngestionKeyResource() resource.Resource {
	return &apiKeyResource{}
}

type ingestionKeyResource struct {
	client ApiClient
}

type ingestionKeyResourceModel struct {
	Id           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	CreatedBy    types.String `tfsdk:"created_by"`
	CreationDate types.String `tfsdk:"creation_date"`
	Key          types.String `tfsdk:"key"`
	Type         types.String `tfsdk:"type"`
	RemoteConfig types.Bool   `tfsdk:"remote_config"`
	Tags         types.List   `tfsdk:"tags"`
}

func (r *ingestionKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestionkey"
}

func (r *ingestionKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Ingestion Key resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the ingestion key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the ingestion key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the ingestion key.",
				Computed:    true,
			},
			"creation_date": schema.StringAttribute{
				Description: "The creation date of the ingestion key.",
				Computed:    true,
			},
			"key": schema.StringAttribute{
				Description: "The actual key value for ingestion.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the ingestion key (e.g., 'ingestion').",
				Computed:    true,
			},
			"remote_config": schema.BoolAttribute{
				Description: "Indicates if the ingestion key is configured for remote configuration.",
				Computed:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags associated with the ingestion key.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ingestionKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected provider.ApiClient, got: "+req.ProviderData.(string)+". Please report this issue to the provider developers.",
		)
		return
	}
	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *ingestionKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ingestionKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating Ingestion key: %s", plan.Name.ValueString()))

	// Prepare the request for the SDK
	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()
	remoteConfig := plan.RemoteConfig.ValueBoolPointer()
	tags := []string{}
	diags := plan.Tags.ElementsAs(ctx, &tags, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &models.CreateIngestionKeyRequest{
		Name:         &nameStr,
		Type:         &typeStr,
		RemoteConfig: remoteConfig,
		Tags:         tags,
	}

	tflog.Debug(ctx, "Sending CreateIngestionKeyRequest to SDK", map[string]any{"name": nameStr, "type": typeStr, "remote_config": remoteConfig, "tags": tags})
	apiResponse, err := r.client.CreateIngestionKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Ingestion Key",
			fmt.Sprintf("Could not create Ingestion Key: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Ingestion Key created %s", apiResponse.Name))

	// Map response back to state
	if apiResponse.Name != nil {
		plan.Name = types.StringValue(*apiResponse.Name)
	}

	if apiResponse.Key != nil {
		plan.Key = types.StringValue(*apiResponse.Key)
	}

	diags = resp.State.Set(ctx, plan)
	if diags.HasError() {
		tflog.Error(ctx, "Error setting state for Ingestion Key", map[string]any{"name": plan.Name.ValueString()})
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ingestionKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ingestionKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading Ingestion Key resource: %s", state.Name.ValueString()))
	response, err := r.client.ListIngestionKeys(ctx, &models.ListIngestionKeysRequest{
		Name: state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Ingestion Key",
			fmt.Sprintf("Could not read Ingestion Key: %s", err.Error()),
		)
		return
	}

	if len(response) == 0 {
		tflog.Warn(ctx, fmt.Sprintf("Ingestion Key not found: %s", state.Name.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Ingestion Key found: %s", state.Name.ValueString()))

	// Update state with the first found key
	ingestionKey := response[0]

	state.Id = types.StringValue(ingestionKey.ID)
	state.CreatedBy = types.StringValue(ingestionKey.CreatedBy)
	state.CreationDate = types.StringValue(ingestionKey.CreationDate.String())
	state.Key = types.StringValue(ingestionKey.Key)
	state.Type = types.StringValue(ingestionKey.Type)
	state.RemoteConfig = types.BoolValue(ingestionKey.RemoteConfig)
	state.Tags, diags = types.ListValueFrom(ctx, types.StringType, ingestionKey.Tags)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Set the refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Error setting state for Ingestion Key", map[string]any{"name": state.Name.ValueString()})
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully read Ingestion Key resource: %s", state.Name.ValueString()))
}

func (r *ingestionKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Ingestion keys are immutable in this implementation.
	tflog.Warn(ctx, "Update operation is not supported for Ingestion Key resource. Please use Create or Delete operations instead.")
}

func (r *ingestionKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ingestionKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ingestionKeyName := state.Name.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Deleting Ingestion Key resource: %s", ingestionKeyName))

	deleteReq := &models.DeleteIngestionKeyRequest{
		Name: &ingestionKeyName,
	}
	if err := r.client.DeleteIngestionKey(ctx, deleteReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Ingestion Key",
			fmt.Sprintf("Could not delete Ingestion Key: %s", err.Error()),
		)
		return
	}

	// Remove the resource from state
	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, fmt.Sprintf("Removed Ingestion Key resource from state: %s", ingestionKeyName))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Error removing Ingestion Key resource from state", map[string]any{"name": ingestionKeyName})
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted Ingestion Key resource: %s", ingestionKeyName))
}

// ImportState imports the resource from its ID.
func (r *ingestionKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
