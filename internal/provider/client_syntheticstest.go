package provider

import (
	"context"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/synthetics"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	syntheticTestResourceId = "syntheticstest"
)

// CreateSyntheticTest creates a new synthetic test
func (c *SdkClientWrapper) CreateSyntheticTest(ctx context.Context, req *models.SyntheticTestCreateRequest) (*models.SyntheticTestCreateResponse, error) {
	logFields := map[string]any{"req": "create_synthetic_test", "name": req.Name}
	tflog.Debug(ctx, "Executing SDK Call: Create Synthetic Test", logFields)

	params := synthetics.NewCreateSyntheticTestParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.Synthetics.CreateSyntheticTest(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateSyntheticTest", syntheticTestResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Synthetic Test", map[string]any{"id": resp.Payload.ID})
	return resp.Payload, nil
}

// GetSyntheticTest retrieves a synthetic test by ID
func (c *SdkClientWrapper) GetSyntheticTest(ctx context.Context, id string) (*models.SyntheticTestCreateRequest, error) {
	logFields := map[string]any{"req": "get_synthetic_test", "id": id}
	tflog.Debug(ctx, "Executing SDK Call: Get Synthetic Test", logFields)

	params := synthetics.NewGetSyntheticTestParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	resp, err := c.sdkClient.Synthetics.GetSyntheticTest(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetSyntheticTest", syntheticTestResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Synthetic Test", logFields)
	return resp.Payload, nil
}

// UpdateSyntheticTest updates an existing synthetic test
func (c *SdkClientWrapper) UpdateSyntheticTest(ctx context.Context, id string, req *models.SyntheticTestCreateRequest) error {
	logFields := map[string]any{"req": "update_synthetic_test", "id": id, "name": req.Name}
	tflog.Debug(ctx, "Executing SDK Call: Update Synthetic Test", logFields)

	params := synthetics.NewUpdateSyntheticTestParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id).
		WithBody(req)

	_, err := c.sdkClient.Synthetics.UpdateSyntheticTest(params, nil)
	if err != nil {
		return handleApiError(ctx, err, "UpdateSyntheticTest", syntheticTestResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Synthetic Test", logFields)
	return nil
}

// DeleteSyntheticTest deletes a synthetic test by ID
func (c *SdkClientWrapper) DeleteSyntheticTest(ctx context.Context, id string) error {
	logFields := map[string]any{"req": "delete_synthetic_test", "id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Synthetic Test", logFields)

	params := synthetics.NewDeleteSyntheticTestParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.Synthetics.DeleteSyntheticTest(params, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteSyntheticTest", syntheticTestResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Synthetic Test", logFields)
	return nil
}
