package observe

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Connector decorates an upjet ExternalConnecter so the ExternalClient it produces
// suppresses semantically-irrelevant drift via the provided Strategy. It implements
// managed.ExternalConnecter and is wired into a resource's controller Setup with
// managed.WithExternalConnecter.
type Connector struct {
	// Inner is the upjet-generated terraform-plugin-framework connector.
	Inner    managed.ExternalConnecter
	Strategy Strategy
	Log      logging.Logger
}

// NewConnector wraps inner with drift suppression governed by strategy.
func NewConnector(inner managed.ExternalConnecter, strategy Strategy, log logging.Logger) *Connector {
	return &Connector{Inner: inner, Strategy: strategy, Log: log}
}

// Connect delegates to the inner connector and wraps the returned client.
func (c *Connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ec, err := c.Inner.Connect(ctx, mg)
	if err != nil {
		return nil, err
	}
	return &client{ExternalClient: ec, strategy: c.Strategy, log: c.Log}, nil
}

// client embeds the inner ExternalClient so Create/Update/Delete (and any
// version-specific methods such as Disconnect) are promoted unchanged; only Observe is
// overridden.
type client struct {
	managed.ExternalClient
	strategy Strategy
	log      logging.Logger
}

// Observe runs the inner observation, then — only when the inner client reports the
// resource exists but is out of date — asks the strategy whether the difference is
// purely cosmetic. If so, it reports the resource as up to date, preventing upjet from
// issuing a no-op Update every reconcile. Any other observation is passed through
// untouched, and a strategy error is logged and treated as "no suppression" so real
// drift is never hidden by a comparison failure.
func (c *client) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	obs, err := c.ExternalClient.Observe(ctx, mg)
	if err != nil || !obs.ResourceExists || obs.ResourceUpToDate {
		return obs, err
	}

	upToDate, sErr := c.strategy.UpToDate(ctx, mg)
	if sErr != nil {
		if c.log != nil {
			c.log.Debug("drift suppression comparison failed; preserving raw observation",
				"error", sErr, "name", mg.GetName())
		}
		return obs, nil
	}

	if upToDate {
		if c.log != nil {
			c.log.Debug("suppressing cosmetic drift; reporting resource up to date",
				"name", mg.GetName())
		}
		obs.ResourceUpToDate = true
	}
	return obs, nil
}
