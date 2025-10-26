package tools

import (
	"encoding/json"
	"fmt"
)

// ArgumentParser parses tool call arguments from JSON.
// This provides a single source of truth for argument parsing across the codebase.
type ArgumentParser struct {
	// AllowEmpty controls whether empty argument strings are allowed.
	// If false, Parse will return an error for empty strings.
	// If true, Parse will return an empty map for empty strings.
	AllowEmpty bool
}

// NewArgumentParser creates a new ArgumentParser with default settings.
// By default, empty arguments are allowed and return an empty map.
func NewArgumentParser() *ArgumentParser {
	return &ArgumentParser{
		AllowEmpty: true,
	}
}

// NewStrictArgumentParser creates a parser that requires non-empty arguments.
// This is useful when arguments are mandatory for a tool call.
func NewStrictArgumentParser() *ArgumentParser {
	return &ArgumentParser{
		AllowEmpty: false,
	}
}

// Parse parses JSON-encoded arguments from a string into a map.
//
// Behavior:
// - If raw is empty and AllowEmpty is true: returns empty map, no error
// - If raw is empty and AllowEmpty is false: returns nil, error
// - If raw is invalid JSON: returns nil, error
// - Otherwise: returns parsed map, no error
func (p *ArgumentParser) Parse(raw string) (map[string]interface{}, error) {
	if raw == "" {
		if p.AllowEmpty {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("tool arguments cannot be empty")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return args, nil
}
