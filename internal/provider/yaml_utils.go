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
	"gopkg.in/yaml.v3"
)

// Constants for AST node markers
const (
	ScalarMarker   = "__is_scalar"
	SequenceMarker = "__is_sequence"
	ItemKeysField  = "__item_keys"
)

// Pre-compiled regexes for time normalization
var (
	timeRegex      = regexp.MustCompile(`\b\d+(?:[hms]\d*)*[hms]\b`)
	componentRegex = regexp.MustCompile(`(\d+)([hms])`)
	zeroRegex      = regexp.MustCompile(`0[hms]`)
)

// NormalizeMonitorYaml sorts keys in a YAML string alphabetically using standard library.
// This approach is more reliable for nested field ordering than AST manipulation.
func NormalizeMonitorYaml(ctx context.Context, yamlString string) (string, error) {
	if yamlString == "" {
		return "", nil
	}

	// Try the standard library approach first as it's more reliable
	normalized, err := NormalizeYamlWithStandardLibrary(yamlString)
	if err != nil {
		tflog.Warn(ctx, "Standard library YAML normalization failed, falling back to AST approach", map[string]interface{}{
			"error":       err.Error(),
			"yaml_length": len(yamlString),
			"error_type":  fmt.Sprintf("%T", err),
		})

		// Fallback to the original AST approach
		file, err := parser.ParseBytes([]byte(yamlString), parser.ParseComments)
		if err != nil {
			tflog.Error(ctx, "Failed to parse YAML with goccy/go-yaml", map[string]interface{}{"error": err, "yaml": yamlString})
			return "", fmt.Errorf("failed to parse YAML with goccy/go-yaml: %w", err)
		}

		for _, doc := range file.Docs {
			sortAstNodeGoccy(doc.Body)
		}

		outputString := file.String()
		if yamlString == "" && outputString == "\n" {
			return "", nil
		}

		return outputString, nil
	}

	return normalized, nil
}

// sortAstNodeGoccy recursively sorts nodes in the AST provided by goccy/go-yaml.
func sortAstNodeGoccy(node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.MappingNode:
		// Sort the key-value pairs alphabetically by key
		sort.Slice(n.Values, func(i, j int) bool {
			keyI := getStringKeyFromNode(n.Values[i].Key)
			keyJ := getStringKeyFromNode(n.Values[j].Key)
			return keyI < keyJ
		})
		// Recursively sort all values
		for _, valNode := range n.Values {
			if valNode.Value != nil {
				sortAstNodeGoccy(valNode.Value)
			}
		}
	case *ast.SequenceNode:
		// Recursively sort each element in the sequence
		for _, valNode := range n.Values {
			sortAstNodeGoccy(valNode)
		}
	case *ast.DocumentNode:
		if n.Body != nil {
			sortAstNodeGoccy(n.Body)
		}
	case *ast.AnchorNode:
		if n.Value != nil {
			sortAstNodeGoccy(n.Value)
		}
	case *ast.AliasNode:
		// Alias nodes don't have children to sort
		return
	}
}

// getStringKeyFromNode extracts string value from a key node, with fallback handling
func getStringKeyFromNode(keyNode ast.Node) string {
	if keyNode == nil {
		return ""
	}

	// Handle the most common case: ScalarNode
	if scalarNode, ok := keyNode.(ast.ScalarNode); ok {
		if value := scalarNode.GetValue(); value != nil {
			if strVal, ok := value.(string); ok {
				return strVal
			}
		}
	}

	// Fallback: try to get the string representation
	str := keyNode.String()
	// Remove quotes if present (common in YAML key representation)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		return str[1 : len(str)-1]
	}
	if len(str) >= 2 && str[0] == '\'' && str[len(str)-1] == '\'' {
		return str[1 : len(str)-1]
	}

	return str
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
	result := timeRegex.ReplaceAllStringFunc(yamlString, func(match string) string {
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

			normalized := strings.Join(filteredComponents, "")

			return normalized
		}
		// If parsing fails, return original
		return match
	})

	return result
}

// NormalizeYamlWithStandardLibrary normalizes YAML using standard library for consistent ordering
func NormalizeYamlWithStandardLibrary(yamlString string) (string, error) {
	if yamlString == "" {
		return "", nil
	}

	var data interface{}
	if err := yaml.Unmarshal([]byte(yamlString), &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Recursively sort all maps
	sortedData := sortYamlData(data)

	// Create a buffer for consistent formatting
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)

	// Set consistent formatting options for proper array indentation
	encoder.SetIndent(2) // Use 2-space indentation consistently

	if err := encoder.Encode(sortedData); err != nil {
		_ = encoder.Close()
		return "", fmt.Errorf("failed to encode YAML: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("failed to close YAML encoder: %w", err)
	}

	result := strings.TrimSpace(buf.String())
	return result, nil
}

// sortYamlData recursively sorts maps and slices in YAML data
func sortYamlData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Extract and sort map keys to ensure deterministic processing
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Build new map with recursively sorted values
		sortedMap := make(map[string]interface{})
		for _, key := range keys {
			sortedMap[key] = sortYamlData(v[key])
		}
		return sortedMap
	case map[interface{}]interface{}:
		// Convert interface{} keys to strings, collect them, and sort
		keyMap := make(map[string]interface{})
		var keys []string

		// First pass: convert keys to strings and collect them
		for key, value := range v {
			var strKey string
			if sk, ok := key.(string); ok {
				strKey = sk
			} else {
				strKey = fmt.Sprintf("%v", key)
			}
			keyMap[strKey] = value
			keys = append(keys, strKey)
		}

		// Sort the keys
		sort.Strings(keys)

		// Build new map with recursively sorted values in key order
		sortedMap := make(map[string]interface{})
		for _, key := range keys {
			sortedMap[key] = sortYamlData(keyMap[key])
		}
		return sortedMap
	case []interface{}:
		// Recursively sort each element in the slice
		sortedSlice := make([]interface{}, len(v))
		for i, item := range v {
			sortedSlice[i] = sortYamlData(item)
		}
		return sortedSlice
	default:
		// Return primitive values as-is
		return v
	}
}

// CompareYamlSemantically compares two YAML strings semantically, ignoring formatting differences
func CompareYamlSemantically(yaml1, yaml2 string) (bool, error) {
	if yaml1 == yaml2 {
		return true, nil
	}

	// Parse both YAMLs into data structures
	var data1, data2 interface{}

	if err := yaml.Unmarshal([]byte(yaml1), &data1); err != nil {
		return false, fmt.Errorf("failed to unmarshal first YAML: %w", err)
	}

	if err := yaml.Unmarshal([]byte(yaml2), &data2); err != nil {
		return false, fmt.Errorf("failed to unmarshal second YAML: %w", err)
	}

	// Normalize time strings in both data structures
	normalizedData1 := normalizeTimeInData(data1)
	normalizedData2 := normalizeTimeInData(data2)

	// Apply default values normalization
	// This handles cases where the server omits default values (like isPaused: false)
	normalizedData1 = applyDefaultValues(normalizedData1)
	normalizedData2 = applyDefaultValues(normalizedData2)

	// Deep compare the normalized data structures
	result := deepEqual(normalizedData1, normalizedData2)

	return result, nil
}

// applyDefaultValues adds default values that the server might omit
func applyDefaultValues(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = applyDefaultValues(value)
		}

		// Add isPaused: false if it's missing at the root level
		if _, hasPaused := result["isPaused"]; !hasPaused {
			// Check if this looks like a monitor root (has required fields)
			if _, hasTitle := result["title"]; hasTitle {
				if _, hasModel := result["model"]; hasModel {
					result["isPaused"] = false
				}
			}
		}

		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			result[key] = applyDefaultValues(value)
		}

		// Add isPaused: false if it's missing at the root level
		if _, hasPaused := result["isPaused"]; !hasPaused {
			// Check if this looks like a monitor root (has required fields)
			if _, hasTitle := result["title"]; hasTitle {
				if _, hasModel := result["model"]; hasModel {
					result["isPaused"] = false
				}
			}
		}

		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = applyDefaultValues(item)
		}
		return result
	default:
		return v
	}
}

// normalizeTimeInData recursively normalizes time strings in a data structure
func normalizeTimeInData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = normalizeTimeInData(value)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			result[key] = normalizeTimeInData(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = normalizeTimeInData(item)
		}
		return result
	case string:
		// Normalize time strings
		return normalizeTimeString(v)
	default:
		return v
	}
}

// normalizeTimeString normalizes a single time string
func normalizeTimeString(s string) string {
	return timeRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Try to parse as duration to validate it's a valid duration
		if duration, err := time.ParseDuration(match); err == nil {
			// Convert back to string and parse components
			durationStr := duration.String()

			// Split into components and filter out zero components
			components := componentRegex.FindAllString(durationStr, -1)

			var filteredComponents []string
			for _, component := range components {
				// Check if this component represents zero (e.g., "0h", "0m", "0s")
				if !zeroRegex.MatchString(component) {
					filteredComponents = append(filteredComponents, component)
				}
			}

			if len(filteredComponents) == 0 {
				return "0s"
			}

			result := strings.Join(filteredComponents, "")
			return result
		}
		// If parsing fails, return original
		return match
	})
}

// deepEqual performs deep comparison of data structures
func deepEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for key, aval := range av {
			bval, exists := bv[key]
			if !exists || !deepEqual(aval, bval) {
				return false
			}
		}
		return true
	case map[interface{}]interface{}:
		bv, ok := b.(map[interface{}]interface{})
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for key, aval := range av {
			bval, exists := bv[key]
			if !exists || !deepEqual(aval, bval) {
				return false
			}
		}
		return true
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i, aval := range av {
			if !deepEqual(aval, bv[i]) {
				return false
			}
		}
		return true
	default:
		return av == b
	}
}
