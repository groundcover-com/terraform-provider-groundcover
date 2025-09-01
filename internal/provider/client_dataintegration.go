package provider

import (
	"context"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/integrations"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	dataIntegrationEndpoint   = "/api/integrations/data/config"
	dataIntegrationResourceId = "dataintegration"
)

// CreateDataIntegration creates a new data integration configuration
func (c *SdkClientWrapper) CreateDataIntegration(ctx context.Context, integrationType string, req *models.CreateDataIntegrationConfigRequest) (*models.DataIntegrationConfig, error) {
	logFields := map[string]any{"req": "create_data_integration", "type": integrationType}
	tflog.Debug(ctx, "Executing SDK Call: Create DataIntegration", logFields)

	createParams := integrations.NewCreateDataIntegrationConfigParamsWithContext(ctx).WithType(integrationType).WithBody(req)
	createResp, err := c.sdkClient.Integrations.CreateDataIntegrationConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateDataIntegration", dataIntegrationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create DataIntegration", logFields)
	return createResp.Payload, nil
}

// GetDataIntegration retrieves a data integration configuration by type and ID
func (c *SdkClientWrapper) GetDataIntegration(ctx context.Context, integrationType string, id string) (*models.DataIntegrationConfig, error) {
	logFields := map[string]any{"req": "get_data_integration", "id": id, "type": integrationType}
	tflog.Debug(ctx, "Executing SDK Call: Get DataIntegration", logFields)

	getParams := integrations.NewGetDataIntegrationConfigParamsWithContext(ctx).WithType(integrationType).WithID(id)
	getResp, emptyGetResp, err := c.sdkClient.Integrations.GetDataIntegrationConfig(getParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetDataIntegration", dataIntegrationResourceId)
	}

	var response *models.DataIntegrationConfig
	if emptyGetResp == nil {
		response = getResp.Payload
	}

	tflog.Debug(ctx, "SDK Call Successful: Get DataIntegration", logFields)
	return response, nil
}

// UpdateDataIntegration updates an existing data integration configuration
func (c *SdkClientWrapper) UpdateDataIntegration(ctx context.Context, integrationType string, id string, req *models.CreateDataIntegrationConfigRequest) (*models.DataIntegrationConfig, error) {
	logFields := map[string]any{"req": "update_data_integration", "id": id, "type": integrationType}
	tflog.Debug(ctx, "Executing SDK Call: Update DataIntegration", logFields)

	updateParams := integrations.NewUpdateDataIntegrationConfigParamsWithContext(ctx).WithType(integrationType).WithID(id).WithBody(req)
	updateResp, err := c.sdkClient.Integrations.UpdateDataIntegrationConfig(updateParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateDataIntegration", dataIntegrationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update DataIntegration", logFields)
	return updateResp.Payload, nil
}

// DeleteDataIntegration deletes a data integration configuration by type and ID
func (c *SdkClientWrapper) DeleteDataIntegration(ctx context.Context, integrationType string, id string, env string, cluster string, instance string) error {
	logFields := map[string]any{"req": "delete_data_integration", "id": id, "type": integrationType, "env": env, "cluster": cluster, "instance": instance}
	tflog.Debug(ctx, "Executing SDK Call: Delete DataIntegration", logFields)

	deleteParams := integrations.NewDeleteDataIntegrationConfigParamsWithContext(ctx).WithType(integrationType).WithID(id)

	// Add optional parameters if they are provided
	if env != "" {
		deleteParams = deleteParams.WithEnv(&env)
	}
	if cluster != "" {
		deleteParams = deleteParams.WithCluster(&cluster)
	}
	if instance != "" {
		deleteParams = deleteParams.WithInstance(&instance)
	}

	_, err := c.sdkClient.Integrations.DeleteDataIntegrationConfig(deleteParams, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteDataIntegration", dataIntegrationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete DataIntegration", logFields)
	return nil
}
