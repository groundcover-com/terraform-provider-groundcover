// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	// NEW SDK IMPORTS
	goclient "github.com/groundcover-com/groundcover-sdk-go/pkg/client"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"                    // Assuming models are here
	gcsdk_transport "github.com/groundcover-com/groundcover-sdk-go/pkg/transport" // Aliased to avoid conflict

	apiruntime "github.com/go-openapi/runtime"
	openapi_client "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/runtime/logger"
	"github.com/go-openapi/strfmt"

	// Terraform specific imports
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// tflogAdapter adapts tflog to the logger.Logger interface expected by go-openapi.
type tflogAdapter struct {
	ctx context.Context
}

func (tla *tflogAdapter) Printf(format string, args ...interface{}) {
	tflog.Debug(tla.ctx, fmt.Sprintf(format, args...), map[string]interface{}{"library": "go-openapi", "level": "printf"})
}

func (tla *tflogAdapter) Debugf(format string, args ...interface{}) {
	tflog.Debug(tla.ctx, fmt.Sprintf(format, args...), map[string]interface{}{"library": "go-openapi", "level": "debugf"})
}

var _ logger.Logger = (*tflogAdapter)(nil) // Verify tflogAdapter implements logger.Logger

// Standard provider error types mapped from API responses.
var (
	ErrNotFound    = errors.New("resource not found")
	ErrConcurrency = errors.New("concurrency conflict detected")
	ErrReadOnly    = errors.New("resource is read-only")
)

const (
	// defaultTimeout is set to 120s to accommodate retry logic with exponential backoff.
	// With 5 retries and backoff delays of ~1s, ~2s, ~4s, ~10s, ~10s plus request times,
	// we need sufficient time for all retry attempts to complete.
	defaultTimeout    = 120 * time.Second
	defaultRetryCount = 5
	minRetryWait      = 1 * time.Second
	maxRetryWait      = 10 * time.Second
	yamlContentType   = "application/x-yaml" // Added for consistency
)

// rateLimitRetryTransport wraps an http.RoundTripper to handle 429 rate limit errors
// with exponential backoff and retry at the HTTP transport level.
// This ensures retries happen before go-openapi processes the response.
type rateLimitRetryTransport struct {
	transport  http.RoundTripper
	maxRetries int
	minWait    time.Duration
	maxWait    time.Duration
}

func (t *rateLimitRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Buffer the request body if present so we can retry
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read request body for retry: %w", err)
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		// Restore the body for each attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err = t.transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// If not a 429, return immediately
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Close the response body before retry
		resp.Body.Close()

		// Don't retry after the last attempt
		if attempt == t.maxRetries {
			// Re-execute to get a fresh response to return
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			return t.transport.RoundTrip(req)
		}

		// Calculate exponential backoff with jitter
		backoff := t.minWait * time.Duration(1<<uint(attempt))
		if backoff > t.maxWait {
			backoff = t.maxWait
		}
		// Add jitter (0-25% of backoff)
		jitter := time.Duration(float64(backoff) * 0.25 * (float64(time.Now().UnixNano()%100) / 100.0))
		delay := backoff + jitter

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return resp, err
}

// ApiClient defines the interface for interacting with the Groundcover API for Terraform resources.
type ApiClient interface {
	// Policies
	CreatePolicy(ctx context.Context, req *models.CreatePolicyRequest) (*models.Policy, error)
	GetPolicy(ctx context.Context, uuid string) (*models.Policy, error)
	UpdatePolicy(ctx context.Context, uuid string, req *models.UpdatePolicyRequest) (*models.Policy, error)
	DeletePolicy(ctx context.Context, uuid string) error

	// Service Accounts
	CreateServiceAccount(ctx context.Context, req *models.CreateServiceAccountRequest) (*models.ServiceAccountCreatePayload, error)
	ListServiceAccounts(ctx context.Context) ([]*models.ServiceAccountsWithPolicy, error)
	UpdateServiceAccount(ctx context.Context, id string, req *models.UpdateServiceAccountRequest) (*models.ServiceAccountsWithPolicy, error)
	DeleteServiceAccount(ctx context.Context, id string) error

	// Monitors (YAML based) - Provider will unmarshal YAML to request models before calling.
	CreateMonitor(ctx context.Context, req *models.CreateMonitorRequest) (*models.CreateMonitorResponse, error)
	GetMonitor(ctx context.Context, id string) ([]byte, error)                            // Returns raw YAML bytes
	UpdateMonitor(ctx context.Context, id string, req *models.UpdateMonitorRequest) error // Update response has no payload
	DeleteMonitor(ctx context.Context, id string) error

	// API Keys
	CreateApiKey(ctx context.Context, req *models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error)
	ListApiKeys(ctx context.Context, withRevoked *bool, withExpired *bool) ([]*models.ListAPIKeysResponseItem, error)
	DeleteApiKey(ctx context.Context, id string) error

	// Logs Pipeline
	CreateLogsPipeline(ctx context.Context, req *models.CreateOrUpdateLogsPipelineConfigRequest) (*models.LogsPipelineConfig, error)
	GetLogsPipeline(ctx context.Context) (*models.LogsPipelineConfig, error)
	UpdateLogsPipeline(ctx context.Context, req *models.CreateOrUpdateLogsPipelineConfigRequest) (*models.LogsPipelineConfig, error)
	DeleteLogsPipeline(ctx context.Context) error

	// Metrics Aggregation
	CreateMetricsAggregation(ctx context.Context, req *models.CreateOrUpdateMetricsAggregatorConfigRequest) (*models.MetricsAggregatorConfig, error)
	GetMetricsAggregation(ctx context.Context) (*models.MetricsAggregatorConfig, error)
	UpdateMetricsAggregation(ctx context.Context, req *models.CreateOrUpdateMetricsAggregatorConfigRequest) (*models.MetricsAggregatorConfig, error)
	DeleteMetricsAggregation(ctx context.Context) error

	// Ingestion Keys
	CreateIngestionKey(ctx context.Context, req *models.CreateIngestionKeyRequest) (*models.IngestionKeyResult, error)
	ListIngestionKeys(ctx context.Context, req *models.ListIngestionKeysRequest) ([]*models.IngestionKeyResult, error)
	DeleteIngestionKey(ctx context.Context, req *models.DeleteIngestionKeyRequest) error

	// Dashboards
	CreateDashboard(ctx context.Context, dashboard *models.CreateDashboardRequest) (*models.View, error)
	GetDashboard(ctx context.Context, uuid string) (*models.View, error)
	UpdateDashboard(ctx context.Context, uuid string, dashboard *models.UpdateDashboardRequest) (*models.View, error)
	DeleteDashboard(ctx context.Context, uuid string) error

	// DataIntegrations
	CreateDataIntegration(ctx context.Context, integrationType string, req *models.CreateDataIntegrationConfigRequest) (*models.DataIntegrationConfig, error)
	GetDataIntegration(ctx context.Context, integrationType string, id string) (*models.DataIntegrationConfig, error)
	UpdateDataIntegration(ctx context.Context, integrationType string, id string, req *models.CreateDataIntegrationConfigRequest) (*models.DataIntegrationConfig, error)
	DeleteDataIntegration(ctx context.Context, integrationType string, id string, cluster *string) error
}

// SdkClientWrapper implements ApiClient using the Groundcover Go SDK.
type SdkClientWrapper struct {
	sdkClient *goclient.GroundcoverAPI
}

var _ ApiClient = (*SdkClientWrapper)(nil)

var getMonitorPathRegex = regexp.MustCompile(`^/api/monitors/[^/]+/?$`)

// overrideYamlContextTypeTransport wraps an http.RoundTripper to ensure that
// successful GET responses for specific monitor YAML endpoints have the
// Content-Type header set to "application/x-yaml".
type overrideYamlContextTypeTransport struct {
	transport http.RoundTripper
}

func (f *overrideYamlContextTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := f.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if req.Method == http.MethodGet && resp.StatusCode == http.StatusOK && getMonitorPathRegex.MatchString(req.URL.Path) {
		currentContentType := resp.Header.Get("Content-Type")
		if currentContentType == "" || !strings.HasPrefix(currentContentType, yamlContentType) {
			resp.Header.Set("Content-Type", yamlContentType)
		}
	}

	return resp, nil
}

func NewSdkClientWrapper(ctx context.Context, baseURLStr, apiKey, backendID string) (ApiClient, error) {
	if baseURLStr == "" {
		return nil, errors.New("GROUNDCOVER_API_URL (api_url) environment variable or provider config is required")
	}
	if apiKey == "" {
		return nil, errors.New("GROUNDCOVER_API_KEY (api_key) environment variable or provider config is required")
	}
	if backendID == "" {
		return nil, errors.New("GROUNDCOVER_BACKEND_ID (backend_id) environment variable or provider config is required")
	}

	userEnabledDebug := os.Getenv("TF_LOG") == "debug"

	tflog.Info(ctx, "Initializing Groundcover SDK client", map[string]any{"baseURL": baseURLStr, "backendID": backendID})

	parsedURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing GROUNDCOVER_API_URL: %w", err)
	}

	host := parsedURL.Host
	basePath := parsedURL.Path
	if basePath == "" || basePath == "/" {
		basePath = goclient.DefaultBasePath
	}
	if !strings.HasPrefix(basePath, "/") && basePath != "" {
		basePath = "/" + basePath
	}

	schemes := []string{parsedURL.Scheme}
	if len(schemes) == 0 || schemes[0] == "" {
		schemes = goclient.DefaultSchemes
	}

	baseHttpTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	retryableStatuses := []int{
		http.StatusServiceUnavailable,
		http.StatusTooManyRequests,
		http.StatusGatewayTimeout,
		http.StatusBadGateway,
	}
	sdkTransportWrapper := gcsdk_transport.NewTransport(
		apiKey,
		backendID,
		baseHttpTransport,
		defaultRetryCount,
		minRetryWait,
		maxRetryWait,
		retryableStatuses,
	)

	// Wrap with rate limit retry transport (handles 429s at HTTP level before go-openapi processes them)
	rateLimitTransport := &rateLimitRetryTransport{
		transport:  sdkTransportWrapper,
		maxRetries: defaultRetryCount,
		minWait:    minRetryWait,
		maxWait:    maxRetryWait,
	}

	monitorContentTypeFixer := &overrideYamlContextTypeTransport{
		transport: rateLimitTransport,
	}

	finalRuntimeTransport := openapi_client.New(host, basePath, schemes)
	finalRuntimeTransport.Transport = monitorContentTypeFixer // Inject our fixer here

	// Add a consumer for application/x-yaml to handle raw YAML responses.
	// This will be used if the Content-Type is correctly identified as application/x-yaml (due to our fixer).
	finalRuntimeTransport.Consumers[yamlContentType] = apiruntime.ByteStreamConsumer()

	// Configure go-openapi to use tflog for its debug messages
	finalRuntimeTransport.SetLogger(&tflogAdapter{ctx: ctx})
	finalRuntimeTransport.SetDebug(userEnabledDebug)

	newSdkClient := goclient.New(finalRuntimeTransport, strfmt.Default)

	return &SdkClientWrapper{sdkClient: newSdkClient}, nil
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
	} else {
		var apiErr *apiruntime.APIError
		if errors.As(err, &apiErr) {
			statusCode = apiErr.Code
			logFields["extracted_status_code_runtime_api_error"] = statusCode
			tflog.Info(ctx, "Extracted status code from runtime.APIError type", logFields)
		}
	}

	// --- Specific Error Mapping ---
	if operation == "CreateServiceAccount" && (statusCode == http.StatusConflict || strings.Contains(lowerErrStr, "conflict")) {
		tflog.Warn(ctx, "Detected 409 Conflict during CreateServiceAccount, likely name collision.", logFields)
		return fmt.Errorf("service account name '%s' was previously used or is currently in use. Please choose a different name", resourceId)
	}

	if operation == "CreateApiKey" && (statusCode == http.StatusConflict || strings.Contains(lowerErrStr, "conflict")) {
		tflog.Warn(ctx, "Detected 409 Conflict during CreateApiKey, likely name collision.", logFields)
		return fmt.Errorf("API Key name '%s' was previously used or is currently in use. Please choose a different name", resourceId)
	}

	if operation == "CreatePolicy" && (statusCode == http.StatusConflict || strings.Contains(lowerErrStr, "conflict")) {
		tflog.Warn(ctx, "Detected 409 Conflict during CreatePolicy, likely name collision.", logFields)
		return fmt.Errorf("policy name '%s' was previously used or is currently in use. Please choose a different name", resourceId)
	}

	if statusCode == http.StatusNotFound {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound based on 404 status code.", logFields)
		return ErrNotFound
	}
	if strings.Contains(lowerErrStr, "not found") || strings.Contains(errStr, " 404 ") || strings.Contains(errStr, "[404]") {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound based on substring match ('not found', ' 404 ', or '[404]').", logFields)
		return ErrNotFound
	}
	// Handle specific case for service account deletion where 400 can mean already deleted or invalid state
	if operation == "DeleteServiceAccount" && (statusCode == http.StatusBadRequest || strings.Contains(errStr, "[400]")) {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound for DeleteServiceAccount 400 error (likely already deleted).", logFields)
		return ErrNotFound
	}

	// Handle specific case for ingestion key deletion where resource not found should be treated as success
	if operation == "DeleteIngestionKey" && (statusCode == http.StatusNotFound || strings.Contains(lowerErrStr, "resource not found") || strings.Contains(lowerErrStr, "not found")) {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound for DeleteIngestionKey not found error (already deleted).", logFields)
		return ErrNotFound
	}

	if strings.Contains(lowerErrStr, "read-only") || strings.Contains(lowerErrStr, "read only") {
		tflog.Warn(ctx, "Mapping SDK error to ErrReadOnly based on substring match.", logFields)
		return ErrReadOnly
	}

	if operation == "UpdatePolicy" && (statusCode == http.StatusConflict || strings.Contains(lowerErrStr, "conflict")) {
		tflog.Warn(ctx, "Mapping SDK error to ErrConcurrency based on status code or substring match (UpdatePolicy).", logFields)
		return ErrConcurrency
	}

	// --- Generic Error Wrapping ---
	tflog.Warn(ctx, "SDK error did not match specific mappings, wrapping original error.", logFields)
	return fmt.Errorf("%s failed: %w", operation, err)
}
