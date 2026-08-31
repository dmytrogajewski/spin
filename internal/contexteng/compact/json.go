package compact

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func filterJSON(_ string, stdout, stderr []byte) (compactedStdout, compactedStderr []byte, err error) {
	text := strings.TrimSpace(string(stdout))
	if text == "" {
		return nil, stderr, nil
	}

	var value any
	if decodeErr := json.Unmarshal([]byte(text), &value); decodeErr != nil {
		return stdout, stderr, fmt.Errorf("decode json: %w", decodeErr)
	}

	return encodeLines(jsonShape(value, "")), stderr, nil
}

func jsonShape(value any, indent string) []string {
	obj, ok := value.(map[string]any)
	if !ok {
		return []string{indent + jsonType(value)}
	}

	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	lines := make([]string, 0, len(keys))

	for _, key := range keys {
		child := obj[key]
		nested, isObj := child.(map[string]any)

		if isObj {
			lines = append(lines, indent+key+": object")
			lines = append(lines, jsonShape(nested, indent+"  ")...)

			continue
		}

		lines = append(lines, indent+key+": "+jsonType(child))
	}

	return lines
}

func jsonType(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}
