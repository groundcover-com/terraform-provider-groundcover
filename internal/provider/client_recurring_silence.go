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

func (c *SdkClientWrapper) CreateRecurringSilence(ctx context.Context, req *models.CreateRecurringSilenceRequest) (*models.RecurringSilenceResponse, error) {
	tflog.Debug(ctx, "Executing SDK Call: Create Recurring Silence")

	params := monitors.NewCreateRecurringSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.Monitors.CreateRecurringSilence(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateRecurringSilence", req.Comment)
	}

	if resp == nil {
		return nil, errors.New("internal SDK error: CreateRecurringSilence returned nil response without error")
	}

	respId := "<empty_id>"
	if resp.Payload != nil && resp.Payload.UUID.String() != "" {
		respId = resp.Payload.UUID.String()
	}
	tflog.Debug(ctx, "SDK Call Successful: Create Recurring Silence", map[string]any{"id": respId})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) GetRecurringSilence(ctx context.Context, id string) (*models.RecurringSilenceResponse, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Get Recurring Silence", logFields)

	// An empty ID would otherwise be sent to GET /recurring-silences/{id}, which redirects
	// to the collection route and returns 200 with the full list (an array) — never a 404.
	// Treat it as not-found so callers don't mistake a list response for an existing resource.
	if id == "" {
		return nil, ErrNotFound
	}

	params := monitors.NewGetRecurringSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	resp, err := c.sdkClient.Monitors.GetRecurringSilence(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetRecurringSilence", id)
	}

	if resp == nil {
		return nil, errors.New("internal SDK error: GetRecurringSilence returned nil response without error")
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Recurring Silence", logFields)
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateRecurringSilence(ctx context.Context, id string, req *models.UpdateRecurringSilenceRequest) (*models.RecurringSilenceResponse, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Update Recurring Silence", logFields)

	params := monitors.NewUpdateRecurringSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id).
		WithBody(req)

	resp, err := c.sdkClient.Monitors.UpdateRecurringSilence(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateRecurringSilence", id)
	}

	if resp == nil {
		return nil, errors.New("internal SDK error: UpdateRecurringSilence returned nil response without error")
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Recurring Silence", logFields)
	return resp.Payload, nil
}

func (c *SdkClientWrapper) DeleteRecurringSilence(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Recurring Silence", logFields)

	params := monitors.NewDeleteRecurringSilenceParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.Monitors.DeleteRecurringSilence(params, nil)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteRecurringSilence", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Recurring Silence Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Recurring Silence", logFields)
	return nil
}
