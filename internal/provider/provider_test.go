// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"groundcover": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
	if v := os.Getenv("GROUNDCOVER_API_KEY"); v == "" {
		t.Fatal("GROUNDCOVER_API_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("GROUNDCOVER_BACKEND_ID"); v == "" {
		t.Fatal("GROUNDCOVER_BACKEND_ID must be set for acceptance tests")
	}
}

func TestNormalizeAPIURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid https URL",
			input:    "https://api.groundcover.com",
			expected: "https://api.groundcover.com",
		},
		{
			name:     "valid http URL",
			input:    "http://api.groundcover.com",
			expected: "http://api.groundcover.com",
		},
		{
			name:     "domain without scheme",
			input:    "api.groundcover.com",
			expected: "https://api.groundcover.com",
		},
		{
			name:     "domain with port without scheme",
			input:    "api.groundcover.com:8080",
			expected: "https://api.groundcover.com:8080",
		},
		{
			name:     "domain with path without scheme",
			input:    "api.groundcover.com/v1/api",
			expected: "https://api.groundcover.com/v1/api",
		},
		{
			name:     "URL with spaces",
			input:    "  https://api.groundcover.com  ",
			expected: "https://api.groundcover.com",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "localhost without scheme",
			input:    "localhost:8080",
			expected: "https://localhost:8080",
		},
		{
			name:     "IP address without scheme",
			input:    "192.168.1.1:8080",
			expected: "https://192.168.1.1:8080",
		},
		{
			name:     "URL with query params without scheme",
			input:    "api.groundcover.com?foo=bar",
			expected: "https://api.groundcover.com?foo=bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeAPIURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeAPIURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
