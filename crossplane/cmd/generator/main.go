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

	// Provider metadata is optional; absence just yields less rich generated docs.
	metadata, err := os.ReadFile(filepath.Join(root, "config", "provider-metadata.yaml"))
	if err != nil {
		metadata = nil
	}

	pipeline.Run(config.GetProvider(schema, metadata), root)
}
