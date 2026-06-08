package provider

import (
	"context"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

func (c *SdkClientWrapper) CreateMonitorV2(ctx context.Context, req *models.CreateMonitorRequest) (*models.CreateMonitorResponse, error) {
	return c.CreateMonitor(ctx, req)
}

func (c *SdkClientWrapper) GetMonitorV2(ctx context.Context, id string) ([]byte, error) {
	return c.GetMonitor(ctx, id)
}

func (c *SdkClientWrapper) UpdateMonitorV2(ctx context.Context, id string, req *models.UpdateMonitorRequest) error {
	return c.UpdateMonitor(ctx, id, req)
}

func (c *SdkClientWrapper) DeleteMonitorV2(ctx context.Context, id string) error {
	return c.DeleteMonitor(ctx, id)
}
