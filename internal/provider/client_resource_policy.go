package provider

import (
	"context"
	"errors"

	// Updated SDK imports
	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/policies" // For params and service client
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"          // For request/response body models

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreatePolicy now takes *models.CreatePolicyRequest and returns *models.Policy
func (c *SdkClientWrapper) CreatePolicy(ctx context.Context, policyReq *models.CreatePolicyRequest) (*models.Policy, error) {
	logFields := map[string]any{"name": policyReq.Name}
	tflog.Debug(ctx, "Executing SDK Call: Create Policy", logFields)

	params := policies.NewCreatePolicyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout). // Using defaultTimeout from client.go
		WithBody(policyReq)

	resp, err := c.sdkClient.Policies.CreatePolicy(params, nil)
	if err != nil {
		// Use Name from the original request model for error reporting
		return nil, handleApiError(ctx, err, "CreatePolicy", *policyReq.Name) // Name is a pointer
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Policy", map[string]any{"uuid": resp.Payload.UUID})
	return resp.Payload, nil
}

// GetPolicy returns *models.Policy
func (c *SdkClientWrapper) GetPolicy(ctx context.Context, uuid string) (*models.Policy, error) {
	logFields := map[string]any{"uuid": uuid}
	tflog.Debug(ctx, "Executing SDK Call: Get Policy", logFields)

	params := policies.NewGetPolicyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(uuid)

	resp, err := c.sdkClient.Policies.GetPolicy(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetPolicy", uuid)
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Policy", logFields)
	return resp.Payload, nil
}

// UpdatePolicy now takes *models.UpdatePolicyRequest and returns *models.Policy
func (c *SdkClientWrapper) UpdatePolicy(ctx context.Context, uuid string, policyReq *models.UpdatePolicyRequest) (*models.Policy, error) {
	logFields := map[string]any{"uuid": uuid, "revision": policyReq.CurrentRevision}
	tflog.Debug(ctx, "Executing SDK Call: Update Policy", logFields)

	params := policies.NewUpdatePolicyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(uuid).
		WithBody(policyReq)

	resp, err := c.sdkClient.Policies.UpdatePolicy(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdatePolicy", uuid)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Policy", map[string]any{"uuid": uuid, "new_revision": resp.Payload.UUID}) // resp.Payload should be a models.Policy, it has UUID not RevisionNumber directly. The Policy model does not show RevisionNumber.
	return resp.Payload, nil
}

func (c *SdkClientWrapper) DeletePolicy(ctx context.Context, uuid string) error {
	logFields := map[string]any{"uuid": uuid}
	tflog.Debug(ctx, "Executing SDK Call: Delete Policy", logFields)

	params := policies.NewDeletePolicyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(uuid)

	_, err := c.sdkClient.Policies.DeletePolicy(params, nil) // DeletePolicyOK has an empty payload (usually)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeletePolicy", uuid)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Policy Not Found during Delete (Idempotent Success)", logFields)
			return nil // Treat NotFound as success
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Policy", logFields)
	return nil
}
