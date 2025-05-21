package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	logsPipelineEndpoint = "/api/pipelines/logs/config"
)

// CreateLogsPipeline creates a new logs pipeline configuration
func (c *SdkClientWrapper) CreateLogsPipeline(ctx context.Context, config *ConfigEntry) (*ConfigEntry, error) {
	logFields := map[string]any{"key": config.Key}
	tflog.Debug(ctx, "Executing SDK Call: Create Logs Pipeline", logFields)

	// Marshal the request to JSON
	reqBody, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("error marshaling logs pipeline config: %w", err)
	}

	// Prepare the HTTP request
	req, err := c.prepareRequest(ctx, http.MethodPost, logsPipelineEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	// Execute the request
	resp, err := c.executeRequest(req)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateLogsPipeline", config.Key)
	}
	defer resp.Body.Close()

	// Read and parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := parseErrorResponse(resp.StatusCode, respBody)
		return nil, handleApiError(ctx, err, "CreateLogsPipeline", config.Key)
	}

	// Parse the response JSON
	var createdConfig ConfigEntry
	if err := json.Unmarshal(respBody, &createdConfig); err != nil {
		return nil, fmt.Errorf("error parsing logs pipeline response: %w", err)
	}

	tflog.Debug(ctx, "SDK Call Successful: Create Logs Pipeline", logFields)
	return &createdConfig, nil
}

// GetLogsPipeline retrieves a logs pipeline configuration by key
func (c *SdkClientWrapper) GetLogsPipeline(ctx context.Context, key string) (*ConfigEntry, error) {
	logFields := map[string]any{"key": key}
	tflog.Debug(ctx, "Executing SDK Call: Get Logs Pipeline", logFields)

	// Prepare the HTTP request
	req, err := c.prepareRequest(ctx, http.MethodGet, fmt.Sprintf("%s/%s", logsPipelineEndpoint, key), nil)
	if err != nil {
		return nil, err
	}

	// Execute the request
	resp, err := c.executeRequest(req)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetLogsPipeline", key)
	}
	defer resp.Body.Close()

	// Read and parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		err := parseErrorResponse(resp.StatusCode, respBody)
		return nil, handleApiError(ctx, err, "GetLogsPipeline", key)
	}

	// Parse the response JSON
	var config ConfigEntry
	if err := json.Unmarshal(respBody, &config); err != nil {
		return nil, fmt.Errorf("error parsing logs pipeline response: %w", err)
	}

	tflog.Debug(ctx, "SDK Call Successful: Get Logs Pipeline", logFields)
	return &config, nil
}

// UpdateLogsPipeline updates an existing logs pipeline configuration
func (c *SdkClientWrapper) UpdateLogsPipeline(ctx context.Context, config *ConfigEntry) (*ConfigEntry, error) {
	logFields := map[string]any{"key": config.Key}
	tflog.Debug(ctx, "Executing SDK Call: Update Logs Pipeline", logFields)

	// Marshal the request to JSON
	reqBody, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("error marshaling logs pipeline config: %w", err)
	}

	// Prepare the HTTP request
	req, err := c.prepareRequest(ctx, http.MethodPut, fmt.Sprintf("%s/%s", logsPipelineEndpoint, config.Key), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	// Execute the request
	resp, err := c.executeRequest(req)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateLogsPipeline", config.Key)
	}
	defer resp.Body.Close()

	// Read and parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		err := parseErrorResponse(resp.StatusCode, respBody)
		return nil, handleApiError(ctx, err, "UpdateLogsPipeline", config.Key)
	}

	// Parse the response JSON
	var updatedConfig ConfigEntry
	if err := json.Unmarshal(respBody, &updatedConfig); err != nil {
		return nil, fmt.Errorf("error parsing logs pipeline response: %w", err)
	}

	tflog.Debug(ctx, "SDK Call Successful: Update Logs Pipeline", logFields)
	return &updatedConfig, nil
}

// DeleteLogsPipeline deletes a logs pipeline configuration by key
func (c *SdkClientWrapper) DeleteLogsPipeline(ctx context.Context, key string) error {
	logFields := map[string]any{"key": key}
	tflog.Debug(ctx, "Executing SDK Call: Delete Logs Pipeline", logFields)

	// Prepare the HTTP request
	req, err := c.prepareRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", logsPipelineEndpoint, key), nil)
	if err != nil {
		return err
	}

	// Execute the request
	resp, err := c.executeRequest(req)
	if err != nil {
		return handleApiError(ctx, err, "DeleteLogsPipeline", key)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 404 is acceptable for delete (already deleted)
		if resp.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, "Resource not found during delete, treating as success", logFields)
			return nil
		}

		respBody, _ := io.ReadAll(resp.Body)
		err := parseErrorResponse(resp.StatusCode, respBody)
		return handleApiError(ctx, err, "DeleteLogsPipeline", key)
	}

	tflog.Debug(ctx, "SDK Call Successful: Delete Logs Pipeline", logFields)
	return nil
}

// Helper functions for HTTP requests

// prepareRequest creates an HTTP request with the proper headers and authentication
func (c *SdkClientWrapper) prepareRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	// Get base URL from environment or provider config
	baseURL := os.Getenv("GROUNDCOVER_API_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("GROUNDCOVER_API_URL environment variable not set")
	}

	// Clean up URL path
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(path, "/") {
		baseURL += "/"
	}
	fullURL := baseURL + path

	// Create the request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Get auth credentials
	apiKey := os.Getenv("GROUNDCOVER_API_KEY")
	backendID := os.Getenv("GROUNDCOVER_ORG_NAME")

	if apiKey == "" || backendID == "" {
		return nil, fmt.Errorf("authentication headers missing: ensure GROUNDCOVER_API_KEY and GROUNDCOVER_ORG_NAME are set")
	}

	req.Header.Set("X-Auth-ApiKey", apiKey)
	req.Header.Set("X-Backend-Id", backendID)

	return req, nil
}

// executeRequest sends the HTTP request and returns the response
func (c *SdkClientWrapper) executeRequest(req *http.Request) (*http.Response, error) {
	// Create a client with similar settings to the SDK's client
	client := &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	// Execute the request
	return client.Do(req)
}

// parseErrorResponse converts an error response to a proper error
func parseErrorResponse(statusCode int, respBody []byte) error {
	if len(respBody) == 0 {
		return fmt.Errorf("status code %d", statusCode)
	}

	var errMsg string
	// Try to parse as JSON error response
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &errResp); err == nil {
		if errResp.Error != "" {
			errMsg = errResp.Error
		} else if errResp.Message != "" {
			errMsg = errResp.Message
		}
	}

	if errMsg == "" {
		// If we couldn't parse it, use the raw body (truncated if too long)
		errMsg = string(respBody)
		if len(errMsg) > 100 {
			errMsg = errMsg[:97] + "..."
		}
	}

	return fmt.Errorf("status code %d: %s", statusCode, errMsg)
}
