package llm

import (
	"encoding/json"
	"testing"
)

// TestFunction_JSONMarshaling tests that Function can be marshaled/unmarshaled with json.RawMessage Parameters
func TestFunction_JSONMarshaling(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)
	fn := Function{
		Name:        "read_file",
		Description: "Reads a file",
		Parameters:  schema,
	}

	// Marshal to JSON
	data, err := json.Marshal(fn)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded Function
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify fields
	if decoded.Name != fn.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, fn.Name)
	}
	if decoded.Description != fn.Description {
		t.Errorf("Description = %q, want %q", decoded.Description, fn.Description)
	}

	// Verify Parameters JSON content matches
	var originalParams, decodedParams map[string]interface{}
	json.Unmarshal(fn.Parameters, &originalParams)
	json.Unmarshal(decoded.Parameters, &decodedParams)

	if originalParams["type"] != decodedParams["type"] {
		t.Errorf("Parameters type mismatch")
	}
}
