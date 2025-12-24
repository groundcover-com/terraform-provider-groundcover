// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/secret"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateSecret(ctx context.Context, req *models.CreateSecretRequest) (*models.SecretResponse, error) {
	identifier := "<unknown>"
	if req.Name != nil {
		identifier = *req.Name
	}
	logFields := map[string]any{"name": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Create Secret", logFields)

	params := secret.NewCreateSecretParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(req)

	resp, err := c.sdkClient.Secret.CreateSecret(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateSecret", identifier)
	}

	respId := "<empty_id>"
	if resp.Payload != nil && resp.Payload.ID != "" {
		respId = resp.Payload.ID
	} else if resp.Payload == nil {
		tflog.Warn(ctx, "CreateSecret response payload was nil", logFields)
	}
	tflog.Debug(ctx, "SDK Call Successful: Create Secret", map[string]any{"id": respId})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateSecret(ctx context.Context, id string, req *models.UpdateSecretRequest) (*models.SecretResponse, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Update Secret", logFields)

	params := secret.NewUpdateSecretParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id).
		WithBody(req)

	resp, err := c.sdkClient.Secret.UpdateSecret(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateSecret", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Secret", logFields)
	return resp.Payload, nil
}

func (c *SdkClientWrapper) DeleteSecret(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Secret", logFields)

	params := secret.NewDeleteSecretParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.Secret.DeleteSecret(params, nil)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteSecret", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Secret Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Secret", logFields)
	return nil
}
