// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/connected_apps"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateConnectedApp(ctx context.Context, req *models.CreateConnectedAppRequest) (*models.CreateConnectedAppResponse, error) {
	identifier := "<unknown>"
	if req.Name != nil && *req.Name != "" {
		identifier = *req.Name
	}
	logFields := map[string]any{"name": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Create Connected App", logFields)

	params := connected_apps.NewCreateConnectedAppParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.ConnectedApps.CreateConnectedApp(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateConnectedApp", identifier)
	}

	if resp == nil {
		tflog.Error(ctx, "SDK CreateConnectedApp returned nil response and nil error, which is unexpected.", logFields)
		return nil, errors.New("internal SDK error: CreateConnectedApp returned nil response without error")
	}

	respId := "<empty_id>"
	if resp.Payload != nil && resp.Payload.ID != "" {
		respId = resp.Payload.ID
	} else if resp.Payload == nil {
		tflog.Warn(ctx, "CreateConnectedApp response payload was nil", logFields)
	}
	tflog.Debug(ctx, "SDK Call Successful: Create Connected App", map[string]any{"id": respId})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) GetConnectedApp(ctx context.Context, id string) (*models.ConnectedAppResponse, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Get Connected App", logFields)

	params := connected_apps.NewGetConnectedAppParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	resp, err := c.sdkClient.ConnectedApps.GetConnectedApp(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetConnectedApp", id)
	}

	if resp == nil {
		tflog.Error(ctx, "SDK GetConnectedApp returned nil response and nil error, which is unexpected.", logFields)
		return nil, errors.New("internal SDK error: GetConnectedApp returned nil response without error")
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Connected App", logFields)
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateConnectedApp(ctx context.Context, id string, req *models.UpdateConnectedAppRequest) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Update Connected App", logFields)

	params := connected_apps.NewUpdateConnectedAppParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id).
		WithBody(req)

	_, err := c.sdkClient.ConnectedApps.UpdateConnectedApp(params, nil)
	if err != nil {
		return handleApiError(ctx, err, "UpdateConnectedApp", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Connected App", logFields)
	return nil
}

func (c *SdkClientWrapper) DeleteConnectedApp(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Connected App", logFields)

	params := connected_apps.NewDeleteConnectedAppParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.ConnectedApps.DeleteConnectedApp(params, nil)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteConnectedApp", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Connected App Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Connected App", logFields)
	return nil
}
