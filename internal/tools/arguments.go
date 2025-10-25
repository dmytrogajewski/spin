package tools

import (
	"encoding/json"
	"errors"
)

// ErrParameterNotFound indicates a required parameter was not found.
var ErrParameterNotFound = errors.New("parameter not found")

// Arguments represents the arguments passed to a tool call.
type Arguments map[string]json.RawMessage

// Get retrieves a typed value from the arguments.
func (a Arguments) Get(key string, dest any) error {
	raw, ok := a[key]
	if !ok {
		return ErrParameterNotFound
	}
	return json.Unmarshal(raw, dest)
}

// GetString retrieves a string value from the arguments.
func (a Arguments) GetString(key string) (string, error) {
	var val string
	err := a.Get(key, &val)
	return val, err
}

// GetInt retrieves an integer value from the arguments.
func (a Arguments) GetInt(key string) (int, error) {
	var val int
	err := a.Get(key, &val)
	return val, err
}

// GetBool retrieves a boolean value from the arguments.
func (a Arguments) GetBool(key string) (bool, error) {
	var val bool
	err := a.Get(key, &val)
	return val, err
}

// ToMap converts arguments to a map for compatibility.
func (a Arguments) ToMap() map[string]any {
	result := make(map[string]any, len(a))
	for k, v := range a {
		var val any
		if err := json.Unmarshal(v, &val); err == nil {
			result[k] = val
		}
	}
	return result
}

// FromMap creates Arguments from a map.
func FromMap(m map[string]any) (Arguments, error) {
	args := make(Arguments)
	for k, v := range m {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		args[k] = json.RawMessage(data)
	}
	return args, nil
}
