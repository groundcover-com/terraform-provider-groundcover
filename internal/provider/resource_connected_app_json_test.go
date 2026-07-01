// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestConnectedAppJsonModelParity guards the typed-clone against drift. Unlike monitor_v2_json,
// groundcover_connected_app_json is a fully independent clone (its own schema and CRUD), so its
// model must be kept in step with connected_app by hand. This fails if connected_app gains or
// changes a field the JSON twin doesn't mirror — except data, which is intentionally a JSON
// string here vs a dynamic object on connected_app.
func TestConnectedAppJsonModelParity(t *testing.T) {
	assertTfsdkFieldParity(t,
		reflect.TypeOf(connectedAppResourceModel{}),
		reflect.TypeOf(connectedAppJsonResourceModel{}),
		map[string]bool{"data": true},
	)
}

const testConnectedAppJsonData = `{"url":"https://hooks.slack.com/services/TEST/WEBHOOK/URL"}`

func TestAccConnectedAppJson_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-slack-app-json")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppJsonConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_connected_app_json.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_connected_app_json.test", "type", "slack-webhook"),
					resource.TestCheckResourceAttr("groundcover_connected_app_json.test", "data", testConnectedAppJsonData),
					resource.TestCheckResourceAttrSet("groundcover_connected_app_json.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_connected_app_json.test", "data_hash"),
				),
			},
			{
				ResourceName:            "groundcover_connected_app_json.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"data"}, // sensitive + redacted on read
			},
		},
	})
}

// TestAccConnectedAppJson_applyLoop verifies repeated applies of an unchanged config
// produce no diff — the redacted `data` must not cause a perpetual apply loop (the same
// guarantee `connected_app` has, here via the JSON-string field + data_hash).
func TestAccConnectedAppJson_applyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-slack-json-apply-loop")
	cfg := testAccConnectedAppJsonConfig_basic(name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttrSet("groundcover_connected_app_json.test", "id")},
			{Config: cfg, Check: resource.TestCheckResourceAttrSet("groundcover_connected_app_json.test", "id")},
			{Config: cfg, Check: resource.TestCheckResourceAttrSet("groundcover_connected_app_json.test", "id")},
		},
	})
}

// TestAccConnectedAppJson_disappears verifies the provider detects external deletion and
// plans to recreate. Reuses the connected_app helpers — both resources delete by ID via
// the same API.
func TestAccConnectedAppJson_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-slack-json-disappears")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedAppJsonConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConnectedAppResourceExists("groundcover_connected_app_json.test"),
					testAccCheckConnectedAppResourceDisappears("groundcover_connected_app_json.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccConnectedAppJsonConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app_json" "test" {
  name = %[1]q
  type = "slack-webhook"
  data = jsonencode({
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  })
}
`, name)
}

// TestUnitJsonStringToMap is a fast, no-network check of the data parsing/validation —
// in particular that a JSON "null" (which json.Unmarshal turns into a nil map without
// error) is rejected rather than sent to the API as Data: nil.
func TestUnitJsonStringToMap(t *testing.T) {
	cases := []struct {
		name    string
		in      types.String
		wantErr bool
	}{
		{"valid object", types.StringValue(`{"url":"x"}`), false},
		{"empty object", types.StringValue(`{}`), false},
		{"json null", types.StringValue(`null`), true},
		{"scalar number", types.StringValue(`123`), true},
		{"scalar string", types.StringValue(`"x"`), true},
		{"malformed", types.StringValue(`{`), true},
		{"null value", types.StringNull(), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := jsonStringToMap(tc.in)
			if diags.HasError() != tc.wantErr {
				t.Fatalf("jsonStringToMap(%q) error = %v, wantErr %v", tc.in, diags.HasError(), tc.wantErr)
			}
		})
	}
}
