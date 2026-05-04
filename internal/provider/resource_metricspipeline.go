package provider

import (
	"context"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &metricsPipelineResource{}
	_ resource.ResourceWithConfigure   = &metricsPipelineResource{}
	_ resource.ResourceWithImportState = &metricsPipelineResource{}
	_ resource.ResourceWithModifyPlan  = &metricsPipelineResource{}
)

func NewMetricsPipelineResource() resource.Resource {
	return &metricsPipelineResource{}
}

type metricsPipelineResource struct {
	client ApiClient
}

type metricsPipelineResourceModel struct {
	Rules     *metricsPipelineRulesModel `tfsdk:"rules"`
	UpdatedAt types.String               `tfsdk:"updated_at"`
}

type metricsPipelineRulesModel struct {
	KeepRegex   []types.String `tfsdk:"keep_regex"`
	DropRegex   []types.String `tfsdk:"drop_regex"`
	AddLabel    types.Map      `tfsdk:"add_label"`
	RemoveLabel []types.String `tfsdk:"remove_label"`
	Raw         types.String   `tfsdk:"raw"`
}

func (r *metricsPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metricspipeline"
}

func (r *metricsPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Metrics Pipeline resource. Singleton resource that configures metrics relabeling rules applied before aggregation.",
		Attributes: map[string]schema.Attribute{
			"rules": schema.SingleNestedAttribute{
				Description: "Relabeling rules to apply to metrics.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"keep_regex": schema.ListAttribute{
						Description: "List of regex patterns. Only metrics whose __name__ matches at least one pattern are kept. All others are dropped.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"drop_regex": schema.ListAttribute{
						Description: "List of regex patterns. Metrics whose __name__ matches any pattern are dropped.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"add_label": schema.MapAttribute{
						Description: "Map of label key-value pairs to add to every metric.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"remove_label": schema.ListAttribute{
						Description: "List of label names to remove from every metric.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"raw": schema.StringAttribute{
						Description: "Raw VictoriaMetrics relabeling rules in YAML format for advanced use cases.",
						Optional:    true,
					},
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the metrics pipeline configuration.",
				Computed:    true,
			},
		},
	}
}

func (r *metricsPipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *metricsPipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan metricsPipelineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating MetricsPipeline")

	sdkReq := planToMetricsPipelineRequest(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateMetricsPipeline(ctx, sdkReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating MetricsPipeline", fmt.Sprintf("Could not create MetricsPipeline: %s", err.Error()))
		return
	}

	plan.UpdatedAt = types.StringValue(created.CreatedTimestamp.String())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, fmt.Sprintf("Successfully created MetricsPipeline resource with UUID %s", created.UUID))
}

func (r *metricsPipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state metricsPipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading MetricsPipeline resource")

	configEntry, err := r.client.GetMetricsPipeline(ctx)
	if err != nil {
		if err == ErrNotFound {
			tflog.Warn(ctx, "MetricsPipeline not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading MetricsPipeline", fmt.Sprintf("Could not read MetricsPipeline: %s", err.Error()))
		return
	}

	metricsPipelineConfigToState(ctx, configEntry, &state, &resp.Diagnostics)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *metricsPipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan metricsPipelineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating MetricsPipeline")

	sdkReq := planToMetricsPipelineRequest(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateMetricsPipeline(ctx, sdkReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating MetricsPipeline", fmt.Sprintf("Could not update MetricsPipeline: %s", err.Error()))
		return
	}

	plan.UpdatedAt = types.StringValue(updated.CreatedTimestamp.String())

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated MetricsPipeline resource with UUID %s", updated.UUID))
}

func (r *metricsPipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state metricsPipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting MetricsPipeline resource")

	err := r.client.DeleteMetricsPipeline(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting MetricsPipeline", fmt.Sprintf("Could not delete MetricsPipeline: %s", err.Error()))
		return
	}

	tflog.Debug(ctx, "Successfully deleted MetricsPipeline resource")
}

func (r *metricsPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, err := r.checkAndImportExisting(ctx, &resp.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing MetricsPipeline", fmt.Sprintf("Could not import MetricsPipeline: %s", err.Error()))
	}
}

func (r *metricsPipelineResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Modifying MetricsPipeline plan")

	_, err := r.checkAndImportExisting(ctx, &req.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing MetricsPipeline", fmt.Sprintf("Could not import MetricsPipeline: %s", err.Error()))
		return
	}

	resp.Diagnostics.AddWarning(
		"MetricsPipeline is a Singleton",
		"Your plan should never include more than one metrics pipeline resource. If it does, only the latest will take place.\n"+
			"Renaming the resource will show an incorrect plan.",
	)
}

func (r *metricsPipelineResource) checkAndImportExisting(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics) (*models.MetricsPipelineConfigInfo, error) {
	existingConfig, err := r.client.GetMetricsPipeline(ctx)
	if err != nil && err != ErrNotFound {
		return nil, err
	}

	var model metricsPipelineResourceModel
	metricsPipelineConfigToState(ctx, existingConfig, &model, diags)

	diags.Append(state.Set(ctx, &model)...)
	return existingConfig, nil
}

func planToMetricsPipelineRequest(ctx context.Context, plan *metricsPipelineResourceModel, diags *diag.Diagnostics) *models.CreateOrUpdateMetricsPipelineConfigRequest {
	req := &models.CreateOrUpdateMetricsPipelineConfigRequest{}

	if plan.Rules == nil {
		return req
	}

	rules := &models.RelabelConfig{}

	for _, v := range plan.Rules.KeepRegex {
		rules.KeepRegex = append(rules.KeepRegex, v.ValueString())
	}

	for _, v := range plan.Rules.DropRegex {
		rules.DropRegex = append(rules.DropRegex, v.ValueString())
	}

	for _, v := range plan.Rules.RemoveLabel {
		rules.RemoveLabel = append(rules.RemoveLabel, v.ValueString())
	}

	if !plan.Rules.AddLabel.IsNull() && !plan.Rules.AddLabel.IsUnknown() {
		addLabel := make(map[string]string)
		diags.Append(plan.Rules.AddLabel.ElementsAs(ctx, &addLabel, false)...)
		rules.AddLabel = addLabel
	}

	if !plan.Rules.Raw.IsNull() && !plan.Rules.Raw.IsUnknown() {
		rules.Raw = plan.Rules.Raw.ValueString()
	}

	req.Rules = rules
	return req
}

func metricsPipelineConfigToState(ctx context.Context, config *models.MetricsPipelineConfigInfo, state *metricsPipelineResourceModel, diags *diag.Diagnostics) {
	if config == nil {
		state.Rules = nil
		state.UpdatedAt = types.StringValue("")
		return
	}

	state.UpdatedAt = types.StringValue(config.CreatedTimestamp.String())

	if config.Rules == nil {
		state.Rules = nil
		return
	}

	rulesModel := &metricsPipelineRulesModel{}

	for _, v := range config.Rules.KeepRegex {
		rulesModel.KeepRegex = append(rulesModel.KeepRegex, types.StringValue(v))
	}

	for _, v := range config.Rules.DropRegex {
		rulesModel.DropRegex = append(rulesModel.DropRegex, types.StringValue(v))
	}

	for _, v := range config.Rules.RemoveLabel {
		rulesModel.RemoveLabel = append(rulesModel.RemoveLabel, types.StringValue(v))
	}

	if len(config.Rules.AddLabel) > 0 {
		addLabelMap, d := types.MapValueFrom(ctx, types.StringType, config.Rules.AddLabel)
		diags.Append(d...)
		rulesModel.AddLabel = addLabelMap
	} else {
		rulesModel.AddLabel = types.MapNull(types.StringType)
	}

	if config.Rules.Raw != "" {
		rulesModel.Raw = types.StringValue(config.Rules.Raw)
	} else {
		rulesModel.Raw = types.StringNull()
	}

	state.Rules = rulesModel
}
