package provider

import (
	"context"

	tracesPipelineClient "github.com/groundcover-com/groundcover-sdk-go/pkg/client/traces_pipeline"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const tracesPipelineResourceId = "traces-pipeline"

// CreateTracesPipeline creates a new traces pipeline configuration
func (c *SdkClientWrapper) CreateTracesPipeline(ctx context.Context, req *models.CreateOrUpdateTracesPipelineConfigRequest) (*models.TracesPipelineConfig, error) {
	logFields := map[string]any{"req": "create_traces_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Create Traces Pipeline", logFields)

	createParams := tracesPipelineClient.NewCreateTracesPipelineConfigParamsWithContext(ctx).WithBody(req)
	createResp, err := c.sdkClient.TracesPipeline.CreateTracesPipelineConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateTracesPipeline", tracesPipelineResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Traces Pipeline", logFields)
	return createResp.Payload, nil
}

// GetTracesPipeline retrieves a traces pipeline configuration by key
func (c *SdkClientWrapper) GetTracesPipeline(ctx context.Context) (*models.TracesPipelineConfig, error) {
	logFields := map[string]any{"req": "get_traces_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Get Traces Pipeline", logFields)

	getParams := tracesPipelineClient.NewGetTracesPipelineConfigParamsWithContext(ctx)
	getResp, emptyGetResp, err := c.sdkClient.TracesPipeline.GetTracesPipelineConfig(getParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetTracesPipeline", tracesPipelineResourceId)
	}

	var response *models.TracesPipelineConfig
	if emptyGetResp == nil {
		response = getResp.Payload
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Traces Pipeline", logFields)
	return response, nil
}

// UpdateTracesPipeline updates an existing traces pipeline configuration
func (c *SdkClientWrapper) UpdateTracesPipeline(ctx context.Context, req *models.CreateOrUpdateTracesPipelineConfigRequest) (*models.TracesPipelineConfig, error) {
	logFields := map[string]any{"req": "update_traces_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Update Traces Pipeline", logFields)

	updateParams := tracesPipelineClient.NewUpdateTracesPipelineConfigParamsWithContext(ctx).WithBody(req)
	updateResp, err := c.sdkClient.TracesPipeline.UpdateTracesPipelineConfig(updateParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateTracesPipeline", tracesPipelineResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Traces Pipeline", logFields)
	return updateResp.Payload, nil
}

// DeleteTracesPipeline deletes a traces pipeline configuration by key
func (c *SdkClientWrapper) DeleteTracesPipeline(ctx context.Context) error {
	logFields := map[string]any{"req": "delete_traces_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Delete Traces Pipeline", logFields)

	deleteParams := tracesPipelineClient.NewDeleteTracesPipelineConfigParamsWithContext(ctx)
	_, err := c.sdkClient.TracesPipeline.DeleteTracesPipelineConfig(deleteParams, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteTracesPipeline", tracesPipelineResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Traces Pipeline", logFields)
	return nil
}
