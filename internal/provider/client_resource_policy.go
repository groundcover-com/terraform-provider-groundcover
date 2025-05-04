package provider

import (
	"context"
	"errors"

	sdkPoliciesReq "github.com/groundcover-com/groundcover-sdk-go/sdk/api/rbac/policies"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreatePolicy(ctx context.Context, req sdkPoliciesReq.CreatePolicyRequest) (*sdkPoliciesReq.Policy, error) {
	logFields := map[string]any{"name": req.Name}
	tflog.Debug(ctx, "Executing SDK Call: Create Policy", logFields)

	policy, err := c.sdkClient.Rbac.Policies.CreatePolicy(ctx, &req)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreatePolicy", req.Name)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Policy", map[string]any{"uuid": policy.UUID})
	return policy, nil
}

func (c *SdkClientWrapper) GetPolicy(ctx context.Context, uuid string) (*sdkPoliciesReq.Policy, error) {
	logFields := map[string]any{"uuid": uuid}
	tflog.Debug(ctx, "Executing SDK Call: Get Policy", logFields)

	policy, err := c.sdkClient.Rbac.Policies.GetPolicy(ctx, uuid)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetPolicy", uuid)
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Policy", logFields)
	return policy, nil
}

func (c *SdkClientWrapper) UpdatePolicy(ctx context.Context, uuid string, req sdkPoliciesReq.UpdatePolicyRequest) (*sdkPoliciesReq.Policy, error) {
	logFields := map[string]any{"uuid": uuid, "revision": req.CurrentRevision}
	tflog.Debug(ctx, "Executing SDK Call: Update Policy", logFields)

	policy, err := c.sdkClient.Rbac.Policies.UpdatePolicy(ctx, uuid, &req)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdatePolicy", uuid)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Policy", map[string]any{"uuid": uuid, "new_revision": policy.RevisionNumber})
	return policy, nil
}

func (c *SdkClientWrapper) DeletePolicy(ctx context.Context, uuid string) error {
	logFields := map[string]any{"uuid": uuid}
	tflog.Debug(ctx, "Executing SDK Call: Delete Policy", logFields)

	_, err := c.sdkClient.Rbac.Policies.DeletePolicy(ctx, uuid)
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
