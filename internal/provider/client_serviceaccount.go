package provider

import (
	"context"
	"errors"

	sdkSA "github.com/groundcover-com/groundcover-sdk-go/sdk/api/rbac/serviceaccounts"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateServiceAccount(ctx context.Context, req sdkSA.CreateServiceAccountRequest) (*sdkSA.CreateServiceAccountResponse, error) {
	identifier := req.Name
	logFields := map[string]any{"name": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Create Service Account", logFields)

	resp, err := c.sdkClient.Rbac.Serviceaccounts.CreateServiceAccount(ctx, &req)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateServiceAccount", identifier)
	}

	respId := resp.ServiceAccountId
	if respId == "" {
		respId = "<empty_id>"
		tflog.Warn(ctx, "CreateServiceAccount response contained an empty ID", logFields)
	}
	tflog.Debug(ctx, "SDK Call Successful: Create Service Account", map[string]any{"id": respId})
	return resp, nil
}

func (c *SdkClientWrapper) ListServiceAccounts(ctx context.Context) ([]sdkSA.ListServiceAccountsResponseItem, error) {
	tflog.Debug(ctx, "Executing SDK Call: List Service Accounts")

	resp, err := c.sdkClient.Rbac.Serviceaccounts.ListServiceAccounts(ctx)
	if err != nil {
		return nil, handleApiError(ctx, err, "ListServiceAccounts", "")
	}

	tflog.Debug(ctx, "SDK Call Successful: List Service Accounts", map[string]any{"count": len(resp)})
	return resp, nil
}

func (c *SdkClientWrapper) UpdateServiceAccount(ctx context.Context, id string, req sdkSA.UpdateServiceAccountRequest) (*sdkSA.UpdateServiceAccountResponse, error) {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Update Service Account", logFields)

	resp, err := c.sdkClient.Rbac.Serviceaccounts.UpdateServiceAccount(ctx, &req)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateServiceAccount", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Service Account", logFields)
	return resp, nil
}

func (c *SdkClientWrapper) DeleteServiceAccount(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Service Account", logFields)

	_, err := c.sdkClient.Rbac.Serviceaccounts.DeleteServiceAccount(ctx, id)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteServiceAccount", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Service Account Not Found during Delete (Idempotent Success)", logFields)
			return nil // Treat NotFound as success
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Service Account", logFields)
	return nil
}
