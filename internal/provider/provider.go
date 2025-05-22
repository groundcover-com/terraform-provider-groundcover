// Copyright (c) HashiCorp, Inc.
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
	ApiKey  types.String `tfsdk:"api_key"`
	OrgName types.String `tfsdk:"org_name"`
	ApiUrl  types.String `tfsdk:"api_url"`
}

func (p *GroundcoverProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "groundcover"
	resp.Version = p.version
}

func (p *GroundcoverProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing Groundcover resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Groundcover API Key. Can also be set via the GROUNDCOVER_API_KEY environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"org_name": schema.StringAttribute{
				MarkdownDescription: "Groundcover Organization Name. Can also be set via the GROUNDCOVER_ORG_NAME environment variable.",
				Optional:            true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Groundcover API URL. Defaults to the Groundcover production URL. Can also be set via the GROUNDCOVER_API_URL environment variable.",
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

	orgName := os.Getenv("GROUNDCOVER_ORG_NAME")
	if !config.OrgName.IsNull() {
		orgName = config.OrgName.ValueString()
	}

	apiUrl := os.Getenv("GROUNDCOVER_API_URL") // Default handled by SDK if empty
	if !config.ApiUrl.IsNull() {
		apiUrl = config.ApiUrl.ValueString()
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
			path.Root("org_name"),
			"Missing Groundcover Organization Name",
			"The provider cannot create the Groundcover API client as no Organization Name was found.\n\n"+
				"Either set the `org_name` provider configuration argument or set the GROUNDCOVER_ORG_NAME environment variable. If both are set, the provider configuration argument takes precedence.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Initializing Groundcover SDK client", map[string]any{"org_name": orgName, "api_url": apiUrl})
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
	}
}

func (p *GroundcoverProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GroundcoverProvider{
			version: version,
		}
	}
}
