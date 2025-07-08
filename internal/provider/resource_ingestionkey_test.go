// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccIngestionKeyResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-ingestionkey")
	updatedName := acctest.RandomWithPrefix("test-ingestionkey-updated")

	resource.Test(t, resource.TestCase{
		PreCheck: func() { 
			testAccPreCheckIngestionKey(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithCloudOrg(t),
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccIngestionKeyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "type", "sensor"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "key"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "creation_date"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "created_by"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_ingestionkey.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"}, // Ingestion key is sensitive and not returned on import
				Config:                  testAccIngestionKeyResourceConfig(name), // Ensure same config for import
			},
			// Update and Read testing
			{
				Config: testAccIngestionKeyResourceConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "name", updatedName),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccIngestionKeyResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-ingestionkey")

	resource.Test(t, resource.TestCase{
		PreCheck: func() { 
			testAccPreCheckIngestionKey(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithCloudOrg(t),
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionKeyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckIngestionKeyResourceExists("groundcover_ingestionkey.test"),
					testAccCheckIngestionKeyResourceDisappears("groundcover_ingestionkey.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccIngestionKeyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_ingestionkey" "test" {
  name = %[1]q
  type = "sensor"
}
`, name)
}

func testAccCheckIngestionKeyResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Ingestion Key ID is set")
		}

		return nil
	}
}

func testAccCheckIngestionKeyResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Ingestion Key ID is set")
		}

		// Create a provider client to delete the resource
		ctx := context.Background()
		
		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_ORG_NAME") // Use current org name (which should be set to cloud org during test)
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.io"
		}

		// Create the client wrapper
		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, orgName)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		// For ingestion keys, the ID is the name (see ImportState implementation)
		ingestionKeyName := rs.Primary.ID
		
		// Create delete request using the SDK pattern
		deleteReq := &models.DeleteIngestionKeyRequest{
			Name: &ingestionKeyName,
		}

		// Delete the resource using the client
		if err := client.DeleteIngestionKey(ctx, deleteReq); err != nil {
			return fmt.Errorf("Failed to delete ingestion key: %v", err)
		}

		// Wait for API consistency - ingestion key deletions take time to propagate
		// This matches the pattern you found in the SDK testing
		time.Sleep(10 * time.Second)

		return nil
	}
}

// testAccPreCheckIngestionKey verifies that required environment variables are set for ingestion key tests
func testAccPreCheckIngestionKey(t *testing.T) {
	if v := os.Getenv("GROUNDCOVER_API_KEY"); v == "" {
		t.Fatal("GROUNDCOVER_API_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("GROUNDCOVER_CLOUD_ORG_NAME"); v == "" {
		t.Skip("Ingestion key tests require GROUNDCOVER_CLOUD_ORG_NAME env var for cloud backend - skipping")
	}
}

// testAccProtoV6ProviderFactoriesWithCloudOrg creates provider factories that use the cloud organization
func testAccProtoV6ProviderFactoriesWithCloudOrg(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	// Skip if not running acceptance tests
	if os.Getenv("TF_ACC") == "" {
		return testAccProtoV6ProviderFactories
	}
	
	// Temporarily override GROUNDCOVER_ORG_NAME with the cloud org name
	cloudOrgName := os.Getenv("GROUNDCOVER_CLOUD_ORG_NAME")
	if cloudOrgName == "" {
		t.Fatal("GROUNDCOVER_CLOUD_ORG_NAME must be set for ingestion key tests")
	}
	
	// Store original value to restore later
	originalOrgName := os.Getenv("GROUNDCOVER_ORG_NAME")
	
	// Set the cloud org name and set up cleanup ONCE for the entire test
	if err := os.Setenv("GROUNDCOVER_ORG_NAME", cloudOrgName); err != nil {
		t.Fatalf("Failed to set GROUNDCOVER_ORG_NAME: %v", err)
	}
	t.Cleanup(func() {
		if originalOrgName != "" {
			if err := os.Setenv("GROUNDCOVER_ORG_NAME", originalOrgName); err != nil {
				t.Errorf("Failed to restore GROUNDCOVER_ORG_NAME: %v", err)
			}
		} else {
			if err := os.Unsetenv("GROUNDCOVER_ORG_NAME"); err != nil {
				t.Errorf("Failed to unset GROUNDCOVER_ORG_NAME: %v", err)
			}
		}
	})
	
	// Create provider factory
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"groundcover": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(New("test")())()
		},
	}
	
	return factories
}