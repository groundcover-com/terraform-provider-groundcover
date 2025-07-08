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

func TestAccServiceAccountResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-serviceaccount")
	updatedEmail := "updated-" + name + "@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceAccountResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_serviceaccount.test", "id"),
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "email", "test-"+name+"@example.com"),
				),
			},
			// Update and Read testing
			{
				Config: testAccServiceAccountResourceConfigUpdated(name, updatedEmail),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "name", name), // Name should stay the same
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "email", updatedEmail),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceAccountResource_withPolicy(t *testing.T) {
	name := acctest.RandomWithPrefix("test-serviceaccount")
	policyName := acctest.RandomWithPrefix("test-policy")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create service account with policy
			{
				Config: testAccServiceAccountResourceConfigWithPolicy(name, policyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_serviceaccount.test", "id"),
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "email", "test-"+name+"@example.com"),
					// Check policy association
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "policy_uuids.#", "1"),
				),
			},
		},
	})
}

func TestAccServiceAccountResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-serviceaccount")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServiceAccountResourceExists("groundcover_serviceaccount.test"),
					testAccCheckServiceAccountResourceDisappears("groundcover_serviceaccount.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccServiceAccountResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test_policy" {
  name        = "%[1]s-policy"
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test" {
  name         = %[1]q
  email        = "test-%[1]s@example.com"
  policy_uuids = [groundcover_policy.test_policy.uuid]
}
`, name)
}

func testAccServiceAccountResourceConfigUpdated(baseName, updatedEmail string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test_policy" {
  name        = "%[1]s-policy"
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test" {
  name         = %[1]q
  email        = %[2]q
  policy_uuids = [groundcover_policy.test_policy.uuid]
}
`, baseName, updatedEmail)
}

func testAccServiceAccountResourceConfigWithPolicy(name, policyName string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test" {
  name        = %[2]q
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test" {
  name         = %[1]q
  email        = "test-%[1]s@example.com"
  policy_uuids = [groundcover_policy.test.uuid]
}
`, name, policyName)
}

func testAccCheckServiceAccountResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service Account ID is set")
		}

		return nil
	}
}

func testAccCheckServiceAccountResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service Account ID is set")
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
		if err := client.DeleteServiceAccount(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete service account: %v", err)
		}

		return nil
	}
}