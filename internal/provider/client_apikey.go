package provider

import (
	"context"
	"errors"
	"fmt"

	apikeys "github.com/groundcover-com/groundcover-sdk-go/sdk/api/rbac/apikeys"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- API Key Methods ---

func (c *SdkClientWrapper) CreateApiKey(ctx context.Context, req *apikeys.CreateApiKeyRequest) (*apikeys.CreateApiKeyResponse, error) {
	logFields := map[string]any{"name": req.Name, "serviceAccountId": req.ServiceAccountId}
	tflog.Debug(ctx, "Executing SDK Call: Create API Key", logFields)

	// Add detailed debug log for the request payload
	tflog.Debug(ctx, fmt.Sprintf("Sending CreateApiKeyRequest to SDK: %+v", req), logFields)

	resp, err := c.sdkClient.Rbac.Apikeys.CreateApiKey(ctx, req)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateApiKey", req.Name) // Use Name as identifier before ID is known
	}

	tflog.Debug(ctx, "SDK Call Successful: Create API Key", map[string]any{"id": resp.Id})
	return resp, nil
}

func (c *SdkClientWrapper) ListApiKeys(ctx context.Context, withRevoked *bool, withExpired *bool) ([]apikeys.ListApiKeysResponseItem, error) {
	logFields := map[string]any{"withRevoked": withRevoked, "withExpired": withExpired}
	tflog.Debug(ctx, "Executing SDK Call: List API Keys", logFields)

	resp, err := c.sdkClient.Rbac.Apikeys.ListApiKeys(ctx, withRevoked, withExpired)
	if err != nil {
		return nil, handleApiError(ctx, err, "ListApiKeys", "-") // No specific ID for list
	}

	tflog.Debug(ctx, "SDK Call Successful: List API Keys", map[string]any{"count": len(resp)})
	return resp, nil
}

func (c *SdkClientWrapper) DeleteApiKey(ctx context.Context, id string) error {
	logFields := map[string]any{"id": id}
	tflog.Debug(ctx, "Executing SDK Call: Delete API Key", logFields)

	_, err := c.sdkClient.Rbac.Apikeys.DeleteApiKey(ctx, id)
	if err != nil {
		mappedErr := handleApiError(ctx, err, "DeleteApiKey", id)
		// Check if the error is NotFound, return nil in that case as deletion is idempotent
		if errors.Is(mappedErr, ErrNotFound) {
			tflog.Warn(ctx, "SDK Call Result: API Key Not Found during Delete (Idempotent Success)", logFields)
			return nil
		}
		return mappedErr
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete API Key", logFields)
	return nil
}
