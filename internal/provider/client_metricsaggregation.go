package provider

import (
	"context"

	metricsAggregationClient "github.com/groundcover-com/groundcover-sdk-go/pkg/client/aggregations_metrics"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	metricsAggregationResourceId = "metrics-aggregation"
)

// CreateMetricsAggregation creates a new metrics aggregation configuration
func (c *SdkClientWrapper) CreateMetricsAggregation(ctx context.Context, req *models.CreateOrUpdateMetricsAggregatorConfigRequest) (*models.MetricsAggregatorConfig, error) {
	logFields := map[string]any{"req": "create_metrics_aggregation"}
	tflog.Debug(ctx, "Executing SDK Call: Create Metrics Aggregation", logFields)

	createParams := metricsAggregationClient.NewCreateMetricsAggregatorConfigParamsWithContext(ctx).WithBody(req)
	createResp, err := c.sdkClient.AggregationsMetrics.CreateMetricsAggregatorConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateMetricsAggregation", metricsAggregationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Metrics Aggregation", logFields)
	return createResp.Payload, nil
}

// GetMetricsAggregation retrieves the metrics aggregation configuration
func (c *SdkClientWrapper) GetMetricsAggregation(ctx context.Context) (*models.MetricsAggregatorConfig, error) {
	logFields := map[string]any{"req": "get_metrics_aggregation"}
	tflog.Debug(ctx, "Executing SDK Call: Get Metrics Aggregation", logFields)

	getParams := metricsAggregationClient.NewGetMetricsAggregatorConfigParamsWithContext(ctx)
	getResp, emptyGetResp, err := c.sdkClient.AggregationsMetrics.GetMetricsAggregatorConfig(getParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetMetricsAggregation", metricsAggregationResourceId)
	}

	var response *models.MetricsAggregatorConfig
	if emptyGetResp == nil {
		response = getResp.Payload
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Metrics Aggregation", logFields)
	return response, nil
}

// UpdateMetricsAggregation updates an existing metrics aggregation configuration
func (c *SdkClientWrapper) UpdateMetricsAggregation(ctx context.Context, req *models.CreateOrUpdateMetricsAggregatorConfigRequest) (*models.MetricsAggregatorConfig, error) {
	logFields := map[string]any{"req": "update_metrics_aggregation"}
	tflog.Debug(ctx, "Executing SDK Call: Update Metrics Aggregation", logFields)

	updateParams := metricsAggregationClient.NewUpdateMetricsAggregatorConfigParamsWithContext(ctx).WithBody(req)
	updateResp, err := c.sdkClient.AggregationsMetrics.UpdateMetricsAggregatorConfig(updateParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateMetricsAggregation", metricsAggregationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Metrics Aggregation", logFields)
	return updateResp.Payload, nil
}

// DeleteMetricsAggregation deletes the metrics aggregation configuration
func (c *SdkClientWrapper) DeleteMetricsAggregation(ctx context.Context) error {
	logFields := map[string]any{"req": "delete_metrics_aggregation"}
	tflog.Debug(ctx, "Executing SDK Call: Delete Metrics Aggregation", logFields)

	deleteParams := metricsAggregationClient.NewDeleteMetricsAggregatorConfigParamsWithContext(ctx)
	_, err := c.sdkClient.AggregationsMetrics.DeleteMetricsAggregatorConfig(deleteParams, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteMetricsAggregation", metricsAggregationResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Metrics Aggregation", logFields)
	return nil
}
