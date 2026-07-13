// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	openapiClient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	goclient "github.com/groundcover-com/groundcover-sdk-go/pkg/client"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	gcsdktransport "github.com/groundcover-com/groundcover-sdk-go/pkg/transport"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestSkillRequestFromModelIsOrganizational(t *testing.T) {
	req := skillRequestFromModel(skillResourceModel{
		Name: types.StringValue("incident-response"), WhenToUse: types.StringValue("When investigating an incident"),
		Description: types.StringValue("Incident workflow"), Instructions: types.StringValue("Inspect alerts and summarize evidence."),
	})
	if req.IsOrganizational == nil || !*req.IsOrganizational {
		t.Fatal("IsOrganizational must always be true")
	}
	if req.Name == nil || *req.Name != "incident-response" || req.Instructions == nil || *req.Instructions == "" {
		t.Fatalf("unexpected request mapping: %#v", req)
	}
}

func TestSkillModelFromAPI(t *testing.T) {
	id, name, whenToUse, instructions := "skill-id", "incident-response", "During incidents", "Follow the runbook"
	createdAt, updatedAt := "2026-07-13T10:00:00Z", "2026-07-13T11:00:00Z"
	revision := int64(2)
	organizational, provisioned := true, true
	model, diags := skillModelFromAPI(&models.AgentSkillDetail{
		ID: &id, Name: &name, WhenToUse: &whenToUse, Instructions: &instructions,
		Description: "description", Identifier: "/incident-response#skill-id", Revision: &revision,
		IsOrganizational: &organizational, IsProvisioned: &provisioned,
		CreatedAt: &createdAt, CreatedBy: "creator", UpdatedAt: &updatedAt, UpdatedBy: "updater",
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}
	if model.ID.ValueString() != id || model.Revision.ValueInt64() != revision || !model.IsProvisioned.ValueBool() {
		t.Fatalf("unexpected model: %#v", model)
	}
}

func TestSkillModelFromAPIRejectsMissingRequiredFields(t *testing.T) {
	_, diags := skillModelFromAPI(&models.AgentSkillDetail{})
	if !diags.HasError() {
		t.Fatal("expected diagnostics for a malformed API response")
	}
}

func TestSkillStringValidators(t *testing.T) {
	tests := []struct {
		name      string
		validator validator.String
		value     string
		wantError bool
	}{
		{name: "instructions accept multiline content", validator: nonWhitespaceStringValidator{}, value: "\nFollow the runbook.\n"},
		{name: "instructions reject whitespace only", validator: nonWhitespaceStringValidator{}, value: " \n\t", wantError: true},
		{name: "trimmed value accepted", validator: trimmedStringValidator{}, value: "incident-response"},
		{name: "leading whitespace rejected", validator: trimmedStringValidator{}, value: " incident-response", wantError: true},
		{name: "empty optional description accepted", validator: trimmedStringValidator{allowEmpty: true}, value: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validator.StringRequest{ConfigValue: types.StringValue(tt.value), Path: path.Root("value")}
			var response validator.StringResponse
			tt.validator.ValidateString(context.Background(), request, &response)
			if response.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("HasError() = %t, want %t; diagnostics: %v", response.Diagnostics.HasError(), tt.wantError, response.Diagnostics)
			}
		})
	}
}

func TestCreateSkillSendsTerraformUserAgent(t *testing.T) {
	var gotUserAgent string
	baseTransport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		gotUserAgent = r.Header.Get("User-Agent")
		return &http.Response{
			StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:    io.NopCloser(strings.NewReader(`{"status":"ok","skill":{"id":"skill-id","name":"name","when_to_use":"when","instructions":"do","revision":1,"is_organizational":true,"is_provisioned":true,"created_at":"now","updated_at":"now"}}`)),
			Request: r,
		}, nil
	})
	client := newSkillSDKTestClient(baseTransport)
	name, whenToUse, instructions, organizational := "name", "when", "do", true
	if _, err := client.CreateSkill(context.Background(), &models.AgentSkillRequest{
		Name: &name, WhenToUse: &whenToUse, Instructions: &instructions, IsOrganizational: &organizational,
	}); err != nil {
		t.Fatalf("CreateSkill() error: %v", err)
	}
	if gotUserAgent != terraformProviderUserAgent {
		t.Fatalf("User-Agent = %q, want %q", gotUserAgent, terraformProviderUserAgent)
	}
}

func TestDeleteSkillTreatsNotFoundAsSuccess(t *testing.T) {
	client := newSkillSDKTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound, Header: http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"status":"error","message":"not found"}`)), Request: r,
		}, nil
	}))
	if err := client.DeleteSkill(context.Background(), "missing-skill-id"); err != nil {
		t.Fatalf("DeleteSkill() error for missing Skill: %v", err)
	}
}

func newSkillSDKTestClient(baseTransport http.RoundTripper) *SdkClientWrapper {
	sdkHTTPTransport := gcsdktransport.NewTransport("api-key", "backend-id", baseTransport, 0, time.Millisecond, time.Millisecond, nil)
	runtimeTransport := openapiClient.New("api.groundcover.com", "/", []string{"https"})
	runtimeTransport.Transport = sdkHTTPTransport
	return &SdkClientWrapper{sdkClient: goclient.New(runtimeTransport, strfmt.Default)}
}

func TestAccSkillResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-skill")
	updatedName := name + "-updated"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSkillConfig(name, "Use while investigating incidents", "Start with active alerts."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("groundcover_skill.test", "id"),
					resource.TestCheckResourceAttr("groundcover_skill.test", "name", name),
					resource.TestCheckResourceAttr("groundcover_skill.test", "is_organizational", "true"),
					resource.TestCheckResourceAttr("groundcover_skill.test", "is_provisioned", "true"),
					resource.TestCheckResourceAttrSet("groundcover_skill.test", "revision"),
				),
			},
			{ResourceName: "groundcover_skill.test", ImportState: true, ImportStateVerify: true},
			{
				Config: testAccSkillConfig(updatedName, "Use during incident response", "Inspect alerts, then summarize evidence."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("groundcover_skill.test", "name", updatedName),
					resource.TestCheckResourceAttr("groundcover_skill.test", "when_to_use", "Use during incident response"),
				),
			},
			{
				Config:   testAccSkillConfig(updatedName, "Use during incident response", "Inspect alerts, then summarize evidence."),
				PlanOnly: true,
			},
		},
	})
}

func TestAccSkillResource_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-skill-disappears")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccSkillConfig(name, "Use during incident response", "Inspect active alerts."),
			Check: resource.ComposeAggregateTestCheckFunc(
				testAccCheckSkillResourceDisappears("groundcover_skill.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccSkillConfig(name, whenToUse, instructions string) string {
	return fmt.Sprintf(`
resource "groundcover_skill" "test" {
  name         = %[1]q
  when_to_use  = %[2]q
  description  = "Managed by Terraform acceptance tests"
  instructions = %[3]q
}
`, name, whenToUse, instructions)
}

func testAccCheckSkillResourceDisappears(name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return fmt.Errorf("Skill resource %q has no ID", name)
		}
		apiURL := os.Getenv("GROUNDCOVER_API_URL")
		if apiURL == "" {
			apiURL = "https://api.groundcover.com"
		}
		client, err := NewSdkClientWrapper(context.Background(), apiURL, os.Getenv("GROUNDCOVER_API_KEY"), os.Getenv("GROUNDCOVER_BACKEND_ID"))
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		return client.DeleteSkill(context.Background(), resourceState.Primary.ID)
	}
}
