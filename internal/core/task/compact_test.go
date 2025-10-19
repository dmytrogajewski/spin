package task

import (
	"testing"
)

func TestNewCompact(t *testing.T) {
	compact := NewCompact()

	if compact == nil {
		t.Fatal("NewCompact() returned nil")
	}

	if compact.config != nil {
		t.Errorf("NewCompact() config = %v, want nil", compact.config)
	}
}

func TestCompact_Name(t *testing.T) {
	compact := NewCompact()

	name := compact.Name()

	if name != "compact" {
		t.Errorf("Compact.Name() = %s, want 'compact'", name)
	}
}

func TestCompact_SystemPrompt(t *testing.T) {
	tests := []struct {
		name   string
		config *CompactConfig
		want   string
	}{
		{
			name:   "default prompt",
			config: nil,
			want:   defaultCompactPrompt,
		},
		{
			name: "custom prompt",
			config: &CompactConfig{
				CustomSystemPrompt: "Custom compact prompt",
			},
			want: "Custom compact prompt",
		},
		{
			name: "empty custom prompt",
			config: &CompactConfig{
				CustomSystemPrompt: "",
			},
			want: defaultCompactPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compact := &Compact{config: tt.config}

			prompt := compact.SystemPrompt()

			if prompt != tt.want {
				t.Errorf("Compact.SystemPrompt() = %s, want %s", prompt, tt.want)
			}
		})
	}
}

func TestCompact_AllowedTools(t *testing.T) {
	tests := []struct {
		name   string
		config *CompactConfig
		want   []string
	}{
		{
			name:   "default tools",
			config: nil,
			want:   []string{"read_file", "get_context", "file_search"},
		},
		{
			name: "with additional tools",
			config: &CompactConfig{
				AdditionalTools: []string{"extra_tool1", "extra_tool2"},
			},
			want: []string{"read_file", "get_context", "file_search", "extra_tool1", "extra_tool2"},
		},
		{
			name: "empty additional tools",
			config: &CompactConfig{
				AdditionalTools: []string{},
			},
			want: []string{"read_file", "get_context", "file_search"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compact := &Compact{config: tt.config}

			tools := compact.AllowedTools()

			if len(tools) != len(tt.want) {
				t.Errorf("Compact.AllowedTools() length = %d, want %d", len(tools), len(tt.want))
			}

			for i, tool := range tools {
				if tool != tt.want[i] {
					t.Errorf("Compact.AllowedTools() [%d] = %s, want %s", i, tool, tt.want[i])
				}
			}
		})
	}
}

func TestCompact_MaxTokens(t *testing.T) {
	tests := []struct {
		name   string
		config *CompactConfig
		want   int
	}{
		{
			name:   "default max tokens",
			config: nil,
			want:   DefaultCompactMaxTokens,
		},
		{
			name: "custom max tokens",
			config: &CompactConfig{
				MaxTokens: 8192,
			},
			want: 8192,
		},
		{
			name: "zero max tokens",
			config: &CompactConfig{
				MaxTokens: 0,
			},
			want: DefaultCompactMaxTokens,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compact := &Compact{config: tt.config}

			maxTokens := compact.MaxTokens()

			if maxTokens != tt.want {
				t.Errorf("Compact.MaxTokens() = %d, want %d", maxTokens, tt.want)
			}
		})
	}
}

func TestCompact_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CompactConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &CompactConfig{
				MaxTokens:          8192,
				AdditionalTools:    []string{"extra_tool"},
				CustomSystemPrompt: "Custom prompt with sufficient length that meets minimum requirements",
			},
			wantErr: false,
		},
		{
			name: "negative max tokens",
			config: &CompactConfig{
				MaxTokens: -1,
			},
			wantErr: true,
		},
		{
			name: "max tokens too high",
			config: &CompactConfig{
				MaxTokens: MaxAllowedTokens + 1,
			},
			wantErr: true,
		},
		{
			name: "empty additional tool",
			config: &CompactConfig{
				AdditionalTools: []string{"valid_tool", ""},
			},
			wantErr: true,
		},
		{
			name: "custom prompt too short",
			config: &CompactConfig{
				CustomSystemPrompt: "short",
			},
			wantErr: true,
		},
		{
			name: "multiple validation errors",
			config: &CompactConfig{
				MaxTokens:          -1,
				AdditionalTools:    []string{""},
				CustomSystemPrompt: "short",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compact := &Compact{config: tt.config}

			err := compact.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Compact.Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Compact.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCompact_Validate_ErrorMessages(t *testing.T) {
	// Test specific error messages
	compact := &Compact{
		config: &CompactConfig{
			MaxTokens:          -1,
			AdditionalTools:    []string{"valid", ""},
			CustomSystemPrompt: "short",
		},
	}

	err := compact.Validate()

	if err == nil {
		t.Fatal("Compact.Validate() expected error, got nil")
	}

	// Check that multiple errors are joined
	errStr := err.Error()
	if !containsCompact(errStr, "max tokens cannot be negative: -1") {
		t.Errorf("Compact.Validate() error should contain max tokens error")
	}

	if !containsCompact(errStr, "additional tool at index 1 cannot be empty") {
		t.Errorf("Compact.Validate() error should contain empty tool error")
	}

	if !containsCompact(errStr, "custom system prompt too short") {
		t.Errorf("Compact.Validate() error should contain prompt length error")
	}
}

func TestCompact_DefaultPrompt(t *testing.T) {
	compact := NewCompact()
	prompt := compact.SystemPrompt()

	// Check that default prompt contains key elements
	expectedElements := []string{
		"Spin in Compact Mode",
		"Minimal context",
		"fast responses",
		"essential operations",
		"Limited token budget",
		"Minimal tool set",
	}

	for _, element := range expectedElements {
		if !containsCompact(prompt, element) {
			t.Errorf("Compact default prompt should contain '%s'", element)
		}
	}
}

func TestCompact_DefaultTools(t *testing.T) {
	compact := NewCompact()
	tools := compact.AllowedTools()

	expectedTools := []string{"read_file", "get_context", "file_search"}

	if len(tools) != len(expectedTools) {
		t.Errorf("Compact.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("Compact.AllowedTools() [%d] = %s, want %s", i, tool, expectedTools[i])
		}
	}
}

func TestCompact_DefaultMaxTokens(t *testing.T) {
	compact := NewCompact()
	maxTokens := compact.MaxTokens()

	if maxTokens != DefaultCompactMaxTokens {
		t.Errorf("Compact.MaxTokens() = %d, want %d", maxTokens, DefaultCompactMaxTokens)
	}
}

func TestCompact_Concurrency(t *testing.T) {
	compact := NewCompact()

	// Test concurrent access to methods
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			// These methods should be safe for concurrent access
			compact.Name()
			compact.SystemPrompt()
			compact.AllowedTools()
			compact.MaxTokens()
			compact.Validate()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper function to check if a string contains a substring
func containsCompact(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && containsCompact(s[1:], substr)
}
