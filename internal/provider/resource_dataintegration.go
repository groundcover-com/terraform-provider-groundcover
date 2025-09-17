package provider

import (
	"context"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure resource implements required interfaces
var (
	_ resource.Resource                = &dataIntegrationResource{}
	_ resource.ResourceWithConfigure   = &dataIntegrationResource{}
	_ resource.ResourceWithImportState = &dataIntegrationResource{}
)

func NewDataIntegrationResource() resource.Resource {
	return &dataIntegrationResource{}
}

type dataIntegrationResource struct {
	client ApiClient
}

type dataIntegrationResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Type      types.String `tfsdk:"type"`
	Env       types.String `tfsdk:"env"`
	Cluster   types.String `tfsdk:"cluster"`
	Instance  types.String `tfsdk:"instance"`
	Config    types.String `tfsdk:"config"`
	IsPaused  types.Bool   `tfsdk:"is_paused"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	UpdatedBy types.String `tfsdk:"updated_by"`
}

func (r *dataIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataintegration"
}

func (r *dataIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "DataIntegration resource for managing data integrations with external services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the data integration configuration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of data integration (e.g., 'cloudwatch', 'datadog', etc.).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"env": schema.StringAttribute{
				Description: "The environment where the data integration runs.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster": schema.StringAttribute{
				Description: "The cluster where the data integration runs.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance": schema.StringAttribute{
				Description: "The instance where the data integration runs.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.StringAttribute{
				Description: "The YAML configuration for the data integration.",
				Required:    true,
			},
			"is_paused": schema.BoolAttribute{
				Description: "Whether the data integration is paused.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the data integration configuration.",
				Computed:    true,
			},
			"updated_by": schema.StringAttribute{
				Description: "The user who last updated the data integration configuration.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *dataIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *dataIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan dataIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating DataIntegration", map[string]any{"type": plan.Type.ValueString()})

	// Create request model
	createReq := &models.CreateDataIntegrationConfigRequest{
		Config:   plan.Config.ValueString(),
		IsPaused: plan.IsPaused.ValueBool(),
	}

	// Set pointer fields only if they are not null
	if !plan.Env.IsNull() {
		env := plan.Env.ValueString()
		createReq.Env = &env
	}
	if !plan.Cluster.IsNull() {
		cluster := plan.Cluster.ValueString()
		createReq.Cluster = &cluster
	}
	if !plan.Instance.IsNull() {
		instance := plan.Instance.ValueString()
		createReq.Instance = &instance
	}

	// Call API client to create the data integration
	createdConfig, err := r.client.CreateDataIntegration(ctx, plan.Type.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating DataIntegration",
			fmt.Sprintf("Could not create DataIntegration: %s", err.Error()),
		)
		return
	}

	// Map response back to plan
	plan.ID = types.StringValue(createdConfig.ID)
	plan.Config = types.StringValue(createdConfig.Config)
	plan.UpdatedAt = types.StringValue(createdConfig.UpdateTimestamp.String())
	plan.UpdatedBy = types.StringValue(createdConfig.UpdatedBy)
	plan.IsPaused = types.BoolValue(createdConfig.IsPaused)

	tflog.Debug(ctx, fmt.Sprintf("DataIntegration created with ID: %s", createdConfig.ID))

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated DataIntegration resource with ID %s", createdConfig.ID))
}

// Read refreshes the Terraform state with the latest data.
func (r *dataIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dataIntegrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading DataIntegration resource", map[string]any{"id": state.ID.ValueString(), "type": state.Type.ValueString()})

	// Call API client to get the data integration
	configEntry, err := r.client.GetDataIntegration(ctx, state.Type.ValueString(), state.ID.ValueString())
	if err != nil {
		if err == ErrNotFound {
			tflog.Warn(ctx, "DataIntegration not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading DataIntegration",
			fmt.Sprintf("Could not read DataIntegration: %s", err.Error()),
		)
		return
	}

	// Handle case where data integration was deleted
	if configEntry == nil {
		tflog.Warn(ctx, "DataIntegration not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state
	state.ID = types.StringValue(configEntry.ID)
	state.Type = types.StringValue(configEntry.Type)
	state.Env = types.StringValue(configEntry.Env)
	state.Cluster = types.StringValue(configEntry.Cluster)
	state.Instance = types.StringValue(configEntry.Instance)
	state.Config = types.StringValue(configEntry.Config)
	state.IsPaused = types.BoolValue(configEntry.IsPaused)
	state.UpdatedAt = types.StringValue(configEntry.UpdateTimestamp.String())
	state.UpdatedBy = types.StringValue(configEntry.UpdatedBy)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully read DataIntegration resource with ID %s", configEntry.ID))
}

// Update updates the resource and sets the updated Terraform state.
func (r *dataIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var plan dataIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating DataIntegration", map[string]any{"id": plan.ID.ValueString(), "type": plan.Type.ValueString()})

	// Create update request model
	updateReq := &models.CreateDataIntegrationConfigRequest{
		Config:   plan.Config.ValueString(),
		IsPaused: plan.IsPaused.ValueBool(),
	}

	// Set pointer fields only if they are not null
	if !plan.Env.IsNull() {
		env := plan.Env.ValueString()
		updateReq.Env = &env
	}
	if !plan.Cluster.IsNull() {
		cluster := plan.Cluster.ValueString()
		updateReq.Cluster = &cluster
	}
	if !plan.Instance.IsNull() {
		instance := plan.Instance.ValueString()
		updateReq.Instance = &instance
	}

	// Call API client to update the data integration
	updatedConfig, err := r.client.UpdateDataIntegration(ctx, plan.Type.ValueString(), plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating DataIntegration",
			fmt.Sprintf("Could not update DataIntegration: %s", err.Error()),
		)
		return
	}

	// Update state
	plan.Env = types.StringValue(updatedConfig.Env)
	plan.Cluster = types.StringValue(updatedConfig.Cluster)
	plan.Instance = types.StringValue(updatedConfig.Instance)
	plan.Config = types.StringValue(updatedConfig.Config)
	plan.IsPaused = types.BoolValue(updatedConfig.IsPaused)
	plan.UpdatedAt = types.StringValue(updatedConfig.UpdateTimestamp.String())
	plan.UpdatedBy = types.StringValue(updatedConfig.UpdatedBy)

	// Set refreshed state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated DataIntegration resource with ID %s", updatedConfig.ID))
}

// Delete deletes the resource from Terraform state.
func (r *dataIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state dataIntegrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting DataIntegration resource", map[string]any{"id": state.ID.ValueString(), "type": state.Type.ValueString()})

	// Call API client to delete the data integration
	var env, cluster, instance *string
	if !state.Env.IsNull() {
		envVal := state.Env.ValueString()
		env = &envVal
	}
	if !state.Cluster.IsNull() {
		clusterVal := state.Cluster.ValueString()
		cluster = &clusterVal
	}
	if !state.Instance.IsNull() {
		instanceVal := state.Instance.ValueString()
		instance = &instanceVal
	}

	err := r.client.DeleteDataIntegration(ctx, state.Type.ValueString(), state.ID.ValueString(), env, cluster, instance)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting DataIntegration",
			fmt.Sprintf("Could not delete DataIntegration: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted DataIntegration resource with ID %s", state.ID.ValueString()))
}

// ImportState imports an existing resource into Terraform state.
func (r *dataIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "type:id" (e.g., "cloudwatch:abc123")
	importParts := len(req.ID)
	if importParts == 0 {
		resp.Diagnostics.AddError(
			"Invalid Import Format",
			"Import ID must be in the format 'type:id' (e.g., 'cloudwatch:abc123')",
		)
		return
	}

	// Find the separator
	var separatorIndex = -1
	for i, char := range req.ID {
		if char == ':' {
			separatorIndex = i
			break
		}
	}

	if separatorIndex == -1 {
		resp.Diagnostics.AddError(
			"Invalid Import Format",
			"Import ID must be in the format 'type:id' (e.g., 'cloudwatch:abc123')",
		)
		return
	}

	integrationType := req.ID[:separatorIndex]
	integrationID := req.ID[separatorIndex+1:]

	if integrationType == "" || integrationID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import Format",
			"Both type and id must be non-empty. Format: 'type:id' (e.g., 'cloudwatch:abc123')",
		)
		return
	}

	// Set the type and id attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), integrationType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), integrationID)...)
}
