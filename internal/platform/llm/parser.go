package llm

import (
	"encoding/json"
	"strings"
)

// StripMarkdown removes markdown code block formatting (e.g. ```json ... ```) from the LLM output.
func StripMarkdown(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "```json") {
		input = strings.TrimPrefix(input, "```json")
		input = strings.TrimSuffix(input, "```")
	} else if strings.HasPrefix(input, "```") {
		input = strings.TrimPrefix(input, "```")
		input = strings.TrimSuffix(input, "```")
	}
	return strings.TrimSpace(input)
}

// UnmarshalAndValidate strips markdown formatting, unmarshals the raw JSON into the target generic type, and returns it.
func UnmarshalAndValidate[T any](input string) (T, error) {
	var result T
	cleaned := StripMarkdown(input)
	err := json.Unmarshal([]byte(cleaned), &result)
	return result, err
}
