package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

var (
	_ resource.Resource                = &apiKeyResource{}
	_ resource.ResourceWithConfigure   = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

func NewApiKeyResource() resource.Resource {
	return &apiKeyResource{}
}

type apiKeyResource struct {
	client ApiClient
}

type apiKeyResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	ServiceAccountId types.String `tfsdk:"service_account_id"`
	Description      types.String `tfsdk:"description"`
	ExpirationDate   types.String `tfsdk:"expiration_date"`
	ApiKey           types.String `tfsdk:"api_key"`
	CreatedBy        types.String `tfsdk:"created_by"`
	CreationDate     types.String `tfsdk:"creation_date"`
	LastActive       types.String `tfsdk:"last_active"`
	RevokedAt        types.String `tfsdk:"revoked_at"`
	ExpiredAt        types.String `tfsdk:"expired_at"`
	Policies         types.List   `tfsdk:"policies"` // List of policyMetadataModel
}

var policyMetadataObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"uuid": types.StringType,
		"name": types.StringType,
	},
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apikey"
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "API Key resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the API key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the API key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_account_id": schema.StringAttribute{
				Description: "The ID of the service account associated with the API key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for the API key.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expiration_date": schema.StringAttribute{
				Description: "The expiration date for the API key (RFC3339 format). If not set, the key never expires.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_key": schema.StringAttribute{
				Description: "The generated API key. This value is only available upon creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(), // Keep the value from state if known
				},
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the API key.",
				Computed:    true,
			},
			"creation_date": schema.StringAttribute{
				Description: "The date the API key was created (RFC3339 format).",
				Computed:    true,
			},
			"last_active": schema.StringAttribute{
				Description: "The last time the API key was active (RFC3339 format).",
				Computed:    true,
			},
			"revoked_at": schema.StringAttribute{
				Description: "The date the API key was revoked (RFC3339 format), if applicable.",
				Computed:    true,
			},
			"expired_at": schema.StringAttribute{
				Description: "The date the API key expired (RFC3339 format), based on the 'expiration_date' set.",
				Computed:    true,
			},
			"policies": schema.ListNestedAttribute{
				Description: "Policies associated with the service account linked to this API key.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Description: "Policy UUID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Policy name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan apiKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating API Key: %s for Service Account: %s", plan.Name.ValueString(), plan.ServiceAccountId.ValueString()))

	// Prepare request to SDK
	nameStr := plan.Name.ValueString()
	saIDStr := plan.ServiceAccountId.ValueString()
	descStr := plan.Description.ValueString()

	createReq := &models.CreateAPIKeyRequest{
		Name:             &nameStr,
		ServiceAccountID: &saIDStr,
		Description:      descStr,
	}

	if !plan.ExpirationDate.IsNull() && !plan.ExpirationDate.IsUnknown() {
		expDateStr := plan.ExpirationDate.ValueString()
		expDate, err := time.Parse(time.RFC3339, expDateStr)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Expiration Date Format",
				fmt.Sprintf("Expected RFC3339 format, got: %s. Error: %s", expDateStr, err.Error()),
			)
			return
		}
		expDateTime := strfmt.DateTime(expDate)
		createReq.ExpirationDate = &expDateTime
	}

	// Call SDK via the ApiClient interface
	apiKeyResp, err := r.client.CreateApiKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Key",
			"Could not create API Key: "+err.Error(),
		)
		return
	}

	// Update model with computed values from create response
	plan.Id = types.StringValue(apiKeyResp.ID)
	plan.ApiKey = types.StringValue(apiKeyResp.APIKey)

	tflog.Debug(ctx, fmt.Sprintf("API Key created with ID: %s", apiKeyResp.ID))

	// Read the full state back to populate computed fields
	diags = r.readApiKey(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated model
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated API Key resource: %s", plan.Id.ValueString()))
}

// Read refreshes the Terraform state with the latest data.
func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading API Key resource: %s", state.Id.ValueString()))

	// Read the latest API key data
	diags = r.readApiKey(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		// If read failed, it might mean the resource was deleted outside Terraform
		for _, diag := range diags {
			if diag.Summary() == "API Key Not Found" {
				tflog.Warn(ctx, fmt.Sprintf("API Key %s not found, removing from state.", state.Id.ValueString()))
				resp.State.RemoveResource(ctx)
				return
			}
		}
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully read API Key resource: %s", state.Id.ValueString()))
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// API Keys are immutable
	tflog.Warn(ctx, "Update operation called for apiKeyResource, but API Keys are mostly immutable. Changes should trigger replacement.")
}

// Delete deletes the resource from Terraform state.
func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state apiKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKeyId := state.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Deleting API Key resource: %s", apiKeyId))

	// Call SDK via the ApiClient interface
	err := r.client.DeleteApiKey(ctx, apiKeyId)
	if err != nil {
		// Use the mapped error from handleApiError (via client wrapper)
		resp.Diagnostics.AddError(
			"Error Deleting API Key",
			fmt.Sprintf("Could not delete API Key %s: %s", apiKeyId, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted API Key resource: %s", apiKeyId))
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to read API Key details using ListApiKeys
func (r *apiKeyResource) readApiKey(ctx context.Context, state *apiKeyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	apiKeyId := state.Id.ValueString()

	tflog.Debug(ctx, fmt.Sprintf("Reading API Key details for ID: %s using ListApiKeys", apiKeyId))

	apiKeys, err := r.client.ListApiKeys(ctx, nil, nil)
	if err != nil {
		diags.AddError("Error Listing API Keys", fmt.Sprintf("Could not list API keys: %s", err.Error()))
		return diags
	}

	var foundKey *models.ListAPIKeysResponseItem
	keyFoundInList := false
	for _, key := range apiKeys {
		if key.ID == apiKeyId {
			// Need to copy the key because 'key' is a loop variable
			tempKey := key
			foundKey = tempKey
			keyFoundInList = true
			break
		}
	}

	// If not found in the initial list (active keys), try again including revoked/expired
	if !keyFoundInList {
		tflog.Debug(ctx, fmt.Sprintf("API Key %s not found in active list, retrying with revoked/expired", apiKeyId))
		withRevoked := true
		withExpired := true
		apiKeysFiltered, errFiltered := r.client.ListApiKeys(ctx, &withRevoked, &withExpired)
		if errFiltered != nil {
			// If even the filtered list fails, report that error (could be permission issue)
			diags.AddError("Error Listing Filtered API Keys", fmt.Sprintf("Could not list API keys with filters: %s", errFiltered.Error()))
			return diags
		}
		for _, key := range apiKeysFiltered {
			if key.ID == apiKeyId {
				tempKey := key
				foundKey = tempKey
				break
			}
		}
	}

	if foundKey == nil {
		diags.AddWarning("API Key Not Found", fmt.Sprintf("API Key with ID %s not found during list operation (checked active and filtered).", apiKeyId))
		return diags
	}

	tflog.Debug(ctx, fmt.Sprintf("Found API Key: %s. Populating state.", apiKeyId))

	state.Id = types.StringValue(foundKey.ID)
	state.Name = types.StringValue(foundKey.Name)
	state.ServiceAccountId = types.StringValue(foundKey.ServiceAccountID)
	state.Description = types.StringValue(foundKey.Description)
	state.CreatedBy = types.StringValue(foundKey.CreatedBy)
	state.CreationDate = types.StringValue(foundKey.CreationDate.String())

	if !foundKey.LastActive.IsZero() {
		state.LastActive = types.StringValue(foundKey.LastActive.String())
	} else {
		state.LastActive = types.StringNull()
	}

	if !foundKey.RevokedAt.IsZero() {
		state.RevokedAt = types.StringValue(foundKey.RevokedAt.String())
	} else {
		state.RevokedAt = types.StringNull()
	}

	if !foundKey.ExpiredAt.IsZero() {
		state.ExpiredAt = types.StringValue(foundKey.ExpiredAt.String())
	} else {
		state.ExpiredAt = types.StringNull()
	}

	policies := make([]attr.Value, 0, len(foundKey.Policies))
	for _, p := range foundKey.Policies {
		if p == nil {
			continue
		}
		var policyNameValue types.String
		if p.Name == "" {
			policyNameValue = types.StringNull()
		} else {
			policyNameValue = types.StringValue(p.Name)
		}

		policyAttrs := map[string]attr.Value{
			"uuid": types.StringValue(p.UUID),
			"name": policyNameValue,
		}
		policyObj, policyDiags := types.ObjectValue(policyMetadataObjectType.AttrTypes, policyAttrs)
		diags.Append(policyDiags...)
		if diags.HasError() {
			return diags // Stop processing if object creation failed
		}
		policies = append(policies, policyObj)
	}

	policiesList, listDiags := types.ListValue(policyMetadataObjectType, policies)
	diags.Append(listDiags...)
	if diags.HasError() {
		return diags
	}
	state.Policies = policiesList

	tflog.Debug(ctx, fmt.Sprintf("API Key %s read complete. State populated.", apiKeyId))

	return diags
}
