package provider

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Constants for AST node markers
const (
	ScalarMarker   = "__is_scalar"
	SequenceMarker = "__is_sequence"
	ItemKeysField  = "__item_keys"
)

// Pre-compiled regexes for time normalization
var (
	timeRegex      = regexp.MustCompile(`\b(?:\d+[hms])+\b`)
	componentRegex = regexp.MustCompile(`(\d+)([hms])`)
	zeroRegex      = regexp.MustCompile(`^0[hms]$`)
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

// FilterYamlKeysBasedOnTemplate filters sourceYaml to only include keys that exist in templateYaml.
// This is useful for drift detection when you only want to compare fields that the user originally specified.
func FilterYamlKeysBasedOnTemplate(ctx context.Context, sourceYaml, templateYaml string) (string, error) {
	if sourceYaml == "" {
		return "", nil
	}
	if templateYaml == "" {
		return sourceYaml, nil // If no template, return source as-is
	}

	// Parse both YAMLs
	sourceFile, err := parser.ParseBytes([]byte(sourceYaml), parser.ParseComments)
	if err != nil {
		tflog.Error(ctx, "Failed to parse source YAML in FilterYamlKeysBasedOnTemplate", map[string]interface{}{"error": err, "yaml": sourceYaml})
		return "", fmt.Errorf("failed to parse source YAML: %w", err)
	}

	templateFile, err := parser.ParseBytes([]byte(templateYaml), parser.ParseComments)
	if err != nil {
		tflog.Error(ctx, "Failed to parse template YAML in FilterYamlKeysBasedOnTemplate", map[string]interface{}{"error": err, "yaml": templateYaml})
		return "", fmt.Errorf("failed to parse template YAML: %w", err)
	}

	// Get template keys structure
	var templateKeys map[string]interface{}
	if len(templateFile.Docs) > 0 && templateFile.Docs[0].Body != nil {
		templateKeys = extractKeysFromAstNode(ctx, templateFile.Docs[0].Body)
	}

	// Filter source based on template keys
	if len(sourceFile.Docs) > 0 && sourceFile.Docs[0].Body != nil {
		filterAstNodeByKeys(ctx, sourceFile.Docs[0].Body, templateKeys)
	}

	filteredYaml := sourceFile.String()
	if sourceYaml == "" && filteredYaml == "\n" {
		return "", nil
	}

	return filteredYaml, nil
}

// extractKeysFromAstNode recursively extracts the key structure from an AST node
func extractKeysFromAstNode(ctx context.Context, node ast.Node) map[string]interface{} {
	if node == nil {
		return nil
	}

	result := make(map[string]interface{})

	switch n := node.(type) {
	case *ast.MappingNode:
		for _, valNode := range n.Values {
			if keyNode, ok := valNode.Key.(ast.ScalarNode); ok {
				if keyValue := keyNode.GetValue(); keyValue != nil {
					if keyStr, ok := keyValue.(string); ok {
						// Recursively get nested structure
						result[keyStr] = extractKeysFromAstNode(ctx, valNode.Value)
					}
				}
			}
		}
	case *ast.SequenceNode:
		// For arrays, extract keys from the first item (assuming array items have similar structure)
		if len(n.Values) > 0 {
			firstItemKeys := extractKeysFromAstNode(ctx, n.Values[0])
			if len(firstItemKeys) > 0 {
				return map[string]interface{}{SequenceMarker: true, ItemKeysField: firstItemKeys}
			}
		}
		return map[string]interface{}{SequenceMarker: true}
	case ast.ScalarNode:
		// Scalar values (strings, numbers, booleans) are leaf nodes - mark them as such
		return map[string]interface{}{ScalarMarker: true}
	default:
		// For other node types, return empty map to indicate no further structure
		return map[string]interface{}{}
	}

	return result
}

// filterAstNodeByKeys recursively filters an AST node to only include keys present in the allowedKeys structure
func filterAstNodeByKeys(ctx context.Context, node ast.Node, allowedKeys map[string]interface{}) {
	if node == nil || allowedKeys == nil {
		return
	}

	// If this is marked as a scalar, don't filter it further
	if _, isScalar := allowedKeys[ScalarMarker]; isScalar {
		return
	}

	switch n := node.(type) {
	case *ast.MappingNode:
		// Filter mapping values
		filteredValues := make([]*ast.MappingValueNode, 0)

		for _, valNode := range n.Values {
			if keyNode, ok := valNode.Key.(ast.ScalarNode); ok {
				if keyValue := keyNode.GetValue(); keyValue != nil {
					if keyStr, ok := keyValue.(string); ok {
						// Check if this key is allowed
						if nestedAllowedKeys, exists := allowedKeys[keyStr]; exists {
							// Key is allowed, keep it and recursively filter its value
							if nestedMap, ok := nestedAllowedKeys.(map[string]interface{}); ok {
								filterAstNodeByKeys(ctx, valNode.Value, nestedMap)
							}
							filteredValues = append(filteredValues, valNode)
						}
						// If key is not in allowedKeys, it gets filtered out (not added to filteredValues)
					}
				}
			}
		}

		// Replace the values with filtered ones
		n.Values = filteredValues

	case *ast.SequenceNode:
		// For sequences, recursively filter each element
		// Check if we have item key information from the template
		var itemKeys map[string]interface{}
		if itemKeysInterface, exists := allowedKeys[ItemKeysField]; exists {
			if itemKeysMap, ok := itemKeysInterface.(map[string]interface{}); ok {
				itemKeys = itemKeysMap
			}
		}

		for _, valNode := range n.Values {
			if itemKeys != nil {
				filterAstNodeByKeys(ctx, valNode, itemKeys)
			} else {
				// Fallback: if no item key info, don't filter (preserve all)
				filterAstNodeByKeys(ctx, valNode, allowedKeys)
			}
		}
	case ast.ScalarNode:
		// Scalar nodes don't need filtering - they are the actual values
		return
	}
}

// NormalizeTimeStringsInYaml normalizes time duration strings in YAML to a consistent format
// e.g., "30m0s" -> "30m", "1h0m0s" -> "1h"
func NormalizeTimeStringsInYaml(yamlString string) string {
	return timeRegex.ReplaceAllStringFunc(yamlString, func(match string) string {
		// Try to parse as duration to validate it's a valid duration
		if _, err := time.ParseDuration(match); err == nil {
			// Split into components and filter out zero components
			components := componentRegex.FindAllString(match, -1)

			var filteredComponents []string
			for _, component := range components {
				// Check if this component starts with "0" followed by unit
				if !zeroRegex.MatchString(component) {
					filteredComponents = append(filteredComponents, component)
				}
			}

			if len(filteredComponents) == 0 {
				return "0s"
			}

			return strings.Join(filteredComponents, "")
		}
		// If parsing fails, return original
		return match
	})
}
