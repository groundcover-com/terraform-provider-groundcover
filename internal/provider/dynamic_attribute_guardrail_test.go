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
// (groundcover-com/crossplane-provider-groundcover, see BE-2207) from this provider's
// schema — cannot represent terraform-plugin-framework dynamic attributes. A dynamic
// field makes the generated resource impossible to reconcile (it fails at Connect when
// upjet rebuilds the TF state), and because the Crossplane provider lives in a separate
// repo generated from the published schema, the breakage is silent until a customer's
// reconcile fails.
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

	checked := 0
	for _, newResource := range p.Resources(ctx) {
		checked++
		r := newResource()

		var md resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerType}, &md)

		var sr resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sr)

		findDynamicAttrs(t, md.TypeName, "", sr.Schema.Attributes)
		findDynamicBlocks(t, md.TypeName, "", sr.Schema.Blocks)
	}

	// Guard against the check passing vacuously (e.g. if Resources ever returns empty).
	if checked == 0 {
		t.Fatal("no resources were checked; the guardrail is ineffective")
	}
}

// findDynamicAttrs flags schema.DynamicAttribute at any nesting depth, recursing through
// nested attributes. Note: it does not flag a dynamic *element* type inside a collection
// (e.g. ListAttribute{ElementType: types.DynamicType}) — that's exotic and unused here;
// widen the type switch if such a field is ever introduced.
func findDynamicAttrs(t *testing.T, resName, prefix string, attrs map[string]rschema.Attribute) {
	t.Helper()
	for name, att := range attrs {
		key := resName + "." + joinPath(prefix, name)
		switch a := att.(type) {
		case rschema.DynamicAttribute:
			if !knownDynamicAttrs[key] {
				t.Errorf("%q is a types.Dynamic attribute, which upjet cannot generate — it breaks the Crossplane provider (crossplane-provider-groundcover, BE-2207). "+
					"Expose this via a JSON-string sibling resource (see groundcover_connected_app_json), "+
					"or, if this resource is intentionally Terraform-only, add %q to knownDynamicAttrs with justification.", key, key)
			}
		case rschema.SingleNestedAttribute:
			findDynamicAttrs(t, resName, joinPath(prefix, name), a.Attributes)
		case rschema.ListNestedAttribute:
			findDynamicAttrs(t, resName, joinPath(prefix, name), a.NestedObject.Attributes)
		case rschema.SetNestedAttribute:
			findDynamicAttrs(t, resName, joinPath(prefix, name), a.NestedObject.Attributes)
		case rschema.MapNestedAttribute:
			findDynamicAttrs(t, resName, joinPath(prefix, name), a.NestedObject.Attributes)
		}
	}
}

// findDynamicBlocks recurses through schema blocks (which the codebase uses heavily, e.g.
// monitor_v2, synthetic_test) so a dynamic attribute can't hide inside a block and slip
// past the guardrail. Blocks contain both attributes and nested blocks.
func findDynamicBlocks(t *testing.T, resName, prefix string, blocks map[string]rschema.Block) {
	t.Helper()
	for name, blk := range blocks {
		path := joinPath(prefix, name)
		switch b := blk.(type) {
		case rschema.SingleNestedBlock:
			findDynamicAttrs(t, resName, path, b.Attributes)
			findDynamicBlocks(t, resName, path, b.Blocks)
		case rschema.ListNestedBlock:
			findDynamicAttrs(t, resName, path, b.NestedObject.Attributes)
			findDynamicBlocks(t, resName, path, b.NestedObject.Blocks)
		case rschema.SetNestedBlock:
			findDynamicAttrs(t, resName, path, b.NestedObject.Attributes)
			findDynamicBlocks(t, resName, path, b.NestedObject.Blocks)
		}
	}
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
