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
	_ resource.Resource                = &metricsAggregationResource{}
	_ resource.ResourceWithConfigure   = &metricsAggregationResource{}
	_ resource.ResourceWithImportState = &metricsAggregationResource{}
	_ resource.ResourceWithModifyPlan  = &metricsAggregationResource{}
)

func NewMetricsAggregationResource() resource.Resource {
	return &metricsAggregationResource{}
}

type metricsAggregationResource struct {
	client ApiClient
}

type metricsAggregationResourceModel struct {
	Value     types.String `tfsdk:"value"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *metricsAggregationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metricsaggregation"
}

func (r *metricsAggregationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Metrics Aggregation resource. This is a singleton resource that configures metrics aggregation rules.",
		Attributes: map[string]schema.Attribute{
			"value": schema.StringAttribute{
				Description: "The YAML representation of the metrics aggregation configuration.",
				Required:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the metrics aggregation configuration.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *metricsAggregationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *metricsAggregationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan metricsAggregationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating MetricsAggregation")

	// Unmarshal to SDK type
	createReq := &models.CreateOrUpdateMetricsAggregatorConfigRequest{
		Value: plan.Value.ValueString(),
	}

	// Call API client to create the metrics aggregation
	createdConfig, err := r.client.CreateMetricsAggregation(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating MetricsAggregation",
			fmt.Sprintf("Could not create MetricsAggregation: %s", err.Error()),
		)
		return
	}

	plan.UpdatedAt = types.StringValue(createdConfig.CreatedTimestamp.String())

	tflog.Debug(ctx, fmt.Sprintf("MetricsAggregation created with UUID: %s", createdConfig.UUID))

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated MetricsAggregation resource with uuid %s", createdConfig.UUID))
}

// Read refreshes the Terraform state with the latest data.
func (r *metricsAggregationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state metricsAggregationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading MetricsAggregation resource")

	// Call API client to get the metrics aggregation
	configEntry, err := r.client.GetMetricsAggregation(ctx)
	if err != nil {
		if err == ErrNotFound {
			tflog.Warn(ctx, "MetricsAggregation not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading MetricsAggregation",
			fmt.Sprintf("Could not read MetricsAggregation: %s", err.Error()),
		)
		return
	}

	value := ""
	createdAt := ""
	uuid := ""
	if configEntry != nil {
		value = configEntry.Value
		createdAt = configEntry.CreatedTimestamp.String()
		uuid = configEntry.UUID
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
	if uuid != "" {
		tflog.Debug(ctx, fmt.Sprintf("Successfully read MetricsAggregation resource with UUID %s", uuid))
	} else {
		tflog.Debug(ctx, "Successfully read MetricsAggregation resource (no config found)")
	}
}

// Update updates the resource and sets the updated Terraform state.
func (r *metricsAggregationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var plan metricsAggregationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating MetricsAggregation")

	// Unmarshal to SDK type
	updateReq := &models.CreateOrUpdateMetricsAggregatorConfigRequest{
		Value: plan.Value.ValueString(),
	}

	// Call API client to update the metrics aggregation
	updatedConfig, err := r.client.UpdateMetricsAggregation(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating MetricsAggregation",
			fmt.Sprintf("Could not update MetricsAggregation: %s", err.Error()),
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
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated MetricsAggregation resource with UUID %s", updatedConfig.UUID))
}

// Delete deletes the resource from Terraform state.
func (r *metricsAggregationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state metricsAggregationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting MetricsAggregation resource")

	// Call API client to delete the metrics aggregation
	err := r.client.DeleteMetricsAggregation(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting MetricsAggregation",
			fmt.Sprintf("Could not delete MetricsAggregation: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Successfully deleted MetricsAggregation resource")
}

// For singleton resources, we don't need the ID for lookups
// But we need to implement a custom import rather than using ImportStatePassthroughID
func (r *metricsAggregationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, err := r.checkAndImportExisting(ctx, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing MetricsAggregation",
			fmt.Sprintf("Could not import MetricsAggregation: %s", err.Error()),
		)
	}
}

func (r *metricsAggregationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Modifying MetricsAggregation plan")

	_, err := r.checkAndImportExisting(ctx, &req.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing MetricsAggregation",
			fmt.Sprintf("Could not import MetricsAggregation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.AddWarning(
		"MetricsAggregation is a Singleton",
		fmt.Sprintf(
			"Your plan should never include more than one metrics aggregation resource. If it does, only the latest will take place.\n"+
				"Renaming the resource will show an incorrect plan.",
		),
	)
}

func (r *metricsAggregationResource) checkAndImportExisting(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics) (*models.MetricsAggregatorConfig, error) {
	existingConfig, err := r.client.GetMetricsAggregation(ctx)
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
