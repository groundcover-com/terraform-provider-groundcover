package provider

import (
	"context"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/dashboards"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateDashboard(ctx context.Context, dashboard *models.CreateDashboardRequest) (*models.View, error) {
	var name string
	if dashboard != nil {
		name = dashboard.Name
	} else {
		name = "<unknown_dashboard_name>"
		tflog.Warn(ctx, "CreateDashboard called with nil dashboard")
	}
	logFields := map[string]any{"name": name}
	tflog.Debug(ctx, "Executing SDK Call: Create Dashboard", logFields)

	tflog.Debug(ctx, fmt.Sprintf("Sending CreateDashboardRequest to SDK: %+v", dashboard), logFields)

	params := dashboards.NewCreateDashboardParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(dashboard)

	resp, err := c.sdkClient.Dashboards.CreateDashboard(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateDashboard", name)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Dashboard", map[string]any{"uuid": resp.Payload.UUID})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) GetDashboard(ctx context.Context, uuid string) (*models.View, error) {
	tflog.Debug(ctx, "Executing SDK Call: Get Dashboard", map[string]any{"uuid": uuid})

	params := dashboards.NewGetDashboardParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(uuid)

	resp, err := c.sdkClient.Dashboards.GetDashboard(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetDashboard", uuid)
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Dashboard", map[string]any{"uuid": uuid})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateDashboard(ctx context.Context, uuid string, dashboard *models.UpdateDashboardRequest) (*models.View, error) {
	tflog.Debug(ctx, "Executing SDK Call: Update Dashboard", map[string]any{"uuid": uuid})

	tflog.Debug(ctx, fmt.Sprintf("Sending UpdateDashboardRequest to SDK: %+v", dashboard), map[string]any{"uuid": uuid})

	params := dashboards.NewUpdateDashboardParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(uuid).
		WithBody(dashboard)

	resp, err := c.sdkClient.Dashboards.UpdateDashboard(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateDashboard", uuid)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Dashboard", map[string]any{"uuid": uuid})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) DeleteDashboard(ctx context.Context, uuid string) error {
	tflog.Debug(ctx, "Executing SDK Call: Delete Dashboard", map[string]any{"uuid": uuid})

	params := dashboards.NewDeleteDashboardParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(uuid)

	_, err := c.sdkClient.Dashboards.DeleteDashboard(params, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteDashboard", uuid)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Dashboard", map[string]any{"uuid": uuid})
	return nil
}

func (c *SdkClientWrapper) ListDashboards(ctx context.Context) ([]*models.View, error) {
	tflog.Debug(ctx, "Executing SDK Call: List Dashboards")

	params := dashboards.NewGetDashboardsParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout)

	resp, err := c.sdkClient.Dashboards.GetDashboards(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "ListDashboards", "")
	}

	tflog.Debug(ctx, "SDK Call Successful: List Dashboards", map[string]any{"count": len(resp.Payload)})
	return resp.Payload, nil
}
