// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
)

func TestAccNotificationRoute_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.connected_apps.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.connected_apps.0.type", "slack-webhook"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "routes.0.connected_apps.0.id"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "notification_settings.renotification_interval", "4h"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "created_by"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "created_at"),
				),
			},
			{
				ResourceName:      "groundcover_notification_route.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNotificationRoute_update(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-update")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRouteConfig_update_step1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
			{
				Config: testAccNotificationRouteConfig_update_step2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:production"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "2"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.1", "Resolved"),
				),
			},
		},
	})
}

func TestAccNotificationRoute_durationNormalization(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-duration")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRouteConfig_durationNormalization(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "notification_settings.renotification_interval", "60m"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
			{
				Config:             testAccNotificationRouteConfig_durationNormalization(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccNotificationRoute_applyLoop tests that applying the same configuration multiple times
// doesn't cause an apply loop due to server-side normalization or formatting differences.
func TestAccNotificationRoute_applyLoop(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-apply-loop")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create notification route
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This is the critical test - if there's an apply loop, this step will show changes
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccNotificationRouteConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
				),
			},
		},
	})
}

func testAccNotificationRouteConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "4h"
  }
}
`, name)
}

func testAccNotificationRouteConfig_update_step1(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "1h"
  }
}
`, name)
}

func testAccNotificationRouteConfig_update_step2(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:production"

  routes = [{
    status = ["Alerting", "Resolved"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "1h"
  }
}
`, name)
}

func testAccNotificationRouteConfig_durationNormalization(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]

  notification_settings = {
    renotification_interval = "60m"
  }
}
`, name)
}

// TestAccNotificationRoute_noNotificationSettings tests that notification routes
// can be created without specifying notification_settings (it's Optional+Computed).
func TestAccNotificationRoute_noNotificationSettings(t *testing.T) {
	name := acctest.RandomWithPrefix("test-route-no-settings")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without notification_settings
			{
				Config: testAccNotificationRouteConfig_noNotificationSettings(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "query", "env:test"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.#", "1"),
					resource.TestCheckResourceAttr("groundcover_notification_route.test", "routes.0.status.0", "Alerting"),
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "id"),
					// notification_settings should be computed with default/empty values
					resource.TestCheckResourceAttrSet("groundcover_notification_route.test", "notification_settings.%"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "groundcover_notification_route.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Apply again - should not cause changes (no apply loop)
			{
				Config:             testAccNotificationRouteConfig_noNotificationSettings(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccNotificationRouteConfig_noNotificationSettings(name string) string {
	return fmt.Sprintf(`
resource "groundcover_connected_app" "test" {
  name = "%[1]s-slack"
  type = "slack-webhook"
  data = {
    url = "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
  }
}

resource "groundcover_notification_route" "test" {
  name  = %[1]q
  query = "env:test"

  routes = [{
    status = ["Alerting"]
    connected_apps = [{
      type = "slack-webhook"
      id   = groundcover_connected_app.test.id
    }]
  }]
}
`, name)
}

// --- Unit tests for routes[*].connected_apps[*].params conversion ---

// routeParamsTestAttrTypes mirrors the expected attribute types of
// routes[*].connected_apps[*].params. Kept local to the tests so they document
// the schema contract independently of the implementation.
func routeParamsTestAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"channels":           types.ListType{ElemType: types.ObjectType{AttrTypes: routeChannelTestAttrTypes()}},
		"team_id":            types.StringType,
		"assignee_id":        types.StringType,
		"delegate_id":        types.StringType,
		"project_id":         types.StringType,
		"resolved_status_id": types.StringType,
		"label_ids":          types.ListType{ElemType: types.StringType},
		"auto_resolve":       types.BoolType,
	}
}

func routeChannelTestAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	}
}

func testRouteChannel(t *testing.T, id string, name attr.Value) attr.Value {
	t.Helper()
	obj, diags := types.ObjectValue(routeChannelTestAttrTypes(), map[string]attr.Value{
		"id":   types.StringValue(id),
		"name": name,
	})
	if diags.HasError() {
		t.Fatalf("failed to build channel object: %v", diags)
	}
	return obj
}

func testRouteChannelsList(t *testing.T, channels ...attr.Value) attr.Value {
	t.Helper()
	list, diags := types.ListValue(types.ObjectType{AttrTypes: routeChannelTestAttrTypes()}, channels)
	if diags.HasError() {
		t.Fatalf("failed to build channels list: %v", diags)
	}
	return list
}

// testRouteParamsObject builds a params object with every attribute null, then
// applies the given overrides.
func testRouteParamsObject(t *testing.T, overrides map[string]attr.Value) types.Object {
	t.Helper()
	values := map[string]attr.Value{
		"channels":           types.ListNull(types.ObjectType{AttrTypes: routeChannelTestAttrTypes()}),
		"team_id":            types.StringNull(),
		"assignee_id":        types.StringNull(),
		"delegate_id":        types.StringNull(),
		"project_id":         types.StringNull(),
		"resolved_status_id": types.StringNull(),
		"label_ids":          types.ListNull(types.StringType),
		"auto_resolve":       types.BoolNull(),
	}
	maps.Copy(values, overrides)
	obj, diags := types.ObjectValue(routeParamsTestAttrTypes(), values)
	if diags.HasError() {
		t.Fatalf("failed to build params object: %v", diags)
	}
	return obj
}

// testRoutesList builds a routes list with a single rule containing a single
// connected app with the given params value.
func testRoutesList(t *testing.T, appType, appID string, params attr.Value) types.List {
	t.Helper()
	ctx := context.Background()

	appObj, diags := types.ObjectValue(routeConnectedAppAttrTypes(), map[string]attr.Value{
		"type":   types.StringValue(appType),
		"id":     types.StringValue(appID),
		"params": params,
	})
	if diags.HasError() {
		t.Fatalf("failed to build connected app object: %v", diags)
	}

	appsList, diags := types.ListValue(types.ObjectType{AttrTypes: routeConnectedAppAttrTypes()}, []attr.Value{appObj})
	if diags.HasError() {
		t.Fatalf("failed to build connected apps list: %v", diags)
	}

	statusList, diags := types.ListValueFrom(ctx, types.StringType, []string{"Alerting"})
	if diags.HasError() {
		t.Fatalf("failed to build status list: %v", diags)
	}

	routeObj, diags := types.ObjectValue(routeRuleAttrTypes(), map[string]attr.Value{
		"status":         statusList,
		"connected_apps": appsList,
	})
	if diags.HasError() {
		t.Fatalf("failed to build route object: %v", diags)
	}

	routesList, diags := types.ListValue(types.ObjectType{AttrTypes: routeRuleAttrTypes()}, []attr.Value{routeObj})
	if diags.HasError() {
		t.Fatalf("failed to build routes list: %v", diags)
	}
	return routesList
}

func assertParamsJSON(t *testing.T, got map[string]any, wantJSON string) {
	t.Helper()
	gotBytes, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}
	var gotAny, wantAny any
	if err := json.Unmarshal(gotBytes, &gotAny); err != nil {
		t.Fatalf("failed to unmarshal marshaled params: %v", err)
	}
	if err := json.Unmarshal([]byte(wantJSON), &wantAny); err != nil {
		t.Fatalf("invalid expected JSON %q: %v", wantJSON, err)
	}
	if !reflect.DeepEqual(gotAny, wantAny) {
		t.Errorf("params mismatch\ngot:  %s\nwant: %s", gotBytes, wantJSON)
	}
}

func TestNotificationRouteConnectedAppParamsToSDK(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		appType  string
		params   func(t *testing.T) attr.Value
		wantJSON string // empty means Params must be nil
	}{
		{
			name:    "slack app channels",
			appType: "slack-app",
			params: func(t *testing.T) attr.Value {
				return testRouteParamsObject(t, map[string]attr.Value{
					"channels": testRouteChannelsList(t,
						testRouteChannel(t, "C123", types.StringValue("#alerts")),
						testRouteChannel(t, "C456", types.StringNull()),
					),
				})
			},
			wantJSON: `{"channels":[{"id":"C123","name":"#alerts"},{"id":"C456"}]}`,
		},
		{
			name:    "linear full options",
			appType: "linear",
			params: func(t *testing.T) attr.Value {
				return testRouteParamsObject(t, map[string]attr.Value{
					"team_id":            types.StringValue("T1"),
					"assignee_id":        types.StringValue("U1"),
					"delegate_id":        types.StringValue("D1"),
					"project_id":         types.StringValue("P1"),
					"resolved_status_id": types.StringValue("S1"),
					"label_ids":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("L1"), types.StringValue("L2")}),
					"auto_resolve":       types.BoolValue(true),
				})
			},
			wantJSON: `{"team_id":"T1","assignee_id":"U1","delegate_id":"D1","project_id":"P1","resolved_status_id":"S1","label_ids":["L1","L2"],"auto_resolve":true}`,
		},
		{
			// The backend defaults auto_resolve to true when unset, so an
			// explicit false must be sent, not dropped.
			name:    "linear explicit auto_resolve false",
			appType: "linear",
			params: func(t *testing.T) attr.Value {
				return testRouteParamsObject(t, map[string]attr.Value{
					"team_id":      types.StringValue("T1"),
					"auto_resolve": types.BoolValue(false),
				})
			},
			wantJSON: `{"team_id":"T1","auto_resolve":false}`,
		},
		{
			name:    "linear unset auto_resolve omitted",
			appType: "linear",
			params: func(t *testing.T) attr.Value {
				return testRouteParamsObject(t, map[string]attr.Value{
					"team_id": types.StringValue("T1"),
				})
			},
			wantJSON: `{"team_id":"T1"}`,
		},
		{
			// During create, unset computed attributes are unknown in the plan
			// and must be omitted from the request.
			name:    "unknown auto_resolve omitted",
			appType: "linear",
			params: func(t *testing.T) attr.Value {
				return testRouteParamsObject(t, map[string]attr.Value{
					"team_id":      types.StringValue("T1"),
					"auto_resolve": types.BoolUnknown(),
				})
			},
			wantJSON: `{"team_id":"T1"}`,
		},
		{
			name:    "no params",
			appType: "slack-webhook",
			params: func(t *testing.T) attr.Value {
				return types.ObjectNull(routeParamsTestAttrTypes())
			},
			wantJSON: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			routesList := testRoutesList(t, tc.appType, "app-1", tc.params(t))

			sdkRoutes, diags := routesListToSDK(ctx, routesList)
			if diags.HasError() {
				t.Fatalf("routesListToSDK returned errors: %v", diags)
			}
			if len(sdkRoutes) != 1 || len(sdkRoutes[0].ConnectedApps) != 1 {
				t.Fatalf("unexpected SDK routes shape: %+v", sdkRoutes)
			}

			app := sdkRoutes[0].ConnectedApps[0]
			if app.Type == nil || *app.Type != tc.appType {
				t.Errorf("unexpected type: %v", app.Type)
			}
			if app.ID == nil || *app.ID != "app-1" {
				t.Errorf("unexpected id: %v", app.ID)
			}

			if tc.wantJSON == "" {
				if app.Params != nil {
					t.Errorf("expected nil Params, got: %+v", app.Params)
				}
				return
			}
			assertParamsJSON(t, app.Params, tc.wantJSON)
		})
	}
}

func TestNotificationRouteConnectedAppParamsFromSDK(t *testing.T) {
	ctx := context.Background()

	sdkRoutes := []*models.RouteRuleResponse{
		{
			Status: []string{"Alerting"},
			ConnectedApps: []*models.RouteConnectedAppResponse{
				{
					ID:   "app-slack",
					Type: "slack-app",
					// Shaped as it arrives from JSON decoding of the API response.
					Params: map[string]any{
						"channels": []any{
							map[string]any{"id": "C123", "name": "#alerts"},
							map[string]any{"id": "C456"},
						},
					},
				},
				{
					ID:   "app-linear",
					Type: "linear",
					Params: map[string]any{
						"team_id":            "T1",
						"resolved_status_id": "S1",
						"label_ids":          []any{"L1"},
						"auto_resolve":       true,
					},
				},
				{
					ID:   "app-webhook",
					Type: "slack-webhook",
				},
			},
		},
	}

	routesList, diags := routesSDKToList(ctx, sdkRoutes)
	if diags.HasError() {
		t.Fatalf("routesSDKToList returned errors: %v", diags)
	}

	routeObj, ok := routesList.Elements()[0].(types.Object)
	if !ok {
		t.Fatalf("unexpected route element type: %T", routesList.Elements()[0])
	}
	appsList, ok := routeObj.Attributes()["connected_apps"].(types.List)
	if !ok {
		t.Fatalf("unexpected connected_apps type: %T", routeObj.Attributes()["connected_apps"])
	}
	apps := appsList.Elements()
	if len(apps) != 3 {
		t.Fatalf("expected 3 connected apps, got %d", len(apps))
	}

	getParams := func(t *testing.T, app attr.Value) attr.Value {
		t.Helper()
		appObj, ok := app.(types.Object)
		if !ok {
			t.Fatalf("unexpected connected app element type: %T", app)
		}
		params, ok := appObj.Attributes()["params"]
		if !ok {
			t.Fatalf("connected app object has no params attribute: %v", appObj)
		}
		return params
	}

	wantSlack := testRouteParamsObject(t, map[string]attr.Value{
		"channels": testRouteChannelsList(t,
			testRouteChannel(t, "C123", types.StringValue("#alerts")),
			testRouteChannel(t, "C456", types.StringNull()),
		),
	})
	if got := getParams(t, apps[0]); !got.Equal(wantSlack) {
		t.Errorf("slack params mismatch\ngot:  %v\nwant: %v", got, wantSlack)
	}

	wantLinear := testRouteParamsObject(t, map[string]attr.Value{
		"team_id":            types.StringValue("T1"),
		"resolved_status_id": types.StringValue("S1"),
		"label_ids":          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("L1")}),
		"auto_resolve":       types.BoolValue(true),
	})
	if got := getParams(t, apps[1]); !got.Equal(wantLinear) {
		t.Errorf("linear params mismatch\ngot:  %v\nwant: %v", got, wantLinear)
	}

	wantNull := types.ObjectNull(routeParamsTestAttrTypes())
	if got := getParams(t, apps[2]); !got.Equal(wantNull) {
		t.Errorf("expected null params for app without params, got: %v", got)
	}
}
