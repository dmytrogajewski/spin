package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

var (
	// ErrParameterNotFound is a sentinel error.
	ErrParameterNotFound = errors.New("parameter not found")
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
func (p ToolParameters) Has(key string) bool {
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
func (p ToolParameters) ToMap() map[string]any {
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
func (p ToolParameters) Keys() []string {
	keys := make([]string, 0, len(p.raw))
	for key := range p.raw {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// Get retrieves a parameter of type T by key.
// Returns an error if the key doesn't exist or the value cannot be unmarshaled to T.
func Get[T any](p ToolParameters, key string) (T, error) {
	var zero T

	rawValue, exists := p.raw[key]
	if !exists {
		return zero, fmt.Errorf("parameter %q not found: %w", key, ErrParameterNotFound)
	}

	var value T

	err := json.Unmarshal(rawValue, &value)
	if err != nil {
		return zero, fmt.Errorf("parameter %q: %w", key, err)
	}

	return value, nil
}

// GetOr retrieves a parameter of type T with a default value.
// Returns the default if the key doesn't exist or the value cannot be unmarshaled to T.
func GetOr[T any](p ToolParameters, key string, defaultValue T) T {
	value, err := Get[T](p, key)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetString retrieves a string parameter.
// Returns an error if the key doesn't exist or the value is not a string.
func (p ToolParameters) GetString(key string) (string, error) { return Get[string](p, key) }

// GetInt retrieves an integer parameter.
// Returns an error if the key doesn't exist or the value is not an integer.
func (p ToolParameters) GetInt(key string) (int, error) { return Get[int](p, key) }

// GetBool retrieves a boolean parameter.
// Returns an error if the key doesn't exist or the value is not a boolean.
func (p ToolParameters) GetBool(key string) (bool, error) { return Get[bool](p, key) }

// GetFloat64 retrieves a float64 parameter.
// Returns an error if the key doesn't exist or the value is not a number.
func (p ToolParameters) GetFloat64(key string) (float64, error) { return Get[float64](p, key) }

// GetObject retrieves a complex object parameter and unmarshals it into dest.
// Returns an error if the key doesn't exist or unmarshaling fails.
func (p ToolParameters) GetObject(key string, dest any) error {
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
func (p ToolParameters) GetStringOr(key, defaultValue string) string {
	return GetOr[string](p, key, defaultValue)
}

// GetIntOr retrieves an integer parameter with a default value.
// Returns the default if the key doesn't exist or the value is not an integer.
func (p ToolParameters) GetIntOr(key string, defaultValue int) int {
	return GetOr[int](p, key, defaultValue)
}

// GetBoolOr retrieves a boolean parameter with a default value.
// Returns the default if the key doesn't exist or the value is not a boolean.
func (p ToolParameters) GetBoolOr(key string, defaultValue bool) bool {
	return GetOr[bool](p, key, defaultValue)
}

// GetFloat64Or retrieves a float64 parameter with a default value.
// Returns the default if the key doesn't exist or the value is not a number.
func (p ToolParameters) GetFloat64Or(key string, defaultValue float64) float64 {
	return GetOr[float64](p, key, defaultValue)
}

// MarshalJSON implements [json.Marshaler].
func (p ToolParameters) MarshalJSON() ([]byte, error) {
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
