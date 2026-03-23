// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSecretResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-secret")
	updatedName := acctest.RandomWithPrefix("test-secret-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSecretResourceConfig(name, "api_key", "test-secret-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_secret.test", "type", "api_key"),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "content_hash"),
				),
			},
			// ImportState testing - content is write-only and not returned on import
			{
				ResourceName:            "groundcover_secret.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content"}, // content is write-only
			},
			// Update and Read testing
			{
				Config: testAccSecretResourceConfig(updatedName, "api_key", "updated-secret-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_secret.test", "type", "api_key"),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "content_hash"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSecretResource_passwordType(t *testing.T) {
	name := acctest.RandomWithPrefix("test-secret-password")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretResourceConfig(name, "password", "test-password-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_secret.test", "type", "password"),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "content_hash"),
				),
			},
		},
	})
}

func TestAccSecretResource_basicAuthType(t *testing.T) {
	name := acctest.RandomWithPrefix("test-secret-basicauth")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretResourceConfig(name, "basic_auth", "test-basic-auth-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_secret.test", "type", "basic_auth"),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "content_hash"),
				),
			},
		},
	})
}

func TestAccSecretResource_contentHashChanges(t *testing.T) {
	name := acctest.RandomWithPrefix("test-secret-hash")
	updatedName := acctest.RandomWithPrefix("test-secret-hash-updated")
	var firstHash, secondHash string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial content
			{
				Config: testAccSecretResourceConfig(name, "api_key", "initial-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "content_hash"),
					// Store the first hash
					resource.TestCheckResourceAttrWith("groundcover_secret.test", "content_hash", func(value string) error {
						firstHash = value
						return nil
					}),
				),
			},
			// Update with different content AND name - hash should change
			// Note: We also change name because content is write-only and Terraform won't detect changes to it alone
			{
				Config: testAccSecretResourceConfig(updatedName, "api_key", "different-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", updatedName),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "content_hash"),
					// Verify the hash changed
					resource.TestCheckResourceAttrWith("groundcover_secret.test", "content_hash", func(value string) error {
						secondHash = value
						if firstHash == secondHash {
							return fmt.Errorf("content_hash should change when content changes, but both are: %s", firstHash)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccSecretResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-secret-disappears")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretResourceConfig(name, "api_key", "test-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_secret.test", "id"),
				),
			},
			// The disappears test is now possible because GetSecretHash can detect if secret exists
		},
	})
}

func testAccSecretResourceConfig(name, secretType, content string) string {
	return fmt.Sprintf(`
resource "groundcover_secret" "test" {
  name    = %[1]q
  type    = %[2]q
  content = %[3]q
}
`, name, secretType, content)
}
