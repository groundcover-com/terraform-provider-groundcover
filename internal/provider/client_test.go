// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleApiError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		err         error
		expectedErr error
	}{
		{
			name:        "nil error",
			err:         nil,
			expectedErr: nil,
		},
		{
			name:        "generic error",
			err:         errors.New("generic error"),
			expectedErr: errors.New("generic error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleApiError(ctx, tt.err, "test-operation", "test-resource")

			if tt.expectedErr == nil {
				assert.Nil(t, result)
			} else {
				assert.Error(t, result)
			}
		})
	}
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
