// Package observe implements the custom Observe drift-suppression that lets the
// upjet-generated Crossplane provider behave like the Terraform provider.
//
// Upjet reconstructs Terraform state from a managed resource's spec.forProvider on
// every reconcile and diffs the raw observed state, so the Terraform provider's
// drift-suppression (which preserves the user's authored value when it is
// semantically equivalent to the server response) is lost — producing perpetual
// drift on connected_apps, monitors, and dashboards.
//
// terraform-plugin-framework resources expose no upjet diff-customization hook
// (TerraformCustomDiff is SDKv2-only), so this package instead decorates the runtime
// managed.ExternalClient: after the underlying Observe runs, it re-evaluates spurious
// drift using the exact same comparison logic as the Terraform provider
// (github.com/groundcover-com/terraform-provider-groundcover/pkg/normalize) and, when
// the difference is purely cosmetic, reports the resource as up to date.
package observe

import (
	"context"

	"github.com/groundcover-com/terraform-provider-groundcover/pkg/normalize"
)

// YAMLUpToDate reports whether an observed YAML document is semantically equivalent to
// the desired one, mirroring the Terraform monitor/dashboard Read path: the observed
// document is first filtered to the keys the user actually authored (so server-added
// fields don't count as drift), then both sides are normalized (key order, durations,
// defaults, empty/ignored fields) and deep-compared.
//
// An empty desired document means "nothing authored to compare against": we treat that
// as up to date so the decorator never manufactures drift from a missing baseline.
func YAMLUpToDate(ctx context.Context, desiredYAML, observedYAML string) (bool, error) {
	if desiredYAML == "" {
		return true, nil
	}

	filteredObserved, err := normalize.FilterYamlKeysBasedOnTemplate(ctx, observedYAML, desiredYAML)
	if err != nil {
		return false, err
	}

	normalizedDesired, err := normalize.NormalizeMonitorYaml(ctx, desiredYAML)
	if err != nil {
		return false, err
	}

	normalizedObserved, err := normalize.NormalizeMonitorYaml(ctx, filteredObserved)
	if err != nil {
		return false, err
	}

	return normalize.CompareYamlSemantically(normalizedDesired, normalizedObserved)
}

// HashUpToDate reports whether a server-computed content hash indicates the resource is
// unchanged, mirroring the Terraform connected_app data_hash contract: with no
// trustworthy baseline (empty recorded or remote hash) it reports up to date rather
// than flagging drift.
func HashUpToDate(recordedHash, remoteHash string) bool {
	return !normalize.HashDrifted(recordedHash, remoteHash)
}
