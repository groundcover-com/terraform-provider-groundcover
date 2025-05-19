package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/serviceaccounts"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) CreateServiceAccount(ctx context.Context, saReq *models.CreateServiceAccountRequest) (*models.ServiceAccountCreatePayload, error) {
	identifier := "<unknown>"
	if saReq.Name != nil {
		identifier = *saReq.Name
	}
	logFields := map[string]any{"name": identifier}
	tflog.Debug(ctx, "Executing SDK Call: Create Service Account", logFields)

	params := serviceaccounts.NewCreateServiceAccountParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(saReq)

	resp, err := c.sdkClient.Serviceaccounts.CreateServiceAccount(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateServiceAccount", identifier)
	}

	respId := "<empty_id>"
	if resp.Payload != nil && resp.Payload.ServiceAccountID != nil {
		respId = *resp.Payload.ServiceAccountID
	} else if resp.Payload != nil {
		tflog.Warn(ctx, "CreateServiceAccount response payload contained a nil ServiceAccountID", logFields)
	} else {
		tflog.Warn(ctx, "CreateServiceAccount response payload was nil", logFields)
	}
	tflog.Debug(ctx, "SDK Call Successful: Create Service Account", map[string]any{"id": respId})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) ListServiceAccounts(ctx context.Context) ([]*models.ServiceAccountsWithPolicy, error) {
	tflog.Debug(ctx, "Executing SDK Call: List Service Accounts")

	params := serviceaccounts.NewListServiceAccountsParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout)

	resp, err := c.sdkClient.Serviceaccounts.ListServiceAccounts(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "ListServiceAccounts", "")
	}

	tflog.Debug(ctx, "SDK Call Successful: List Service Accounts", map[string]any{"count": len(resp.Payload)})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateServiceAccount(ctx context.Context, id string, saReq *models.UpdateServiceAccountRequest) (*models.ServiceAccountsWithPolicy, error) {
	if saReq.ServiceAccountID == nil || *saReq.ServiceAccountID == "" {
		saReq.ServiceAccountID = &id
	} else if *saReq.ServiceAccountID != id {
		return nil, fmt.Errorf("mismatch between ID parameter ('%s') and ServiceAccountID in request body ('%s') for UpdateServiceAccount", id, *saReq.ServiceAccountID)
	}

	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Update Service Account", logFields)

	params := serviceaccounts.NewUpdateServiceAccountParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(saReq)

	_, err := c.sdkClient.Serviceaccounts.UpdateServiceAccount(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateServiceAccount", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Service Account", logFields)

	tflog.Debug(ctx, "Re-fetching service account after update to return full details", logFields)
	saList, err := c.ListServiceAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts after update for ID %s: %w", id, err)
	}

	for _, sa := range saList {
		if sa != nil && sa.ServiceAccountID == id {
			tflog.Debug(ctx, "Found updated service account in list", map[string]any{"id": id})
			return sa, nil
		}
	}

	tflog.Warn(ctx, "Updated service account not found in list after update", map[string]any{"id": id})
	return nil, fmt.Errorf("service account with ID '%s' updated but not found in subsequent list operation", id)
}

func (c *SdkClientWrapper) DeleteServiceAccount(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete Service Account", logFields)

	params := serviceaccounts.NewDeleteServiceAccountParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.Serviceaccounts.DeleteServiceAccount(params, nil)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteServiceAccount", id)
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: Service Account Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Service Account", logFields)
	return nil
}
