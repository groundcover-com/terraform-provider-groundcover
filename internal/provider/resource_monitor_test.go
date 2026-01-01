// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMonitorResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor")
	updatedName := acctest.RandomWithPrefix("test-monitor-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMonitorResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					// Check the YAML contains our title
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(name)),
				),
			},
			// ImportState testing
			{
				ResourceName:            "groundcover_monitor.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"monitor_yaml"}, // YAML may not be identical after import
			},
			// Update and Read testing
			{
				Config: testAccMonitorResourceConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(updatedName)),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccMonitorResource_complex(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-complex")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with complex monitor configuration
			{
				Config: testAccMonitorResourceConfigComplex(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					// Check that YAML contains expected elements
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile("title")),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile("model")),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile("severity")),
				),
			},
		},
	})
}

func TestAccMonitorResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMonitorResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMonitorResourceExists("groundcover_monitor.test"),
					testAccCheckMonitorResourceDisappears("groundcover_monitor.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccMonitorResource_trailingNewline tests that monitors with trailing newlines
// do not cause apply loops. This simulates the issue where the server returns YAML
// with different formatting (including trailing newlines) and verifies that the
// normalization fixes prevent unnecessary updates.
func TestAccMonitorResource_trailingNewline(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-newline")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with trailing newlines in YAML
			{
				Config: testAccMonitorResourceConfigWithTrailingNewline(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(name)),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This verifies that the normalization fixes prevent false drift detection
			{
				Config: testAccMonitorResourceConfigWithTrailingNewline(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(name)),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccMonitorResourceConfigWithTrailingNewline(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccMonitorResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: %[1]s
display:
  header: %[1]s Test Monitor
  description: Test monitor created by acceptance tests
severity: critical
model:
  queries:
    - name: test_query
      dataType: metrics
      pipeline:
        function:
          name: sum_over_time
          pipelines:
            - metric: up
          args:
          - 5m
  thresholds:
    - name: threshold_1
      inputName: test_query
      operator: gt
      values:
        - 1
evaluationInterval:
  interval: 1m
  pendingFor: 1m
measurementType: state
YAML
}
`, name)
}

func testAccMonitorResourceConfigComplex(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: %[1]s Complex Monitor
display:
  header: %[1]s Complex Test Monitor
  description: Complex test monitor with multiple conditions
  resourceHeaderLabels:
    - namespace
    - workload
  contextHeaderLabels:
    - cluster
severity: critical
model:
  queries:
    - name: complex_query
      dataType: metrics
      pipeline:
        function:
          name: avg_over_time
          pipelines:
            - function:
                name: avg_by
                args:
                  - container
                  - env
                pipelines:
                  - function:
                      name: rate
                      pipelines:
                        - metric: http_requests_total
          args:
          - 5m
      conditions:
        - filters:
            - op: match
              value: production
          key: environment
          origin: root
          type: string
  thresholds:
    - name: high_rate_threshold
      inputName: complex_query
      operator: gt
      values:
        - 1000
labels:
  team: platform
  environment: production
annotations: 
  description: High request rate detected for {{ .Labels.workload }}
  summary: Request rate exceeds threshold
  runbook_url: https://example.com/runbooks/high-rate
executionErrorState: OK
noDataState: OK
evaluationInterval:
  interval: 5m
  pendingFor: 2m
measurementType: state
YAML
}
`, name)
}

func testAccCheckMonitorResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Monitor ID is set")
		}

		return nil
	}
}

// testAccCheckMonitorResourcePrintDetails prints the monitor ID and YAML for verification
func testAccCheckMonitorResourcePrintDetails(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		monitorID := rs.Primary.ID
		monitorYaml := rs.Primary.Attributes["monitor_yaml"]

		fmt.Printf("\n✅ Monitor created successfully!\n")
		fmt.Printf("   Monitor ID: %s\n", monitorID)
		fmt.Printf("   Monitor YAML (first 200 chars): %s\n", func() string {
			if len(monitorYaml) > 200 {
				return monitorYaml[:200] + "..."
			}
			return monitorYaml
		}())
		fmt.Printf("   Full YAML length: %d characters\n\n", len(monitorYaml))

		return nil
	}
}

func testAccCheckMonitorResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Monitor ID is set")
		}

		// Create a provider client to delete the resource
		ctx := context.Background()

		// Get environment variables for client configuration
		apiKey := os.Getenv("GROUNDCOVER_API_KEY")
		orgName := os.Getenv("GROUNDCOVER_BACKEND_ID")
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
		if err := client.DeleteMonitor(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete monitor: %v", err)
		}

		return nil
	}
}

// testAccMonitorResourceConfigWithTrailingNewline creates a monitor config with trailing newlines
// to simulate the issue where YAML formatting differences cause apply loops
func testAccMonitorResourceConfigWithTrailingNewline(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: %[1]s
display:
  header: %[1]s Test Monitor
  description: Test monitor with trailing newlines
severity: critical
model:
  queries:
    - name: test_query
      dataType: metrics
      pipeline:
        function:
          name: sum_over_time
          pipelines:
            - metric: up
          args:
          - 5m
  thresholds:
    - name: threshold_1
      inputName: test_query
      operator: gt
      values:
        - 1
evaluationInterval:
  interval: 1m
  pendingFor: 1m
measurementType: state

YAML
}
`, name)
}

// TestAccMonitorResource_multilinePipeSyntax tests that monitors with multiline pipe syntax (|)
// for title and header fields do not cause apply loops. This simulates the issue shown in the
// image where `title: |` followed by the value on the next line should be normalized to `title: value`.
func TestAccMonitorResource_multilinePipeSyntax(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-pipe")
	titleValue := fmt.Sprintf("CloudSql Connection Count %s", name)
	headerValue := fmt.Sprintf("CloudSql Connection Count %s", name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with multiline pipe syntax (|) for title and header
			{
				Config: testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(titleValue))),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This verifies that semantic comparison treats `title: |\nvalue` and `title: value` as equivalent
			{
				Config: testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(titleValue))),
				),
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
				),
			},
		},
	})
}

// testAccMonitorResourceConfigWithMultilinePipe creates a monitor config using multiline pipe syntax (|)
// for title and header fields. This tests that single-line values using multiline pipe syntax
// (e.g., `title: |\n  value`) are normalized to simple string format (e.g., `title: value`)
// because Grafana/monitor API doesn't accept multiline pipe syntax for single-line values.
func testAccMonitorResourceConfigWithMultilinePipe(titleValue, headerValue string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: |
  %s
display:
  header: |
    %s
  description: Test monitor with multiline pipe syntax
severity: critical
model:
  queries:
    - name: test_query
      dataType: metrics
      pipeline:
        function:
          name: sum_over_time
          pipelines:
            - metric: up
          args:
          - 5m
  thresholds:
    - name: threshold_1
      inputName: test_query
      operator: gt
      values:
        - 1
evaluationInterval:
  interval: 1m
  pendingFor: 1m
measurementType: state
YAML
}
`, titleValue, headerValue)
}

func TestAccMonitorResource_applyLoopIssue(t *testing.T) {
	title := acctest.RandomWithPrefix("k8s eu-povs node not ready")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with the exact YAML from monitor test.yml
			{
				Config: testAccMonitorResourceConfigApplyLoop(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(title))),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Apply the same config again - should not detect changes (no apply loop)
			// This is the critical test - if there's an apply loop, this step will show changes
			{
				Config: testAccMonitorResourceConfigApplyLoop(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(title))),
				),
				// ExpectNonEmptyPlan is false by default, meaning we expect no changes
				// If there were an apply loop, this step would fail or show changes
			},
			// Step 3: Apply one more time to be absolutely sure there's no apply loop
			{
				Config: testAccMonitorResourceConfigApplyLoop(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
				),
			},
		},
	})
}

// testAccMonitorResourceConfigApplyLoop creates a monitor config that matches the user's
// monitor test.yml file.
func testAccMonitorResourceConfigApplyLoop(title string) string {
	// This is the exact YAML from monitor test.yml
	yaml := fmt.Sprintf(`title: %s
display:
  header: %s
  contextHeaderLabels:
  - cluster
  - node
  - environment
  - cluster
  - env
  description: is not ready
severity: error
measurementType: state
model:
  queries:
  - name: threshold_input_query
    # trailing space after status="true"}) tests yaml normalization
    expression: sum(kube_node_status_condition{cluster="eu-povs", condition="Ready",status="true"}) 
      by (cluster, node, environment) > 0
    datasourceType: prometheus
    queryType: instant
    rollup:
      function: last
      time: 10m
  thresholds:
  - name: threshold_1
    inputName: threshold_input_query
    operator: gt
    values:
    - 0
annotations:
  Pagerduty_Incidents: enabled
  Slack-Prod-Alerts: enabled
executionErrorState: Error
noDataState: OK
evaluationInterval:
  interval: 5m0s
  pendingFor: 5m0s
notificationSettings:
    renotificationInterval: 2h
isPaused: true`, title, title)

	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
%s
YAML
}
`, yaml)
}

// TestAccMonitorResource_multilineExpression tests that monitors with multiline expressions
// (where the expression spans multiple lines with trailing spaces) do not cause apply loops.
// This tests the specific issue where the API returns expressions on a single line, but the
// input has them split across lines with trailing spaces.
func TestAccMonitorResource_multilineExpression(t *testing.T) {
	name := acctest.RandomWithPrefix("test-monitor-multiline-expr")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create monitor with multiline expression (trailing space + continuation)
			{
				Config: testAccMonitorResourceConfigMultilineExpression(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(name))),
					testAccCheckMonitorResourcePrintDetails("groundcover_monitor.test"),
				),
			},
			// Step 2: Plan with the same config - should show no changes (no apply loop)
			// This explicitly verifies that multiline expressions are normalized correctly
			// and don't cause drift. If there were an apply loop, this would show changes.
			{
				Config:             testAccMonitorResourceConfigMultilineExpression(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // Explicitly expect no changes - this is the key check for apply loop
			},
			// Step 3: Apply the same config again - should not detect changes (no apply loop)
			// This verifies that applying doesn't trigger updates
			{
				Config: testAccMonitorResourceConfigMultilineExpression(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("groundcover_monitor.test", "monitor_yaml"),
					resource.TestMatchResourceAttr("groundcover_monitor.test", "monitor_yaml", regexp.MustCompile(regexp.QuoteMeta(name))),
				),
				ExpectNonEmptyPlan: false, // Explicitly expect no changes
			},
			// Step 4: Plan one more time to be absolutely sure there's no apply loop
			{
				Config:             testAccMonitorResourceConfigMultilineExpression(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // Explicitly expect no changes
			},
		},
	})
}

// testAccMonitorResourceConfigMultilineExpression creates a monitor config with a multiline expression
// that has a trailing space on the first line and continuation on the next line. This simulates
// the exact issue where the API returns the expression on a single line, but the input has it
// split across lines with trailing spaces.
func testAccMonitorResourceConfigMultilineExpression(name string) string {
	return fmt.Sprintf(`
resource "groundcover_monitor" "test" {
  monitor_yaml = <<-YAML
title: %s
display:
  header: %s
  description: Test monitor with multiline expression
severity: error
measurementType: state
model:
  queries:
  - name: threshold_input_query
    expression: sum(kube_node_status_condition{cluster="test", condition="Ready",status="true"}) 
      by (cluster, node, environment) > 0
    datasourceType: prometheus
    queryType: instant
    rollup:
      function: last
      time: 10m
  thresholds:
  - name: threshold_1
    inputName: threshold_input_query
    operator: gt
    values:
    - 0
executionErrorState: Error
noDataState: OK
evaluationInterval:
  interval: 5m0s
  pendingFor: 5m0s
isPaused: true
YAML
}
`, name, name)
}
