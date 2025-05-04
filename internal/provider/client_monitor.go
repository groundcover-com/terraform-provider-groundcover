package provider

import (
	"context"
	"errors"

	sdkmonitor "github.com/groundcover-com/groundcover-sdk-go/sdk/api/monitors"
	"github.com/groundcover-com/groundcover-sdk-go/sdk/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateMonitorYaml(ctx context.Context, monitorYaml []byte) (*sdkmonitor.CreateMonitorResponse, error) {
	logFields := map[string]any{"yaml_length": len(monitorYaml)}
	tflog.Debug(ctx, "Executing SDK Call: Create Monitor YAML", logFields)

	// Pass the call directly to the SDK's monitor service
	resp, err := c.sdkClient.Monitors.CreateMonitorYaml(ctx, monitorYaml)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateMonitorYaml", "<from_yaml>") // ID is unknown during create
	}

	respId := "<nil_or_empty_response>"
	// Use the correct field MonitorID based on the provided struct definition
	if resp != nil && resp.MonitorID != "" {
		respId = resp.MonitorID
	} else if resp != nil {
		tflog.Warn(ctx, "CreateMonitorYaml response contained an empty MonitorID", logFields)
	} else {
		tflog.Warn(ctx, "CreateMonitorYaml response was nil", logFields)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Monitor YAML", map[string]any{"id": respId})
	return resp, nil
}

func (c *SdkClientWrapper) GetMonitor(ctx context.Context, id string) ([]byte, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Get Monitor YAML", logFields)

	// Pass the call directly to the SDK's monitor service
	resp, err := c.sdkClient.Monitors.GetMonitor(ctx, id)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetMonitor", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Monitor YAML", map[string]any{"id": id, "yaml_length": len(resp)})
	return resp, nil
}

func (c *SdkClientWrapper) UpdateMonitorYaml(ctx context.Context, id string, monitorYaml []byte) (*models.EmptyResponse, error) {
	logFields := map[string]any{"id": id, "yaml_length": len(monitorYaml)}
	tflog.Debug(ctx, "Executing SDK Call: Update Monitor YAML", logFields)

	// Pass the call directly to the SDK's monitor service
	resp, err := c.sdkClient.Monitors.UpdateMonitorYaml(ctx, id, monitorYaml)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateMonitorYaml", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Monitor YAML", logFields)
	return resp, nil
}

func (c *SdkClientWrapper) DeleteMonitor(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Monitor", logFields)

	// Pass the call directly to the SDK's monitor service
	_, err := c.sdkClient.Monitors.DeleteMonitor(ctx, id)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteMonitor", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Monitor Not Found during Delete (Idempotent Success)", logFields)
			return nil // Treat NotFound as success
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Monitor", logFields)
	return nil
}
