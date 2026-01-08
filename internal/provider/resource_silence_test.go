// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSilenceResource(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-silence")
	updatedComment := acctest.RandomWithPrefix("test-silence-updated")

	// Use future dates to ensure the silence is valid
	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	updatedStartsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	updatedEndsAt := time.Now().Add(4 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSilenceResourceConfig(comment, startsAt, endsAt, "service", "test-service", true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.#", "1"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.name", "service"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.value", "test-service"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_equal", "true"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_regex", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_silence.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccSilenceResourceConfig(updatedComment, updatedStartsAt, updatedEndsAt, "workload", "updated-workload", true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", updatedComment),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.#", "1"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.name", "workload"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.value", "updated-workload"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSilenceResource_multipleMatchers(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-silence-multi")

	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSilenceResourceConfigMultipleMatchers(comment, startsAt, endsAt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.#", "3"),
					// First matcher - equal
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.name", "equal"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.value", "test-service"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_equal", "true"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_regex", "false"),
					// Second matcher - not contains regex
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.1.name", "not-contains"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.1.value", ".*test-service.*"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.1.is_equal", "false"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.1.is_regex", "true"),
					// Third matcher - defaults
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.2.name", "empty-equal"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.2.value", "test-service"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.2.is_equal", "true"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.2.is_regex", "false"),
				),
			},
		},
	})
}

func TestAccSilenceResource_regexMatcher(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-silence-regex")

	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSilenceResourceConfig(comment, startsAt, endsAt, "service", ".*-test$", true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.name", "service"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.value", ".*-test$"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_equal", "true"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_regex", "true"),
				),
			},
		},
	})
}

func TestAccSilenceResource_negation(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-silence-negation")

	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSilenceResourceConfig(comment, startsAt, endsAt, "environment", "production", false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.name", "environment"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.value", "production"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_equal", "false"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "matchers.0.is_regex", "false"),
				),
			},
		},
	})
}

func TestAccSilenceResource_disappears(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-silence-disappears")

	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSilenceResourceConfig(comment, startsAt, endsAt, "service", "test-service", true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSilenceResourceExists("groundcover_silence.test"),
					testAccCheckSilenceResourceDisappears("groundcover_silence.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccSilenceResource_applyLoop tests that consecutive applies do not require
// plan changes. This verifies that there is no apply loop caused by server-side
// normalization or formatting differences.
func TestAccSilenceResource_applyLoop(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-silence-apply-loop")

	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create silence
			{
				Config: testAccSilenceResourceConfig(comment, startsAt, endsAt, "service", "test-service", true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", comment),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			{
				Config: testAccSilenceResourceConfig(comment, startsAt, endsAt, "service", "test-service", true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", comment),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccSilenceResourceConfig(comment, startsAt, endsAt, "service", "test-service", true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_silence.test", "comment", comment),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccSilenceResourceConfig(comment, startsAt, endsAt, matcherName, matcherValue string, isEqual, isRegex bool) string {
	return fmt.Sprintf(`
resource "groundcover_silence" "test" {
  starts_at = %[1]q
  ends_at   = %[2]q
  comment   = %[3]q

  matchers = [
    {
      name     = %[4]q
      value    = %[5]q
      is_equal = %[6]t
      is_regex = %[7]t
    }
  ]
}
`, startsAt, endsAt, comment, matcherName, matcherValue, isEqual, isRegex)
}

func testAccSilenceResourceConfigMultipleMatchers(comment, startsAt, endsAt string) string {
	return fmt.Sprintf(`
resource "groundcover_silence" "test" {
  starts_at = %[1]q
  ends_at   = %[2]q
  comment   = %[3]q

  matchers = [
    {
      name     = "equal"
      value    = "test-service"
      is_equal = true
      is_regex = false
    },
    {
      name     = "not-contains"
      value    = "test-service"
      is_equal = false
      is_regex = true
    },
    {
      name     = "empty-equal"
      value    = "test-service"
      is_equal = true
      is_regex = false
    }
  ]
}
`, startsAt, endsAt, comment)
}

func testAccCheckSilenceResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Silence ID is set")
		}

		return nil
	}
}

func testAccCheckSilenceResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Silence ID is set")
		}

		// Create a provider client to delete the resource
		ctx := context.Background()

		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		backendID := os.Getenv("GROUNDCOVER_BACKEND_ID")
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}

		// Create the client wrapper
		client, err := NewSdkClientWrapper(ctx, apiURL, apiKey, backendID)
		if err != nil {
			return fmt.Errorf("Failed to create client: %v", err)
		}

		// Delete the resource using the client
		if err := client.DeleteSilence(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete silence: %v", err)
		}

		return nil
	}
}
