// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPolicyResourceSimple tests only policy creation which should work in most environments
func TestAccPolicyResourceSimple(t *testing.T) {
	name := acctest.RandomWithPrefix("test-policy-simple")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyResourceSimpleConfig(name),
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
			// Update testing
			{
				Config: testAccPolicyResourceSimpleConfigUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_policy.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_policy.test", "role.admin", "admin"),
				),
			},
		},
	})
}

func testAccPolicyResourceSimpleConfig(name string) string {
	return `
resource "groundcover_policy" "test" {
  name        = "` + name + `"
  description = "Simple test policy for acceptance tests"
  role = {
    read = "read"
  }
}
`
}

func testAccPolicyResourceSimpleConfigUpdated(name string) string {
	return `
resource "groundcover_policy" "test" {
  name        = "` + name + `"
  description = "Updated simple test policy"
  role = {
    admin = "admin"
  }
}
`
}
