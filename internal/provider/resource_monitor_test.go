// Copyright (c) HashiCorp, Inc.
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
		orgName := os.Getenv("GROUNDCOVER_ORG_NAME")
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