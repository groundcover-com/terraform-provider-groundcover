// Package connectedapp configures the groundcover_connected_app resource. Its data is
// sensitive and redacted on read, so drift is detected via a server-computed data_hash
// rather than field comparison; the controller-layer observe decorator applies that
// hash contract (internal/observe).
package connectedapp

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_connected_app resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_connected_app", func(r *config.Resource) {
		r.ShortGroup = "integrations"
		r.Kind = "ConnectedApp"
	})
}
