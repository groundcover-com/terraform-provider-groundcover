// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

// TestProvider returns a configured provider for testing
func TestProvider() provider.Provider {
	return New("test")()
}


// testAccResourceName generates a unique resource name for tests
func testAccResourceName(prefix string) string {
	// Generate a unique ID using timestamp and random number
	timestamp := time.Now().Unix()
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%s-%d-%s", prefix, timestamp, randomNum.String())
}

// TestAccExampleData provides common test data structures
type TestAccExampleData struct {
	ApiKeyName         string
	IngestionKeyName   string
	MonitorName        string
	PolicyName         string
	ServiceAccountName string
}

// NewTestAccExampleData creates a new test data structure with unique names
func NewTestAccExampleData() TestAccExampleData {
	return TestAccExampleData{
		ApiKeyName:         testAccResourceName("test-apikey"),
		IngestionKeyName:   testAccResourceName("test-ingestionkey"),
		MonitorName:        testAccResourceName("test-monitor"),
		PolicyName:         testAccResourceName("test-policy"),
		ServiceAccountName: testAccResourceName("test-serviceaccount"),
	}
}

