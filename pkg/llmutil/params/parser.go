package params

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrToolArgumentsCannotBeEmpty is a sentinel error.
var ErrToolArgumentsCannotBeEmpty = errors.New("tool arguments cannot be empty")

// ArgumentParser parses tool call arguments from JSON.
// This provides a single source of truth for argument parsing across the codebase.
type ArgumentParser struct {
	// AllowEmpty controls whether empty argument strings are allowed.
	// If false, Parse will return an error for empty strings.
	// If true, Parse will return an empty map for empty strings.
	AllowEmpty bool
}

// NewStrictArgumentParser creates a parser that requires non-empty arguments.
// This is useful when arguments are mandatory for a tool call.
func NewStrictArgumentParser() *ArgumentParser {
	return &ArgumentParser{
		AllowEmpty: false,
	}
}

// Parse parses JSON-encoded arguments from a string into ToolParameters.
//
// Behavior:
// - If raw is empty and AllowEmpty is true: returns empty ToolParameters, no error
// - If raw is empty and AllowEmpty is false: returns zero ToolParameters, error
// - If raw is invalid JSON: returns zero ToolParameters, error
// - Otherwise: returns parsed ToolParameters, no error.
func (p *ArgumentParser) Parse(raw string) (ToolParameters, error) {
	if raw == "" {
		if p.AllowEmpty {
			return ToolParameters{}, nil
		}

		return ToolParameters{}, ErrToolArgumentsCannotBeEmpty
	}

	var args map[string]any

	err := json.Unmarshal([]byte(raw), &args)
	if err != nil {
		return ToolParameters{}, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return FromMap(args)
}
