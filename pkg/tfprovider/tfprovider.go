// Package tfprovider exposes the groundcover Terraform provider's plugin-framework
// provider constructor outside of internal/, so the Crossplane code generator (a
// separate Go module under crossplane/) can hand the provider to upjet for schema
// introspection during generation.
package tfprovider

import (
	"github.com/hashicorp/terraform-plugin-framework/provider"

	internalprovider "github.com/groundcover-com/terraform-provider-groundcover/internal/provider"
)

// New returns the plugin-framework provider constructor for the given version. It is a
// thin re-export of internal/provider.New.
func New(version string) func() provider.Provider {
	return internalprovider.New(version)
}
