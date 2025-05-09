package provider

import (
	"context"
	"fmt"
	"sort" // Added for sort.Slice for a more robust sort

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

// kvPair is a helper struct for sorting YAML map entries.
// It holds a key-value pair of yaml.Node pointers.
type kvPair struct {
	key   *yaml.Node
	value *yaml.Node
}

// NormalizeMonitorYaml sorts keys in a YAML string alphabetically.
// It also handles potential errors during parsing and marshalling.
// This function is exported for use in other files.
func NormalizeMonitorYaml(ctx context.Context, yamlString string) (string, error) {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlString), &node)
	if err != nil {
		tflog.Error(ctx, "Failed to unmarshal YAML", map[string]interface{}{"error": err, "yaml": yamlString})
		return "", fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// The top-level element is typically a DocumentNode, whose content is the actual root map/sequence.
	if len(node.Content) == 1 {
		sortYamlNodeRecursively(node.Content[0])
	} else {
		// Handle cases where YAML might not be a single document or has unexpected structure
		tflog.Warn(ctx, "YAML content is not a single document node, attempting to sort directly if it's a map or sequence.")
		sortYamlNodeRecursively(&node) // Pass the address of the node
	}

	out, err := yaml.Marshal(&node)
	if err != nil {
		tflog.Error(ctx, "Failed to marshal YAML", map[string]interface{}{"error": err})
		return "", fmt.Errorf("failed to marshal sorted YAML: %w", err)
	}

	return string(out), nil
}

// sortYamlNodeRecursively sorts map keys within a yaml.Node.
// It traverses the YAML structure and applies sorting to all mapping nodes.
// This function is unexported as it's a helper for NormalizeMonitorYaml.
func sortYamlNodeRecursively(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			// This shouldn't happen for valid YAML maps
			tflog.Error(context.Background(), "YAML MappingNode has odd number of children, cannot sort keys.")
			return
		}
		pairs := make([]kvPair, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			pairs[i/2] = kvPair{key: node.Content[i], value: node.Content[i+1]}
		}

		customSortPairs(pairs) // Use the more robust sort

		newContent := make([]*yaml.Node, 0, len(node.Content))
		for _, p := range pairs {
			newContent = append(newContent, p.key, p.value)
		}
		node.Content = newContent

		for i := 1; i < len(node.Content); i += 2 {
			sortYamlNodeRecursively(node.Content[i])
		}

	case yaml.SequenceNode:
		for _, elem := range node.Content {
			sortYamlNodeRecursively(elem)
		}
	case yaml.DocumentNode:
		for _, elem := range node.Content {
			sortYamlNodeRecursively(elem)
		}
		// ScalarNode and AliasNode don't need sorting of their internal structure
	}
}

// customSortPairs sorts key-value pairs based on the key's string value.
// This uses sort.Slice for better performance and stability over a manual bubble sort.
// This function is unexported.
func customSortPairs(pairs []kvPair) {
	sort.Slice(pairs, func(i, j int) bool {
		// Ensure keys are scalar nodes and have string values for comparison
		if pairs[i].key.Kind == yaml.ScalarNode && pairs[j].key.Kind == yaml.ScalarNode {
			return pairs[i].key.Value < pairs[j].key.Value
		}
		// Fallback for non-scalar keys or other complex scenarios:
		// maintain original order or implement more sophisticated comparison.
		// For simplicity, if keys are not comparable scalars, they are treated as equal in terms of sort order.
		return false
	})
}
