package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/ingestionkeys"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- API Key Methods ---
func (c *SdkClientWrapper) CreateIngestionKey(ctx context.Context, req *models.CreateIngestionKeyRequest) (*models.IngestionKeyResult, error) {
	var name string
	if req != nil && req.Name != nil {
		name = *req.Name
	} else {
		name = "<unknown_ingestion_key_name>"
		tflog.Warn(ctx, "CreateIngestionKey called with nil req or nil Name")
	}
	logFields := map[string]any{"name": name}
	tflog.Debug(ctx, "Executing SDK Call: Create Ingestion Key", logFields)

	tflog.Debug(ctx, fmt.Sprintf("Sending CreateIngestionKeyRequest to SDK: %+v", req), logFields)

	params := ingestionkeys.NewCreateIngestionKeyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.Ingestionkeys.CreateIngestionKey(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateIngestionKey", name)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Ingestion Key", map[string]any{"name": resp.Payload.Name})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) ListIngestionKeys(ctx context.Context, req *models.ListIngestionKeysRequest) ([]*models.IngestionKeyResult, error) {
	tflog.Debug(ctx, "Executing SDK Call: List Ingestion Keys", map[string]any{"request": req})

	params := ingestionkeys.NewListIngestionKeysParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.Ingestionkeys.ListIngestionKeys(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "ListIngestionKeys", "")
	}

	tflog.Debug(ctx, "SDK Call Successful: List Ingestion Keys", map[string]any{"count": len(resp.Payload)})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) DeleteIngestionKey(ctx context.Context, req *models.DeleteIngestionKeyRequest) error {
	var name string
	if req != nil && req.Name != nil {
		name = *req.Name
	} else {
		name = "<unknown_ingestion_key_name>"
		tflog.Warn(ctx, "DeleteIngestionKey called with nil req or nil Name")
	}
	logFields := map[string]any{"name": name}
	tflog.Debug(ctx, "Executing SDK Call: Delete Ingestion Key", logFields)

	params := ingestionkeys.NewDeleteIngestionKeyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	if _, err := c.sdkClient.Ingestionkeys.DeleteIngestionKey(params, nil); err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteIngestionKey", name)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Ingestion Key Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Ingestion Key", logFields)
	return nil
}
