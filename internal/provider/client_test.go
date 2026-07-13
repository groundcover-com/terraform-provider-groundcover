// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}
}

func TestHandleApiError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		err         error
		operation   string
		expectedErr error
		checkIs     error
	}{
		{
			name:        "nil error",
			err:         nil,
			operation:   "test-operation",
			expectedErr: nil,
		},
		{
			name:        "generic error",
			err:         errors.New("generic error"),
			operation:   "test-operation",
			expectedErr: errors.New("generic error"),
		},
		{
			name:        "go-swagger 404 maps to ErrNotFound",
			err:         errors.New(`[GET /api/connected-apps/v1/abc][404] getConnectedAppNotFound`),
			operation:   "GetConnectedApp",
			expectedErr: ErrNotFound,
			checkIs:     ErrNotFound,
		},
		{
			name:        "go-swagger 400 with not found in message should NOT map to ErrNotFound",
			err:         errors.New(`[POST /api/connected-apps/v1][400] createConnectedAppBadRequest {"message":"invalid custom payload template: Tag '22id' not found (or beginning tag not provided)"}`),
			operation:   "CreateConnectedApp",
			expectedErr: errors.New("CreateConnectedApp failed"),
			checkIs:     nil,
		},
		{
			name:        "go-swagger 400 should surface real error message",
			err:         errors.New(`[POST /api/monitors/v1][400] createMonitorBadRequest {"message":"invalid monitor configuration: field 'query' is required"}`),
			operation:   "CreateMonitor",
			expectedErr: errors.New("CreateMonitor failed"),
			checkIs:     nil,
		},
		{
			name:        "go-swagger 500 with not found in message should NOT map to ErrNotFound",
			err:         errors.New(`[PUT /api/dashboards/v1/abc][500] updateDashboardInternalServerError {"message":"widget not found in layout"}`),
			operation:   "UpdateDashboard",
			expectedErr: errors.New("UpdateDashboard failed"),
			checkIs:     nil,
		},
		{
			name:        "status code format maps correctly",
			err:         errors.New("request failed with status code 404"),
			operation:   "GetMonitor",
			expectedErr: ErrNotFound,
			checkIs:     ErrNotFound,
		},
		{
			name:        "fallback not found substring works when no status code extractable",
			err:         errors.New("resource not found"),
			operation:   "GetMonitor",
			expectedErr: ErrNotFound,
			checkIs:     ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleApiError(ctx, tt.err, tt.operation, "test-resource")

			if tt.expectedErr == nil {
				assert.Nil(t, result)
			} else {
				require.Error(t, result)
				if tt.checkIs != nil {
					assert.ErrorIs(t, result, tt.checkIs, "expected error to wrap %v, got: %v", tt.checkIs, result)
				} else {
					assert.NotErrorIs(t, result, ErrNotFound, "error should NOT be mapped to ErrNotFound: %v", result)
					assert.Contains(t, result.Error(), tt.operation+" failed")
				}
			}
		})
	}
}

func TestHandleApiErrorSkillDiagnostics(t *testing.T) {
	forbidden := handleApiError(context.Background(), errors.New(`[POST /api/agent/skills][403] agentCreateSkillForbidden`), "CreateSkill", "my-skill")
	require.Error(t, forbidden)
	assert.Contains(t, forbidden.Error(), "admin role")

	conflict := handleApiError(context.Background(), errors.New(`[POST /api/agent/skills][409] agentCreateSkillConflict`), "CreateSkill", "my-skill")
	require.Error(t, conflict)
	assert.Contains(t, conflict.Error(), `Skill name "my-skill" is already in use`)
}

func TestProviderErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "ErrNotFound",
			err:  ErrNotFound,
		},
		{
			name: "ErrConcurrency",
			err:  ErrConcurrency,
		},
		{
			name: "ErrReadOnly",
			err:  ErrReadOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

func TestRateLimitRetryTransportRetriesSafeMethodInternalServerError(t *testing.T) {
	attempts := 0
	transport := &rateLimitRetryTransport{
		transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return testHTTPResponse(http.StatusInternalServerError), nil
			}
			return testHTTPResponse(http.StatusOK), nil
		}),
		maxRetries: 1,
	}

	req, err := http.NewRequest(http.MethodDelete, "https://example.com/resource", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, attempts)
}

func TestRateLimitRetryTransportDoesNotRetryPostInternalServerError(t *testing.T) {
	attempts := 0
	transport := &rateLimitRetryTransport{
		transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			return testHTTPResponse(http.StatusInternalServerError), nil
		}),
		maxRetries: 1,
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/resource", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, 1, attempts)
}

func TestRateLimitRetryTransportRetriesRateLimitForPost(t *testing.T) {
	attempts := 0
	transport := &rateLimitRetryTransport{
		transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return testHTTPResponse(http.StatusTooManyRequests), nil
			}
			return testHTTPResponse(http.StatusOK), nil
		}),
		maxRetries: 1,
	}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/resource", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, attempts)
}

func TestContextValidation(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "valid context",
			ctx:  context.Background(),
			want: true,
		},
		{
			name: "nil context",
			ctx:  nil,
			want: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ctx == nil {
				assert.False(t, tt.want)
				return
			}

			select {
			case <-tt.ctx.Done():
				assert.False(t, tt.want)
			default:
				assert.True(t, tt.want)
			}
		})
	}
}

func TestApiClientInterface(t *testing.T) {
	// Test that our client wrapper implements the ApiClient interface
	var _ ApiClient = (*SdkClientWrapper)(nil)

	// This test ensures that our interface contracts are maintained
	assert.True(t, true, "ApiClient interface implementation check passed")
}

func TestErrorConstants(t *testing.T) {
	// Test that error constants are properly defined
	require.NotNil(t, ErrNotFound)
	require.NotNil(t, ErrConcurrency)
	require.NotNil(t, ErrReadOnly)

	// Test that they have meaningful messages
	assert.NotEmpty(t, ErrNotFound.Error())
	assert.NotEmpty(t, ErrConcurrency.Error())
	assert.NotEmpty(t, ErrReadOnly.Error())

	// Test that they are distinct
	assert.NotEqual(t, ErrNotFound, ErrConcurrency)
	assert.NotEqual(t, ErrNotFound, ErrReadOnly)
	assert.NotEqual(t, ErrConcurrency, ErrReadOnly)
}
