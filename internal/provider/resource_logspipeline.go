package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
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

// OttlRuleConfigList represents a list of OTTL rules configuration for validation purposes
type OttlRuleConfigList struct {
	OttlRules []OttlRuleConfig `yaml:"ottlRules"`
}

// OttlRuleConfig represents a single OTTL rule configuration for validation purposes
type OttlRuleConfig struct {
	RuleName               string   `yaml:"ruleName"`
	RuleDisabled           bool     `yaml:"ruleDisabled,omitempty"`
	Conditions             []string `yaml:"conditions"`
	ConditionLogicOperator string   `yaml:"conditionLogicOperator"`
	Statements             []string `yaml:"statements"`
	StatementsErrorMode    string   `yaml:"statementsErrorMode"`
}

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

	// Prepare the SDK request using the ConfigEntry fields
	keyStr := plan.Key.ValueString()
	valueStr := plan.Value.ValueString()
	descStr := plan.Description.ValueString()

	// Create a request using available fields (based on JSON serialization)
	reqData := map[string]interface{}{
		"key":         keyStr,
		"value":       valueStr,
		"description": descStr,
	}

	// Convert to SDK model via JSON
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Marshaling Request",
			fmt.Sprintf("Could not marshal request: %s", err.Error()),
		)
		return
	}

	// Unmarshal to SDK type
	var createReq models.CreateOrUpdateConfigRequest
	if err := json.Unmarshal(reqBytes, &createReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Request",
			fmt.Sprintf("Could not create request: %s", err.Error()),
		)
		return
	}

	// Call API client to create the logs pipeline
	createdConfig, err := r.client.CreateLogsPipeline(ctx, &createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating LogsPipeline",
			fmt.Sprintf("Could not create LogsPipeline: %s", err.Error()),
		)
		return
	}

	// Convert response to map to extract fields
	respBytes, err := json.Marshal(createdConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Response",
			fmt.Sprintf("Could not process response: %s", err.Error()),
		)
		return
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(respBytes, &respData); err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Response",
			fmt.Sprintf("Could not process response: %s", err.Error()),
		)
		return
	}

	// Update model with computed values from the response map
	if id, ok := respData["id"].(string); ok && id != "" {
		plan.Id = types.StringValue(id)
	}

	plan.Key = types.StringValue(keyStr)
	plan.Value = types.StringValue(valueStr)
	plan.Description = types.StringValue(descStr)

	if createdAt, ok := respData["createdAt"].(string); ok && createdAt != "" {
		plan.CreatedAt = types.StringValue(createdAt)
	}

	if updatedAt, ok := respData["updatedAt"].(string); ok && updatedAt != "" {
		plan.UpdatedAt = types.StringValue(updatedAt)
	}

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
	configEntry, err := r.client.GetLogsPipeline(ctx)
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

	// Convert response to map to extract fields
	respBytes, err := json.Marshal(configEntry)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Response",
			fmt.Sprintf("Could not process response: %s", err.Error()),
		)
		return
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(respBytes, &respData); err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Response",
			fmt.Sprintf("Could not process response: %s", err.Error()),
		)
		return
	}

	// Update state with values from the response map
	if id, ok := respData["id"].(string); ok && id != "" {
		state.Id = types.StringValue(id)
	}

	if key, ok := respData["key"].(string); ok && key != "" {
		state.Key = types.StringValue(key)
	}

	if value, ok := respData["value"].(string); ok {
		state.Value = types.StringValue(value)
	}

	if description, ok := respData["description"].(string); ok {
		state.Description = types.StringValue(description)
	}

	if createdAt, ok := respData["createdAt"].(string); ok && createdAt != "" {
		state.CreatedAt = types.StringValue(createdAt)
	}

	if updatedAt, ok := respData["updatedAt"].(string); ok && updatedAt != "" {
		state.UpdatedAt = types.StringValue(updatedAt)
	}

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

	// Prepare the SDK request using the ConfigEntry fields
	valueStr := plan.Value.ValueString()
	descStr := plan.Description.ValueString()

	// Create a request using available fields (based on JSON serialization)
	reqData := map[string]interface{}{
		"key":         keyStr,
		"value":       valueStr,
		"description": descStr,
	}

	// Convert to SDK model via JSON
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Marshaling Request",
			fmt.Sprintf("Could not marshal request: %s", err.Error()),
		)
		return
	}

	// Unmarshal to SDK type
	var updateReq models.CreateOrUpdateConfigRequest
	if err := json.Unmarshal(reqBytes, &updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Request",
			fmt.Sprintf("Could not create request: %s", err.Error()),
		)
		return
	}

	// Call API client to update the logs pipeline
	updatedConfig, err := r.client.UpdateLogsPipeline(ctx, &updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating LogsPipeline",
			fmt.Sprintf("Could not update LogsPipeline %s: %s", keyStr, err.Error()),
		)
		return
	}

	// Convert response to map to extract fields
	respBytes, err := json.Marshal(updatedConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Response",
			fmt.Sprintf("Could not process response: %s", err.Error()),
		)
		return
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(respBytes, &respData); err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Response",
			fmt.Sprintf("Could not process response: %s", err.Error()),
		)
		return
	}

	// Update model with computed values from the response map
	if id, ok := respData["id"].(string); ok && id != "" {
		plan.Id = types.StringValue(id)
	}

	if createdAt, ok := respData["createdAt"].(string); ok && createdAt != "" {
		plan.CreatedAt = types.StringValue(createdAt)
	}

	if updatedAt, ok := respData["updatedAt"].(string); ok && updatedAt != "" {
		plan.UpdatedAt = types.StringValue(updatedAt)
	}

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
	err := r.client.DeleteLogsPipeline(ctx)
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
