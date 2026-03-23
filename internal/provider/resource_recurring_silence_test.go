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

func TestAccRecurringSilenceResource(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence")
	updatedComment := acctest.RandomWithPrefix("test-recurring-silence-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - daily recurrence
			{
				Config: testAccRecurringSilenceResourceConfig_daily(comment, "09:00", "17:00", "UTC", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "recurrence_type", "daily"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "start_time", "09:00"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "end_time", "17:00"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "timezone", "UTC"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "enabled", "true"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.#", "1"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.name", "service"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.value", "test-service"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.is_equal", "true"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.is_contains", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_recurring_silence.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing - change comment and time window
			{
				Config: testAccRecurringSilenceResourceConfig_daily(updatedComment, "10:00", "18:00", "America/New_York", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", updatedComment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "start_time", "10:00"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "end_time", "18:00"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "timezone", "America/New_York"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccRecurringSilenceResource_weekly(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence-weekly")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecurringSilenceResourceConfig_weekly(comment, "08:00", "12:00", "UTC", []int{1, 3, 5}, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "recurrence_type", "weekly"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "days_of_week.#", "3"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "days_of_week.0", "1"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "days_of_week.1", "3"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "days_of_week.2", "5"),
				),
			},
		},
	})
}

func TestAccRecurringSilenceResource_monthly(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence-monthly")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecurringSilenceResourceConfig_monthly(comment, "22:00", "06:00", "UTC", []int{1, 15}, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "recurrence_type", "monthly"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "days_of_month.#", "2"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "days_of_month.0", "1"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "days_of_month.1", "15"),
				),
			},
		},
	})
}

func TestAccRecurringSilenceResource_multipleMatchers(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence-multi")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecurringSilenceResourceConfigMultipleMatchers(comment),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.#", "3"),
					// First matcher - equal
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.name", "equal"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.value", "test-service"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.is_equal", "true"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.0.is_contains", "false"),
					// Second matcher - not contains
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.1.name", "not-contains"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.1.value", "test-service"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.1.is_equal", "false"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.1.is_contains", "true"),
					// Third matcher - defaults
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.2.name", "empty-equal"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.2.value", "test-service"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.2.is_equal", "true"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.2.is_contains", "false"),
				),
			},
		},
	})
}

func TestAccRecurringSilenceResource_disappears(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence-disappears")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecurringSilenceResourceConfig_daily(comment, "09:00", "17:00", "UTC", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRecurringSilenceResourceExists("groundcover_recurring_silence.test"),
					testAccCheckRecurringSilenceResourceDisappears("groundcover_recurring_silence.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccRecurringSilenceResource_applyLoop(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccRecurringSilenceResourceConfig_daily(comment, "09:00", "17:00", "UTC", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
				),
			},
			// Step 2: Apply same config - should not detect changes
			{
				Config: testAccRecurringSilenceResourceConfig_daily(comment, "09:00", "17:00", "UTC", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
				),
			},
			// Step 3: Apply one more time to be sure
			{
				Config: testAccRecurringSilenceResourceConfig_daily(comment, "09:00", "17:00", "UTC", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
				),
			},
		},
	})
}

// TestAccRecurringSilenceResource_applyLoop_noComment tests that consecutive applies
// do not require plan changes when the comment attribute is omitted from the config.
// This verifies that UseStateForUnknown prevents drift from server-generated comments.
func TestAccRecurringSilenceResource_applyLoop_noComment(t *testing.T) {
	config := testAccRecurringSilenceResourceConfig_daily_noComment("09:00", "17:00", "UTC", true)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create without comment (server auto-generates one)
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "comment"),
				),
			},
			// Step 2: Apply same config — expect no changes
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "comment"),
				),
			},
			// Step 3: Apply again — expect no changes
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "comment"),
				),
			},
		},
	})
}

func TestAccRecurringSilenceResource_disabled(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence-disabled")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecurringSilenceResourceConfig_daily(comment, "09:00", "17:00", "UTC", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "enabled", "false"),
				),
			},
		},
	})
}

// --- Config Helpers ---

func intsToTerraformList(values []int) string {
	result := "["
	for i, v := range values {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d", v)
	}
	result += "]"
	return result
}

func testAccRecurringSilenceResourceConfig_daily_noComment(startTime, endTime, timezone string, enabled bool) string {
	return fmt.Sprintf(`
resource "groundcover_recurring_silence" "test" {
  recurrence_type = "daily"
  start_time      = %[1]q
  end_time        = %[2]q
  timezone        = %[3]q
  enabled         = %[4]t

  matchers = [
    {
      name        = "service"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    }
  ]
}
`, startTime, endTime, timezone, enabled)
}

func testAccRecurringSilenceResourceConfig_daily(comment, startTime, endTime, timezone string, enabled bool) string {
	return fmt.Sprintf(`
resource "groundcover_recurring_silence" "test" {
  comment         = %[1]q
  recurrence_type = "daily"
  start_time      = %[2]q
  end_time        = %[3]q
  timezone        = %[4]q
  enabled         = %[5]t

  matchers = [
    {
      name        = "service"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    }
  ]
}
`, comment, startTime, endTime, timezone, enabled)
}

func testAccRecurringSilenceResourceConfig_weekly(comment, startTime, endTime, timezone string, daysOfWeek []int, enabled bool) string {
	daysStr := intsToTerraformList(daysOfWeek)

	return fmt.Sprintf(`
resource "groundcover_recurring_silence" "test" {
  comment         = %[1]q
  recurrence_type = "weekly"
  start_time      = %[2]q
  end_time        = %[3]q
  timezone        = %[4]q
  days_of_week    = %[5]s
  enabled         = %[6]t

  matchers = [
    {
      name        = "service"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    }
  ]
}
`, comment, startTime, endTime, timezone, daysStr, enabled)
}

func testAccRecurringSilenceResourceConfig_monthly(comment, startTime, endTime, timezone string, daysOfMonth []int, enabled bool) string {
	daysStr := intsToTerraformList(daysOfMonth)

	return fmt.Sprintf(`
resource "groundcover_recurring_silence" "test" {
  comment         = %[1]q
  recurrence_type = "monthly"
  start_time      = %[2]q
  end_time        = %[3]q
  timezone        = %[4]q
  days_of_month   = %[5]s
  enabled         = %[6]t

  matchers = [
    {
      name        = "service"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    }
  ]
}
`, comment, startTime, endTime, timezone, daysStr, enabled)
}

func testAccRecurringSilenceResourceConfigMultipleMatchers(comment string) string {
	return fmt.Sprintf(`
resource "groundcover_recurring_silence" "test" {
  comment         = %[1]q
  recurrence_type = "daily"
  start_time      = "09:00"
  end_time        = "17:00"
  timezone        = "UTC"

  matchers = [
    {
      name        = "equal"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    },
    {
      name        = "not-contains"
      value       = "test-service"
      is_equal    = false
      is_contains = true
    },
    {
      name        = "empty-equal"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    }
  ]
}
`, comment)
}

// --- Check Helpers ---

func testAccCheckRecurringSilenceResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Recurring Silence ID is set")
		}

		return nil
	}
}

func testAccCheckRecurringSilenceResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Recurring Silence ID is set")
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

		if err := client.DeleteRecurringSilence(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete recurring silence: %v", err)
		}

		return nil
	}
}
