package provider

import (
	"context"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/integrations"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	integrationEndpoint   = "/api/integrations/config"
	integrationResourceId = "integration"
)

// CreateIntegration creates a new integration configuration
func (c *SdkClientWrapper) CreateIntegration(ctx context.Context, req *models.CreateIntegrationConfigRequest) (*models.IntegrationConfig, error) {
	logFields := map[string]any{"req": "create_integration", "type": req.Type}
	tflog.Debug(ctx, "Executing SDK Call: Create Integration", logFields)

	createParams := integrations.NewCreateIntegrationConfigParamsWithContext(ctx).WithBody(req)
	createResp, err := c.sdkClient.Integrations.CreateIntegrationConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateIntegration", integrationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Integration", logFields)
	return createResp.Payload, nil
}

// GetIntegration retrieves an integration configuration by ID and type
func (c *SdkClientWrapper) GetIntegration(ctx context.Context, integrationType string, id string) (*models.IntegrationConfig, error) {
	logFields := map[string]any{"req": "get_integration", "id": id, "type": integrationType}
	tflog.Debug(ctx, "Executing SDK Call: Get Integration", logFields)

	getParams := integrations.NewGetIntegrationConfigParamsWithContext(ctx).WithID(id).WithType(integrationType)
	getResp, emptyGetResp, err := c.sdkClient.Integrations.GetIntegrationConfig(getParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetIntegration", integrationResourceId)
	}

	var response *models.IntegrationConfig
	if emptyGetResp == nil {
		response = getResp.Payload
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Integration", logFields)
	return response, nil
}

// UpdateIntegration updates an existing integration configuration
func (c *SdkClientWrapper) UpdateIntegration(ctx context.Context, req *models.UpdateIntegrationConfigRequest) (*models.IntegrationConfig, error) {
	logFields := map[string]any{"req": "update_integration", "id": req.ID, "type": req.Type}
	tflog.Debug(ctx, "Executing SDK Call: Update Integration", logFields)

	updateParams := integrations.NewUpdateIntegrationConfigParamsWithContext(ctx).WithBody(req)
	updateResp, err := c.sdkClient.Integrations.UpdateIntegrationConfig(updateParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateIntegration", integrationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Integration", logFields)
	return updateResp.Payload, nil
}

// DeleteIntegration deletes an integration configuration by ID and type
func (c *SdkClientWrapper) DeleteIntegration(ctx context.Context, integrationType string, id string) error {
	logFields := map[string]any{"req": "delete_integration", "id": id, "type": integrationType}
	tflog.Debug(ctx, "Executing SDK Call: Delete Integration", logFields)

	deleteParams := integrations.NewDeleteIntegrationConfigParamsWithContext(ctx).WithID(id).WithType(integrationType)
	_, err := c.sdkClient.Integrations.DeleteIntegrationConfig(deleteParams, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteIntegration", integrationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Integration", logFields)
	return nil
}
