package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// NormalizeMonitorYaml sorts keys in a YAML string alphabetically using goccy/go-yaml.
// It also handles potential errors during parsing and marshalling.
func NormalizeMonitorYaml(ctx context.Context, yamlString string) (string, error) {
	file, err := parser.ParseBytes([]byte(yamlString), parser.ParseComments)
	if err != nil {
		tflog.Error(ctx, "Failed to parse YAML with goccy/go-yaml", map[string]interface{}{"error": err, "yaml": yamlString})
		return "", fmt.Errorf("failed to parse YAML with goccy/go-yaml: %w", err)
	}

	for _, doc := range file.Docs {
		sortAstNodeGoccy(doc.Body)
	}

	outputString := file.String()
	// goccy/go-yaml might add a trailing newline if the input didn't have one and was just a single line key: value for example.
	// For consistency, especially in tests, let's ensure we are strict about the output matching the normalized input.
	// However, typically YAML tools are flexible with trailing newlines.
	// If the original string was empty or just whitespace, String() might return "\n".
	if yamlString == "" && outputString == "\n" {
		return "", nil
	}

	return outputString, nil
}

// sortAstNodeGoccy recursively sorts nodes in the AST provided by goccy/go-yaml.
func sortAstNodeGoccy(node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.MappingNode:
		sort.Slice(n.Values, func(i, j int) bool {
			keyI, okI := n.Values[i].Key.(ast.ScalarNode)
			keyJ, okJ := n.Values[j].Key.(ast.ScalarNode)
			if okI && okJ {
				valI := keyI.GetValue()
				valJ := keyJ.GetValue()
				if valI != nil && valJ != nil {
					strValI, okStrI := valI.(string)
					strValJ, okStrJ := valJ.(string)
					if okStrI && okStrJ {
						return strValI < strValJ
					}
				}
			}
			return false // Default to not swapping if keys are not comparable strings
		})
		for _, valNode := range n.Values {
			sortAstNodeGoccy(valNode.Value)
		}
	case *ast.SequenceNode:
		for _, valNode := range n.Values {
			sortAstNodeGoccy(valNode)
		}
	case *ast.DocumentNode:
		sortAstNodeGoccy(n.Body)
	}
}
