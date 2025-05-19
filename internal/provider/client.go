// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	// NEW SDK IMPORTS
	goclient "github.com/groundcover-com/groundcover-sdk-go/pkg/client"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"                    // Assuming models are here
	gcsdk_transport "github.com/groundcover-com/groundcover-sdk-go/pkg/transport" // Aliased to avoid conflict

	"github.com/go-openapi/runtime"
	apiruntime "github.com/go-openapi/runtime"            // Used for APIError type assertion
	openapi_client "github.com/go-openapi/runtime/client" // Used for New() and transport debugging
	"github.com/go-openapi/runtime/logger"                // Used for the logger.Logger interface
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
	defaultTimeout    = 30 * time.Second
	defaultRetryCount = 5
	minRetryWait      = 1 * time.Second
	maxRetryWait      = 5 * time.Second
	yamlContentType   = "application/x-yaml" // Added for consistency
)

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
}

// SdkClientWrapper implements ApiClient using the Groundcover Go SDK.
type SdkClientWrapper struct {
	sdkClient *goclient.GroundcoverAPI
}

var _ ApiClient = (*SdkClientWrapper)(nil)

var getMonitorPathRegex = regexp.MustCompile(`^/api/monitors/[^/]+/?$`)

// fixMonitorContentTypeTransport wraps an http.RoundTripper to ensure that
// successful GET responses for specific monitor YAML endpoints have the
// Content-Type header set to "application/x-yaml".
type fixMonitorContentTypeTransport struct {
	transport http.RoundTripper
}

func (f *fixMonitorContentTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
		return nil, errors.New("GC_BASE_URL (api_url) environment variable or provider config is required")
	}
	if apiKey == "" {
		return nil, errors.New("GC_API_KEY (api_key) environment variable or provider config is required")
	}
	if backendID == "" {
		return nil, errors.New("GC_BACKEND_ID (org_name) environment variable or provider config is required")
	}

	tflog.Info(ctx, "Initializing Groundcover SDK v1.1.0 client", map[string]any{"baseURL": baseURLStr, "backendID": backendID})

	parsedURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing GC_BASE_URL: %w", err)
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

	monitorContentTypeFixer := &fixMonitorContentTypeTransport{
		transport: sdkTransportWrapper,
	}

	finalRuntimeTransport := openapi_client.New(host, basePath, schemes)
	finalRuntimeTransport.Transport = monitorContentTypeFixer // Inject our fixer here

	// Add a consumer for application/x-yaml to handle raw YAML responses.
	// This will be used if the Content-Type is correctly identified as application/x-yaml (due to our fixer).
	finalRuntimeTransport.Consumers[yamlContentType] = runtime.ByteStreamConsumer()

	// Configure go-openapi to use tflog for its debug messages
	finalRuntimeTransport.SetLogger(&tflogAdapter{ctx: ctx})
	finalRuntimeTransport.SetDebug(true)

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
		return fmt.Errorf("Policy name '%s' was previously used or is currently in use. Please choose a different name", resourceId)
	}

	if statusCode == http.StatusNotFound {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound based on 404 status code.", logFields)
		return ErrNotFound
	}
	if strings.Contains(lowerErrStr, "not found") || strings.Contains(errStr, " 404 ") {
		tflog.Warn(ctx, "Mapping SDK error to ErrNotFound based on substring match ('not found' or ' 404 ').", logFields)
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
