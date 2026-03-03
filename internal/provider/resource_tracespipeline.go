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
	_ resource.Resource                = &tracesPipelineResource{}
	_ resource.ResourceWithConfigure   = &tracesPipelineResource{}
	_ resource.ResourceWithImportState = &tracesPipelineResource{}
	_ resource.ResourceWithModifyPlan  = &tracesPipelineResource{}
)

func NewTracesPipelineResource() resource.Resource {
	return &tracesPipelineResource{}
}

type tracesPipelineResource struct {
	client ApiClient
}

type tracesPipelineResourceModel struct {
	Value     types.String `tfsdk:"value"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *tracesPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tracespipeline"
}

func (r *tracesPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Traces Pipeline resource. This is a singleton resource.",
		Attributes: map[string]schema.Attribute{
			"value": schema.StringAttribute{
				Description: "The YAML representation of the traces pipeline configuration.",
				Required:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the traces pipeline configuration.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *tracesPipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *tracesPipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan tracesPipelineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating TracesPipeline")

	// Unmarshal to SDK type
	createReq := &models.CreateOrUpdateTracesPipelineConfigRequest{
		Value: plan.Value.ValueString(),
	}

	// Call API client to create the traces pipeline
	createdConfig, err := r.client.CreateTracesPipeline(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating TracesPipeline",
			fmt.Sprintf("Could not create TracesPipeline: %s", err.Error()),
		)
		return
	}

	plan.UpdatedAt = types.StringValue(createdConfig.CreatedTimestamp.String())

	tflog.Debug(ctx, fmt.Sprintf("TracesPipeline created with UUID: %s", createdConfig.UUID))

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated TracesPipeline resource with uuid %s", createdConfig.UUID))
}

// Read refreshes the Terraform state with the latest data.
func (r *tracesPipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tracesPipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading TracesPipeline resource")

	// Call API client to get the traces pipeline
	configEntry, err := r.client.GetTracesPipeline(ctx)
	if err != nil {
		if err == ErrNotFound {
			tflog.Warn(ctx, "TracesPipeline not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading TracesPipeline",
			fmt.Sprintf("Could not read TracesPipeline: %s", err.Error()),
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
	tflog.Debug(ctx, fmt.Sprintf("Successfully read TracesPipeline resource with UUID %s", configEntry.UUID))
}

// Update updates the resource and sets the updated Terraform state.
func (r *tracesPipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var plan tracesPipelineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating TracesPipeline")

	// Unmarshal to SDK type
	updateReq := &models.CreateOrUpdateTracesPipelineConfigRequest{
		Value: plan.Value.ValueString(),
	}

	// Call API client to update the traces pipeline
	updatedConfig, err := r.client.UpdateTracesPipeline(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating TracesPipeline",
			fmt.Sprintf("Could not update TracesPipeline: %s", err.Error()),
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
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated TracesPipeline resource with UUID %s", updatedConfig.UUID))
}

// Delete deletes the resource from Terraform state.
func (r *tracesPipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state tracesPipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting TracesPipeline resource")

	// Call API client to delete the traces pipeline
	err := r.client.DeleteTracesPipeline(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting TracesPipeline",
			fmt.Sprintf("Could not delete TracesPipeline: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Successfully deleted TracesPipeline resource")
}

// For singleton resources, we don't need the ID for lookups
// But we need to implement a custom import rather than using ImportStatePassthroughID
func (r *tracesPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, err := r.checkAndImportExisting(ctx, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing TracesPipeline",
			fmt.Sprintf("Could not import TracesPipeline: %s", err.Error()),
		)
	}
}

func (r *tracesPipelineResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Modifying TracesPipeline plan")

	_, err := r.checkAndImportExisting(ctx, &req.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing TracesPipeline",
			fmt.Sprintf("Could not import TracesPipeline: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.AddWarning(
		"TracesPipeline is a Singleton",
		fmt.Sprintf(
			"Your plan should never include more than one traces pipeline resource. If it does, only the latest will take place.\n"+
				"Renaming the resource will show an incorrect plan.",
		),
	)
}

func (r *tracesPipelineResource) checkAndImportExisting(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics) (*models.TracesPipelineConfig, error) {
	existingConfig, err := r.client.GetTracesPipeline(ctx)
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
