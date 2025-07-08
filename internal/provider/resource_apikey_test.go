// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApiKeyResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-apikey")
	updatedName := acctest.RandomWithPrefix("test-apikey-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApiKeyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_apikey.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_apikey.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_apikey.test", "api_key"),
					resource.TestCheckResourceAttrSet("groundcover_apikey.test", "creation_date"),
					resource.TestCheckResourceAttrSet("groundcover_apikey.test", "created_by"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_apikey.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"}, // API key is sensitive and not returned on import
			},
			// Update and Read testing
			{
				Config: testAccApiKeyResourceConfigUpdated(name, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_apikey.test", "name", updatedName),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApiKeyResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-apikey")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApiKeyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckApiKeyResourceExists("groundcover_apikey.test"),
					testAccCheckApiKeyResourceDisappears("groundcover_apikey.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccApiKeyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test_policy" {
  name        = "%[1]s-policy"
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test_sa" {
  name         = "%[1]s-sa"
  email        = "test-%[1]s@example.com"
  policy_uuids = [groundcover_policy.test_policy.uuid]
}

resource "groundcover_apikey" "test" {
  name               = %[1]q
  description        = "Test API key created by acceptance tests"
  service_account_id = groundcover_serviceaccount.test_sa.id
}
`, name)
}

func testAccApiKeyResourceConfigUpdated(baseName, apiKeyName string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test_policy" {
  name        = "%[1]s-policy"
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test_sa" {
  name         = "%[1]s-sa"
  email        = "test-%[1]s@example.com"
  policy_uuids = [groundcover_policy.test_policy.uuid]
}

resource "groundcover_apikey" "test" {
  name               = %[2]q
  description        = "Updated test API key"
  service_account_id = groundcover_serviceaccount.test_sa.id
}
`, baseName, apiKeyName)
}

func testAccCheckApiKeyResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Key ID is set")
		}

		return nil
	}
}

func testAccCheckApiKeyResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Key ID is set")
		}

		// Create a provider client to delete the resource
		ctx := context.Background()
		
		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_ORG_NAME")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.io"
		}

		// Create the client wrapper
		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, orgName)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		// Delete the resource using the client
		if err := client.DeleteApiKey(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete API key: %v", err)
		}

		return nil
	}
}