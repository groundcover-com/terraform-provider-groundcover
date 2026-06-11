// Command generator runs the upjet code-generation pipeline for the groundcover
// Crossplane provider. It reads the Terraform provider schema (and optional provider
// metadata), builds the provider configuration, and generates the CRD API types,
// controllers, and example manifests under the module root.
//
// It expects:
//   - config/schema.json          Terraform provider schema, produced by
//     `terraform providers schema -json` against the groundcover provider.
//   - config/provider-metadata.yaml  (optional) provider metadata for richer docs.
//
// Invoke via `make generate`, which prepares the schema and runs `go run` on this
// command from the module root.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/crossplane/upjet/pkg/pipeline"

	"github.com/groundcover-com/terraform-provider-groundcover/crossplane/config"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	schema, err := os.ReadFile(filepath.Join(root, "config", "schema.json"))
	if err != nil {
		panic("read provider schema (run `make generate`, which produces config/schema.json): " + err.Error())
	}

	// upjet's NewProvider converts the whole schema to an SDKv2 resource map and panics
	// on cty DynamicPseudoType attributes (e.g. connected_app.data, a JSON object). Coerce
	// dynamic attributes to JSON strings before generation: the CRD then carries the data
	// as an opaque JSON string, which is how a Crossplane user supplies it and pairs with
	// the hash-based observe strategy for that resource.
	schema = coerceDynamicAttributesToString(schema)

	// Provider metadata is optional; absence just yields less rich generated docs.
	metadata, err := os.ReadFile(filepath.Join(root, "config", "provider-metadata.yaml"))
	if err != nil {
		metadata = nil
	}

	pipeline.Run(config.GetProvider(schema, metadata), root)
}

// coerceDynamicAttributesToString rewrites every attribute whose Terraform type is the
// dynamic pseudo-type to "string" across all resource, data-source, and provider blocks.
func coerceDynamicAttributesToString(schema []byte) []byte {
	var doc map[string]any
	if err := json.Unmarshal(schema, &doc); err != nil {
		panic("parse provider schema JSON: " + err.Error())
	}

	providerSchemas, _ := doc["provider_schemas"].(map[string]any)
	for _, ps := range providerSchemas {
		psMap, ok := ps.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"resource_schemas", "data_source_schemas"} {
			schemas, ok := psMap[key].(map[string]any)
			if !ok {
				continue
			}
			for _, rs := range schemas {
				if rsMap, ok := rs.(map[string]any); ok {
					coerceBlock(rsMap["block"])
				}
			}
		}
		if prov, ok := psMap["provider"].(map[string]any); ok {
			coerceBlock(prov["block"])
		}
	}

	out, err := json.Marshal(doc)
	if err != nil {
		panic("re-marshal provider schema JSON: " + err.Error())
	}
	return out
}

func coerceBlock(block any) {
	b, ok := block.(map[string]any)
	if !ok {
		return
	}
	if attrs, ok := b["attributes"].(map[string]any); ok {
		for _, attr := range attrs {
			am, ok := attr.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := am["type"].(string); ok && t == "dynamic" {
				am["type"] = "string"
			}
		}
	}
	if blockTypes, ok := b["block_types"].(map[string]any); ok {
		for _, bt := range blockTypes {
			if btMap, ok := bt.(map[string]any); ok {
				coerceBlock(btMap["block"])
			}
		}
	}
}
