package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

// Ensure resource implements required interfaces
var (
	_ resource.Resource                = &logsPipelineResource{}
	_ resource.ResourceWithConfigure   = &logsPipelineResource{}
	_ resource.ResourceWithImportState = &logsPipelineResource{}
)

// ottl constants and types
type LogicOperation string
type ErrorMode string

const (
	LogicOperationAnd LogicOperation = "and"
	LogicOperationOr  LogicOperation = "or"

	ErrorModeIgnore    ErrorMode = "ignore"
	ErrorModeSilent    ErrorMode = "silent"
	ErrorModePropagate ErrorMode = "propagate"
)

// OttlRuleConfigList represents a list of OTTL rules configuration
type OttlRuleConfigList struct {
	OttlRules []OttlRuleConfig `json:"ottlRules" yaml:"ottlRules"`
}

// OttlRuleConfig represents a single OTTL rule configuration
type OttlRuleConfig struct {
	RuleName               string         `json:"ruleName" yaml:"ruleName"`
	RuleDisabled           bool           `json:"ruleDisabled,omitempty" yaml:"ruleDisabled,omitempty"`
	Conditions             []string       `json:"conditions" yaml:"conditions"`
	ConditionLogicOperator LogicOperation `json:"conditionLogicOperator" yaml:"conditionLogicOperator"`
	Statements             []string       `json:"statements" yaml:"statements"`
	StatementsErrorMode    ErrorMode      `json:"statementsErrorMode" yaml:"statementsErrorMode"`
}

// Note: Using ConfigEntry from client.go

func NewLogsPipelineResource() resource.Resource {
	return &logsPipelineResource{}
}

type logsPipelineResource struct {
	client ApiClient
}

type logsPipelineResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *logsPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logspipeline"
}

func (r *logsPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Logs Pipeline resource for configuring OTTL transformation rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the logs pipeline.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The key (name) of the logs pipeline configuration.",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "The YAML representation of the OTTL rule configuration list.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description for the logs pipeline configuration.",
				Optional:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the logs pipeline configuration.",
				Computed:    true,
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

	tflog.Debug(ctx, fmt.Sprintf("Creating LogsPipeline: %s", plan.Key.ValueString()))

	// Validate the YAML is parseable as OttlRuleConfigList
	var ottlConfig OttlRuleConfigList
	err := yaml.Unmarshal([]byte(plan.Value.ValueString()), &ottlConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid OTTL Configuration YAML",
			fmt.Sprintf("Unable to parse YAML as OttlRuleConfigList: %s", err.Error()),
		)
		return
	}

	// Prepare the config entry
	configEntry := &ConfigEntry{
		Key:         plan.Key.ValueString(),
		Value:       plan.Value.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Call API client to create the logs pipeline
	createdConfig, err := r.client.CreateLogsPipeline(ctx, configEntry)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating LogsPipeline",
			fmt.Sprintf("Could not create LogsPipeline: %s", err.Error()),
		)
		return
	}

	// Update model with computed values from create response
	plan.Id = types.StringValue(createdConfig.ID)
	plan.CreatedAt = types.StringValue(createdConfig.CreatedAt)
	plan.UpdatedAt = types.StringValue(createdConfig.UpdatedAt)

	tflog.Debug(ctx, fmt.Sprintf("LogsPipeline created with ID: %s", plan.Id.ValueString()))

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated LogsPipeline resource: %s", plan.Id.ValueString()))
}

// Read refreshes the Terraform state with the latest data.
func (r *logsPipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state logsPipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyStr := state.Key.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Reading LogsPipeline resource: %s", keyStr))

	// Call API client to get the logs pipeline
	configEntry, err := r.client.GetLogsPipeline(ctx, keyStr)
	if err != nil {
		if err == ErrNotFound {
			tflog.Warn(ctx, fmt.Sprintf("LogsPipeline %s not found, removing from state.", keyStr))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading LogsPipeline",
			fmt.Sprintf("Could not read LogsPipeline %s: %s", keyStr, err.Error()),
		)
		return
	}

	// Update state with values from response
	state.Id = types.StringValue(configEntry.ID)
	state.Key = types.StringValue(configEntry.Key)
	state.Value = types.StringValue(configEntry.Value)
	state.Description = types.StringValue(configEntry.Description)
	state.CreatedAt = types.StringValue(configEntry.CreatedAt)
	state.UpdatedAt = types.StringValue(configEntry.UpdatedAt)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully read LogsPipeline resource: %s", keyStr))
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

	keyStr := plan.Key.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Updating LogsPipeline: %s", keyStr))

	// Validate the YAML is parseable as OttlRuleConfigList
	var ottlConfig OttlRuleConfigList
	err := yaml.Unmarshal([]byte(plan.Value.ValueString()), &ottlConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid OTTL Configuration YAML",
			fmt.Sprintf("Unable to parse YAML as OttlRuleConfigList: %s", err.Error()),
		)
		return
	}

	// Prepare the config entry
	configEntry := &ConfigEntry{
		Key:         keyStr,
		Value:       plan.Value.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Call API client to update the logs pipeline
	updatedConfig, err := r.client.UpdateLogsPipeline(ctx, configEntry)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating LogsPipeline",
			fmt.Sprintf("Could not update LogsPipeline %s: %s", keyStr, err.Error()),
		)
		return
	}

	// Update model with computed values
	plan.Id = types.StringValue(updatedConfig.ID)
	plan.CreatedAt = types.StringValue(updatedConfig.CreatedAt)
	plan.UpdatedAt = types.StringValue(updatedConfig.UpdatedAt)

	tflog.Debug(ctx, fmt.Sprintf("LogsPipeline updated: %s", keyStr))

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully updated LogsPipeline resource: %s", keyStr))
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

	keyStr := state.Key.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Deleting LogsPipeline resource: %s", keyStr))

	// Call API client to delete the logs pipeline
	err := r.client.DeleteLogsPipeline(ctx, keyStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting LogsPipeline",
			fmt.Sprintf("Could not delete LogsPipeline %s: %s", keyStr, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted LogsPipeline resource: %s", keyStr))
}

func (r *logsPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}
