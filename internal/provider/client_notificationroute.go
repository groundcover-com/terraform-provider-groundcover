// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/notification_routes"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateNotificationRoute(ctx context.Context, req *models.CreateNotificationRouteRequest) (*models.CreateNotificationRouteResponse, error) {
	identifier := "<unknown>"
	if req.Name != nil && *req.Name != "" {
		identifier = *req.Name
	}
	logFields := map[string]any{"name": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Create Notification Route", logFields)

	params := notification_routes.NewCreateNotificationRouteParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.NotificationRoutes.CreateNotificationRoute(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateNotificationRoute", identifier)
	}

	if resp == nil {
		tflog.Error(ctx, "SDK CreateNotificationRoute returned nil response and nil error, which is unexpected.", logFields)
		return nil, errors.New("internal SDK error: CreateNotificationRoute returned nil response without error")
	}

	respId := "<empty_id>"
	if resp.Payload != nil && resp.Payload.ID != "" {
		respId = resp.Payload.ID
	} else if resp.Payload == nil {
		tflog.Warn(ctx, "CreateNotificationRoute response payload was nil", logFields)
	}
	tflog.Debug(ctx, "SDK Call Successful: Create Notification Route", map[string]any{"id": respId})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) GetNotificationRoute(ctx context.Context, id string) (*models.NotificationRouteResponse, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Get Notification Route", logFields)

	params := notification_routes.NewGetNotificationRouteParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	resp, err := c.sdkClient.NotificationRoutes.GetNotificationRoute(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetNotificationRoute", id)
	}

	if resp == nil {
		tflog.Error(ctx, "SDK GetNotificationRoute returned nil response and nil error, which is unexpected.", logFields)
		return nil, errors.New("internal SDK error: GetNotificationRoute returned nil response without error")
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Notification Route", logFields)
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateNotificationRoute(ctx context.Context, id string, req *models.UpdateNotificationRouteRequest) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Update Notification Route", logFields)

	params := notification_routes.NewUpdateNotificationRouteParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id).
		WithBody(req)

	_, err := c.sdkClient.NotificationRoutes.UpdateNotificationRoute(params, nil)
	if err != nil {
		return handleApiError(ctx, err, "UpdateNotificationRoute", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Notification Route", logFields)
	return nil
}

func (c *SdkClientWrapper) DeleteNotificationRoute(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Notification Route", logFields)

	params := notification_routes.NewDeleteNotificationRouteParamsWithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.NotificationRoutes.DeleteNotificationRoute(params, nil)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteNotificationRoute", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Notification Route Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Notification Route", logFields)
	return nil
}
