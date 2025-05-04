// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	sdkapi "github.com/groundcover-com/groundcover-sdk-go/sdk/api"
	monitors "github.com/groundcover-com/groundcover-sdk-go/sdk/api/monitors"
	apikeys "github.com/groundcover-com/groundcover-sdk-go/sdk/api/rbac/apikeys"
	policies "github.com/groundcover-com/groundcover-sdk-go/sdk/api/rbac/policies"
	serviceaccounts "github.com/groundcover-com/groundcover-sdk-go/sdk/api/rbac/serviceaccounts"
	"github.com/groundcover-com/groundcover-sdk-go/sdk/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Standard provider error types mapped from API responses.
var (
	ErrNotFound    = errors.New("resource not found")
	ErrConcurrency = errors.New("concurrency conflict detected")
	ErrReadOnly    = errors.New("resource is read-only")
)

// ApiClient defines the interface for interacting with the Groundcover API for Terraform resources.
type ApiClient interface {
	// Policies
	CreatePolicy(ctx context.Context, req policies.CreatePolicyRequest) (*policies.Policy, error)
	GetPolicy(ctx context.Context, uuid string) (*policies.Policy, error)
	UpdatePolicy(ctx context.Context, uuid string, req policies.UpdatePolicyRequest) (*policies.Policy, error)
	DeletePolicy(ctx context.Context, uuid string) error

	// Service Accounts
	CreateServiceAccount(ctx context.Context, req serviceaccounts.CreateServiceAccountRequest) (*serviceaccounts.CreateServiceAccountResponse, error)
	ListServiceAccounts(ctx context.Context) ([]serviceaccounts.ListServiceAccountsResponseItem, error)
	UpdateServiceAccount(ctx context.Context, id string, req serviceaccounts.UpdateServiceAccountRequest) (*serviceaccounts.UpdateServiceAccountResponse, error)
	DeleteServiceAccount(ctx context.Context, id string) error

	// Monitors (YAML based)
	CreateMonitorYaml(ctx context.Context, monitorYaml []byte) (*monitors.CreateMonitorResponse, error)
	GetMonitor(ctx context.Context, id string) ([]byte, error)
	UpdateMonitorYaml(ctx context.Context, id string, monitorYaml []byte) (*models.EmptyResponse, error)
	DeleteMonitor(ctx context.Context, id string) error // Removed *models.EmptyResponse, Delete should just return error

	// API Keys
	CreateApiKey(ctx context.Context, req *apikeys.CreateApiKeyRequest) (*apikeys.CreateApiKeyResponse, error)
	ListApiKeys(ctx context.Context, withRevoked *bool, withExpired *bool) ([]apikeys.ListApiKeysResponseItem, error)
	DeleteApiKey(ctx context.Context, id string) error
}

// SdkClientWrapper implements ApiClient using the Groundcover Go SDK.
type SdkClientWrapper struct {
	sdkClient *sdkapi.Client // Use the api.Client from the imported sdkapi package
}

var _ ApiClient = (*SdkClientWrapper)(nil)

// NewSdkClientWrapper creates a new API client wrapper.
func NewSdkClientWrapper(sdkClient *sdkapi.Client) (ApiClient, error) {
	if sdkClient == nil {
		return nil, errors.New("SDK client cannot be nil")
	}
	return &SdkClientWrapper{sdkClient: sdkClient}, nil
}

// statusCodeRegex extracts the HTTP status code from SDK error strings.
var statusCodeRegex = regexp.MustCompile(`status code (\d+)`)

// handleApiError maps SDK error strings to standard provider errors (ErrNotFound, etc.)
// or returns a wrapped generic error if no specific mapping applies.
func handleApiError(ctx context.Context, err error, operation string, resourceId string) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	logFields := map[string]any{
		"operation":   operation,
		"resource_id": resourceId,
		"sdk_error":   errStr,
	}

	tflog.Error(ctx, "SDK Error occurred (pre-mapping)", logFields)

	lowerErrStr := strings.ToLower(errStr)
	statusCode := -1

	// Attempt to extract status code from the error string using regex
	// This is a fallback as the specific StatusCodeError type seems problematic.
	match := statusCodeRegex.FindStringSubmatch(errStr)
	if len(match) > 1 {
		extractedCode, parseErr := strconv.Atoi(match[1])
		if parseErr == nil {
			statusCode = extractedCode
			logFields["extracted_status_code_regex"] = statusCode
			tflog.Info(ctx, "Extracted status code from error string via regex", logFields)
		} else {
			tflog.Warn(ctx, "Failed to parse status code from SDK error string via regex", logFields)
		}
	}

	// --- Specific Error Mapping ---

	// Handle 409 Conflict specifically for CreateServiceAccount (name collision)
	if operation == "CreateServiceAccount" && (statusCode == http.StatusConflict || strings.Contains(lowerErrStr, "conflict")) {
		tflog.Warn(ctx, "Detected 409 Conflict during CreateServiceAccount, likely name collision.", logFields)
		return fmt.Errorf("service account name '%s' was previously used or is currently in use. Please choose a different name", resourceId)
	}

	// Handle 409 Conflict specifically for CreateApiKey (name collision)
	if operation == "CreateApiKey" && (statusCode == http.StatusConflict || strings.Contains(lowerErrStr, "conflict")) {
		tflog.Warn(ctx, "Detected 409 Conflict during CreateApiKey, likely name collision.", logFields)
		// resourceId here is the proposed name from the request
		return fmt.Errorf("API Key name '%s' was previously used or is currently in use. Please choose a different name", resourceId)
	}

	// Handle 404 Not Found for state management (all operations where it matters)
	if statusCode == http.StatusNotFound {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound based on 404 status code.", logFields)
		return ErrNotFound
	}
	// Fallback for cases where status code extraction might fail but text indicates not found
	// Check the error string directly for "404" or "not found"
	if strings.Contains(lowerErrStr, "not found") || strings.Contains(errStr, " 404 ") {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound based on substring match ('not found' or ' 404 ').", logFields)
		return ErrNotFound
	}

	// Handle read-only errors
	if strings.Contains(lowerErrStr, "read-only") || strings.Contains(lowerErrStr, "read only") {
		tflog.Warn(ctx, "Mapping SDK error to ErrReadOnly based on substring match.", logFields)
		return ErrReadOnly
	}

	// Handle 409 Conflict for UpdatePolicy (concurrency)
	if operation == "UpdatePolicy" && (statusCode == http.StatusConflict || strings.Contains(lowerErrStr, "conflict")) {
		tflog.Warn(ctx, "Mapping SDK error to ErrConcurrency based on status code or substring match (UpdatePolicy).", logFields)
		return ErrConcurrency
	}

	// --- Generic Error Wrapping ---
	tflog.Warn(ctx, "SDK error did not match specific mappings, wrapping original error.", logFields)
	return fmt.Errorf("%s failed: %w", operation, err)
}
