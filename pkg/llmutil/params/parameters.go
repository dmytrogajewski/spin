// Package params provides LLM parameter handling and validation.
package params

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

var (
	// ErrParameterNotFound is a sentinel error.
	ErrParameterNotFound = errors.New("parameter  not found")
)

// ToolParameters provides type-safe access to tool parameters.
// It wraps a map of JSON-encoded values and provides typed accessors
// with proper error handling.
//
// The internal representation uses [json.RawMessage] to preserve the original
// JSON encoding, allowing for efficient re-marshaling and complex object handling.
type ToolParameters struct {
	raw map[string]json.RawMessage
}

// Has checks if a parameter key exists.
func (p *ToolParameters) Has(key string) bool {
	_, exists := p.raw[key]

	return exists
}

// FromMap creates ToolParameters from a map[string]interface{}.
// This is the primary way to create ToolParameters from tool call arguments.
func FromMap(m map[string]any) (ToolParameters, error) {
	raw := make(map[string]json.RawMessage, len(m))
	for key, value := range m {
		jsonData, err := json.Marshal(value)
		if err != nil {
			return ToolParameters{}, fmt.Errorf("marshaling parameter %q: %w", key, err)
		}

		raw[key] = jsonData
	}

	return ToolParameters{raw: raw}, nil
}

// ToMap converts ToolParameters back to map[string]interface{}.
// This is useful for backwards compatibility or debugging.
func (p *ToolParameters) ToMap() map[string]any {
	result := make(map[string]any, len(p.raw))
	for key, rawValue := range p.raw {
		var value any

		err := json.Unmarshal(rawValue, &value)
		if err != nil {
			// If unmarshal fails, store the raw JSON string.
			result[key] = string(rawValue)
		} else {
			result[key] = value
		}
	}

	return result
}

// Keys returns all parameter keys in sorted order.
func (p *ToolParameters) Keys() []string {
	keys := make([]string, 0, len(p.raw))
	for key := range p.raw {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// GetString retrieves a string parameter.
// Returns an error if the key doesn't exist or the value is not a string.
func (p *ToolParameters) GetString(key string) (string, error) {
	rawValue, exists := p.raw[key]
	if !exists {
		return "", fmt.Errorf("parameter %q not found: %w", key, ErrParameterNotFound)
	}

	var value string

	err := json.Unmarshal(rawValue, &value)
	if err != nil {
		return "", fmt.Errorf("parameter %q is not a string: %w", key, err)
	}

	return value, nil
}

// GetInt retrieves an integer parameter.
// Returns an error if the key doesn't exist or the value is not an integer.
func (p *ToolParameters) GetInt(key string) (int, error) {
	rawValue, exists := p.raw[key]
	if !exists {
		return 0, fmt.Errorf("parameter %q not found: %w", key, ErrParameterNotFound)
	}

	var value int

	err := json.Unmarshal(rawValue, &value)
	if err != nil {
		return 0, fmt.Errorf("parameter %q is not an integer: %w", key, err)
	}

	return value, nil
}

// GetBool retrieves a boolean parameter.
// Returns an error if the key doesn't exist or the value is not a boolean.
func (p *ToolParameters) GetBool(key string) (bool, error) {
	rawValue, exists := p.raw[key]
	if !exists {
		return false, fmt.Errorf("parameter %q not found: %w", key, ErrParameterNotFound)
	}

	var value bool

	err := json.Unmarshal(rawValue, &value)
	if err != nil {
		return false, fmt.Errorf("parameter %q is not a boolean: %w", key, err)
	}

	return value, nil
}

// GetFloat64 retrieves a float64 parameter.
// Returns an error if the key doesn't exist or the value is not a number.
func (p *ToolParameters) GetFloat64(key string) (float64, error) {
	rawValue, exists := p.raw[key]
	if !exists {
		return 0, fmt.Errorf("parameter %q not found: %w", key, ErrParameterNotFound)
	}

	var value float64

	err := json.Unmarshal(rawValue, &value)
	if err != nil {
		return 0, fmt.Errorf("parameter %q is not a number: %w", key, err)
	}

	return value, nil
}

// GetObject retrieves a complex object parameter and unmarshals it into dest.
// Returns an error if the key doesn't exist or unmarshaling fails.
func (p *ToolParameters) GetObject(key string, dest any) error {
	rawValue, exists := p.raw[key]
	if !exists {
		return fmt.Errorf("parameter %q not found: %w", key, ErrParameterNotFound)
	}

	err := json.Unmarshal(rawValue, dest)
	if err != nil {
		return fmt.Errorf("parameter %q unmarshal failed: %w", key, err)
	}

	return nil
}

// GetStringOr retrieves a string parameter with a default value.
// Returns the default if the key doesn't exist or the value is not a string.
func (p *ToolParameters) GetStringOr(key, defaultValue string) string {
	value, err := p.GetString(key)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetIntOr retrieves an integer parameter with a default value.
// Returns the default if the key doesn't exist or the value is not an integer.
func (p *ToolParameters) GetIntOr(key string, defaultValue int) int {
	value, err := p.GetInt(key)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetBoolOr retrieves a boolean parameter with a default value.
// Returns the default if the key doesn't exist or the value is not a boolean.
func (p *ToolParameters) GetBoolOr(key string, defaultValue bool) bool {
	value, err := p.GetBool(key)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetFloat64Or retrieves a float64 parameter with a default value.
// Returns the default if the key doesn't exist or the value is not a number.
func (p *ToolParameters) GetFloat64Or(key string, defaultValue float64) float64 {
	value, err := p.GetFloat64(key)
	if err != nil {
		return defaultValue
	}

	return value
}

// MarshalJSON implements [json.Marshaler].
func (p *ToolParameters) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(p.raw)
	if err != nil {
		return nil, fmt.Errorf("marshal tool parameters: %w", err)
	}

	return data, nil
}

// UnmarshalJSON implements [json.Unmarshaler].
func (p *ToolParameters) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &p.raw); err != nil {
		return fmt.Errorf("unmarshal tool parameters: %w", err)
	}

	return nil
}
