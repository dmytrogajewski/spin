// Package types provides common types used across the application.
package types

import (
	"encoding/json"
	"errors"
)

// ErrParameterNotFound indicates a required parameter was not found.
var ErrParameterNotFound = errors.New("parameter not found")

// ToolCallArguments represents the arguments passed to a tool call.
type ToolCallArguments map[string]json.RawMessage

// Get retrieves a typed value from the arguments.
func (t ToolCallArguments) Get(key string, dest any) error {
	raw, ok := t[key]
	if !ok {
		return ErrParameterNotFound
	}
	return json.Unmarshal(raw, dest)
}

// GetString retrieves a string value from the arguments.
func (t ToolCallArguments) GetString(key string) (string, error) {
	var val string
	err := t.Get(key, &val)
	return val, err
}

// GetInt retrieves an integer value from the arguments.
func (t ToolCallArguments) GetInt(key string) (int, error) {
	var val int
	err := t.Get(key, &val)
	return val, err
}

// GetBool retrieves a boolean value from the arguments.
func (t ToolCallArguments) GetBool(key string) (bool, error) {
	var val bool
	err := t.Get(key, &val)
	return val, err
}

// ToMap converts arguments to a map for compatibility.
func (t ToolCallArguments) ToMap() map[string]any {
	result := make(map[string]any, len(t))
	for k, v := range t {
		var val any
		if err := json.Unmarshal(v, &val); err == nil {
			result[k] = val
		}
	}
	return result
}

// FromMap creates ToolCallArguments from a map.
func FromMap(m map[string]any) (ToolCallArguments, error) {
	args := make(ToolCallArguments)
	for k, v := range m {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		args[k] = json.RawMessage(data)
	}
	return args, nil
}