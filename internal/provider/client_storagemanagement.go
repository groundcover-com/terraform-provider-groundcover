// Copyright groundcover 2026
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"

	storageManagementClient "github.com/groundcover-com/groundcover-sdk-go/pkg/client/storage_management"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *SdkClientWrapper) GetStorageManagementPolicy(ctx context.Context, dataType string) (*models.StorageManagementPolicyResponse, error) {
	tflog.Debug(ctx, "Executing SDK Call: Get Storage Management Policy", map[string]any{"data_type": dataType})
	params := storageManagementClient.NewGetStorageManagementPolicyByTypeParams().WithContext(ctx).WithTimeout(defaultTimeout).WithDataType(dataType)
	resp, err := c.sdkClient.StorageManagement.GetStorageManagementPolicyByType(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetStorageManagementPolicy", dataType)
	}
	if resp == nil || resp.Payload == nil {
		return nil, errors.New("get storage management policy response payload was nil")
	}
	return resp.Payload, nil
}

func (c *SdkClientWrapper) UpdateStorageManagementPolicy(ctx context.Context, dataType string, req *models.StorageManagementPolicyRequest) (*models.StorageManagementPolicyResponse, error) {
	tflog.Debug(ctx, "Executing SDK Call: Update Storage Management Policy", map[string]any{"data_type": dataType})
	params := storageManagementClient.NewUpdateStorageManagementPolicyByTypeParams().WithContext(ctx).WithTimeout(defaultTimeout).WithDataType(dataType).WithBody(req)
	resp, err := c.sdkClient.StorageManagement.UpdateStorageManagementPolicyByType(params, nil)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateStorageManagementPolicy", dataType)
	}
	if resp == nil || resp.Payload == nil {
		return nil, errors.New("update storage management policy response payload was nil")
	}
	return resp.Payload, nil
}
