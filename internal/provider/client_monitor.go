package provider

import (
	"context"
	"errors"

	// Updated SDK imports
	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/monitors"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateMonitor now takes *models.CreateMonitorRequest and returns *models.CreateMonitorResponse (which contains the ID)
// The provider will be responsible for unmarshaling YAML into models.CreateMonitorRequest before calling this.
func (c *SdkClientWrapper) CreateMonitor(ctx context.Context, monitorReq *models.CreateMonitorRequest) (*models.CreateMonitorResponse, error) {
	// The identifier for logging might need to come from monitorReq.Title or similar, assuming it's set.
	identifier := "<unknown_monitor>"
	if monitorReq != nil && monitorReq.Title != nil {
		identifier = *monitorReq.Title
	}
	logFields := map[string]any{"title": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Create Monitor", logFields)

	params := monitors.NewCreateMonitorParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout). // Using defaultTimeout from client.go
		WithBody(monitorReq)         // SDK will marshal this to YAML if Content-Type is application/x-yaml

	// We must use WithContentTypeApplicationxYaml client option for the SDK to send YAML
	resp, err := c.sdkClient.Monitors.CreateMonitor(params, nil, monitors.WithContentTypeApplicationxYaml)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateMonitor", identifier)
	}

	respId := "<nil_or_empty_response>"
	if resp != nil && resp.Payload != nil && resp.Payload.MonitorID != "" {
		respId = resp.Payload.MonitorID
	} else if resp != nil && resp.Payload != nil {
		tflog.Warn(ctx, "CreateMonitor response payload contained an empty MonitorID", logFields)
	} else {
		tflog.Warn(ctx, "CreateMonitor response or payload was nil", logFields)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Monitor", map[string]any{"id": respId})
	return resp.Payload, nil
}

// GetMonitor returns raw YAML bytes ([]byte)
func (c *SdkClientWrapper) GetMonitor(ctx context.Context, id string) ([]byte, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Get Monitor YAML", logFields)

	params := monitors.NewGetMonitorParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	// GetMonitor in SDK produces "application/x-yaml"
	// Revert to not explicitly setting Accept header, to match E2E test behavior.
	resp, err := c.sdkClient.Monitors.GetMonitor(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetMonitor", id)
	}

	// Log the raw YAML response for debugging
	if resp != nil && resp.Payload != nil {
		tflog.Debug(ctx, "SDK GetMonitor Response YAML", map[string]any{"id": id, "yaml_content": string(resp.Payload)})
	} else {
		tflog.Warn(ctx, "SDK GetMonitor returned nil response or payload", map[string]any{"id": id})
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Monitor YAML", map[string]any{"id": id, "yaml_length": len(resp.Payload)})
	return resp.Payload, nil // resp.Payload is []uint8
}

// UpdateMonitor now takes *models.UpdateMonitorRequest and returns error (UpdateMonitorAccepted has no payload)
// The provider will be responsible for unmarshaling YAML into models.UpdateMonitorRequest.
func (c *SdkClientWrapper) UpdateMonitor(ctx context.Context, id string, monitorReq *models.UpdateMonitorRequest) error {
	identifier := "<unknown_monitor>"
	if monitorReq != nil && monitorReq.Title != nil {
		identifier = *monitorReq.Title
	}
	logFields := map[string]any{"id": id, "title": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Update Monitor", logFields)

	params := monitors.NewUpdateMonitorParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id).
		WithBody(monitorReq) // SDK will marshal this to YAML if Content-Type is application/x-yaml

	// We must use WithContentTypeApplicationxYaml client option for the SDK to send YAML
	_, err := c.sdkClient.Monitors.UpdateMonitor(params, nil, monitors.WithContentTypeApplicationxYaml)
	if err != nil {
		return handleApiError(ctx, err, "UpdateMonitor", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Monitor", logFields)
	return nil
}

func (c *SdkClientWrapper) DeleteMonitor(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Monitor", logFields)

	params := monitors.NewDeleteMonitorParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.Monitors.DeleteMonitor(params, nil) // DeleteMonitorOK has an empty payload
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
