// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	_ resource.Resource                = &connectedAppResource{}
	_ resource.ResourceWithConfigure   = &connectedAppResource{}
	_ resource.ResourceWithImportState = &connectedAppResource{}
)

func NewConnectedAppResource() resource.Resource {
	return &connectedAppResource{}
}

type connectedAppResource struct {
	client ApiClient
}

type connectedAppResourceModel struct {
	Id        types.String  `tfsdk:"id"`
	Name      types.String  `tfsdk:"name"`
	Type      types.String  `tfsdk:"type"`
	Data      types.Dynamic `tfsdk:"data"`
	CreatedBy types.String  `tfsdk:"created_by"`
	CreatedAt types.String  `tfsdk:"created_at"`
	UpdatedBy types.String  `tfsdk:"updated_by"`
	UpdatedAt types.String  `tfsdk:"updated_at"`
}

func (r *connectedAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app"
}

func (r *connectedAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Connected App resource for managing integrations with external services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the connected app.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the connected app.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of connected app (slack-webhook or pagerduty).",
				Required:    true,
			},
			"data": schema.DynamicAttribute{
				Description: "Type-specific configuration. Supports nested structures. For slack-webhook: {url = \"https://...\"}. For pagerduty: {routing_key = \"...\", severity_mapping = {critical = \"P1\", ...}}.",
				Required:    true,
				Sensitive:   true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the connected app.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date the connected app was created (RFC3339 format).",
				Computed:    true,
			},
			"updated_by": schema.StringAttribute{
				Description: "The user who last updated the connected app.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The date the connected app was last updated (RFC3339 format).",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *connectedAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *connectedAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan connectedAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating connected app: %s", plan.Name.ValueString()))

	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()

	dataAny, diags := dynamicValueToMap(ctx, plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &models.CreateConnectedAppRequest{
		Name: &nameStr,
		Type: &typeStr,
		Data: dataAny,
	}

	createResp, err := r.client.CreateConnectedApp(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating connected app", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Connected app created with ID: %s", createResp.ID))

	connectedApp, err := r.client.GetConnectedApp(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created connected app", err.Error())
		return
	}

	mapConnectedAppResponseToModel(ctx, connectedApp, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully created and populated connected app resource: %s", plan.Id.ValueString()))
}

// Read refreshes the Terraform state with the latest data.
func (r *connectedAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state connectedAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading connected app resource: %s", state.Id.ValueString()))

	connectedApp, err := r.client.GetConnectedApp(ctx, state.Id.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Connected app %s not found, removing from state.", state.Id.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading connected app", err.Error())
		return
	}

	mapConnectedAppResponseToModel(ctx, connectedApp, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully read connected app resource: %s", state.Id.ValueString()))
}

// Update updates the resource.
func (r *connectedAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan connectedAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Updating connected app: %s", plan.Id.ValueString()))

	nameStr := plan.Name.ValueString()
	typeStr := plan.Type.ValueString()

	dataAny, diags := dynamicValueToMap(ctx, plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &models.UpdateConnectedAppRequest{
		Name: &nameStr,
		Type: &typeStr,
		Data: dataAny,
	}

	err := r.client.UpdateConnectedApp(ctx, plan.Id.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating connected app", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Connected app updated: %s", plan.Id.ValueString()))

	connectedApp, err := r.client.GetConnectedApp(ctx, plan.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated connected app", err.Error())
		return
	}

	mapConnectedAppResponseToModel(ctx, connectedApp, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully updated connected app resource: %s", plan.Id.ValueString()))
}

func (r *connectedAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state connectedAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectedAppId := state.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Deleting connected app resource: %s", connectedAppId))

	err := r.client.DeleteConnectedApp(ctx, connectedAppId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Connected app already deleted externally, treating as success", map[string]any{"id": connectedAppId})
			return
		}
		resp.Diagnostics.AddError("Error deleting connected app", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted connected app resource: %s", connectedAppId))
	// Terraform automatically removes the resource from state when Delete returns no error.
}

func (r *connectedAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapConnectedAppResponseToModel(ctx context.Context, app *models.ConnectedAppResponse, model *connectedAppResourceModel) {
	model.Id = types.StringValue(app.ID)
	model.Name = types.StringValue(app.Name)
	model.Type = types.StringValue(app.Type)

	if app.Data != nil {
		dataMap, ok := app.Data.(map[string]any)
		if !ok {
			tflog.Warn(ctx, fmt.Sprintf("Connected app data is not a map[string]any, got %T", app.Data))
			model.Data = types.DynamicNull()
		} else {
			dynamicValue, err := mapToDynamicValue(ctx, dataMap)
			if err != nil {
				tflog.Error(ctx, fmt.Sprintf("Error converting connected app data to dynamic: %v", err))
				model.Data = types.DynamicNull()
			} else {
				model.Data = dynamicValue
			}
		}
	} else {
		model.Data = types.DynamicNull()
	}

	model.CreatedBy = types.StringValue(app.CreatedBy)
	if !time.Time(app.CreatedAt).IsZero() {
		model.CreatedAt = types.StringValue(time.Time(app.CreatedAt).Format(time.RFC3339))
	} else {
		model.CreatedAt = types.StringNull()
	}

	model.UpdatedBy = types.StringValue(app.UpdatedBy)
	if !time.Time(app.UpdatedAt).IsZero() {
		model.UpdatedAt = types.StringValue(time.Time(app.UpdatedAt).Format(time.RFC3339))
	} else {
		model.UpdatedAt = types.StringNull()
	}
}

func dynamicValueToMap(ctx context.Context, dynamic types.Dynamic) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if dynamic.IsNull() || dynamic.IsUnknown() {
		diags.AddError("Missing required attribute", "The 'data' attribute is required and cannot be null or unknown")
		return nil, diags
	}

	underlyingValue := dynamic.UnderlyingValue()
	if underlyingValue == nil {
		diags.AddError("Missing required attribute", "The 'data' attribute is required and cannot be empty")
		return nil, diags
	}

	result, err := attrValueToGo(ctx, underlyingValue)
	if err != nil {
		diags.AddError("Error converting dynamic value", err.Error())
		return nil, diags
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		diags.AddError("Invalid data type", fmt.Sprintf("Expected map, got %T", result))
		return nil, diags
	}

	return resultMap, diags
}

func mapToDynamicValue(ctx context.Context, data map[string]any) (types.Dynamic, error) {
	attrValue, err := goToAttrValue(ctx, data)
	if err != nil {
		return types.DynamicNull(), err
	}

	return types.DynamicValue(attrValue), nil
}

func attrValueToGo(ctx context.Context, value attr.Value) (any, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	switch v := value.(type) {
	case types.String:
		return v.ValueString(), nil
	case types.Bool:
		return v.ValueBool(), nil
	case types.Int64:
		return v.ValueInt64(), nil
	case types.Float64:
		return v.ValueFloat64(), nil
	case types.Number:
		f, _ := v.ValueBigFloat().Float64()
		return f, nil
	case types.Map:
		result := make(map[string]any)
		for k, elem := range v.Elements() {
			converted, err := attrValueToGo(ctx, elem)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case types.Object:
		result := make(map[string]any)
		for k, elem := range v.Attributes() {
			converted, err := attrValueToGo(ctx, elem)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case types.List:
		var result []any
		for _, elem := range v.Elements() {
			converted, err := attrValueToGo(ctx, elem)
			if err != nil {
				return nil, err
			}
			result = append(result, converted)
		}
		return result, nil
	case types.Tuple:
		var result []any
		for _, elem := range v.Elements() {
			converted, err := attrValueToGo(ctx, elem)
			if err != nil {
				return nil, err
			}
			result = append(result, converted)
		}
		return result, nil
	case types.Set:
		var result []any
		for _, elem := range v.Elements() {
			converted, err := attrValueToGo(ctx, elem)
			if err != nil {
				return nil, err
			}
			result = append(result, converted)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported attr.Value type: %T", value)
	}
}

func goToAttrValue(ctx context.Context, value any) (attr.Value, error) {
	if value == nil {
		return types.StringNull(), nil
	}

	switch v := value.(type) {
	case string:
		return types.StringValue(v), nil
	case bool:
		return types.BoolValue(v), nil
	case int:
		return types.Int64Value(int64(v)), nil
	case int64:
		return types.Int64Value(v), nil
	case float64:
		return types.Float64Value(v), nil
	case map[string]any:
		attrTypes := make(map[string]attr.Type)
		attrValues := make(map[string]attr.Value)
		for k, elem := range v {
			converted, err := goToAttrValue(ctx, elem)
			if err != nil {
				return nil, err
			}
			attrTypes[k] = converted.Type(ctx)
			attrValues[k] = converted
		}
		objValue, diags := types.ObjectValue(attrTypes, attrValues)
		if diags.HasError() {
			return nil, fmt.Errorf("error creating object value: %v", diags.Errors())
		}
		return objValue, nil
	case []any:
		var attrValues []attr.Value
		for _, elem := range v {
			converted, err := goToAttrValue(ctx, elem)
			if err != nil {
				return nil, err
			}
			attrValues = append(attrValues, converted)
		}
		elemTypes := make([]attr.Type, len(attrValues))
		for i, av := range attrValues {
			elemTypes[i] = av.Type(ctx)
		}
		tupleValue, diags := types.TupleValue(elemTypes, attrValues)
		if diags.HasError() {
			return nil, fmt.Errorf("error creating tuple value: %v", diags.Errors())
		}
		return tupleValue, nil
	default:
		return types.StringValue(fmt.Sprintf("%v", v)), nil
	}
}
