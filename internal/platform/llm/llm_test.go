package llm_test

import (
	"testing"

	"queenx/internal/platform/llm"

	"github.com/stretchr/testify/assert"
)

type TestStats struct {
	Glamour int `json:"glamour"`
	Comedy  int `json:"comedy"`
}

type TestPersona struct {
	DragName string    `json:"dragName"`
	Bio      string    `json:"bio"`
	Stats    TestStats `json:"stats"`
	Tags     []string  `json:"tags"`
}

func TestGenerateJSONInstruction(t *testing.T) {
	expected := "You MUST return a JSON object with the following fields and types:\n" +
		"- dragName (string)\n" +
		"- bio (string)\n" +
		"- stats (object):\n" +
		"  - glamour (int)\n" +
		"  - comedy (int)\n" +
		"- tags (array of string)\n"

	result := llm.GenerateJSONInstruction(TestPersona{})
	assert.Equal(t, expected, result)
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wrapped in json block",
			input:    "```json\n{\"dragName\": \"Gigi\"}\n```",
			expected: "{\"dragName\": \"Gigi\"}",
		},
		{
			name:     "wrapped in plain code block",
			input:    "```\n{\"dragName\": \"Gigi\"}\n```",
			expected: "{\"dragName\": \"Gigi\"}",
		},
		{
			name:     "no wrapper",
			input:    "{\"dragName\": \"Gigi\"}",
			expected: "{\"dragName\": \"Gigi\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, llm.StripMarkdown(tt.input))
		})
	}
}

func TestUnmarshalAndValidate(t *testing.T) {
	input := "```json\n{\"dragName\": \"Gigi\", \"bio\": \"Iconic\", \"stats\": {\"glamour\": 10, \"comedy\": 8}, \"tags\": [\"fashion\"]}\n```"
	res, err := llm.UnmarshalAndValidate[TestPersona](input)
	assert.NoError(t, err)
	assert.Equal(t, "Gigi", res.DragName)
	assert.Equal(t, "Iconic", res.Bio)
	assert.Equal(t, 10, res.Stats.Glamour)
	assert.Equal(t, []string{"fashion"}, res.Tags)
}
