package provider

import (
	"context"

	metricsPipelineClient "github.com/groundcover-com/groundcover-sdk-go/pkg/client/metrics_pipeline"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	metricsPipelineResourceId = "metrics-pipeline"
)

func (c *SdkClientWrapper) CreateMetricsPipeline(ctx context.Context, req *models.CreateOrUpdateMetricsPipelineConfigRequest) (*models.MetricsPipelineConfigInfo, error) {
	logFields := map[string]any{"req": "create_metrics_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Create Metrics Pipeline", logFields)

	createParams := metricsPipelineClient.NewCreateMetricsPipelineConfigParamsWithContext(ctx).WithBody(req)
	createResp, err := c.sdkClient.MetricsPipeline.CreateMetricsPipelineConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateMetricsPipeline", metricsPipelineResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Metrics Pipeline", logFields)
	return createResp.Payload, nil
}

func (c *SdkClientWrapper) GetMetricsPipeline(ctx context.Context) (*models.MetricsPipelineConfigInfo, error) {
	logFields := map[string]any{"req": "get_metrics_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Get Metrics Pipeline", logFields)

	getParams := metricsPipelineClient.NewGetMetricsPipelineConfigParamsWithContext(ctx)
	getResp, emptyGetResp, err := c.sdkClient.MetricsPipeline.GetMetricsPipelineConfig(getParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetMetricsPipeline", metricsPipelineResourceId)
	}

	var response *models.MetricsPipelineConfigInfo
	if emptyGetResp == nil {
		response = getResp.Payload
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Metrics Pipeline", logFields)
	return response, nil
}

func (c *SdkClientWrapper) UpdateMetricsPipeline(ctx context.Context, req *models.CreateOrUpdateMetricsPipelineConfigRequest) (*models.MetricsPipelineConfigInfo, error) {
	logFields := map[string]any{"req": "update_metrics_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Update Metrics Pipeline", logFields)

	updateParams := metricsPipelineClient.NewUpdateMetricsPipelineConfigParamsWithContext(ctx).WithBody(req)
	updateResp, err := c.sdkClient.MetricsPipeline.UpdateMetricsPipelineConfig(updateParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateMetricsPipeline", metricsPipelineResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Metrics Pipeline", logFields)
	return updateResp.Payload, nil
}

func (c *SdkClientWrapper) DeleteMetricsPipeline(ctx context.Context) error {
	logFields := map[string]any{"req": "delete_metrics_pipeline"}
	tflog.Debug(ctx, "Executing SDK Call: Delete Metrics Pipeline", logFields)

	deleteParams := metricsPipelineClient.NewDeleteMetricsPipelineConfigParamsWithContext(ctx)
	_, err := c.sdkClient.MetricsPipeline.DeleteMetricsPipelineConfig(deleteParams, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteMetricsPipeline", metricsPipelineResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Metrics Pipeline", logFields)
	return nil
}
