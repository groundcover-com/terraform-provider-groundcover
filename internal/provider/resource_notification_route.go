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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

var (
	_ resource.Resource                = &notificationRouteResource{}
	_ resource.ResourceWithConfigure   = &notificationRouteResource{}
	_ resource.ResourceWithImportState = &notificationRouteResource{}
)

func NewNotificationRouteResource() resource.Resource {
	return &notificationRouteResource{}
}

type notificationRouteResource struct {
	client ApiClient
}

type notificationRouteResourceModel struct {
	Id                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Query                types.String `tfsdk:"query"`
	Routes               types.List   `tfsdk:"routes"`
	NotificationSettings types.Object `tfsdk:"notification_settings"`
	CreatedBy            types.String `tfsdk:"created_by"`
	CreatedAt            types.String `tfsdk:"created_at"`
	ModifiedBy           types.String `tfsdk:"modified_by"`
	ModifiedAt           types.String `tfsdk:"modified_at"`
}

type routeRuleModel struct {
	Status        types.List `tfsdk:"status"`
	ConnectedApps types.List `tfsdk:"connected_apps"`
}

type routeConnectedAppModel struct {
	Type types.String `tfsdk:"type"`
	Id   types.String `tfsdk:"id"`
}

type notificationSettingsModel struct {
	RenotificationInterval types.String `tfsdk:"renotification_interval"`
}

func (r *notificationRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_route"
}

func (r *notificationRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Notification Route resource for managing issue routing to connected apps.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the notification route.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the notification route.",
				Required:    true,
			},
			"query": schema.StringAttribute{
				Description: "gcQL query to match issues.",
				Required:    true,
			},
			"routes": schema.ListNestedAttribute{
				Description: "List of routing rules that define which connected apps receive notifications based on issue status.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"status": schema.ListAttribute{
							Description: "List of issue statuses that trigger this route (e.g., 'open', 'resolved').",
							ElementType: types.StringType,
							Required:    true,
						},
						"connected_apps": schema.ListNestedAttribute{
							Description: "List of connected apps to notify for this route.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Description: "Type of connected app (e.g., 'slack-webhook', 'pagerduty').",
										Required:    true,
									},
									"id": schema.StringAttribute{
										Description: "ID of the connected app.",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"notification_settings": schema.SingleNestedAttribute{
				Description: "Notification settings for this route.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"renotification_interval": schema.StringAttribute{
						Description: "Duration between renotifications (e.g., '1h', '30m'). The API may normalize this value.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the notification route.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date the notification route was created (RFC3339 format).",
				Computed:    true,
			},
			"modified_by": schema.StringAttribute{
				Description: "The user who last modified the notification route.",
				Computed:    true,
			},
			"modified_at": schema.StringAttribute{
				Description: "The date the notification route was last modified (RFC3339 format).",
				Computed:    true,
			},
		},
	}
}

func (r *notificationRouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *notificationRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationRouteResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating notification route: %s", plan.Name.ValueString()))

	// Convert TF model to SDK request
	createReq, convDiags := planToCreateRequest(ctx, &plan)
	resp.Diagnostics.Append(convDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	createResp, err := r.client.CreateNotificationRoute(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating notification route", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Notification route created with ID: %s", createResp.ID))

	// API returns only ID, must GET to populate full state
	route, err := r.client.GetNotificationRoute(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created notification route", err.Error())
		return
	}

	// Save the original plan's notification_settings for duration comparison
	originalSettings := plan.NotificationSettings

	// Populate state from GET response
	mapNotificationRouteResponseToModel(ctx, route, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve plan's notification_settings.renotification_interval if semantically equivalent
	plan.NotificationSettings = preserveEquivalentDuration(ctx, originalSettings, plan.NotificationSettings)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully created notification route resource: %s", plan.Id.ValueString()))
}

func (r *notificationRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationRouteResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading notification route resource: %s", state.Id.ValueString()))

	route, err := r.client.GetNotificationRoute(ctx, state.Id.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Notification route %s not found, removing from state.", state.Id.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading notification route", err.Error())
		return
	}

	originalSettings := state.NotificationSettings

	mapNotificationRouteResponseToModel(ctx, route, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	state.NotificationSettings = preserveEquivalentDuration(ctx, originalSettings, state.NotificationSettings)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully read notification route resource: %s", state.Id.ValueString()))
}

func (r *notificationRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan notificationRouteResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Updating notification route: %s", plan.Id.ValueString()))

	// Convert to SDK request
	updateReq, convDiags := planToUpdateRequest(ctx, &plan)
	resp.Diagnostics.Append(convDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	err := r.client.UpdateNotificationRoute(ctx, plan.Id.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating notification route", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Notification route updated: %s", plan.Id.ValueString()))

	// GET to refresh state
	route, err := r.client.GetNotificationRoute(ctx, plan.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated notification route", err.Error())
		return
	}

	originalSettings := plan.NotificationSettings

	mapNotificationRouteResponseToModel(ctx, route, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.NotificationSettings = preserveEquivalentDuration(ctx, originalSettings, plan.NotificationSettings)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully updated notification route resource: %s", plan.Id.ValueString()))
}

func (r *notificationRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state notificationRouteResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	routeId := state.Id.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Deleting notification route resource: %s", routeId))

	err := r.client.DeleteNotificationRoute(ctx, routeId)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			resp.Diagnostics.AddError("Error deleting notification route", err.Error())
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted notification route resource: %s", routeId))
}

func (r *notificationRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper functions for TF/SDK conversion

func planToCreateRequest(ctx context.Context, plan *notificationRouteResourceModel) (*models.CreateNotificationRouteRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	nameStr := plan.Name.ValueString()
	queryStr := plan.Query.ValueString()

	req := &models.CreateNotificationRouteRequest{
		Name:  &nameStr,
		Query: &queryStr,
	}

	// Convert routes
	routes, routeDiags := routesListToSDK(ctx, plan.Routes)
	diags.Append(routeDiags...)
	if diags.HasError() {
		return nil, diags
	}
	req.Routes = routes

	settings, settingsDiags := notificationSettingsToSDK(ctx, plan.NotificationSettings)
	diags.Append(settingsDiags...)
	if diags.HasError() {
		return nil, diags
	}
	req.NotificationSettings = settings

	return req, diags
}

func planToUpdateRequest(ctx context.Context, plan *notificationRouteResourceModel) (*models.UpdateNotificationRouteRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	nameStr := plan.Name.ValueString()
	queryStr := plan.Query.ValueString()

	req := &models.UpdateNotificationRouteRequest{
		Name:  &nameStr,
		Query: &queryStr,
	}

	// Convert routes
	routes, routeDiags := routesListToSDK(ctx, plan.Routes)
	diags.Append(routeDiags...)
	if diags.HasError() {
		return nil, diags
	}
	req.Routes = routes

	settings, settingsDiags := notificationSettingsToSDK(ctx, plan.NotificationSettings)
	diags.Append(settingsDiags...)
	if diags.HasError() {
		return nil, diags
	}
	req.NotificationSettings = settings

	return req, diags
}

func routesListToSDK(ctx context.Context, routesList types.List) ([]*models.RouteRuleRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	if routesList.IsNull() || routesList.IsUnknown() {
		return []*models.RouteRuleRequest{}, diags
	}

	var routeModels []routeRuleModel
	diags.Append(routesList.ElementsAs(ctx, &routeModels, false)...)
	if diags.HasError() {
		return nil, diags
	}

	sdkRoutes := make([]*models.RouteRuleRequest, len(routeModels))
	for i, routeModel := range routeModels {
		// Convert status list
		var statusList []string
		if !routeModel.Status.IsNull() && !routeModel.Status.IsUnknown() {
			diags.Append(routeModel.Status.ElementsAs(ctx, &statusList, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}

		// Convert connected apps list
		var connectedAppModels []routeConnectedAppModel
		if !routeModel.ConnectedApps.IsNull() && !routeModel.ConnectedApps.IsUnknown() {
			diags.Append(routeModel.ConnectedApps.ElementsAs(ctx, &connectedAppModels, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}

		sdkConnectedApps := make([]*models.RouteConnectedAppRequest, len(connectedAppModels))
		for j, appModel := range connectedAppModels {
			typeStr := appModel.Type.ValueString()
			idStr := appModel.Id.ValueString()
			sdkConnectedApps[j] = &models.RouteConnectedAppRequest{
				Type: &typeStr,
				ID:   &idStr,
			}
		}

		sdkRoutes[i] = &models.RouteRuleRequest{
			Status:        statusList,
			ConnectedApps: sdkConnectedApps,
		}
	}

	return sdkRoutes, diags
}

func routesSDKToList(ctx context.Context, sdkRoutes []*models.RouteRuleResponse) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(sdkRoutes) == 0 {
		emptyList, listDiags := types.ListValue(types.ObjectType{
			AttrTypes: routeRuleAttrTypes(),
		}, []attr.Value{})
		diags.Append(listDiags...)
		return emptyList, diags
	}

	routeElements := make([]attr.Value, len(sdkRoutes))
	for i, sdkRoute := range sdkRoutes {
		// Convert status list
		statusList, statusDiags := types.ListValueFrom(ctx, types.StringType, sdkRoute.Status)
		diags.Append(statusDiags...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: routeRuleAttrTypes()}), diags
		}

		// Convert connected apps list
		connectedAppElements := make([]attr.Value, len(sdkRoute.ConnectedApps))
		for j, sdkApp := range sdkRoute.ConnectedApps {
			appObj, appDiags := types.ObjectValue(
				routeConnectedAppAttrTypes(),
				map[string]attr.Value{
					"type": types.StringValue(sdkApp.Type),
					"id":   types.StringValue(sdkApp.ID),
				},
			)
			diags.Append(appDiags...)
			if diags.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: routeRuleAttrTypes()}), diags
			}
			connectedAppElements[j] = appObj
		}

		connectedAppsList, appListDiags := types.ListValue(
			types.ObjectType{AttrTypes: routeConnectedAppAttrTypes()},
			connectedAppElements,
		)
		diags.Append(appListDiags...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: routeRuleAttrTypes()}), diags
		}

		routeObj, routeDiags := types.ObjectValue(
			routeRuleAttrTypes(),
			map[string]attr.Value{
				"status":         statusList,
				"connected_apps": connectedAppsList,
			},
		)
		diags.Append(routeDiags...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: routeRuleAttrTypes()}), diags
		}

		routeElements[i] = routeObj
	}

	routesList, listDiags := types.ListValue(
		types.ObjectType{AttrTypes: routeRuleAttrTypes()},
		routeElements,
	)
	diags.Append(listDiags...)

	return routesList, diags
}

func notificationSettingsToSDK(ctx context.Context, settings types.Object) (*models.NotificationSettingsRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	if settings.IsNull() || settings.IsUnknown() {
		return nil, diags
	}

	var settingsModel notificationSettingsModel
	diags.Append(settings.As(ctx, &settingsModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	req := &models.NotificationSettingsRequest{}
	if !settingsModel.RenotificationInterval.IsNull() && !settingsModel.RenotificationInterval.IsUnknown() {
		req.RenotificationInterval = settingsModel.RenotificationInterval.ValueString()
	}

	return req, diags
}

func notificationSettingsSDKToObject(ctx context.Context, sdkSettings *models.NotificationSettingsResponse) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if sdkSettings == nil {
		return types.ObjectNull(notificationSettingsAttrTypes()), diags
	}

	// Normalize duration if present
	renotificationInterval := types.StringNull()
	if sdkSettings.RenotificationInterval != "" {
		normalized := normalizeDuration(sdkSettings.RenotificationInterval)
		renotificationInterval = types.StringValue(normalized)
	}

	obj, objDiags := types.ObjectValue(
		notificationSettingsAttrTypes(),
		map[string]attr.Value{
			"renotification_interval": renotificationInterval,
		},
	)
	diags.Append(objDiags...)

	return obj, diags
}

func mapNotificationRouteResponseToModel(ctx context.Context, route *models.NotificationRouteResponse, model *notificationRouteResourceModel, diags *diag.Diagnostics) {
	model.Id = types.StringValue(route.ID)
	model.Name = types.StringValue(route.Name)
	model.Query = types.StringValue(route.Query)

	// Convert routes
	routesList, routesDiags := routesSDKToList(ctx, route.Routes)
	diags.Append(routesDiags...)
	if diags.HasError() {
		return
	}
	model.Routes = routesList

	// Convert notification settings
	settingsObj, settingsDiags := notificationSettingsSDKToObject(ctx, route.NotificationSettings)
	diags.Append(settingsDiags...)
	if diags.HasError() {
		return
	}
	model.NotificationSettings = settingsObj

	model.CreatedBy = types.StringValue(route.CreatedBy)
	if !time.Time(route.CreatedAt).IsZero() {
		model.CreatedAt = types.StringValue(time.Time(route.CreatedAt).Format(time.RFC3339))
	} else {
		model.CreatedAt = types.StringNull()
	}

	model.ModifiedBy = types.StringValue(route.ModifiedBy)
	if !time.Time(route.ModifiedAt).IsZero() {
		model.ModifiedAt = types.StringValue(time.Time(route.ModifiedAt).Format(time.RFC3339))
	} else {
		model.ModifiedAt = types.StringNull()
	}
}

func normalizeDuration(d string) string {
	if d == "" {
		return d
	}

	parsed, err := time.ParseDuration(d)
	if err != nil {
		return d
	}

	hours := int(parsed.Hours())
	minutes := int(parsed.Minutes()) % 60
	seconds := int(parsed.Seconds()) % 60

	if hours > 0 && minutes == 0 && seconds == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	if hours > 0 && seconds == 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if hours == 0 && minutes > 0 && seconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	return parsed.String()
}

func durationsEqual(d1, d2 string) bool {
	if d1 == d2 {
		return true
	}
	if d1 == "" || d2 == "" {
		return false
	}

	parsed1, err1 := time.ParseDuration(d1)
	parsed2, err2 := time.ParseDuration(d2)
	if err1 != nil || err2 != nil {
		return false
	}

	return parsed1 == parsed2
}

func preserveEquivalentDuration(ctx context.Context, original, updated types.Object) types.Object {
	if original.IsNull() || original.IsUnknown() {
		return updated
	}
	if updated.IsNull() {
		return updated
	}

	var origSettings, updSettings notificationSettingsModel
	if diags := original.As(ctx, &origSettings, basetypes.ObjectAsOptions{}); diags.HasError() {
		return updated
	}
	if diags := updated.As(ctx, &updSettings, basetypes.ObjectAsOptions{}); diags.HasError() {
		return updated
	}

	origInterval := origSettings.RenotificationInterval.ValueString()
	updInterval := updSettings.RenotificationInterval.ValueString()

	if durationsEqual(origInterval, updInterval) {
		return original
	}

	return updated
}

// Attribute type definitions for nested structures

func routeConnectedAppAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
		"id":   types.StringType,
	}
}

func routeRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"status": types.ListType{ElemType: types.StringType},
		"connected_apps": types.ListType{
			ElemType: types.ObjectType{AttrTypes: routeConnectedAppAttrTypes()},
		},
	}
}

func notificationSettingsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"renotification_interval": types.StringType,
	}
}
