package neo4j

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapStringProp(t *testing.T) {
	tests := []struct {
		name     string
		props    map[string]any
		key      string
		expected string
	}{
		{
			name:     "key not present",
			props:    map[string]any{},
			key:      "birthPlace",
			expected: "",
		},
		{
			name:     "key present and is string",
			props:    map[string]any{"birthPlace": "Chicago"},
			key:      "birthPlace",
			expected: "Chicago",
		},
		{
			name:     "key present but not a string",
			props:    map[string]any{"birthPlace": 12345},
			key:      "birthPlace",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := mapStringProp(tt.props, tt.key)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestMapStringSliceProp(t *testing.T) {
	tests := []struct {
		name     string
		props    map[string]any
		key      string
		expected []string
	}{
		{
			name:     "key not present",
			props:    map[string]any{},
			key:      "classifications",
			expected: nil,
		},
		{
			name:     "key is []string",
			props:    map[string]any{"classifications": []string{"fashion", "comedy"}},
			key:      "classifications",
			expected: []string{"fashion", "comedy"},
		},
		{
			name:     "key is []any of strings",
			props:    map[string]any{"classifications": []any{"fashion", "comedy"}},
			key:      "classifications",
			expected: []string{"fashion", "comedy"},
		},
		{
			name:     "key is []any with mixed types",
			props:    map[string]any{"classifications": []any{"fashion", 123, "comedy"}},
			key:      "classifications",
			expected: []string{"fashion", "comedy"},
		},
		{
			name:     "key is not a slice",
			props:    map[string]any{"classifications": "not-a-slice"},
			key:      "classifications",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := mapStringSliceProp(tt.props, tt.key)
			assert.Equal(t, tt.expected, res)
		})
	}
}
