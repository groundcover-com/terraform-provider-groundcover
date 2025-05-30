package provider

import (
	"context"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure resource implements required interfaces
var (
	_ resource.Resource                = &logsPipelineResource{}
	_ resource.ResourceWithConfigure   = &logsPipelineResource{}
	_ resource.ResourceWithImportState = &logsPipelineResource{}
	_ resource.ResourceWithModifyPlan  = &logsPipelineResource{}
)

func NewLogsPipelineResource() resource.Resource {
	return &logsPipelineResource{}
}

type logsPipelineResource struct {
	client ApiClient
}

type logsPipelineResourceModel struct {
	Value     types.String `tfsdk:"value"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *logsPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logspipeline"
}

func (r *logsPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Logs Pipeline resource. This is a singleton resource.",
		Attributes: map[string]schema.Attribute{
			"value": schema.StringAttribute{
				Description: "The YAML representation of the logs pipeline configuration.",
				Required:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the logs pipeline configuration.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *logsPipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *logsPipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan logsPipelineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating LogsPipeline")

	// Unmarshal to SDK type
	createReq := &models.CreateOrUpdateLogsPipelineConfigRequest{
		Value: plan.Value.ValueString(),
	}

	// Call API client to create the logs pipeline
	createdConfig, err := r.client.CreateLogsPipeline(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating LogsPipeline",
			fmt.Sprintf("Could not create LogsPipeline: %s", err.Error()),
		)
		return
	}

	plan.UpdatedAt = types.StringValue(createdConfig.CreatedTimestamp.String())

	tflog.Debug(ctx, fmt.Sprintf("LogsPipeline created with UUID: %s", createdConfig.UUID))

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated LogsPipeline resource with uuid %s", createdConfig.UUID))
}

// Read refreshes the Terraform state with the latest data.
func (r *logsPipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state logsPipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading LogsPipeline resource")

	// Call API client to get the logs pipeline
	configEntry, err := r.client.GetLogsPipeline(ctx)
	if err != nil {
		if err == ErrNotFound {
			tflog.Warn(ctx, "LogsPipeline not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading LogsPipeline",
			fmt.Sprintf("Could not read LogsPipeline: %s", err.Error()),
		)
		return
	}

	value := ""
	createdAt := ""
	if configEntry != nil {
		value = configEntry.Value
		createdAt = configEntry.CreatedTimestamp.String()
	}

	// Update state
	state.UpdatedAt = types.StringValue(createdAt)
	state.Value = types.StringValue(value)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully read LogsPipeline resource with UUID %s", configEntry.UUID))
}

// Update updates the resource and sets the updated Terraform state.
func (r *logsPipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var plan logsPipelineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating LogsPipeline")

	// Unmarshal to SDK type
	updateReq := &models.CreateOrUpdateLogsPipelineConfigRequest{
		Value: plan.Value.ValueString(),
	}

	// Call API client to update the logs pipeline
	updatedConfig, err := r.client.UpdateLogsPipeline(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating LogsPipeline",
			fmt.Sprintf("Could not update LogsPipeline: %s", err.Error()),
		)
		return
	}

	// Update state
	plan.UpdatedAt = types.StringValue(updatedConfig.CreatedTimestamp.String())

	// Set refreshed state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated LogsPipeline resource with UUID %s", updatedConfig.UUID))
}

// Delete deletes the resource from Terraform state.
func (r *logsPipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state logsPipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting LogsPipeline resource")

	// Call API client to delete the logs pipeline
	err := r.client.DeleteLogsPipeline(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting LogsPipeline",
			fmt.Sprintf("Could not delete LogsPipeline: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Successfully deleted LogsPipeline resource")
}

// For singleton resources, we don't need the ID for lookups
// But we need to implement a custom import rather than using ImportStatePassthroughID
func (r *logsPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, err := r.checkAndImportExisting(ctx, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing LogsPipeline",
			fmt.Sprintf("Could not import LogsPipeline: %s", err.Error()),
		)
	}
}

func (r *logsPipelineResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Modifying LogsPipeline plan")

	_, err := r.checkAndImportExisting(ctx, &req.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing LogsPipeline",
			fmt.Sprintf("Could not import LogsPipeline: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.AddWarning(
		"LogsPipeline is a Singleton",
		fmt.Sprintf(
			"Your plan should never include more than one logs pipeline resource. If it does, only the latest will take place.\n"+
				"Renaming the resource will show an incorrect plan.",
		),
	)
}

func (r *logsPipelineResource) checkAndImportExisting(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics) (*models.LogsPipelineConfig, error) {
	existingConfig, err := r.client.GetLogsPipeline(ctx)
	if err != nil && err != ErrNotFound {
		return nil, err
	}

	value := ""
	createdAt := ""
	if existingConfig != nil {
		value = existingConfig.Value
		createdAt = existingConfig.CreatedTimestamp.String()
	}

	diags.Append(state.SetAttribute(ctx, path.Root("value"), value)...)
	diags.Append(state.SetAttribute(ctx, path.Root("updated_at"), createdAt)...)
	return existingConfig, nil
}
