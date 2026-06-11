// Package monitor configures the groundcover_monitor resource for the Crossplane
// provider. The monitor definition is an opaque YAML string, so all meaningful drift
// suppression happens at the controller layer (internal/observe) rather than in the
// generated schema.
package monitor

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_monitor resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_monitor", func(r *config.Resource) {
		r.ShortGroup = "monitoring"
		r.Kind = "Monitor"
	})
}
