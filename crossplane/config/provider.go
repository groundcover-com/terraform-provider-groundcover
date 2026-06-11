package config

import (
	"github.com/crossplane/upjet/pkg/config"

	"github.com/groundcover-com/terraform-provider-groundcover/pkg/tfprovider"

	"github.com/groundcover-com/terraform-provider-groundcover/crossplane/config/connectedapp"
	"github.com/groundcover-com/terraform-provider-groundcover/crossplane/config/dashboard"
	"github.com/groundcover-com/terraform-provider-groundcover/crossplane/config/monitor"
)

const (
	// resourcePrefix is the Terraform provider's resource name prefix.
	resourcePrefix = "groundcover"
	// modulePath is this Crossplane provider module's import path; upjet uses it to
	// generate import statements in the generated API and controller code.
	modulePath = "github.com/groundcover-com/terraform-provider-groundcover/crossplane"
	// rootGroup is the API group suffix for all generated CRDs.
	rootGroup = "groundcover.com"
)

// GetProvider builds the upjet provider configuration used for code generation.
//
// schema is the Terraform provider schema JSON (produced by `terraform providers
// schema -json` against the groundcover provider) and metadata is the provider metadata
// produced by upjet's scraper. Both are supplied by the generator entrypoint
// (cmd/generator) so this function stays free of embedded build artifacts.
//
// The provider is a terraform-plugin-framework provider, so only the
// TerraformPluginFramework include list is populated; the SDKv2/CLI include lists stay
// empty. The include list is scoped to the POC resources via ExternalNameConfigured.
func GetProvider(schema []byte, metadata []byte) *config.Provider {
	pc := config.NewProvider(
		schema,
		resourcePrefix,
		modulePath,
		metadata,
		config.WithRootGroup(rootGroup),
		config.WithShortName(resourcePrefix),
		// This is a terraform-plugin-framework provider, so all resources are sourced
		// through the PF include list. The CLI and SDKv2 include lists default to ".+"
		// (match everything), which would double-register every resource — empty them.
		config.WithIncludeList(nil),
		config.WithTerraformPluginSDKIncludeList(nil),
		config.WithTerraformPluginFrameworkIncludeList(ExternalNameConfigured()),
		// upjet introspects PF resource schemas through the live provider instance.
		config.WithTerraformPluginFrameworkProvider(tfprovider.New("dev")()),
		config.WithDefaultResourceOptions(ExternalNameConfigurations()),
	)

	for _, configure := range []func(*config.Provider){
		monitor.Configure,
		dashboard.Configure,
		connectedapp.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}
