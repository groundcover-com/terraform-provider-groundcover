// Package config holds the upjet provider configuration that drives CRD/controller
// generation for the groundcover Crossplane provider.
package config

import "github.com/crossplane/upjet/pkg/config"

// ExternalNameConfigs maps each Terraform resource to its external-name handling.
// groundcover resources are identified by a server-assigned UUID returned in the
// Terraform "id" field, so they all use IdentifierFromProvider: the external name is
// whatever the provider assigns on create, and no name field is sent on the request.
var ExternalNameConfigs = map[string]config.ExternalName{
	"groundcover_monitor":       config.IdentifierFromProvider,
	"groundcover_dashboard":     config.IdentifierFromProvider,
	"groundcover_connected_app": config.IdentifierFromProvider,
}

// ExternalNameConfigured returns the list of Terraform resources that have an
// external-name configuration, in the regex form upjet's include lists expect (anchored
// with a trailing "$"). It feeds WithTerraformPluginFrameworkIncludeList so only the
// POC resources are generated.
func ExternalNameConfigured() []string {
	l := make([]string, 0, len(ExternalNameConfigs))
	for name := range ExternalNameConfigs {
		l = append(l, name+"$")
	}
	return l
}

// ExternalNameConfigurations applies the ExternalNameConfigs to the matching resource
// during configuration. Registered as a default resource option.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}
