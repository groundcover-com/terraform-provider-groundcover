package provider

import (
	"context"
	"testing"
)

func TestCompareJSONSemantically(t *testing.T) {
	tests := []struct {
		name   string
		json1  string
		json2  string
		expect bool
	}{
		{
			name:   "identical JSON strings",
			json1:  `{"name":"test","value":123}`,
			json2:  `{"name":"test","value":123}`,
			expect: true,
		},
		{
			name:   "different key ordering",
			json1:  `{"name":"test","value":123}`,
			json2:  `{"value":123,"name":"test"}`,
			expect: true,
		},
		{
			name:   "different whitespace",
			json1:  `{"name":"test","value":123}`,
			json2:  `{ "name" : "test" , "value" : 123 }`,
			expect: true,
		},
		{
			name:   "different values",
			json1:  `{"name":"test","value":123}`,
			json2:  `{"name":"test","value":456}`,
			expect: false,
		},
		{
			name:   "nested objects with different ordering",
			json1:  `{"outer":{"inner":"value","other":123}}`,
			json2:  `{"outer":{"other":123,"inner":"value"}}`,
			expect: true,
		},
		{
			name:   "arrays with same order",
			json1:  `{"items":[1,2,3]}`,
			json2:  `{"items":[1,2,3]}`,
			expect: true,
		},
		{
			name:   "arrays with different order",
			json1:  `{"items":[1,2,3]}`,
			json2:  `{"items":[3,2,1]}`,
			expect: false, // Array order matters
		},
		{
			name:   "empty objects",
			json1:  `{}`,
			json2:  `{}`,
			expect: true,
		},
		{
			name:   "empty arrays",
			json1:  `{"items":[]}`,
			json2:  `{"items":[]}`,
			expect: true,
		},
		{
			name:   "null values",
			json1:  `{"value":null}`,
			json2:  `{"value":null}`,
			expect: true,
		},
		{
			name:   "boolean values",
			json1:  `{"enabled":true}`,
			json2:  `{"enabled":true}`,
			expect: true,
		},
		{
			name:   "complex nested structure",
			json1:  `{"spec":{"layoutType":"ordered"},"layout":[{"h":5,"w":24,"x":0,"y":0}],"widgets":[]}`,
			json2:  `{"layout":[{"y":0,"x":0,"w":24,"h":5}],"spec":{"layoutType":"ordered"},"widgets":[]}`,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareJSONSemantically(tt.json1, tt.json2)
			if err != nil {
				t.Errorf("CompareJSONSemantically() error = %v", err)
				return
			}
			if result != tt.expect {
				t.Errorf("CompareJSONSemantically() = %v, want %v", result, tt.expect)
				t.Logf("JSON1: %s", tt.json1)
				t.Logf("JSON2: %s", tt.json2)
			}
		})
	}
}

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // We'll check that normalization is consistent
	}{
		{
			name:     "simple object",
			input:    `{"b":2,"a":1}`,
			expected: `{
  "a": 1,
  "b": 2
}`,
		},
		{
			name:     "nested object",
			input:    `{"outer":{"c":3,"a":1,"b":2}}`,
			expected: `{
  "outer": {
    "a": 1,
    "b": 2,
    "c": 3
  }
}`,
		},
		{
			name:     "array with objects",
			input:    `{"items":[{"b":2,"a":1}]}`,
			expected: `{
  "items": [
    {
      "a": 1,
      "b": 2
    }
  ]
}`,
		},
		{
			name:     "empty string",
			input:    ``,
			expected: ``,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeJSON(ctx, tt.input)
			if err != nil {
				t.Errorf("NormalizeJSON() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("NormalizeJSON() = %q, want %q", result, tt.expected)
			}
			// Verify that normalizing twice produces the same result
			result2, err := NormalizeJSON(ctx, result)
			if err != nil {
				t.Errorf("NormalizeJSON() second call error = %v", err)
				return
			}
			if result2 != result {
				t.Errorf("NormalizeJSON() is not idempotent: first=%q, second=%q", result, result2)
			}
		})
	}
}

func TestCompareJSONSemantically_RealWorldDashboardPreset(t *testing.T) {
	// Test with actual preset JSON from the trace data
	preset1 := `{"spec":{"layoutType":"ordered"},"layout":[{"h":5,"w":24,"x":0,"y":0,"id":"A","minH":2,"children":[]}],"widgets":[{"id":"A","name":"Test","type":"section"}],"duration":"Last hour","variables":[],"schemaVersion":7}`
	
	// Same preset with different key ordering and whitespace
	preset2 := `{
  "layout": [
    {
      "y": 0,
      "x": 0,
      "w": 24,
      "h": 5,
      "id": "A",
      "minH": 2,
      "children": []
    }
  ],
  "spec": {
    "layoutType": "ordered"
  },
  "widgets": [
    {
      "id": "A",
      "name": "Test",
      "type": "section"
    }
  ],
  "duration": "Last hour",
  "variables": [],
  "schemaVersion": 7
}`

	result, err := CompareJSONSemantically(preset1, preset2)
	if err != nil {
		t.Errorf("CompareJSONSemantically() error = %v", err)
		return
	}
	if !result {
		t.Errorf("CompareJSONSemantically() = false, want true - presets should be semantically identical")
	}
}

func TestNormalizeJSON_ThenCompare(t *testing.T) {
	// Test that normalization + comparison works correctly
	json1 := `{"b":2,"a":1,"c":{"z":26,"y":25}}`
	json2 := `{"a":1,"c":{"y":25,"z":26},"b":2}`

	ctx := context.Background()
	norm1, err := NormalizeJSON(ctx, json1)
	if err != nil {
		t.Fatalf("NormalizeJSON() error = %v", err)
	}

	norm2, err := NormalizeJSON(ctx, json2)
	if err != nil {
		t.Fatalf("NormalizeJSON() error = %v", err)
	}

	// After normalization, they should be identical strings
	if norm1 != norm2 {
		t.Errorf("Normalized JSONs should be identical: %q != %q", norm1, norm2)
	}

	// And semantically the same
	same, err := CompareJSONSemantically(norm1, norm2)
	if err != nil {
		t.Fatalf("CompareJSONSemantically() error = %v", err)
	}
	if !same {
		t.Errorf("Normalized JSONs should be semantically the same")
	}
}

