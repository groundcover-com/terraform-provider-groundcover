package provider

import (
	"context"
	"fmt"

	// Required for HTTP status codes
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &monitorResource{}
var _ resource.ResourceWithImportState = &monitorResource{}
var _ resource.ResourceWithConfigure = &monitorResource{}

func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

// monitorResource defines the resource implementation.
type monitorResource struct {
	client ApiClient
}

// monitorResourceModel describes the resource data model.
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data monitorResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating monitor resource from YAML")

	monitorYamlBytes := []byte(data.MonitorYaml.ValueString())

	// Use the SDK's CreateMonitorYaml function via the ApiClient interface
	createResp, err := r.client.CreateMonitorYaml(ctx, monitorYamlBytes)
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

	// Use the SDK's GetMonitor function via the ApiClient interface
	monitorYamlBytes, err := r.client.GetMonitor(ctx, monitorId)
	if err != nil {
		// Use errors.Is to check for the wrapped ErrNotFound from handleApiError
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Monitor %s not found (handled via ErrNotFound), removing from state", monitorId))
			resp.State.RemoveResource(ctx)
			return
		}
		// Handle other errors
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read monitor %s YAML, got error: %s", monitorId, err))
		return
	}

	// Update the model with the fetched YAML
	data.MonitorYaml = types.StringValue(string(monitorYamlBytes))

	tflog.Trace(ctx, "Read monitor resource YAML", map[string]interface{}{"id": monitorId})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorResourceModel
	var state monitorResourceModel

	// Read Terraform plan and state data into the models
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := state.Id.ValueString() // ID comes from prior state
	tflog.Debug(ctx, "Updating monitor resource from YAML", map[string]interface{}{"id": monitorId})

	monitorYamlBytes := []byte(plan.MonitorYaml.ValueString())

	// Use the SDK's UpdateMonitorYaml function via the ApiClient interface
	_, err := r.client.UpdateMonitorYaml(ctx, monitorId, monitorYamlBytes)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update monitor %s using YAML, got error: %s", monitorId, err))
		return
	}

	tflog.Trace(ctx, "Updated monitor resource from YAML", map[string]interface{}{"id": monitorId})

	// Update the state with the planned YAML (reflecting the user's intent)
	plan.Id = state.Id // Ensure ID is carried over from state to the updated state model
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data monitorResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	monitorId := data.Id.ValueString()
	tflog.Debug(ctx, "Deleting monitor resource", map[string]interface{}{"id": monitorId})

	// Use the SDK's DeleteMonitor function via the ApiClient interface
	err := r.client.DeleteMonitor(ctx, monitorId)
	if err != nil {
		// We check if the *wrapped* error is ErrNotFound. DeleteMonitor itself returns nil on success (including 404).
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("DeleteMonitor returned ErrNotFound for %s, which should have been handled by the wrapper. Removing from state anyway.", monitorId))
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete monitor %s, got error: %s", monitorId, err))
			return // Keep resource in state if delete fails unexpectedly
		}
	}

	tflog.Trace(ctx, "Deleted monitor resource", map[string]interface{}{"id": monitorId})
	// State removal is handled by the framework implicitly upon successful completion.
}

func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
