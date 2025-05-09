package provider

import (
	"context"
	"fmt"

	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

type kvPair struct {
	key   *yaml.Node
	value *yaml.Node
}

var _ resource.Resource = &monitorResource{}
var _ resource.ResourceWithImportState = &monitorResource{}
var _ resource.ResourceWithConfigure = &monitorResource{}
var _ resource.ResourceWithModifyPlan = &monitorResource{}

func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

type monitorResource struct {
	client ApiClient
}

type monitorResourceModel struct {
	Id          types.String `tfsdk:"id"`
	MonitorYaml types.String `tfsdk:"monitor_yaml"`
}

func (r *monitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *monitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Groundcover Monitor resource managed via raw YAML.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Monitor identifier (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"monitor_yaml": schema.StringAttribute{
				MarkdownDescription: "The monitor definition in YAML format.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{},
			},
		},
	}
}

func (r *monitorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "monitor resource configured successfully")
}

// normalizeMonitorYaml sorts keys in a YAML string alphabetically.
// It also handles potential errors during parsing and marshalling.
func normalizeMonitorYaml(ctx context.Context, yamlString string) (string, error) {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlString), &node)
	if err != nil {
		tflog.Error(ctx, "Failed to unmarshal YAML", map[string]interface{}{"error": err, "yaml": yamlString})
		return "", fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	if len(node.Content) == 1 {
		sortYamlNodeRecursively(node.Content[0])
	} else {
		// Handle cases where YAML might not be a single document or has unexpected structure
		tflog.Warn(ctx, "YAML content is not a single document node, attempting to sort directly if it's a map or sequence.")
		sortYamlNodeRecursively(&node)
	}

	out, err := yaml.Marshal(&node)
	if err != nil {
		tflog.Error(ctx, "Failed to marshal YAML", map[string]interface{}{"error": err})
		return "", fmt.Errorf("failed to marshal sorted YAML: %w", err)
	}

	return string(out), nil
}

// sortYamlNodeRecursively sorts map keys within a yaml.Node.
// It traverses the YAML structure and applies sorting to all mapping nodes.
func sortYamlNodeRecursively(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			// This shouldn't happen for valid YAML maps
			return
		}
		pairs := make([]kvPair, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			pairs[i/2] = kvPair{key: node.Content[i], value: node.Content[i+1]}
		}

		// Sort pairs by key's string value
		customSortPairs(pairs)

		// Rebuild node.Content from sorted pairs
		newContent := make([]*yaml.Node, 0, len(node.Content))
		for _, p := range pairs {
			newContent = append(newContent, p.key, p.value)
		}
		node.Content = newContent

		for i := 1; i < len(node.Content); i += 2 {
			sortYamlNodeRecursively(node.Content[i])
		}

	case yaml.SequenceNode:
		for _, elem := range node.Content {
			sortYamlNodeRecursively(elem)
		}
	case yaml.DocumentNode:
		for _, elem := range node.Content {
			sortYamlNodeRecursively(elem)
		}
	}
}

func customSortPairs(pairs []kvPair) {
	n := len(pairs)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if pairs[j].key.Kind == yaml.ScalarNode && pairs[j+1].key.Kind == yaml.ScalarNode {
				if pairs[j].key.Value > pairs[j+1].key.Value {
					pairs[j], pairs[j+1] = pairs[j+1], pairs[j]
				}
			}
		}
	}
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data monitorResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating monitor resource from YAML")

	userInputMonitorYaml := data.MonitorYaml.ValueString() // Get user's original input
	normalizedApiYaml, err := normalizeMonitorYaml(ctx, userInputMonitorYaml)
	if err != nil {
		resp.Diagnostics.AddError("YAML Normalization Error", fmt.Sprintf("Unable to normalize monitor YAML during Create: %s", err))
		return
	}

	monitorYamlBytesForApi := []byte(normalizedApiYaml)

	// Use the SDK's CreateMonitorYaml function via the ApiClient interface
	createResp, err := r.client.CreateMonitorYaml(ctx, monitorYamlBytesForApi)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create monitor using YAML, got error: %s", err))
		return
	}

	// Use the correct field MonitorID
	if createResp == nil || createResp.MonitorID == "" {
		resp.Diagnostics.AddError("API Error", "Monitor creation response did not contain a MonitorID")
		return
	}

	// Set the ID from the response using MonitorID
	data.Id = types.StringValue(createResp.MonitorID)
	// data.MonitorYaml is already set from req.Plan.Get, which is the user's original input.
	// We ensure it remains the user's original input.
	data.MonitorYaml = types.StringValue(userInputMonitorYaml) // Explicitly set to user's original input for clarity

	tflog.Trace(ctx, "Created monitor resource from YAML", map[string]interface{}{"id": data.Id.ValueString()})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data monitorResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := data.Id.ValueString()
	tflog.Debug(ctx, "Reading monitor resource YAML", map[string]interface{}{"id": monitorId})

	_, err := r.client.GetMonitor(ctx, monitorId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Monitor %s not found (handled via ErrNotFound), removing from state", monitorId))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read monitor %s YAML, got error: %s", monitorId, err))
		return
	}

	tflog.Trace(ctx, "Read monitor resource YAML (confirmed existence)", map[string]interface{}{"id": monitorId})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorResourceModel
	var state monitorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := state.Id.ValueString()
	tflog.Debug(ctx, "Updating monitor resource from YAML", map[string]interface{}{"id": monitorId})

	userInputMonitorYaml := plan.MonitorYaml.ValueString()
	normalizedApiYaml, err := normalizeMonitorYaml(ctx, userInputMonitorYaml)
	if err != nil {
		resp.Diagnostics.AddError("YAML Normalization Error", fmt.Sprintf("Unable to normalize monitor YAML during Update for monitor %s: %s", monitorId, err))
		return
	}

	monitorYamlBytesForApi := []byte(normalizedApiYaml)

	_, err = r.client.UpdateMonitorYaml(ctx, monitorId, monitorYamlBytesForApi)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update monitor %s using YAML, got error: %s", monitorId, err))
		return
	}

	tflog.Trace(ctx, "Updated monitor resource from YAML", map[string]interface{}{"id": monitorId})

	updatedState := plan
	updatedState.Id = state.Id
	updatedState.MonitorYaml = types.StringValue(userInputMonitorYaml)

	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data monitorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := data.Id.ValueString()
	tflog.Debug(ctx, "Deleting monitor resource", map[string]interface{}{"id": monitorId})

	err := r.client.DeleteMonitor(ctx, monitorId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("DeleteMonitor returned ErrNotFound for %s, which should have been handled by the wrapper. Removing from state anyway.", monitorId))
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete monitor %s, got error: %s", monitorId, err))
			return // Keep resource in state if delete fails unexpectedly
		}
	}

	tflog.Trace(ctx, "Deleted monitor resource", map[string]interface{}{"id": monitorId})
}

func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan is used to compare the normalized versions of the monitor_yaml
// and adjust the plan if they are semantically equivalent to avoid unnecessary diffs.
func (r *monitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "ModifyPlan: Skipping custom YAML diff for new or destroyed resource.")
		return
	}

	var plannedYaml types.String
	diags := req.Plan.GetAttribute(ctx, path.Root("monitor_yaml"), &plannedYaml)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateYaml types.String
	diags = req.State.GetAttribute(ctx, path.Root("monitor_yaml"), &stateYaml)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If either YAML is null or unknown at this stage (shouldn't happen for updates if properly configured),
	// skip custom logic.
	if plannedYaml.IsNull() || plannedYaml.IsUnknown() || stateYaml.IsNull() || stateYaml.IsUnknown() {
		tflog.Debug(ctx, "ModifyPlan: Planned or State YAML is null/unknown, skipping custom diff.")
		return
	}

	plannedYamlString := plannedYaml.ValueString()
	stateYamlString := stateYaml.ValueString()

	if plannedYamlString == stateYamlString {
		tflog.Debug(ctx, "ModifyPlan: Raw YAML strings are identical.")
		return
	}

	normalizedPlannedYaml, err := normalizeMonitorYaml(ctx, plannedYamlString)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("monitor_yaml"),
			"Plan YAML Normalization Error",
			fmt.Sprintf("Failed to normalize planned monitor_yaml: %s. Input: %s", err, plannedYamlString),
		)
		return
	}

	normalizedStateYaml, err := normalizeMonitorYaml(ctx, stateYamlString)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("monitor_yaml"),
			"State YAML Normalization Error",
			fmt.Sprintf("Failed to normalize state monitor_yaml: %s. Input: %s", err, stateYamlString),
		)
		return
	}

	// If the normalized versions are the same, there's no semantic diff.
	// In this case, we tell Terraform that the planned value for monitor_yaml
	// should be considered the same as the state value to prevent a diff.
	if normalizedPlannedYaml == normalizedStateYaml {
		tflog.Info(ctx, "ModifyPlan: Normalized YAMLs are identical. Setting plan's monitor_yaml to state's monitor_yaml to suppress diff.")
		diags := resp.Plan.SetAttribute(ctx, path.Root("monitor_yaml"), stateYaml)
		resp.Diagnostics.Append(diags...)
	} else {
		tflog.Info(ctx, "ModifyPlan: Normalized YAMLs differ. Plan will proceed with update for monitor_yaml.")
	}
}
