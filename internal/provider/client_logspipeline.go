package provider

import (
	"context"

	logsPipelineClient "github.com/groundcover-com/groundcover-sdk-go/pkg/client/logs_pipeline"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	logsPipelineEndpoint = "/api/pipelines/logs/config"
	resourceId           = "logs-pipeline"
)

// CreateLogsPipeline creates a new logs pipeline configuration
func (c *SdkClientWrapper) CreateLogsPipeline(ctx context.Context, req *models.CreateOrUpdateConfigRequest) (*models.ManageConfigResponseEntry, error) {
	logFields := map[string]any{"req": "create_logs_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Create Logs Pipeline", logFields)

	createParams := logsPipelineClient.NewCreateConfigParamsWithContext(ctx).WithBody(req)
	createResp, err := c.sdkClient.LogsPipeline.CreateConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateLogsPipeline", resourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Logs Pipeline", logFields)
	return createResp.Payload, nil
}

// GetLogsPipeline retrieves a logs pipeline configuration by key
func (c *SdkClientWrapper) GetLogsPipeline(ctx context.Context) (*models.ManageConfigResponseEntry, error) {
	logFields := map[string]any{"req": "get_logs_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Get Logs Pipeline", logFields)

	getParams := logsPipelineClient.NewGetConfigParamsWithContext(ctx)
	getResp, emptyGetResp, err := c.sdkClient.LogsPipeline.GetConfig(getParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetLogsPipeline", resourceId)
	}

	var response *models.ManageConfigResponseEntry
	if emptyGetResp == nil {
		response = getResp.Payload
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Logs Pipeline", logFields)
	return response, nil
}

// UpdateLogsPipeline updates an existing logs pipeline configuration
func (c *SdkClientWrapper) UpdateLogsPipeline(ctx context.Context, req *models.CreateOrUpdateConfigRequest) (*models.ManageConfigResponseEntry, error) {
	logFields := map[string]any{"req": "update_logs_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Update Logs Pipeline", logFields)

	createParams := logsPipelineClient.NewCreateConfigParamsWithContext(ctx).WithBody(req)
	createResp, err := c.sdkClient.LogsPipeline.CreateConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateLogsPipeline", resourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Logs Pipeline", logFields)
	return createResp.Payload, nil
}

// DeleteLogsPipeline deletes a logs pipeline configuration by key
func (c *SdkClientWrapper) DeleteLogsPipeline(ctx context.Context) error {
	logFields := map[string]any{"req": "delete_logs_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Delete Logs Pipeline", logFields)

	deleteParams := logsPipelineClient.NewDeleteConfigParamsWithContext(ctx)
	_, err := c.sdkClient.LogsPipeline.DeleteConfig(deleteParams, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteLogsPipeline", resourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Logs Pipeline", logFields)
	return nil
}
