// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &GroundcoverProvider{}

// GroundcoverProvider defines the provider implementation.
type GroundcoverProvider struct {
	// version is set dynamically by the build process.
	version string
}

// GroundcoverProviderModel describes the provider data model.
type GroundcoverProviderModel struct {
	ApiKey    types.String `tfsdk:"api_key"`
	OrgName   types.String `tfsdk:"org_name"` // Kept for backwards compatibility
	BackendId types.String `tfsdk:"backend_id"`
	ApiUrl    types.String `tfsdk:"api_url"`
}

func (p *GroundcoverProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "groundcover"
	resp.Version = p.version
}

func (p *GroundcoverProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing groundcover resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "groundcover API Key. Can also be set via the GROUNDCOVER_API_KEY environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"org_name": schema.StringAttribute{
				MarkdownDescription: "groundcover Organization Name. Can also be set via the GROUNDCOVER_ORG_NAME environment variable. Deprecated: Use backend_id instead.",
				Optional:            true,
			},
			"backend_id": schema.StringAttribute{
				MarkdownDescription: "groundcover Backend ID. Can also be set via the GROUNDCOVER_BACKEND_ID environment variable.",
				Optional:            true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "groundcover API URL. Defaults to the groundcover production URL. Can also be set via the GROUNDCOVER_API_URL environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *GroundcoverProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Groundcover provider")

	var config GroundcoverProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("GROUNDCOVER_API_KEY")
	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}

	// Handle backwards compatibility: org_name is kept for backwards compatibility
	orgName := os.Getenv("GROUNDCOVER_ORG_NAME")
	if !config.OrgName.IsNull() {
		orgName = config.OrgName.ValueString()
	}

	// Check for backend_id (new preferred option)
	backendId := os.Getenv("GROUNDCOVER_BACKEND_ID")
	if !config.BackendId.IsNull() {
		backendId = config.BackendId.ValueString()
	}

	// Use backend_id if provided, otherwise fall back to org_name
	if backendId != "" {
		orgName = backendId
	}

	apiUrl := os.Getenv("GROUNDCOVER_API_URL")
	if !config.ApiUrl.IsNull() {
		apiUrl = config.ApiUrl.ValueString()
	}
	// Set default API URL if not provided
	if apiUrl == "" {
		apiUrl = "https://api.groundcover.com"
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Groundcover API Key",
			"The provider cannot create the Groundcover API client as no API Key was found.\n\n"+
				"Either set the `api_key` provider configuration argument or set the GROUNDCOVER_API_KEY environment variable. If both are set, the provider configuration argument takes precedence.",
		)
	}

	if orgName == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("backend_id"),
			"Missing Groundcover Backend ID",
			"The provider cannot create the Groundcover API client as no Backend ID was found.\n\n"+
				"Either set the `backend_id` provider configuration argument or set the GROUNDCOVER_BACKEND_ID environment variable.\n"+
				"For backwards compatibility, you can also use `org_name` or GROUNDCOVER_ORG_NAME environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Initializing Groundcover SDK client", map[string]any{"backend_id": orgName, "api_url": apiUrl})
	clientWrapper, err := NewSdkClientWrapper(ctx, apiUrl, apiKey, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Create API Client Wrapper",
			"The provider failed to initialize the internal SDK client wrapper: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = clientWrapper
	resp.ResourceData = clientWrapper

	tflog.Info(ctx, "Groundcover provider configured successfully")
}

func (p *GroundcoverProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPolicyResource,
		NewServiceAccountResource,
		NewMonitorResource,
		NewApiKeyResource,
		NewLogsPipelineResource,
		NewIngestionKeyResource,
		NewIntegrationResource,
	}
}

func (p *GroundcoverProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// CloseEphemeralResource closes an opened ephemeral resource.
// This method is required by the tfprotov5.ProviderServer interface in recent versions of the framework.
func (p *GroundcoverProvider) CloseEphemeralResource(ctx context.Context, req *tfprotov5.CloseEphemeralResourceRequest) (*tfprotov5.CloseEphemeralResourceResponse, error) {
	return &tfprotov5.CloseEphemeralResourceResponse{}, nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GroundcoverProvider{
			version: version,
		}
	}
}
