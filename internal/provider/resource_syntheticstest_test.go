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

func TestAccSyntheticTestResource_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "enabled", "true"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "version", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.url", "https://httpbin.org/status/200"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.method", "GET"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "http_check.timeout", "10s"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "2"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "statusCode"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.operator", "eq"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.target", "200"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_synthetic_test.test",
				ImportState:       true,
				ImportStateVerify: true,
				// auth token/password are sensitive and not returned from API
				ImportStateVerifyIgnore: []string{"http_check.auth"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSyntheticTestResource_update(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth")
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "1m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "2"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Update name and interval
			{
				Config: testAccSyntheticTestResourceConfig_updated(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "interval", "5m"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.#", "1"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "assertion.0.source", "statusCode"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSyntheticTestResource_withRetry(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-retry")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_withRetry(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "retry.count", "3"),
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "retry.interval", "500ms"),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_applyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 2: Apply same config - should detect no changes
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
			// Step 3: One more time to be sure
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_synthetic_test.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_synthetic_test.test", "id"),
				),
			},
		},
	})
}

func TestAccSyntheticTestResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-synth-disappear")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticTestResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSyntheticTestResourceExists("groundcover_synthetic_test.test"),
					testAccCheckSyntheticTestResourceDisappears("groundcover_synthetic_test.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// --- Config helpers ---

func testAccSyntheticTestResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "1m"

  http_check {
    url     = "https://httpbin.org/status/200"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  assertion {
    source   = "responseTime"
    operator = "lt"
    target   = "5000"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  enabled  = true
  interval = "5m"

  http_check {
    url     = "https://httpbin.org/status/200"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
}
`, name)
}

func testAccSyntheticTestResourceConfig_withRetry(name string) string {
	return fmt.Sprintf(`
resource "groundcover_synthetic_test" "test" {
  name     = %[1]q
  interval = "1m"

  http_check {
    url     = "https://httpbin.org/status/200"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  retry {
    count    = 3
    interval = "500ms"
  }
}
`, name)
}

// --- Test check helpers ---

func testAccCheckSyntheticTestResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Synthetic Test ID is set")
		}

		return nil
	}
}

func testAccCheckSyntheticTestResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Synthetic Test ID is set")
		}

		ctx := context.Background()

		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		backendID := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}

		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, backendID)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		if err := client.DeleteSyntheticTest(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete Synthetic Test: %v", err)
		}

		return nil
	}
}
