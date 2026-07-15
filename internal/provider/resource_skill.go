// Copyright groundcover 2026
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &skillResource{}
var _ resource.ResourceWithConfigure = &skillResource{}
var _ resource.ResourceWithImportState = &skillResource{}

func NewSkillResource() resource.Resource { return &skillResource{} }

type skillResource struct{ client ApiClient }

type skillResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	WhenToUse        types.String `tfsdk:"when_to_use"`
	Description      types.String `tfsdk:"description"`
	Instructions     types.String `tfsdk:"instructions"`
	Identifier       types.String `tfsdk:"identifier"`
	Revision         types.Int64  `tfsdk:"revision"`
	IsOrganizational types.Bool   `tfsdk:"is_organizational"`
	IsProvisioned    types.Bool   `tfsdk:"is_provisioned"`
	CreatedAt        types.String `tfsdk:"created_at"`
	CreatedBy        types.String `tfsdk:"created_by"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	UpdatedBy        types.String `tfsdk:"updated_by"`
}

func (r *skillResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

func (r *skillResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an organizational groundcover Agent Skill. Managing organizational Skills requires an admin service account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{MarkdownDescription: "Skill UUID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name": schema.StringAttribute{
				MarkdownDescription: "Skill name. Organizational skill names are unique case-insensitively.", Required: true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 255), trimmedStringValidator{}},
			},
			"when_to_use": schema.StringAttribute{
				MarkdownDescription: "Guidance that tells the Agent when to use the Skill.", Required: true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 5000), trimmedStringValidator{}},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional human-readable description of the Skill.", Optional: true, Computed: true,
				Default: stringdefault.StaticString(""), Validators: []validator.String{stringvalidator.LengthAtMost(5000), trimmedStringValidator{allowEmpty: true}},
			},
			"instructions": schema.StringAttribute{
				MarkdownDescription: "Instructions the Agent follows when it uses the Skill.", Required: true,
				Validators: []validator.String{nonWhitespaceStringValidator{}},
			},
			"identifier":        schema.StringAttribute{MarkdownDescription: "Stable display identifier returned by the API.", Computed: true},
			"revision":          schema.Int64Attribute{MarkdownDescription: "Current Skill revision.", Computed: true},
			"is_organizational": schema.BoolAttribute{MarkdownDescription: "Whether the Skill is available to the organization. Terraform-managed Skills are always organizational.", Computed: true},
			"is_provisioned":    schema.BoolAttribute{MarkdownDescription: "Whether the Skill is managed by an external provisioner such as Terraform.", Computed: true},
			"created_at":        schema.StringAttribute{MarkdownDescription: "Creation timestamp returned by the API.", Computed: true},
			"created_by":        schema.StringAttribute{MarkdownDescription: "Creator identifier returned by the API.", Computed: true},
			"updated_at":        schema.StringAttribute{MarkdownDescription: "Last update timestamp returned by the API.", Computed: true},
			"updated_by":        schema.StringAttribute{MarkdownDescription: "Last updater identifier returned by the API.", Computed: true},
		},
	}
}

func (r *skillResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(ApiClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected provider.ApiClient, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *skillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan skillResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	detail, err := r.client.CreateSkill(ctx, skillRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Skill", fmt.Sprintf("Failed to create Skill %q: %s", plan.Name.ValueString(), err))
		return
	}
	state, mapDiags := skillModelFromAPI(detail)
	resp.Diagnostics.Append(mapDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *skillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state skillResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	detail, err := r.client.GetSkill(ctx, state.ID.ValueString())
	if errors.Is(err, ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Skill", fmt.Sprintf("Failed to read Skill %q: %s", state.ID.ValueString(), err))
		return
	}
	state, mapDiags := skillModelFromAPI(detail)
	resp.Diagnostics.Append(mapDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *skillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan skillResourceModel
	var state skillResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	detail, err := r.client.UpdateSkill(ctx, state.ID.ValueString(), skillRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Skill", fmt.Sprintf("Failed to update Skill %q: %s", state.ID.ValueString(), err))
		return
	}
	updated, mapDiags := skillModelFromAPI(detail)
	resp.Diagnostics.Append(mapDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *skillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state skillResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSkill(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Skill", fmt.Sprintf("Failed to delete Skill %q: %s", state.ID.ValueString(), err))
	}
}

func (r *skillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func skillRequestFromModel(model skillResourceModel) *models.AgentSkillRequest {
	name, whenToUse, instructions := model.Name.ValueString(), model.WhenToUse.ValueString(), model.Instructions.ValueString()
	isOrganizational := true
	return &models.AgentSkillRequest{
		Name: &name, WhenToUse: &whenToUse, Description: model.Description.ValueString(), Instructions: &instructions,
		IsOrganizational: &isOrganizational,
	}
}

func skillModelFromAPI(detail *models.AgentSkillDetail) (skillResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if detail == nil {
		diags.AddError("Invalid Skill API Response", "Skill payload was nil")
		return skillResourceModel{}, diags
	}
	missing := make([]string, 0)
	if detail.ID == nil {
		missing = append(missing, "id")
	}
	if detail.Name == nil {
		missing = append(missing, "name")
	}
	if detail.WhenToUse == nil {
		missing = append(missing, "when_to_use")
	}
	if detail.Instructions == nil {
		missing = append(missing, "instructions")
	}
	if detail.Revision == nil {
		missing = append(missing, "revision")
	}
	if detail.IsOrganizational == nil {
		missing = append(missing, "is_organizational")
	}
	if detail.IsProvisioned == nil {
		missing = append(missing, "is_provisioned")
	}
	if detail.CreatedAt == nil {
		missing = append(missing, "created_at")
	}
	if detail.UpdatedAt == nil {
		missing = append(missing, "updated_at")
	}
	if len(missing) > 0 {
		diags.AddError("Invalid Skill API Response", "Skill payload is missing required fields: "+strings.Join(missing, ", "))
		return skillResourceModel{}, diags
	}
	return skillResourceModel{
		ID: types.StringValue(*detail.ID), Name: types.StringValue(*detail.Name), WhenToUse: types.StringValue(*detail.WhenToUse),
		Description: types.StringValue(detail.Description), Instructions: types.StringValue(*detail.Instructions),
		Identifier: types.StringValue(detail.Identifier), Revision: types.Int64Value(*detail.Revision),
		IsOrganizational: types.BoolValue(*detail.IsOrganizational), IsProvisioned: types.BoolValue(*detail.IsProvisioned),
		CreatedAt: types.StringValue(*detail.CreatedAt), CreatedBy: types.StringValue(detail.CreatedBy),
		UpdatedAt: types.StringValue(*detail.UpdatedAt), UpdatedBy: types.StringValue(detail.UpdatedBy),
	}, diags
}

type nonWhitespaceStringValidator struct{}

func (nonWhitespaceStringValidator) Description(context.Context) string {
	return "value must contain at least one non-whitespace character"
}
func (v nonWhitespaceStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (nonWhitespaceStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if strings.TrimSpace(req.ConfigValue.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Empty Value", "Value must contain at least one non-whitespace character.")
	}
}

type trimmedStringValidator struct{ allowEmpty bool }

func (trimmedStringValidator) Description(context.Context) string {
	return "value must not have leading or trailing whitespace"
}
func (v trimmedStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v trimmedStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if v.allowEmpty && value == "" {
		return
	}
	if value == "" || strings.TrimSpace(value) != value {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Whitespace", "Value must be non-empty and must not have leading or trailing whitespace.")
	}
}
