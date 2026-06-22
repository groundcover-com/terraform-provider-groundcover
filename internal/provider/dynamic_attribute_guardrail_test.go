// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// knownDynamicAttrs is the allowlist of resource attributes that are intentionally
// types.Dynamic (schema.DynamicAttribute).
//
// WHY THIS GUARDRAIL EXISTS: upjet — which generates the standalone Crossplane provider
// (groundcover-com/provider-groundcover, see BE-2207) from this provider's schema —
// cannot represent terraform-plugin-framework dynamic attributes. A dynamic field makes
// the generated resource impossible to reconcile (it fails at Connect when upjet rebuilds
// the TF state), and because the Crossplane provider lives in a separate repo generated
// from the published schema, the breakage is silent until a customer's reconcile fails.
//
// So a NEW dynamic attribute must be a deliberate, reviewed choice. If you are adding one:
// the Crossplane-friendly path is a JSON-string sibling resource (see
// groundcover_connected_app_json mirroring datadog_dashboard_json). Only add an entry here
// if that resource is intentionally Terraform-only / not exposed via Crossplane.
var knownDynamicAttrs = map[string]bool{
	// The dynamic original; groundcover_connected_app_json is its Crossplane-friendly twin.
	"groundcover_connected_app.data": true,
}

// TestNoUnexpectedDynamicAttributes fails when a resource exposes a types.Dynamic attribute
// that is not on the knownDynamicAttrs allowlist, preventing accidental breakage of the
// upjet-generated Crossplane provider.
func TestNoUnexpectedDynamicAttributes(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	var pmd fwprovider.MetadataResponse
	p.Metadata(ctx, fwprovider.MetadataRequest{}, &pmd)
	providerType := pmd.TypeName
	if providerType == "" {
		providerType = "groundcover"
	}

	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var md resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerType}, &md)

		var sr resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sr)

		findDynamicAttrs(t, md.TypeName, "", sr.Schema.Attributes)
	}
}

// findDynamicAttrs flags schema.DynamicAttribute at any nesting depth.
// ponytail: it does not flag a dynamic *element* type inside a collection
// (e.g. ListAttribute{ElementType: types.DynamicType}) — that's exotic and unused here;
// widen the type switch if such a field is ever introduced.
func findDynamicAttrs(t *testing.T, resName, prefix string, attrs map[string]rschema.Attribute) {
	t.Helper()
	for name, att := range attrs {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		key := resName + "." + path

		switch a := att.(type) {
		case rschema.DynamicAttribute:
			if !knownDynamicAttrs[key] {
				t.Errorf("%q is a types.Dynamic attribute, which upjet cannot generate — it breaks the Crossplane provider (provider-groundcover, BE-2207). "+
					"Expose this via a JSON-string sibling resource (see groundcover_connected_app_json), "+
					"or, if this resource is intentionally Terraform-only, add %q to knownDynamicAttrs with justification.", key, key)
			}
		// Recurse into nested attributes so a dynamic field can't hide inside a block.
		case rschema.SingleNestedAttribute:
			findDynamicAttrs(t, resName, path, a.Attributes)
		case rschema.ListNestedAttribute:
			findDynamicAttrs(t, resName, path, a.NestedObject.Attributes)
		case rschema.SetNestedAttribute:
			findDynamicAttrs(t, resName, path, a.NestedObject.Attributes)
		case rschema.MapNestedAttribute:
			findDynamicAttrs(t, resName, path, a.NestedObject.Attributes)
		}
	}
}
