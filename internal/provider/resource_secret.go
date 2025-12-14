// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	models "github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &secretResource{}
var _ resource.ResourceWithConfigure = &secretResource{}
var _ resource.ResourceWithImportState = &secretResource{}

func NewSecretResource() resource.Resource {
	return &secretResource{}
}

type secretResource struct {
	client ApiClient
}

type secretResourceModel struct {
	ID      types.String `tfsdk:"id"`      // Secret reference ID (computed) - use this in other resources
	Name    types.String `tfsdk:"name"`    // Secret Name (required)
	Type    types.String `tfsdk:"type"`    // Secret Type: api_key, password, basic_auth (required)
	Content types.String `tfsdk:"content"` // Secret Content (required, sensitive, write-only)
}

func (r *secretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *secretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a groundcover Secret.

Secrets allow you to securely store sensitive values (like API keys, passwords, or credentials) and receive a reference ID that can be used in other resources (such as data integrations) as a placeholder instead of the actual secret value.

**Note:** The secret content is write-only and will not be returned by the API after creation. The content is stored in the Terraform state (encrypted if using a remote backend with encryption).`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique reference ID for the secret. Use this ID in other resources (e.g., data integrations) as a placeholder for the secret value.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the secret.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the secret. Valid values are: `api_key`, `password`, `basic_auth`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("api_key", "password", "basic_auth"),
				},
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The secret content/value. This is write-only and will not be returned by the API.",
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
		},
	}
}

func (r *secretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "Secret resource configured successfully")
}

// --- CRUD Operations ---

func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameVal := plan.Name.ValueString()
	typeVal := plan.Type.ValueString()
	contentVal := plan.Content.ValueString()

	apiRequest := &models.CreateSecretRequest{
		Name:              &nameVal,
		Type:              &typeVal,
		Content:           &contentVal,
		ManagedByProvider: models.CreateSecretRequestManagedByProviderTerraform,
	}

	tflog.Debug(ctx, "CreateSecret SDK Call Request constructed", map[string]any{"name": nameVal, "type": typeVal})
	apiResponse, err := r.client.CreateSecret(ctx, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Create Secret Error", fmt.Sprintf("Failed to create secret '%s': %s", nameVal, err.Error()))
		return
	}

	if apiResponse == nil || apiResponse.ID == "" {
		resp.Diagnostics.AddError("SDK Client Create Secret Error", "Create response missing secret ID")
		return
	}

	tflog.Info(ctx, "Secret created successfully via SDK", map[string]any{"id": apiResponse.ID})

	// Map response back to state
	plan.ID = types.StringValue(apiResponse.ID)
	// Name, Type, Content are preserved from the plan

	tflog.Info(ctx, "Saving new secret to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretID := state.ID.ValueString()
	tflog.Debug(ctx, "Reading Secret info", map[string]any{"id": secretID})

	// Note: There is no Get/List endpoint for secrets in the API.
	// The secret content is write-only and cannot be retrieved.
	// We preserve the state as-is since we cannot verify the secret still exists
	// without a read endpoint. The secret will be validated on next update/delete.
	tflog.Info(ctx, "Secret read completed (no API verification available - preserving state)", map[string]any{"id": secretID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan secretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state secretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretID := state.ID.ValueString()
	tflog.Debug(ctx, "Updating Secret", map[string]any{"id": secretID})

	nameVal := plan.Name.ValueString()
	typeVal := plan.Type.ValueString()
	contentVal := plan.Content.ValueString()

	apiRequest := &models.UpdateSecretRequest{
		Name:              &nameVal,
		Type:              &typeVal,
		Content:           &contentVal,
		ManagedByProvider: models.UpdateSecretRequestManagedByProviderTerraform,
	}

	tflog.Debug(ctx, "UpdateSecret SDK Call Request constructed", map[string]any{"id": secretID})
	apiResponse, err := r.client.UpdateSecret(ctx, secretID, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Update Secret Error", fmt.Sprintf("Failed to update secret ID %s: %s", secretID, err.Error()))
		return
	}

	tflog.Info(ctx, "Secret updated successfully via SDK", map[string]any{"id": secretID})

	// Update state with response data
	if apiResponse != nil && apiResponse.ID != "" {
		plan.ID = types.StringValue(apiResponse.ID)
	} else {
		plan.ID = state.ID // Preserve existing ID
	}

	tflog.Info(ctx, "Saving updated secret to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretID := state.ID.ValueString()
	tflog.Debug(ctx, "Deleting Secret", map[string]any{"id": secretID})

	err := r.client.DeleteSecret(ctx, secretID)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Delete Secret Error", fmt.Sprintf("Failed to delete secret %s: %s", secretID, err.Error()))
		return
	}

	tflog.Info(ctx, "Secret deleted successfully via SDK", map[string]any{"id": secretID})
	// Terraform automatically removes the resource from state when Delete returns no error.
}

func (r *secretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ID - but note that content cannot be imported since it's not retrievable
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Warn user that content must be provided after import
	tflog.Warn(ctx, "Secret imported by ID. The 'content' attribute must be set in your configuration as it cannot be retrieved from the API.", map[string]any{"id": req.ID})
}
