package params

import (
	"encoding/json"
	"testing"
)

func TestFromMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{
			name:    "empty map",
			input:   map[string]any{},
			wantErr: false,
		},
		{
			name: "simple types",
			input: map[string]any{
				"string": "hello",
				"int":    42,
				"bool":   true,
				"float":  3.14,
			},
			wantErr: false,
		},
		{
			name: "complex objects",
			input: map[string]any{
				"object": map[string]any{
					"nested": "value",
				},
				"array": []string{"a", "b", "c"},
			},
			wantErr: false,
		},
		{
			name: "nil values",
			input: map[string]any{
				"null": nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params, err := FromMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromMap() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && len(params.raw) != len(tt.input) {
				t.Errorf("expected %d parameters, got %d", len(tt.input), len(params.raw))
			}
		})
	}
}

func TestHas(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{
		"exists": "value",
	})

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "existing key",
			key:  "exists",
			want: true,
		},
		{
			name: "non-existing key",
			key:  "missing",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := params.Has(tt.key); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToMap(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"string": "hello",
		"int":    float64(42), // JSON unmarshals numbers as float64.
		"bool":   true,
	}

	params, err := FromMap(input)
	if err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	output := params.ToMap()
	if len(output) != len(input) {
		t.Errorf("expected %d items, got %d", len(input), len(output))
	}

	for key, expectedValue := range input {
		gotValue, exists := output[key]
		if !exists {
			t.Errorf("missing key %q in output", key)

			continue
		}

		if gotValue != expectedValue {
			t.Errorf("key %q: got %v, want %v", key, gotValue, expectedValue)
		}
	}
}

func TestKeys(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{
		"c": 1,
		"a": 2,
		"b": 3,
	})

	keys := params.Keys()
	expected := []string{"a", "b", "c"}

	if len(keys) != len(expected) {
		t.Errorf("expected %d keys, got %d", len(expected), len(keys))

		return
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] = %q, want %q", i, key, expected[i])
		}
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{
		"valid":   "hello",
		"number":  42,
		"boolean": true,
	})

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid string",
			key:     "valid",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "missing key",
			key:     "missing",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type - number",
			key:     "number",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type - boolean",
			key:     "boolean",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := params.GetString(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetString() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{
		"valid":   42,
		"string":  "hello",
		"boolean": true,
		"float":   3.14,
	})

	tests := []struct {
		name    string
		key     string
		want    int
		wantErr bool
	}{
		{
			name:    "valid int",
			key:     "valid",
			want:    42,
			wantErr: false,
		},
		{
			name:    "missing key",
			key:     "missing",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type - string",
			key:     "string",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type - boolean",
			key:     "boolean",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type - float",
			key:     "float",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := params.GetInt(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInt() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{
		"valid":  true,
		"string": "hello",
		"number": 42,
	})

	tests := []struct {
		name    string
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:    "valid bool",
			key:     "valid",
			want:    true,
			wantErr: false,
		},
		{
			name:    "missing key",
			key:     "missing",
			want:    false,
			wantErr: true,
		},
		{
			name:    "wrong type - string",
			key:     "string",
			want:    false,
			wantErr: true,
		},
		{
			name:    "wrong type - number",
			key:     "number",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := params.GetBool(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBool() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFloat64(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{
		"valid":   3.14,
		"int":     42,
		"string":  "hello",
		"boolean": true,
	})

	tests := []struct {
		name    string
		key     string
		want    float64
		wantErr bool
	}{
		{
			name:    "valid float",
			key:     "valid",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "int as float",
			key:     "int",
			want:    42.0,
			wantErr: false,
		},
		{
			name:    "missing key",
			key:     "missing",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type - string",
			key:     "string",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type - boolean",
			key:     "boolean",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := params.GetFloat64(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFloat64() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("GetFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetObject(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	params, _ := FromMap(map[string]any{
		"object": map[string]any{
			"field1": "value",
			"field2": 42,
		},
		"string": "not an object",
	})

	tests := []struct {
		name    string
		key     string
		dest    any
		want    any
		wantErr bool
	}{
		{
			name:    "valid object",
			key:     "object",
			dest:    &testStruct{},
			want:    &testStruct{Field1: "value", Field2: 42},
			wantErr: false,
		},
		{
			name:    "missing key",
			key:     "missing",
			dest:    &testStruct{},
			want:    &testStruct{},
			wantErr: true,
		},
		{
			name:    "wrong type",
			key:     "string",
			dest:    &testStruct{},
			want:    &testStruct{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := params.GetObject(tt.key, tt.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetObject() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				destStruct, _ := tt.dest.(*testStruct)

				wantStruct, _ := tt.want.(*testStruct)
				if destStruct.Field1 != wantStruct.Field1 || destStruct.Field2 != wantStruct.Field2 {
					t.Errorf("GetObject() = %+v, want %+v", destStruct, wantStruct)
				}
			}
		})
	}
}

// getOrCase is a generic test case for GetXxxOr methods.
type getOrCase[T comparable] struct {
	name         string
	key          string
	defaultValue T
	want         T
}

func runGetOrTests[T comparable](
	t *testing.T, params ToolParameters, cases []getOrCase[T],
	opName string, op func(*ToolParameters, string, T) T,
) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := op(&params, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("%s() = %v, want %v", opName, got, tt.want)
			}
		})
	}
}

func TestGetStringOr(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{"valid": "hello", "number": 42})
	runGetOrTests(t, params, []getOrCase[string]{
		{"valid string", "valid", "default", "hello"},
		{"missing key", "missing", "default", "default"},
		{"wrong type", "number", "default", "default"},
	}, "GetStringOr", (*ToolParameters).GetStringOr)
}

func TestGetIntOr(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{"valid": 42, "string": "hello"})
	runGetOrTests(t, params, []getOrCase[int]{
		{"valid int", "valid", 10, 42},
		{"missing key", "missing", 10, 10},
		{"wrong type", "string", 10, 10},
	}, "GetIntOr", (*ToolParameters).GetIntOr)
}

func TestGetBoolOr(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{"valid": true, "string": "hello"})
	runGetOrTests(t, params, []getOrCase[bool]{
		{"valid bool", "valid", false, true},
		{"missing key", "missing", false, false},
		{"wrong type", "string", false, false},
	}, "GetBoolOr", (*ToolParameters).GetBoolOr)
}

func TestGetFloat64Or(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{"valid": 3.14, "string": "hello"})
	runGetOrTests(t, params, []getOrCase[float64]{
		{"valid float", "valid", 1.0, 3.14},
		{"missing key", "missing", 1.0, 1.0},
		{"wrong type", "string", 1.0, 1.0},
	}, "GetFloat64Or", (*ToolParameters).GetFloat64Or)
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	params, _ := FromMap(map[string]any{
		"string": "hello",
		"number": 42,
		"bool":   true,
	})

	data, err := json.Marshal(&params)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Unmarshal to verify structure.
	var result map[string]any

	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result["string"] != "hello" {
		t.Errorf("string = %v, want hello", result["string"])
	}

	if result["number"] != float64(42) {
		t.Errorf("number = %v, want 42", result["number"])
	}

	boolVal, _ := result["bool"].(bool)
	if !boolVal {
		t.Errorf("bool = %v, want true", result["bool"])
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	jsonData := []byte(`{"string":"hello","number":42,"bool":true}`)

	var params ToolParameters

	err := json.Unmarshal(jsonData, &params)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	str, err := params.GetString("string")
	if err != nil || str != "hello" {
		t.Errorf("GetString(string) = %v, %v; want hello, nil", str, err)
	}

	num, err := params.GetInt("number")
	if err != nil || num != 42 {
		t.Errorf("GetInt(number) = %v, %v; want 42, nil", num, err)
	}

	b, err := params.GetBool("bool")
	if err != nil || !b {
		t.Errorf("GetBool(bool) = %v, %v; want true, nil", b, err)
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	original := map[string]any{
		"string": "hello",
		"number": 42,
		"bool":   true,
		"object": map[string]any{
			"nested": "value",
		},
	}

	// Convert to ToolParameters.
	params, err := FromMap(original)
	if err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	// Convert back to map.
	result := params.ToMap()

	// Verify all keys exist.
	for key := range original {
		if _, exists := result[key]; !exists {
			t.Errorf("missing key %q in result", key)
		}
	}
}
