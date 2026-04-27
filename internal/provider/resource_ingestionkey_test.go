// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccIngestionKeyResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-ingestionkey")
	updatedName := acctest.RandomWithPrefix("test-ingestionkey-updated")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckIngestionKey(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccIngestionKeyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "type", "sensor"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "key"),
					// creation_date is deprecated and no longer provided by API v1.84.0+
					// The field exists in schema but is always null/empty
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "created_by"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_ingestionkey.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},                         // Ingestion key is sensitive and not returned on import
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

func TestAccIngestionKeyResource_typeRum(t *testing.T) {
	name := acctest.RandomWithPrefix("test-ingestionkey-rum")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckIngestionKey(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionKeyResourceConfigWithType(name, "rum"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "type", "rum"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "key"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "created_by"),
				),
			},
		},
	})
}

func TestAccIngestionKeyResource_typeThirdParty(t *testing.T) {
	name := acctest.RandomWithPrefix("test-ingestionkey-tp")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckIngestionKey(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionKeyResourceConfigWithType(name, "thirdParty"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_ingestionkey.test", "type", "thirdParty"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "key"),
					resource.TestCheckResourceAttrSet("groundcover_ingestionkey.test", "created_by"),
				),
			},
		},
	})
}

func TestAccIngestionKeyResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-ingestionkey")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckIngestionKey(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
	return testAccIngestionKeyResourceConfigWithType(name, "sensor")
}

// testAccIngestionKeyResourceConfigWithType emits an HCL config that pins the
// provider's backend_id to GROUNDCOVER_INCLOUD_BACKEND_ID. This avoids mutating
// the process-global GROUNDCOVER_BACKEND_ID env var, which would corrupt other
// tests running in parallel.
func testAccIngestionKeyResourceConfigWithType(name, keyType string) string {
	return fmt.Sprintf(`
provider "groundcover" {
  backend_id = %[1]q
}

resource "groundcover_ingestionkey" "test" {
  name = %[2]q
  type = %[3]q
}
`, os.Getenv("GROUNDCOVER_INCLOUD_BACKEND_ID"), name, keyType)
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

		// Get environment variables for client configuration. Use the in-cloud
		// backend ID directly — the regular GROUNDCOVER_BACKEND_ID points at
		// the dev backend, which doesn't host ingestion keys.
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_INCLOUD_BACKEND_ID")
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

		// Poll the in-cloud backend until the ingestion key is no longer
		// returned by List. The in-cloud backend has eventually-consistent
		// read-after-delete (the gap was historically ~10s; now it's longer
		// and variable). A fixed sleep was flaky — poll-with-timeout returns
		// as soon as the delete is visible, with a bounded ceiling.
		const (
			pollInterval = 2 * time.Second
			pollTimeout  = 30 * time.Second
		)
		deadline := time.Now().Add(pollTimeout)
		for {
			keys, err := client.ListIngestionKeys(ctx, &models.ListIngestionKeysRequest{Name: ingestionKeyName})
			if err != nil {
				return fmt.Errorf("Failed to list ingestion keys while waiting for delete propagation: %v", err)
			}

			stillThere := false
			for _, k := range keys {
				if k != nil && k.Name == ingestionKeyName {
					stillThere = true
					break
				}
			}
			if !stillThere {
				return nil
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("ingestion key %q still visible %s after delete; in-cloud backend consistency exceeded budget", ingestionKeyName, pollTimeout)
			}
			time.Sleep(pollInterval)
		}
	}
}

// testAccPreCheckIngestionKey verifies that required environment variables are set for ingestion key tests
func testAccPreCheckIngestionKey(t *testing.T) {
	if v := os.Getenv("GROUNDCOVER_API_KEY"); v == "" {
		t.Fatal("GROUNDCOVER_API_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("GROUNDCOVER_INCLOUD_BACKEND_ID"); v == "" {
		t.Skip("Ingestion key tests require GROUNDCOVER_INCLOUD_BACKEND_ID env var for in-cloud backend - skipping")
	}
}
