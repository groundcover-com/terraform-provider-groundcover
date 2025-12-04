package provider

import (
	"context"

	metricsAggregatorClient "github.com/groundcover-com/groundcover-sdk-go/pkg/client/aggregations_metrics"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	metricsAggregatorResourceId = "metrics-aggregator"
)

// CreateMetricsAggregator creates a new metrics aggregator configuration
func (c *SdkClientWrapper) CreateMetricsAggregator(ctx context.Context, req *models.CreateOrUpdateMetricsAggregatorConfigRequest) (*models.MetricsAggregatorConfig, error) {
	logFields := map[string]any{"req": "create_metrics_aggregator"}
	tflog.Debug(ctx, "Executing SDK Call: Create Metrics Aggregator", logFields)

	createParams := metricsAggregatorClient.NewCreateMetricsAggregatorConfigParamsWithContext(ctx).WithBody(req)
	createResp, err := c.sdkClient.AggregationsMetrics.CreateMetricsAggregatorConfig(createParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateMetricsAggregator", metricsAggregatorResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Metrics Aggregator", logFields)
	return createResp.Payload, nil
}

// GetMetricsAggregator retrieves the metrics aggregator configuration
func (c *SdkClientWrapper) GetMetricsAggregator(ctx context.Context) (*models.MetricsAggregatorConfig, error) {
	logFields := map[string]any{"req": "get_metrics_aggregator"}
	tflog.Debug(ctx, "Executing SDK Call: Get Metrics Aggregator", logFields)

	getParams := metricsAggregatorClient.NewGetMetricsAggregatorConfigParamsWithContext(ctx)
	getResp, emptyGetResp, err := c.sdkClient.AggregationsMetrics.GetMetricsAggregatorConfig(getParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetMetricsAggregator", metricsAggregatorResourceId)
	}

	var response *models.MetricsAggregatorConfig
	if emptyGetResp == nil {
		response = getResp.Payload
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Metrics Aggregator", logFields)
	return response, nil
}

// UpdateMetricsAggregator updates an existing metrics aggregator configuration
func (c *SdkClientWrapper) UpdateMetricsAggregator(ctx context.Context, req *models.CreateOrUpdateMetricsAggregatorConfigRequest) (*models.MetricsAggregatorConfig, error) {
	logFields := map[string]any{"req": "update_metrics_aggregator"}
	tflog.Debug(ctx, "Executing SDK Call: Update Metrics Aggregator", logFields)

	updateParams := metricsAggregatorClient.NewUpdateMetricsAggregatorConfigParamsWithContext(ctx).WithBody(req)
	updateResp, err := c.sdkClient.AggregationsMetrics.UpdateMetricsAggregatorConfig(updateParams, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateMetricsAggregator", metricsAggregatorResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Metrics Aggregator", logFields)
	return updateResp.Payload, nil
}

// DeleteMetricsAggregator deletes the metrics aggregator configuration
func (c *SdkClientWrapper) DeleteMetricsAggregator(ctx context.Context) error {
	logFields := map[string]any{"req": "delete_metrics_aggregator"}
	tflog.Debug(ctx, "Executing SDK Call: Delete Metrics Aggregator", logFields)

	deleteParams := metricsAggregatorClient.NewDeleteMetricsAggregatorConfigParamsWithContext(ctx)
	_, err := c.sdkClient.AggregationsMetrics.DeleteMetricsAggregatorConfig(deleteParams, nil)
	if err != nil {
		return handleApiError(ctx, err, "DeleteMetricsAggregator", metricsAggregatorResourceId)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Metrics Aggregator", logFields)
	return nil
}
