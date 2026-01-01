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
	// Match valid Go duration patterns: h, m, s in proper order (not all required)
	// Examples: "1h", "30m", "45s", "1h30m", "2h15m30s", "5m30s"
	timeRegex      = regexp.MustCompile(`\d+h(?:\d+m)?(?:\d+s)?|\d+m(?:\d+s)?|\d+s`)
	componentRegex = regexp.MustCompile(`(\d+)([hms])`)
	zeroRegex      = regexp.MustCompile(`^0[hms]$`)
)

// NormalizeMonitorYaml sorts keys in a YAML string alphabetically using AST manipulation.
// This approach preserves comments and handles complex YAML structures consistently.
func NormalizeMonitorYaml(ctx context.Context, yamlString string) (string, error) {
	if yamlString == "" {
		return "", nil
	}

	// Parse with AST approach to preserve comments and handle complex structures
	file, err := parser.ParseBytes([]byte(yamlString), parser.ParseComments)
	if err != nil {
		tflog.Error(ctx, "Failed to parse YAML in NormalizeMonitorYaml", map[string]interface{}{
			"error":       err.Error(),
			"yaml_length": len(yamlString),
		})
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Sort all documents
	for _, doc := range file.Docs {
		sortAstNodeGoccy(doc.Body)
	}

	// Generate normalized output
	outputString := file.String()
	if yamlString == "" && outputString == "\n" {
		return "", nil
	}

	// Remove extra blank lines between fields and trailing newlines
	outputString = removeExtraNewlines(outputString)

	// Convert single-line multiline pipe syntax to simple strings
	// This converts `title: |\n  value` to `title: value` for Grafana compatibility
	outputString = convertSingleLineMultilinePipeToSimpleString(outputString)

	return outputString, nil
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
		// NOTE: We do NOT sort the sequence elements themselves (n.Values slice)
		// because array order often matters semantically in configuration files.
		// We only recursively sort the internal structure of each element.
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

			// Validate that the normalized string is still a valid duration
			if _, err := time.ParseDuration(normalized); err == nil {
				return normalized
			}

			// If normalized string is invalid, return original
			return match
		}
		// If parsing fails, return original
		return match
	})

	return result
}

// DefaultValueRule defines a rule for applying default values
type DefaultValueRule struct {
	// RequiredFields that must exist to trigger this rule
	RequiredFields []string
	// DefaultField is the field to add if missing
	DefaultField string
	// DefaultValue is the value to set for the default field
	DefaultValue interface{}
}

// Fields that the server doesn't persist/return and should be ignored during comparison
var ignoredFields = map[string]bool{
	"link": true, // Server doesn't return the link field
}

// Fields that should be ignored if they are empty (nil, empty string, empty map, empty slice)
var ignoreIfEmptyFields = map[string]bool{
	"description": true,
	"annotations": true,
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

	// Normalize string values (especially expression fields) to handle multiline formatting differences
	normalizedData1 = normalizeStringValues(normalizedData1)
	normalizedData2 = normalizeStringValues(normalizedData2)

	// Remove empty fields that the server commonly adds
	normalizedData1 = removeEmptyFields(normalizedData1)
	normalizedData2 = removeEmptyFields(normalizedData2)

	// Remove fields that the server doesn't return
	normalizedData1 = removeIgnoredFields(normalizedData1)
	normalizedData2 = removeIgnoredFields(normalizedData2)

	// Apply monitor-specific default values normalization
	// This handles cases where the server omits default values (like isPaused: false)
	monitorDefaultRules := []DefaultValueRule{
		{
			RequiredFields: []string{"title", "model"},
			DefaultField:   "isPaused",
			DefaultValue:   false,
		},
	}

	normalizedData1 = applyDefaultValuesWithRules(normalizedData1, monitorDefaultRules)
	normalizedData2 = applyDefaultValuesWithRules(normalizedData2, monitorDefaultRules)

	// Deep compare the normalized data structures
	result := deepEqual(normalizedData1, normalizedData2)

	return result, nil
}

// removeIgnoredFields recursively removes fields that should be ignored during comparison
func removeIgnoredFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			// Skip fields that are always ignored
			if ignoredFields[key] {
				continue
			}
			// Skip fields that are ignored when empty
			if ignoreIfEmptyFields[key] && isEmpty(value) {
				continue
			}
			result[key] = removeIgnoredFields(value)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			keyStr, isString := key.(string)
			if isString {
				// Skip fields that are always ignored
				if ignoredFields[keyStr] {
					continue
				}
				// Skip fields that are ignored when empty
				if ignoreIfEmptyFields[keyStr] && isEmpty(value) {
					continue
				}
			}
			result[key] = removeIgnoredFields(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = removeIgnoredFields(item)
		}
		return result
	default:
		return v
	}
}

// applyDefaultValuesWithRules adds default values based on configurable rules
func applyDefaultValuesWithRules(data interface{}, rules []DefaultValueRule) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = applyDefaultValuesWithRules(value, rules)
		}

		// Apply each default value rule
		for _, rule := range rules {
			// Check if this map has all required fields
			hasAllRequiredFields := true
			for _, requiredField := range rule.RequiredFields {
				if _, exists := result[requiredField]; !exists {
					hasAllRequiredFields = false
					break
				}
			}

			// If all required fields exist and default field is missing, add it
			if hasAllRequiredFields {
				if _, hasDefaultField := result[rule.DefaultField]; !hasDefaultField {
					result[rule.DefaultField] = rule.DefaultValue
				}
			}
		}

		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			result[key] = applyDefaultValuesWithRules(value, rules)
		}

		// Apply each default value rule (convert interface{} keys to strings for comparison)
		for _, rule := range rules {
			// Check if this map has all required fields
			hasAllRequiredFields := true
			for _, requiredField := range rule.RequiredFields {
				found := false
				for key := range result {
					if keyStr, ok := key.(string); ok && keyStr == requiredField {
						found = true
						break
					}
				}
				if !found {
					hasAllRequiredFields = false
					break
				}
			}

			// If all required fields exist and default field is missing, add it
			if hasAllRequiredFields {
				defaultFieldExists := false
				for key := range result {
					if keyStr, ok := key.(string); ok && keyStr == rule.DefaultField {
						defaultFieldExists = true
						break
					}
				}
				if !defaultFieldExists {
					result[rule.DefaultField] = rule.DefaultValue
				}
			}
		}

		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = applyDefaultValuesWithRules(item, rules)
		}
		return result
	default:
		return v
	}
}

// removeEmptyFields recursively removes empty maps, nil values, and empty slices from data structures
// This helps ignore server-added fields that are empty/nil
func removeEmptyFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			cleaned := removeEmptyFields(value)
			// Only keep non-empty values
			if !isEmpty(cleaned) {
				result[key] = cleaned
			}
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			cleaned := removeEmptyFields(value)
			// Only keep non-empty values
			if !isEmpty(cleaned) {
				result[key] = cleaned
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, item := range v {
			cleaned := removeEmptyFields(item)
			if !isEmpty(cleaned) {
				result = append(result, cleaned)
			}
		}
		return result
	default:
		return v
	}
}

// isEmpty checks if a value is considered empty (nil, empty map, empty slice)
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case map[string]interface{}:
		return len(v) == 0
	case map[interface{}]interface{}:
		return len(v) == 0
	case []interface{}:
		return len(v) == 0
	case string:
		return v == ""
	default:
		return false
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

// normalizeStringValues recursively normalizes string values in a data structure
// This collapses whitespace in multiline strings, especially for expression fields.
// For expression fields, it normalizes whitespace to handle cases where the API returns
// expressions on a single line but the input has them split across lines with trailing spaces.
func normalizeStringValues(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			// Special handling for expression fields - normalize whitespace
			// This handles multiline expressions where trailing spaces and line breaks
			// should be treated as equivalent to single-line expressions
			if key == "expression" {
				if strVal, ok := value.(string); ok {
					// Collapse multiple whitespace characters (spaces, tabs, newlines) into single spaces
					// This makes "value1 \n  value2" equivalent to "value1 value2"
					normalized := strings.Join(strings.Fields(strVal), " ")
					result[key] = normalized
					continue
				}
			}
			result[key] = normalizeStringValues(value)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			keyStr, isString := key.(string)
			// Special handling for expression fields - normalize whitespace
			if isString && keyStr == "expression" {
				if strVal, ok := value.(string); ok {
					// Collapse multiple whitespace characters (spaces, tabs, newlines) into single spaces
					normalized := strings.Join(strings.Fields(strVal), " ")
					result[key] = normalized
					continue
				}
			}
			result[key] = normalizeStringValues(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = normalizeStringValues(item)
		}
		return result
	case string:
		// Don't normalize all strings, only expression fields
		// Other strings should preserve their formatting
		return v
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

			// Validate that the normalized string is still a valid duration
			if _, err := time.ParseDuration(result); err == nil {
				return result
			}

			// If normalized string is invalid, return original
			return match
		}
		// If parsing fails, return original
		return match
	})
}

// convertSingleLineMultilinePipeToSimpleString converts single-line multiline pipe syntax
// (e.g., `title: |\n  value`) to simple string format (e.g., `title: value`)
// This is needed because Grafana/monitor API doesn't accept multiline pipe syntax for single-line values
func convertSingleLineMultilinePipeToSimpleString(s string) string {
	lines := strings.Split(s, "\n")
	result := make([]string, 0, len(lines))
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Check if this line matches the pattern: "key: |" (with optional whitespace)
		pipePattern := regexp.MustCompile(`^(\s*)([a-zA-Z_][a-zA-Z0-9_]*):\s*\|\s*$`)
		if matches := pipePattern.FindStringSubmatch(line); matches != nil {
			indent := matches[1]
			key := matches[2]

			// Check if the next line exists and contains a single-line value
			if i+1 < len(lines) {
				nextLine := lines[i+1]
				nextLineTrimmed := strings.TrimSpace(nextLine)

				// Check if the next line is indented more than the key (value line)
				// The value should be indented at least one level more than the key
				if nextLineTrimmed != "" {
					// Calculate minimum indentation for the value (key indent + at least 2 spaces)
					minValueIndent := len(indent) + 2
					actualValueIndent := len(nextLine) - len(strings.TrimLeft(nextLine, " \t"))

					if actualValueIndent >= minValueIndent {
						value := strings.TrimSpace(nextLine)

						// Check if the line after the value is empty, less indented (next key at same or less level), or doesn't exist
						isSingleLine := true
						if i+2 < len(lines) {
							nextNextLine := lines[i+2]
							nextNextLineTrimmed := strings.TrimSpace(nextNextLine)

							if nextNextLineTrimmed != "" {
								// Calculate indentation of the next line
								nextNextIndent := len(nextNextLine) - len(strings.TrimLeft(nextNextLine, " \t"))

								// If the next line is also indented at the same or deeper level as the value,
								// it means this is actually a multiline value
								if nextNextIndent >= actualValueIndent {
									isSingleLine = false
								}
							}
						}

						if isSingleLine {
							// Convert to simple string format
							// Check if value needs quoting (only for special cases that would break YAML)
							// Most values don't need quotes, but we quote if they contain special YAML characters
							needsQuotes := strings.Contains(value, ": ") ||
								(strings.Contains(value, "#") && !strings.HasPrefix(value, "#")) ||
								strings.Contains(value, "|") || strings.Contains(value, "&") ||
								strings.Contains(value, "*") || strings.Contains(value, "!") ||
								strings.Contains(value, "%") || strings.Contains(value, "@") ||
								strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ")

							if needsQuotes && !strings.HasPrefix(value, `"`) && !strings.HasPrefix(value, `'`) {
								escapedValue := strings.ReplaceAll(value, `"`, `\"`)
								result = append(result, fmt.Sprintf("%s%s: \"%s\"", indent, key, escapedValue))
							} else {
								result = append(result, fmt.Sprintf("%s%s: %s", indent, key, value))
							}
							i += 2 // Skip both the pipe line and the value line
							continue
						}
					}
				}
			}
		}

		// Keep the line as-is
		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// removeExtraNewlines removes trailing newlines and reduces multiple consecutive blank lines to zero
// This ensures consistent YAML formatting without extra spacing between fields
func removeExtraNewlines(s string) string {
	// Split into lines
	lines := strings.Split(s, "\n")

	// Remove trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// Filter out blank lines between fields (but preserve indented content)
	filteredLines := make([]string, 0, len(lines))
	for _, line := range lines {
		// Skip completely blank lines (lines with only whitespace)
		if strings.TrimSpace(line) == "" {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	// Join back with newlines and add a single trailing newline
	if len(filteredLines) == 0 {
		return ""
	}

	return strings.Join(filteredLines, "\n") + "\n"
}

// isNumeric checks if a value is a numeric type
func isNumeric(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

// toFloat64 converts a numeric value to float64 for comparison
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// deepEqual performs deep comparison of data structures
// It handles numeric type differences (int vs float64) by comparing values as float64
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
		// Handle numeric type comparison (int vs float64)
		// YAML parsers may return int for "0" but float64 for "0.0" or scientific notation
		// These should be considered equal if they represent the same value
		if isNumeric(a) && isNumeric(b) {
			aFloat, aOk := toFloat64(a)
			bFloat, bOk := toFloat64(b)
			if aOk && bOk {
				return aFloat == bFloat
			}
		}
		return av == b
	}
}
