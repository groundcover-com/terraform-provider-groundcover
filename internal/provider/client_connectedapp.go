// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/connected_apps"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// deleteConnectedAppRefRetryAttempts and deleteConnectedAppRefRetryDelay control the retry
// behavior for the "connected app is still referenced by notification routes" case.
// The synthetic API auto-creates internal notification_routes when a synthetic_test runs
// in `connectedApps` mode; cleanup of those routes on synthetic destroy is eventually
// consistent. Without retry, a Terraform that destroys synthetic + connected_app together
// can hit a window where the route purge hasn't completed yet → 409.
//
// Total budget: ~30s (6 attempts × 5s) — enough for the backend to settle in practice,
// while bounded so genuine "still in use" cases (user has manually-created routes
// pointing at the connected_app) surface as errors in reasonable time.
const (
	deleteConnectedAppRefRetryAttempts = 6
	deleteConnectedAppRefRetryDelay    = 5 * time.Second
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

	// Retry on the specific "referenced by notification routes" 409. This race
	// happens when a synthetic_test using the connected_app in `connectedApps`
	// mode is destroyed alongside the connected_app: the synthetic backend's
	// async cleanup of its internal notification_route hasn't completed yet.
	// We retry with a fixed delay to give the backend time to settle. Other
	// 409s (e.g. user has real routes referencing the app) still surface
	// after the retry budget is exhausted.
	var lastErr error
	for attempt := 0; attempt < deleteConnectedAppRefRetryAttempts; attempt++ {
		_, err := c.sdkClient.ConnectedApps.DeleteConnectedApp(params, nil)
		if err == nil {
			tflog.Debug(ctx, "SDK Call Successful: Delete Connected App", logFields)
			return nil
		}

		mappedErr := handleApiError(ctx, err, "DeleteConnectedApp", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Connected App Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}

		lastErr = mappedErr

		// Only retry on the specific eventually-consistent reference case.
		// Match on the friendly message produced by handleApiError for this 409.
		if !strings.Contains(strings.ToLower(mappedErr.Error()), "referenced by one or more notification routes") {
			return mappedErr
		}

		if attempt < deleteConnectedAppRefRetryAttempts-1 {
			tflog.Warn(ctx, "Connected app still referenced by notification routes; retrying delete after backoff",
				map[string]any{"id": id, "attempt": attempt + 1, "max_attempts": deleteConnectedAppRefRetryAttempts})
			select {
			case <-time.After(deleteConnectedAppRefRetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}
