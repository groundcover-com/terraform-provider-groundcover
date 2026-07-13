// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestGetRecurringSilenceEmptyIDReturnsNotFound(t *testing.T) {
	c := &SdkClientWrapper{}
	resp, err := c.GetRecurringSilence(context.Background(), "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for empty id, got %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response for empty id, got %v", resp)
	}
}

func TestValidateTimeframeDay(t *testing.T) {
	cases := []struct {
		recurrence string
		day        string
		wantErr    bool
	}{
		{recurrenceTypeDaily, "every_day", false},
		{recurrenceTypeDaily, "monday", true},
		{recurrenceTypeWeekly, "monday", false},
		{recurrenceTypeWeekly, "sunday", false},
		{recurrenceTypeWeekly, "Monday", true}, // must be lowercase
		{recurrenceTypeWeekly, "funday", true},
		{recurrenceTypeMonthly, "1", false},
		{recurrenceTypeMonthly, "31", false},
		{recurrenceTypeMonthly, "0", true},
		{recurrenceTypeMonthly, "32", true},
		{recurrenceTypeMonthly, "abc", true},
	}
	for _, c := range cases {
		err := validateTimeframeDay(c.recurrence, c.day)
		if (err != nil) != c.wantErr {
			t.Errorf("validateTimeframeDay(%q, %q) err=%v, wantErr=%v", c.recurrence, c.day, err, c.wantErr)
		}
	}
}

// TestTimeframesRoundTrip verifies flat-set <-> map[day][]TimeRange conversion preserves data.
func TestTimeframesRoundTrip(t *testing.T) {
	ctx := context.Background()

	mk := func(day, start, end string) attr.Value {
		obj, _ := types.ObjectValue(timeframeObjectAttrTypes, map[string]attr.Value{
			"day":        types.StringValue(day),
			"start_time": types.StringValue(start),
			"end_time":   types.StringValue(end),
		})
		return obj
	}
	set, diags := types.SetValue(timeframeObjectType, []attr.Value{
		mk("wednesday", "09:00", "11:00"),
		mk("thursday", "09:00", "11:00"),
		mk("thursday", "13:00", "14:30"),
	})
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}

	m, err := timeframesFromModel(ctx, set)
	if err != nil {
		t.Fatalf("timeframesFromModel: %v", err)
	}
	if len(m["wednesday"]) != 1 || len(m["thursday"]) != 2 {
		t.Fatalf("unexpected grouping: %#v", m)
	}
	gotThursdayStarts := map[string]bool{}
	for _, tr := range m["thursday"] {
		if tr.StartTime != nil {
			gotThursdayStarts[*tr.StartTime] = true
		}
	}
	if !gotThursdayStarts["09:00"] || !gotThursdayStarts["13:00"] {
		t.Errorf("thursday ranges lost start times: %#v", m["thursday"])
	}

	// Back to a set, then flatten again — grouping must be identical (set is order-independent).
	roundTripSet, err := timeframesToModel(m)
	if err != nil {
		t.Fatalf("timeframesToModel: %v", err)
	}
	m2, err := timeframesFromModel(ctx, roundTripSet)
	if err != nil {
		t.Fatalf("timeframesFromModel (round 2): %v", err)
	}
	if len(m2["wednesday"]) != 1 || len(m2["thursday"]) != 2 {
		t.Errorf("round-trip changed grouping: %#v", m2)
	}
}

func TestTimeframesFromModel_empty(t *testing.T) {
	ctx := context.Background()
	m, err := timeframesFromModel(ctx, types.SetNull(timeframeObjectType))
	if err != nil || m != nil {
		t.Errorf("expected nil,nil for null set, got %#v, %v", m, err)
	}
}

func TestAccRecurringSilenceResource(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence")
	updatedComment := acctest.RandomWithPrefix("test-recurring-silence-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecurringSilenceResourceConfigWeekly(comment),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "recurrence_type", "weekly"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "timezone", "UTC"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "enabled", "true"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", comment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "timeframes.#", "2"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "matchers.#", "1"),
				),
			},
			{
				ResourceName:      "groundcover_recurring_silence.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRecurringSilenceResourceConfigDaily(updatedComment),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "recurrence_type", "daily"),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "comment", updatedComment),
					resource.TestCheckResourceAttr("groundcover_recurring_silence.test", "timeframes.#", "1"),
				),
			},
		},
	})
}

// TestAccRecurringSilenceResource_applyLoop guards against a re-apply diff caused by the
// set<->map timeframe conversion or server-side normalization.
func TestAccRecurringSilenceResource_applyLoop(t *testing.T) {
	comment := acctest.RandomWithPrefix("test-recurring-silence-loop")
	config := testAccRecurringSilenceResourceConfigWeekly(comment)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.TestCheckResourceAttrSet("groundcover_recurring_silence.test", "id")},
			{Config: config},
			{Config: config},
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
				Config: testAccRecurringSilenceResourceConfigWeekly(comment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRecurringSilenceResourceExists("groundcover_recurring_silence.test"),
					testAccCheckRecurringSilenceResourceDisappears("groundcover_recurring_silence.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRecurringSilenceResourceConfigWeekly(comment string) string {
	return fmt.Sprintf(`
resource "groundcover_recurring_silence" "test" {
  recurrence_type = "weekly"
  timezone        = "UTC"
  comment         = %[1]q

  timeframes = [
    { day = "wednesday", start_time = "09:00", end_time = "11:00" },
    { day = "thursday",  start_time = "09:00", end_time = "11:00" },
  ]

  matchers = [
    {
      name        = "service"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    }
  ]
}
`, comment)
}

func testAccRecurringSilenceResourceConfigDaily(comment string) string {
	return fmt.Sprintf(`
resource "groundcover_recurring_silence" "test" {
  recurrence_type = "daily"
  timezone        = "UTC"
  comment         = %[1]q

  timeframes = [
    { day = "every_day", start_time = "03:00", end_time = "03:30" },
  ]

  matchers = [
    {
      name        = "service"
      value       = "test-service"
      is_equal    = true
      is_contains = false
    }
  ]
}
`, comment)
}

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
