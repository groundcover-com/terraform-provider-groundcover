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

// Ensure resource implements required interfaces
var (
	_ resource.Resource                = &integrationResource{}
	_ resource.ResourceWithConfigure   = &integrationResource{}
	_ resource.ResourceWithImportState = &integrationResource{}
)

func NewIntegrationResource() resource.Resource {
	return &integrationResource{}
}

type integrationResource struct {
	client ApiClient
}

type integrationResourceModel struct {
	// The resource key is type+id. Logically, type is the primary key, but for terraform to work, the IDs are unique across
	// all types.
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`

	CreatedAt types.String `tfsdk:"created_at"`
	Value     types.String `tfsdk:"value"`
}

func (r *integrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (r *integrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Integration resource for managing integrations with external services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the integration configuration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of integration (e.g., 'cloudwatch', 'datadog', etc.).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Description: "The YAML configuration value for the integration.",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the integration configuration.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *integrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *integrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan integrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Integration", map[string]any{"type": plan.Type.ValueString()})

	// Create request model
	createReq := &models.CreateIntegrationConfigRequest{
		Type:  plan.Type.ValueString(),
		Value: plan.Value.ValueString(),
	}

	// Call API client to create the integration
	createdConfig, err := r.client.CreateIntegration(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Integration",
			fmt.Sprintf("Could not create Integration: %s", err.Error()),
		)
		return
	}

	// Map response back to plan
	plan.ID = types.StringValue(createdConfig.ID)
	plan.CreatedAt = types.StringValue(createdConfig.CreatedTimestamp.String())

	tflog.Debug(ctx, fmt.Sprintf("Integration created with ID: %s", createdConfig.ID))

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated Integration resource with ID %s", createdConfig.ID))
}

// Read refreshes the Terraform state with the latest data.
func (r *integrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Integration resource", map[string]any{"id": state.ID.ValueString(), "type": state.Type.ValueString()})

	// Call API client to get the integration
	configEntry, err := r.client.GetIntegration(ctx, state.Type.ValueString(), state.ID.ValueString())
	if err != nil {
		if err == ErrNotFound {
			tflog.Warn(ctx, "Integration not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Integration",
			fmt.Sprintf("Could not read Integration: %s", err.Error()),
		)
		return
	}

	// Handle case where integration was deleted
	if configEntry == nil {
		tflog.Warn(ctx, "Integration not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state
	state.Value = types.StringValue(configEntry.Value)
	state.CreatedAt = types.StringValue(configEntry.CreatedTimestamp.String())

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully read Integration resource with ID %s", configEntry.ID))
}

// Update updates the resource and sets the updated Terraform state.
func (r *integrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var plan integrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Integration", map[string]any{"id": plan.ID.ValueString(), "type": plan.Type.ValueString()})

	// Create update request model
	updateReq := &models.UpdateIntegrationConfigRequest{
		ID:    plan.ID.ValueString(),
		Type:  plan.Type.ValueString(),
		Value: plan.Value.ValueString(),
	}

	// Call API client to update the integration
	updatedConfig, err := r.client.UpdateIntegration(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Integration",
			fmt.Sprintf("Could not update Integration: %s", err.Error()),
		)
		return
	}

	// Update state
	plan.CreatedAt = types.StringValue(updatedConfig.CreatedTimestamp.String())

	// Set refreshed state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated Integration resource with ID %s", updatedConfig.ID))
}

// Delete deletes the resource from Terraform state.
func (r *integrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state integrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Integration resource", map[string]any{"id": state.ID.ValueString(), "type": state.Type.ValueString()})

	// Call API client to delete the integration
	err := r.client.DeleteIntegration(ctx, state.Type.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Integration",
			fmt.Sprintf("Could not delete Integration: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted Integration resource with ID %s", state.ID.ValueString()))
}

// ImportState imports an existing resource into Terraform state.
func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
