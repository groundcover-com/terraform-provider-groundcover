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
				),
			},
			// ImportState testing - note: name, type, content cannot be retrieved from API (no GET endpoint)
			{
				ResourceName:            "groundcover_secret.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name", "type", "content"}, // These are write-only and not returned on import
			},
			// Update and Read testing
			{
				Config: testAccSecretResourceConfig(updatedName, "api_key", "updated-secret-content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_secret.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_secret.test", "type", "api_key"),
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
				),
			},
		},
	})
}

// Note: TestAccSecretResource_disappears is not implemented because the secret API
// has no GET endpoint, so Read cannot detect if the secret was deleted externally.
// The resource preserves state as-is, which means Terraform cannot detect drift.

func testAccSecretResourceConfig(name, secretType, content string) string {
	return fmt.Sprintf(`
resource "groundcover_secret" "test" {
  name    = %[1]q
  type    = %[2]q
  content = %[3]q
}
`, name, secretType, content)
}
