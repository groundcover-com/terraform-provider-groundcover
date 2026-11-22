// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
)

func TestDeleteServiceAccountHTTPResponses(t *testing.T) {
	tests := []struct {
		name         string
		deleteStatus int
		listStatus   int
		listBody     string
		wantError    bool
	}{
		{
			name:         "bad request with account still present",
			deleteStatus: http.StatusBadRequest,
			listStatus:   http.StatusOK,
			listBody:     `[{"serviceAccountId":"test-id"}]`,
			wantError:    true,
		},
		{
			name:         "bad request with account already gone",
			deleteStatus: http.StatusBadRequest,
			listStatus:   http.StatusOK,
			listBody:     `[{"serviceAccountId":"other-id"}]`,
		},
		{
			name:         "bad request with failed lookup",
			deleteStatus: http.StatusBadRequest,
			listStatus:   http.StatusForbidden,
			wantError:    true,
		},
		{
			name:         "not found",
			deleteStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodDelete && r.URL.Path == "/api/rbac/service-account/test-id":
					w.WriteHeader(tt.deleteStatus)
				case tt.listStatus != 0 && r.Method == http.MethodGet && r.URL.Path == "/api/rbac/service-accounts/list":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.listStatus)
					if _, err := fmt.Fprint(w, tt.listBody); err != nil {
						t.Errorf("write list response: %v", err)
					}
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client, err := NewSdkClientWrapper(t.Context(), server.URL, "test-key", "test-backend")
			require.NoError(t, err)

			err = client.DeleteServiceAccount(t.Context(), "test-id")
			if tt.wantError {
				require.EqualError(t, err, `cannot delete service account "test-id"; revoke any active API keys and retry`)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAccServiceAccountResource(t *testing.T) {
	name := acctest.RandomWithPrefix("test-serviceaccount")
	updatedEmail := "updated-" + name + "@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceAccountResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "name", name),
					resource.TestCheckResourceAttrSet("groundcover_serviceaccount.test", "id"),
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "email", "test-"+name+"@example.com"),
				),
			},
			// Update and Read testing
			{
				Config: testAccServiceAccountResourceConfigUpdated(name, updatedEmail),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "name", name), // Name should stay the same
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "email", updatedEmail),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceAccountResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("test-serviceaccount")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServiceAccountResourceExists("groundcover_serviceaccount.test"),
					testAccCheckServiceAccountResourceDisappears("groundcover_serviceaccount.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccServiceAccountResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test_policy" {
  name        = "%[1]s-policy"
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test" {
  name         = %[1]q
  email        = "test-%[1]s@example.com"
  policy_uuids = [groundcover_policy.test_policy.uuid]
}
`, name)
}

func testAccServiceAccountResourceConfigUpdated(baseName, updatedEmail string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test_policy" {
  name        = "%[1]s-policy"
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test" {
  name         = %[1]q
  email        = %[2]q
  policy_uuids = [groundcover_policy.test_policy.uuid]
}
`, baseName, updatedEmail)
}

func testAccCheckServiceAccountResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service Account ID is set")
		}

		return nil
	}
}

func testAccCheckServiceAccountResourceDisappears(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service Account ID is set")
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
		if err := client.DeleteServiceAccount(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("Failed to delete service account: %v", err)
		}

		return nil
	}
}

// TestAccServiceAccountResource_edgeValues guards against a perpetual diff when optional
// fields are set to empty values and the API echoes them back differently.
func TestAccServiceAccountResource_edgeValues(t *testing.T) {
	name := acctest.RandomWithPrefix("test-serviceaccount-edge")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfig_edgeValues(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_serviceaccount.test", "description", ""),
				),
			},
		},
	})
}

func testAccServiceAccountResourceConfig_edgeValues(name string) string {
	return fmt.Sprintf(`
resource "groundcover_policy" "test_policy" {
  name        = "%[1]s-policy"
  description = "Test policy for service account"
  role = {
    read = "read"
  }
}

resource "groundcover_serviceaccount" "test" {
  name         = %[1]q
  email        = "test-%[1]s@example.com"
  description  = ""
  policy_uuids = [groundcover_policy.test_policy.uuid]
}
`, name)
}
