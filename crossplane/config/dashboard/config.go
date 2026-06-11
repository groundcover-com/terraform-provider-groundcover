// Package dashboard configures the groundcover_dashboard resource. Like monitors,
// dashboards are stored as opaque YAML, so drift suppression is handled by the
// controller-layer observe decorator (internal/observe).
package dashboard

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_dashboard resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_dashboard", func(r *config.Resource) {
		r.ShortGroup = "dashboards"
		r.Kind = "Dashboard"
	})
}
