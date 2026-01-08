// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/monitors"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateSilence(ctx context.Context, req *models.CreateSilenceRequest) (*models.Silence, error) {
	identifier := "<unknown>"
	if req.Comment != "" {
		identifier = req.Comment
	}
	logFields := map[string]any{"comment": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Create Silence", logFields)

	params := monitors.NewCreateSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.Monitors.CreateSilence(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateSilence", identifier)
	}

	if resp == nil {
		tflog.Error(ctx, "SDK CreateSilence returned nil response and nil error, which is unexpected.", logFields)
		return nil, errors.New("internal SDK error: CreateSilence returned nil response without error")
	}

	respId := "<empty_id>"
	if resp.Payload != nil && resp.Payload.UUID.String() != "" {
		respId = resp.Payload.UUID.String()
	} else if resp.Payload == nil {
		tflog.Warn(ctx, "CreateSilence response payload was nil", logFields)
	}
	tflog.Debug(ctx, "SDK Call Successful: Create Silence", map[string]any{"id": respId})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) GetSilence(ctx context.Context, id string) (*models.Silence, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Get Silence", logFields)

	params := monitors.NewGetSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	resp, err := c.sdkClient.Monitors.GetSilence(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetSilence", id)
	}

	if resp == nil {
		tflog.Error(ctx, "SDK GetSilence returned nil response and nil error, which is unexpected.", logFields)
		return nil, errors.New("internal SDK error: GetSilence returned nil response without error")
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Silence", logFields)
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateSilence(ctx context.Context, id string, req *models.UpdateSilenceRequest) (*models.Silence, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Update Silence", logFields)

	params := monitors.NewUpdateSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id).
		WithBody(req)

	resp, err := c.sdkClient.Monitors.UpdateSilence(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateSilence", id)
	}

	if resp == nil {
		tflog.Error(ctx, "SDK UpdateSilence returned nil response and nil error, which is unexpected.", logFields)
		return nil, errors.New("internal SDK error: UpdateSilence returned nil response without error")
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Silence", logFields)
	return resp.Payload, nil
}

func (c *SdkClientWrapper) DeleteSilence(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Silence", logFields)

	params := monitors.NewDeleteSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.Monitors.DeleteSilence(params, nil)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteSilence", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Silence Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Silence", logFields)
	return nil
}
