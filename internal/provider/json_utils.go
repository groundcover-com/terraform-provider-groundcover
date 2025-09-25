package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// NormalizeJSON normalizes a JSON string by parsing and re-marshaling it with sorted keys
func NormalizeJSON(ctx context.Context, jsonString string) (string, error) {
	if jsonString == "" {
		return "", nil
	}

	var data interface{}
	if err := json.Unmarshal([]byte(jsonString), &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Sort keys recursively
	sortedData := sortJSONKeys(data)

	// Marshal back to JSON with consistent formatting
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(sortedData); err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	result := buf.String()
	// Remove trailing newline added by encoder
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// sortJSONKeys recursively sorts all map keys in the JSON structure
func sortJSONKeys(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Create a new map with sorted keys
		result := make(map[string]interface{})
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			result[k] = sortJSONKeys(v[k])
		}
		return result
	case []interface{}:
		// Process array elements but don't sort them (order matters)
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = sortJSONKeys(item)
		}
		return result
	default:
		// Return primitive values as-is
		return v
	}
}

// CompareJSONSemantically compares two JSON strings semantically
func CompareJSONSemantically(json1, json2 string) (bool, error) {
	if json1 == json2 {
		return true, nil
	}

	var data1, data2 interface{}
	if err := json.Unmarshal([]byte(json1), &data1); err != nil {
		return false, fmt.Errorf("failed to parse first JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(json2), &data2); err != nil {
		return false, fmt.Errorf("failed to parse second JSON: %w", err)
	}

	return reflect.DeepEqual(data1, data2), nil
}

// FilterJSONKeysBasedOnTemplate filters sourceJSON to only include keys that exist in templateJSON
func FilterJSONKeysBasedOnTemplate(ctx context.Context, sourceJSON, templateJSON string) (string, error) {
	if sourceJSON == "" {
		return "", nil
	}
	if templateJSON == "" {
		return sourceJSON, nil
	}

	var sourceData, templateData interface{}
	if err := json.Unmarshal([]byte(sourceJSON), &sourceData); err != nil {
		return "", fmt.Errorf("failed to parse source JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(templateJSON), &templateData); err != nil {
		return "", fmt.Errorf("failed to parse template JSON: %w", err)
	}

	filteredData := filterJSONData(sourceData, templateData)

	// Marshal back to JSON
	result, err := json.Marshal(filteredData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal filtered JSON: %w", err)
	}

	return string(result), nil
}

// filterJSONData recursively filters sourceData to only include keys present in templateData
func filterJSONData(sourceData, templateData interface{}) interface{} {
	switch templateVal := templateData.(type) {
	case map[string]interface{}:
		sourceMap, ok := sourceData.(map[string]interface{})
		if !ok {
			return nil
		}
		result := make(map[string]interface{})
		for key, tmplValue := range templateVal {
			if sourceValue, exists := sourceMap[key]; exists {
				result[key] = filterJSONData(sourceValue, tmplValue)
			}
		}
		return result
	case []interface{}:
		sourceArray, ok := sourceData.([]interface{})
		if !ok {
			return templateData
		}
		// For arrays, keep the same structure but filter nested objects
		result := make([]interface{}, 0)
		for i, item := range sourceArray {
			if i < len(templateVal) {
				filtered := filterJSONData(item, templateVal[i])
				result = append(result, filtered)
			} else if len(templateVal) > 0 {
				// Use first template item as template for remaining source items
				filtered := filterJSONData(item, templateVal[0])
				result = append(result, filtered)
			} else {
				result = append(result, item)
			}
		}
		return result
	default:
		// For primitive values, return the source value
		return sourceData
	}
}
