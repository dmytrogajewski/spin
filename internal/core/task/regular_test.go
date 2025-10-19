package task

import (
	"testing"
)

func TestNewRegular(t *testing.T) {
	regular := NewRegular()

	if regular == nil {
		t.Fatal("NewRegular() returned nil")
	}

	if regular.Name() != "regular" {
		t.Errorf("NewRegular().Name() = %v, want %v", regular.Name(), "regular")
	}
}

func TestRegular_Name(t *testing.T) {
	regular := NewRegular()

	if regular.Name() != "regular" {
		t.Errorf("Regular.Name() = %v, want %v", regular.Name(), "regular")
	}
}

func TestRegular_SystemPrompt(t *testing.T) {
	regular := NewRegular()
	result := regular.SystemPrompt()

	if len(result) == 0 {
		t.Error("Regular.SystemPrompt() returned empty string")
	}
}

func TestRegular_AllowedTools(t *testing.T) {
	tests := []struct {
		name         string
		regular      *Regular
		wantTools    []string
		wantExcluded bool
	}{
		{
			name:    "default tools",
			regular: NewRegular(),
			wantTools: []string{
				"read_file", "write_file", "list_directory",
				"execute_command", "get_context", "file_search",
				"apply_patch", "git_context",
			},
			wantExcluded: false,
		},
		{
			name: "with excluded tools",
			regular: func() *Regular {
				r := NewRegular()
				r.config = &RegularConfig{
					ExcludedTools: []string{"execute_command", "write_file"},
				}
				return r
			}(),
			wantTools: []string{
				"read_file", "list_directory", "get_context",
				"file_search", "apply_patch", "git_context",
			},
			wantExcluded: true,
		},
		{
			name: "with all tools excluded",
			regular: func() *Regular {
				r := NewRegular()
				r.config = &RegularConfig{
					ExcludedTools: []string{
						"read_file", "write_file", "list_directory",
						"execute_command", "get_context", "file_search",
						"apply_patch", "git_context",
					},
				}
				return r
			}(),
			wantTools:    []string{},
			wantExcluded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := tt.regular.AllowedTools()

			if tools == nil {
				t.Error("Regular.AllowedTools() returned nil")
				return
			}

			if tt.wantExcluded {
				// Check that excluded tools are not present
				for _, tool := range tt.wantTools {
					found := false
					for _, allowed := range tools {
						if allowed == tool {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Regular.AllowedTools() missing expected tool: %s", tool)
					}
				}
			} else {
				// Check that all expected tools are present
				if len(tools) != len(tt.wantTools) {
					t.Errorf("Regular.AllowedTools() got %d tools, want %d", len(tools), len(tt.wantTools))
				}

				for _, wantTool := range tt.wantTools {
					found := false
					for _, tool := range tools {
						if tool == wantTool {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Regular.AllowedTools() missing tool: %s", wantTool)
					}
				}
			}
		})
	}
}

func TestRegular_MaxTokens(t *testing.T) {
	regular := NewRegular()
	result := regular.MaxTokens()

	if result <= 0 {
		t.Errorf("Regular.MaxTokens() = %v, want > 0", result)
	}
}

func TestRegular_Validate(t *testing.T) {
	regular := NewRegular()
	err := regular.Validate()

	if err != nil {
		t.Errorf("Regular.Validate() unexpected error: %v", err)
	}
}

func TestRegular_Validate_WithConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *RegularConfig
		wantError bool
	}{
		{
			name:      "nil config is valid",
			config:    nil,
			wantError: false,
		},
		{
			name: "valid config",
			config: &RegularConfig{
				MaxTokens:     16000,
				ExcludedTools: []string{"execute_command"},
			},
			wantError: false,
		},
		{
			name: "negative max tokens",
			config: &RegularConfig{
				MaxTokens: -100,
			},
			wantError: true,
		},
		{
			name: "max tokens exceeds limit",
			config: &RegularConfig{
				MaxTokens: MaxAllowedTokens + 1000,
			},
			wantError: true,
		},
		{
			name: "empty excluded tool",
			config: &RegularConfig{
				ExcludedTools: []string{"tool1", "", "tool2"},
			},
			wantError: true,
		},
		{
			name: "custom prompt too short",
			config: &RegularConfig{
				CustomSystemPrompt: "Short",
			},
			wantError: true,
		},
		{
			name: "valid custom prompt",
			config: &RegularConfig{
				CustomSystemPrompt: "This is a sufficiently long custom system prompt that meets all requirements",
			},
			wantError: false,
		},
		{
			name: "multiple errors",
			config: &RegularConfig{
				MaxTokens:          -50,
				ExcludedTools:      []string{"", "valid"},
				CustomSystemPrompt: "x",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regular := &Regular{config: tt.config}
			err := regular.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Regular.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestRegular_SystemPrompt_WithCustom(t *testing.T) {
	customPrompt := "Custom regular system prompt that is sufficiently long to meet minimum requirements"
	regular := &Regular{
		config: &RegularConfig{
			CustomSystemPrompt: customPrompt,
		},
	}

	result := regular.SystemPrompt()
	if result != customPrompt {
		t.Errorf("Regular.SystemPrompt() = %v, want %v", result, customPrompt)
	}
}

func TestRegular_MaxTokens_WithCustom(t *testing.T) {
	customTokens := 20000
	regular := &Regular{
		config: &RegularConfig{
			MaxTokens: customTokens,
		},
	}

	result := regular.MaxTokens()
	if result != customTokens {
		t.Errorf("Regular.MaxTokens() = %v, want %v", result, customTokens)
	}
}

func TestRegular_Concurrency(t *testing.T) {
	regular := NewRegular()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = regular.Name()
			_ = regular.SystemPrompt()
			_ = regular.AllowedTools()
			_ = regular.MaxTokens()
			_ = regular.Validate()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test Regular validation methods
func TestRegular_ValidateMaxTokens(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
		wantErrs  int
	}{
		{
			name:      "valid max tokens",
			maxTokens: 8192,
			wantErrs:  0,
		},
		{
			name:      "negative max tokens",
			maxTokens: -1,
			wantErrs:  1,
		},
		{
			name:      "zero max tokens",
			maxTokens: 0,
			wantErrs:  0,
		},
		{
			name:      "max allowed tokens",
			maxTokens: MaxAllowedTokens,
			wantErrs:  0,
		},
		{
			name:      "exceeds max allowed tokens",
			maxTokens: MaxAllowedTokens + 1,
			wantErrs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regular := NewRegular()
			regular.config = &RegularConfig{
				MaxTokens: tt.maxTokens,
			}

			errs := regular.validateMaxTokens()
			if len(errs) != tt.wantErrs {
				t.Errorf("validateMaxTokens() got %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestRegular_ValidateExcludedTools(t *testing.T) {
	tests := []struct {
		name          string
		excludedTools []string
		wantErrs      int
	}{
		{
			name:          "valid excluded tools",
			excludedTools: []string{"tool1", "tool2"},
			wantErrs:      0,
		},
		{
			name:          "empty excluded tools",
			excludedTools: []string{},
			wantErrs:      0,
		},
		{
			name:          "empty tool in middle",
			excludedTools: []string{"tool1", "", "tool3"},
			wantErrs:      1,
		},
		{
			name:          "empty tool at start",
			excludedTools: []string{"", "tool2"},
			wantErrs:      1,
		},
		{
			name:          "multiple empty tools",
			excludedTools: []string{"tool1", "", "tool3", ""},
			wantErrs:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regular := NewRegular()
			regular.config = &RegularConfig{
				ExcludedTools: tt.excludedTools,
			}

			errs := regular.validateExcludedTools()
			if len(errs) != tt.wantErrs {
				t.Errorf("validateExcludedTools() got %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestRegular_ValidateCustomSystemPrompt(t *testing.T) {
	tests := []struct {
		name               string
		customSystemPrompt string
		wantErrs           int
	}{
		{
			name:               "valid custom prompt",
			customSystemPrompt: "This is a valid custom system prompt that is long enough",
			wantErrs:           0,
		},
		{
			name:               "empty custom prompt",
			customSystemPrompt: "",
			wantErrs:           0, // Empty is valid, falls back to default
		},
		{
			name:               "short custom prompt",
			customSystemPrompt: "short",
			wantErrs:           1,
		},
		{
			name:               "minimum length prompt",
			customSystemPrompt: "This is exactly fifty characters long prompt that meets minimum requirements",
			wantErrs:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regular := NewRegular()
			regular.config = &RegularConfig{
				CustomSystemPrompt: tt.customSystemPrompt,
			}

			errs := regular.validateCustomSystemPrompt()
			if len(errs) != tt.wantErrs {
				t.Errorf("validateCustomSystemPrompt() got %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

// Test Regular constants
func TestRegular_Constants(t *testing.T) {
	if DefaultMaxTokens != 16384 {
		t.Errorf("DefaultMaxTokens = %v, want 16384", DefaultMaxTokens)
	}
	if MaxAllowedTokens != 100000 {
		t.Errorf("MaxAllowedTokens = %v, want 100000", MaxAllowedTokens)
	}
	if MinPromptLength != 50 {
		t.Errorf("MinPromptLength = %v, want 50", MinPromptLength)
	}
}
