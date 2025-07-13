package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	return &ingestionKeyResource{}
}

type ingestionKeyResource struct {
	client ApiClient
}

type ingestionKeyResourceModel struct {
	ID           types.String `tfsdk:"id"`
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
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"remote_config": schema.BoolAttribute{
				Description: "Indicates if the ingestion key is configured for remote configuration.",
				Optional:    true,
				Computed:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags associated with the ingestion key.",
				ElementType: types.StringType,
				Optional:    true,
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
			fmt.Sprintf("Expected provider.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *ingestionKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ingestionKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating Ingestion key: %s", plan.Name.ValueString()))
	tags, diags := r.tagsFromList(ctx, plan.Tags)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()
	var remoteConfig *bool
	if !plan.RemoteConfig.IsNull() && !plan.RemoteConfig.IsUnknown() {
		val := plan.RemoteConfig.ValueBool()
		remoteConfig = &val
	}

	createReq := &models.CreateIngestionKeyRequest{
		Name:         &nameStr,
		Type:         &typeStr,
		RemoteConfig: remoteConfig,
		Tags:         tags,
	}

	tflog.Debug(ctx, "Sending CreateIngestionKeyRequest to SDK", map[string]any{"name": nameStr, "type": typeStr, "remote_config": remoteConfig, "tags": tags})
	result, err := r.client.CreateIngestionKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Ingestion Key",
			fmt.Sprintf("Could not create Ingestion Key: %s", err.Error()),
		)
		return
	}

	// Map response back to state
	state := ingestionKeyResourceModel{
		ID:           types.StringValue(result.Name), // Use name as ID for ingestion keys
		Name:         types.StringValue(result.Name),
		CreatedBy:    types.StringValue(result.CreatedBy),
		CreationDate: types.StringValue(result.CreationDate.String()),
		Key:          types.StringValue(result.Key),
		Type:         types.StringValue(result.Type),
		RemoteConfig: types.BoolValue(result.RemoteConfig), // API always returns a bool
	}

	state.Tags, diags = r.tagsToList(ctx, result.Tags)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ingestionKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ingestionKeyResourceModel
	diags := req.State.Get(ctx, &state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading Ingestion Key resource: %s", state.Name.ValueString()))

	// Retry logic to handle timing issues - match SDK e2e pattern exactly
	targetName := state.Name.ValueString()

	// Retry logic for API consistency - use 10 seconds like SDK e2e pattern
	timeout := time.Now().Add(10 * time.Second)
	var response []*models.IngestionKeyResult

	for {
		// List ingestion keys by name like the SDK e2e test
		listResp, err := r.client.ListIngestionKeys(ctx, &models.ListIngestionKeysRequest{
			Name: targetName,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Ingestion Key",
				fmt.Sprintf("Could not read Ingestion Key: %s", err.Error()),
			)
			return
		}

		if len(listResp) > 0 || time.Now().After(timeout) {
			response = listResp
			break
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Ingestion Key %s to be listed, retrying...", targetName))
		time.Sleep(1 * time.Second)
	}

	if len(response) == 0 {
		tflog.Warn(ctx, fmt.Sprintf("Ingestion Key not found after timeout: %s", state.Name.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Ingestion Key found: %s", state.Name.ValueString()))

	// Update state with the first found key
	ingestionKey := response[0]
	state.ID = types.StringValue(ingestionKey.Name) // Ensure ID is set
	state.CreatedBy = types.StringValue(ingestionKey.CreatedBy)
	state.CreationDate = types.StringValue(ingestionKey.CreationDate.String())
	state.Key = types.StringValue(ingestionKey.Key)
	state.Type = types.StringValue(ingestionKey.Type)
	state.RemoteConfig = types.BoolValue(ingestionKey.RemoteConfig)
	state.Tags, diags = r.tagsToList(ctx, ingestionKey.Tags)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
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
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *ingestionKeyResource) tagsFromList(ctx context.Context, listVal types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if listVal.IsNull() || listVal.IsUnknown() {
		return nil, diags
	}

	var tags []string
	diags = listVal.ElementsAs(ctx, &tags, false)
	return tags, diags
}

func (r *ingestionKeyResource) tagsToList(ctx context.Context, tags []string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if tags == nil {
		return types.ListNull(types.StringType), diags
	}

	if len(tags) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{}), diags
	}

	elements := make([]attr.Value, len(tags))
	for i, tag := range tags {
		elements[i] = types.StringValue(tag)
	}

	listVal, diagsNew := types.ListValue(types.StringType, elements)
	diags.Append(diagsNew...)
	return listVal, diags
}
