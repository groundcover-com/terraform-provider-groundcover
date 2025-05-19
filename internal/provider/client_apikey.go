package provider

import (
	"context"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/apikeys"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- API Key Methods ---

func (c *SdkClientWrapper) CreateApiKey(ctx context.Context, apiKeyReq *models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	var name string
	if apiKeyReq != nil && apiKeyReq.Name != nil {
		name = *apiKeyReq.Name
	} else {
		name = "<unknown_apikey_name>"
		tflog.Warn(ctx, "CreateApiKey called with nil apiKeyReq or nil Name")
	}
	logFields := map[string]any{"name": name}
	tflog.Debug(ctx, "Executing SDK Call: Create API Key", logFields)

	tflog.Debug(ctx, fmt.Sprintf("Sending CreateAPIKeyRequest to SDK: %+v", apiKeyReq), logFields)

	params := apikeys.NewCreateAPIKeyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithBody(apiKeyReq)

	resp, err := c.sdkClient.Apikeys.CreateAPIKey(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateApiKey", name)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create API Key", map[string]any{"id": resp.Payload.ID})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) ListApiKeys(ctx context.Context, withRevoked *bool, withExpired *bool) ([]*models.ListAPIKeysResponseItem, error) {
	tflog.Debug(ctx, "Executing SDK Call: List API Keys", map[string]any{"withRevoked": withRevoked, "withExpired": withExpired})

	params := apikeys.NewListAPIKeysParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithWithRevoked(withRevoked).
		WithWithExpired(withExpired)

	resp, err := c.sdkClient.Apikeys.ListAPIKeys(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "ListApiKeys", "")
	}

	tflog.Debug(ctx, "SDK Call Successful: List API Keys", map[string]any{"count": len(resp.Payload)})
	return resp.Payload, nil
}

func (c *SdkClientWrapper) DeleteApiKey(ctx context.Context, id string) error {
	tflog.Debug(ctx, "Executing SDK Call: Delete API Key", map[string]any{"id": id})

	params := apikeys.NewDeleteAPIKeyParams().
		WithContext(ctx).
		WithTimeout(defaultTimeout).
		WithID(id)

	_, err := c.sdkClient.Apikeys.DeleteAPIKey(params, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteApiKey", id)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete API Key", map[string]any{"id": id})
	return nil
}
