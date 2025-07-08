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

func TestAccPolicyResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-policy")
	updatedName := acctest.RandomWithPrefix("test-policy-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_policy.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_policy.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_policy.test", "uuid"),
					resource.TestCheckResourceAttr("groundcover_policy.test", "role.read", "read"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_policy.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"deprecated", "is_system_defined", "read_only", "revision_number", "description", "role"},
			},
			// Update and Read testing
			{
				Config: testAccPolicyResourceConfigUpdated(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_policy.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_policy.test", "role.admin", "admin"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPolicyResource_complex(t *testing.T) {
	name := acctest.RandomWithPrefix("test-policy-complex")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with multiple roles
			{
				Config: testAccPolicyResourceConfigComplex(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_policy.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_policy.test", "role.admin", "admin"),
					resource.TestCheckResourceAttr("groundcover_policy.test", "claim_role", "sso-test-role"),
					resource.TestCheckResourceAttrSet("groundcover_policy.test", "data_scope.simple.operator"),
				),
			},
		},
	})
}

func TestAccPolicyResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-policy")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPolicyResourceExists("groundcover_policy.test"),
					testAccCheckPolicyResourceDisappears("groundcover_policy.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccPolicyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test" {
  name        = %[1]q
  description = "Test policy created by acceptance tests"
  role = {
    read = "read"
  }
}
`, name)
}

func testAccPolicyResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test" {
  name        = %[1]q
  description = "Updated test policy"
  role = {
    admin = "admin"
  }
}
`, name)
}

func testAccPolicyResourceConfigComplex(name string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test" {
  name        = %[1]q
  description = "Complex test policy with data scope"
  claim_role  = "sso-test-role"
  role = {
    admin = "admin"
  }
  data_scope = {
    simple = {
      operator = "and"
      conditions = [
        {
          key    = "cluster"
          origin = "root"
          type   = "string"
          filters = [
            {
              op    = "match"
              value = "test-cluster"
            }
          ]
        }
      ]
    }
  }
}
`, name)
}

func testAccCheckPolicyResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Policy ID is set")
		}

		return nil
	}
}

func testAccCheckPolicyResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Policy ID is set")
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
		if err := client.DeletePolicy(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete policy: %v", err)
		}

		return nil
	}
}
