package llm

import (
	"fmt"
	"reflect"
	"strings"
)

// GenerateJSONInstruction uses reflection to generate a description of the expected JSON structure from a Go struct.
func GenerateJSONInstruction(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "The response must be valid JSON."
	}

	var sb strings.Builder
	sb.WriteString("You MUST return a JSON object with the following fields and types:\n")
	describeStruct(t, &sb, 0)
	return sb.String()
}

func describeStruct(t reflect.Type, sb *strings.Builder, indent int) {
	indentStr := strings.Repeat("  ", indent)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		jsonTag := field.Tag.Get("json")
		var fieldName string
		if jsonTag == "" || jsonTag == "-" {
			fieldName = field.Name
		} else {
			// strip options like omitempty
			parts := strings.Split(jsonTag, ",")
			fieldName = parts[0]
		}

		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			_, _ = fmt.Fprintf(sb, "%s- %s (object):\n", indentStr, fieldName)
			describeStruct(fieldType, sb, indent+1)
		case reflect.Slice:
			_, _ = fmt.Fprintf(sb, "%s- %s (array of %s)\n", indentStr, fieldName, fieldType.Elem().Kind().String())
		case reflect.Map:
			_, _ = fmt.Fprintf(sb, "%s- %s (object map from %s to %s)\n", indentStr, fieldName, fieldType.Key().Kind().String(), fieldType.Elem().Kind().String())
		default:
			_, _ = fmt.Fprintf(sb, "%s- %s (%s)\n", indentStr, fieldName, fieldType.Kind().String())
		}
	}
}
