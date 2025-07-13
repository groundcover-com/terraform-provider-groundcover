// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	// SDK Imports
	models "github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &serviceAccountResource{}
var _ resource.ResourceWithConfigure = &serviceAccountResource{}
var _ resource.ResourceWithImportState = &serviceAccountResource{}

func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

type serviceAccountResource struct {
	client ApiClient
}

type serviceAccountResourceModel struct {
	ID          types.String `tfsdk:"id"`           // Service Account ID (computed)
	Name        types.String `tfsdk:"name"`         // Service Account Name (required)
	Email       types.String `tfsdk:"email"`        // Service Account Email (required)
	PolicyUUIDs types.List   `tfsdk:"policy_uuids"` // List of Policy UUIDs (required)
	Description types.String `tfsdk:"description"`  // Optional description
}

func (r *serviceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serviceaccount"
}

func (r *serviceAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Groundcover Service Account.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the service account.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(), // Keep ID persistent across updates
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the service account.",
				Required:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email associated with the service account.",
				Required:            true,
			},
			"policy_uuids": schema.ListAttribute{
				MarkdownDescription: "List of policy UUIDs to assign to the service account.",
				ElementType:         types.StringType,
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "An optional description for the service account.",
				Optional:            true,
			},
			// Secret attribute removed
		},
	}
}

func (r *serviceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "ServiceAccount resource configured successfully")
}

// --- CRUD Operations ---

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map plan to SDK request
	var policyUUIDs []string
	diags := plan.PolicyUUIDs.ElementsAs(ctx, &policyUUIDs, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameVal := plan.Name.ValueString()
	emailVal := plan.Email.ValueString()

	apiRequest := &models.CreateServiceAccountRequest{
		Name:        &nameVal,
		Email:       &emailVal,
		PolicyUUIDs: policyUUIDs,
	}

	tflog.Debug(ctx, "CreateServiceAccount SDK Call Request constructed", map[string]any{"name": plan.Name.ValueString(), "email": plan.Email.ValueString()})
	apiResponse, err := r.client.CreateServiceAccount(ctx, apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Create Service Account Error", fmt.Sprintf("Failed to create service account '%s': %s", plan.Name.ValueString(), err.Error()))
		return
	}
	// Use ServiceAccountID (string) from response
	saGeneratedIdPtr := apiResponse.ServiceAccountID
	tflog.Info(ctx, "Service Account created successfully via SDK", map[string]any{"id_ptr": saGeneratedIdPtr})

	// Map response back to state
	if saGeneratedIdPtr == nil {
		resp.Diagnostics.AddError("SDK Client Create Service Account Error", "Create response missing ServiceAccountID")
		return
	}
	plan.ID = types.StringValue(*saGeneratedIdPtr)
	// Persist Name, Email, PolicyUUIDs, Description from the plan as they were the desired state

	tflog.Info(ctx, "Saving new service account to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	saID := state.ID.ValueString()
	tflog.Debug(ctx, "Reading Service Account info", map[string]any{"id": saID})

	// Use List endpoint as there's no direct Get endpoint
	apiResponseList, err := r.client.ListServiceAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client List Service Accounts Error", fmt.Sprintf("Failed to list service accounts to find ID %s: %s", saID, err.Error()))
		return
	}

	var foundSA *models.ServiceAccountsWithPolicy
	for _, item := range apiResponseList {
		if item != nil && item.ServiceAccountID != "" && item.ServiceAccountID == saID {
			foundSA = item
			break
		}
	}

	if foundSA == nil {
		tflog.Warn(ctx, "Service Account not found via SDK List, removing from state", map[string]any{"id": saID})
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Debug(ctx, "Service Account found via SDK List", map[string]any{"id": saID})

	// Update state from the found service account info
	state.ID = types.StringValue(foundSA.ServiceAccountID)
	state.Name = types.StringValue(foundSA.Name)
	state.Email = types.StringValue(foundSA.Email)

	// Extract policy UUIDs from the service account policies
	var policyUUIDs []attr.Value
	for _, policy := range foundSA.Policies {
		if policy.UUID != "" {
			policyUUIDs = append(policyUUIDs, types.StringValue(policy.UUID))
		}
	}
	policyList, diags := types.ListValue(types.StringType, policyUUIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.PolicyUUIDs = policyList

	tflog.Info(ctx, "Saving updated service account to state", map[string]any{"id": state.ID.ValueString(), "policies_count": len(policyUUIDs)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	saID := state.ID.ValueString()
	tflog.Debug(ctx, "Updating Service Account", map[string]any{"id": saID})

	var policyUUIDs []string
	diags := plan.PolicyUUIDs.ElementsAs(ctx, &policyUUIDs, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	saIDForRequest := saID
	emailVal := plan.Email.ValueString()

	apiRequest := models.UpdateServiceAccountRequest{
		ServiceAccountID: &saIDForRequest,
		Email:            emailVal,
		PolicyUUIDs:      policyUUIDs,
	}

	tflog.Debug(ctx, "UpdateServiceAccount SDK Call Request constructed", map[string]any{"id": saID})
	_, err := r.client.UpdateServiceAccount(ctx, saID, &apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("SDK Client Update Service Account Error", fmt.Sprintf("Failed to update service account ID %s: %s", saID, err.Error()))
		return
	}

	tflog.Info(ctx, "Service Account updated successfully via SDK", map[string]any{"id": saID})

	plan.ID = types.StringValue(saID)

	tflog.Info(ctx, "Saving updated service account to state", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	saID := state.ID.ValueString()
	tflog.Debug(ctx, "Deleting Service Account", map[string]any{"id": saID})

	err := r.client.DeleteServiceAccount(ctx, saID)
	if err != nil {
		// Delete returns nil if ErrNotFound is handled by the client wrapper
		resp.Diagnostics.AddError("SDK Client Delete Service Account Error", fmt.Sprintf("Failed to delete service account %s: %s", saID, err.Error()))
		return // Keep in state if delete fails unexpectedly
	}

	tflog.Info(ctx, "Service Account deleted successfully via SDK (or was already gone)", map[string]any{"id": saID})
	// Terraform automatically removes the resource from state when Delete returns no error.
}

func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
