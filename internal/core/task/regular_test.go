package task

import (
	"strings"
	"testing"
)

// TestNewRegular tests the default constructor
func TestNewRegular(t *testing.T) {
	r := NewRegular()
	if r == nil {
		t.Fatal("NewRegular() returned nil")
	}

	// Should have nil config for defaults
	if r.config != nil {
		t.Error("NewRegular() should have nil config for defaults")
	}
}

// TestNewRegularWithConfig tests constructor with custom config
func TestNewRegularWithConfig(t *testing.T) {
	config := &RegularConfig{
		MaxTokens:     8192,
		ExcludedTools: []string{"shell"},
	}

	r := NewRegularWithConfig(config)
	if r == nil {
		t.Fatal("NewRegularWithConfig() returned nil")
	}

	if r.config != config {
		t.Error("NewRegularWithConfig() did not set config")
	}
}

// TestNewRegularWithConfig_Nil tests constructor with nil config
func TestNewRegularWithConfig_Nil(t *testing.T) {
	r := NewRegularWithConfig(nil)
	if r == nil {
		t.Fatal("NewRegularWithConfig(nil) returned nil")
	}

	if r.config != nil {
		t.Error("NewRegularWithConfig(nil) should have nil config")
	}
}

// TestRegular_Name tests the Name() method
func TestRegular_Name(t *testing.T) {
	tests := []struct {
		name   string
		config *RegularConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &RegularConfig{
				MaxTokens: 8192,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegularWithConfig(tt.config)
			if got := r.Name(); got != "regular" {
				t.Errorf("Name() = %q, want %q", got, "regular")
			}
		})
	}
}

// TestRegular_SystemPrompt tests the SystemPrompt() method
func TestRegular_SystemPrompt(t *testing.T) {
	tests := []struct {
		name         string
		config       *RegularConfig
		wantContains string
		minLength    int
	}{
		{
			name:         "default prompt",
			config:       nil,
			wantContains: "Spin",
			minLength:    100,
		},
		{
			name: "custom prompt",
			config: &RegularConfig{
				CustomSystemPrompt: "Custom agent prompt for testing purposes that is long enough",
			},
			wantContains: "Custom agent prompt",
			minLength:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegularWithConfig(tt.config)
			prompt := r.SystemPrompt()

			if prompt == "" {
				t.Error("SystemPrompt() returned empty string")
			}

			if len(prompt) < tt.minLength {
				t.Errorf("SystemPrompt() length = %d, want >= %d", len(prompt), tt.minLength)
			}

			if !strings.Contains(prompt, tt.wantContains) {
				t.Errorf("SystemPrompt() should contain %q", tt.wantContains)
			}
		})
	}
}

// TestRegular_SystemPrompt_DefaultContent tests default prompt content
func TestRegular_SystemPrompt_DefaultContent(t *testing.T) {
	r := NewRegular()
	prompt := r.SystemPrompt()

	requiredPhrases := []string{
		"Spin",
		"CAPABILITIES",
		"BEHAVIOR",
		"CONSTRAINTS",
		"WORKFLOW",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("Default system prompt should contain %q", phrase)
		}
	}
}

// TestRegular_AllowedTools tests the AllowedTools() method
func TestRegular_AllowedTools(t *testing.T) {
	tests := []struct {
		name         string
		config       *RegularConfig
		wantContains []string
		wantExcludes []string
		minCount     int
	}{
		{
			name:   "default tools",
			config: nil,
			wantContains: []string{
				"read_file",
				"write_file",
				"shell",
				"git_status",
				"search_code",
				"list_dir",
			},
			wantExcludes: nil,
			minCount:     10,
		},
		{
			name: "excluded shell",
			config: &RegularConfig{
				ExcludedTools: []string{"shell"},
			},
			wantContains: []string{
				"read_file",
				"write_file",
			},
			wantExcludes: []string{"shell"},
			minCount:     5,
		},
		{
			name: "multiple exclusions",
			config: &RegularConfig{
				ExcludedTools: []string{"shell", "write_file"},
			},
			wantContains: []string{
				"read_file",
			},
			wantExcludes: []string{"shell", "write_file"},
			minCount:     3,
		},
		{
			name: "empty exclusion list",
			config: &RegularConfig{
				ExcludedTools: []string{},
			},
			wantContains: []string{
				"read_file",
				"shell",
			},
			wantExcludes: nil,
			minCount:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegularWithConfig(tt.config)
			tools := r.AllowedTools()

			if len(tools) == 0 {
				t.Error("AllowedTools() returned empty list")
			}

			if len(tools) < tt.minCount {
				t.Errorf("AllowedTools() count = %d, want >= %d", len(tools), tt.minCount)
			}

			// Check expected tools are present
			for _, want := range tt.wantContains {
				if !contains(tools, want) {
					t.Errorf("AllowedTools() missing %q, got: %v", want, tools)
				}
			}

			// Check excluded tools are not present
			for _, exclude := range tt.wantExcludes {
				if contains(tools, exclude) {
					t.Errorf("AllowedTools() should not contain excluded tool %q", exclude)
				}
			}
		})
	}
}

// TestRegular_AllowedTools_NoDuplicates tests no duplicate tools
func TestRegular_AllowedTools_NoDuplicates(t *testing.T) {
	r := NewRegular()
	tools := r.AllowedTools()

	seen := make(map[string]bool)
	for _, tool := range tools {
		if seen[tool] {
			t.Errorf("AllowedTools() contains duplicate: %q", tool)
		}
		seen[tool] = true
	}
}

// TestRegular_MaxTokens tests the MaxTokens() method
func TestRegular_MaxTokens(t *testing.T) {
	tests := []struct {
		name   string
		config *RegularConfig
		want   int
	}{
		{
			name:   "default",
			config: nil,
			want:   DefaultMaxTokens,
		},
		{
			name: "custom",
			config: &RegularConfig{
				MaxTokens: 8192,
			},
			want: 8192,
		},
		{
			name: "zero uses default",
			config: &RegularConfig{
				MaxTokens: 0,
			},
			want: DefaultMaxTokens,
		},
		{
			name: "large budget",
			config: &RegularConfig{
				MaxTokens: 32768,
			},
			want: 32768,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegularWithConfig(tt.config)
			if got := r.MaxTokens(); got != tt.want {
				t.Errorf("MaxTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestRegular_Validate tests the Validate() method
func TestRegular_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RegularConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "default config",
			config:  &RegularConfig{},
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: &RegularConfig{
				MaxTokens:          8192,
				ExcludedTools:      []string{"shell"},
				CustomSystemPrompt: "This is a valid custom system prompt for testing purposes that meets the minimum length",
			},
			wantErr: false,
		},
		{
			name: "negative max tokens",
			config: &RegularConfig{
				MaxTokens: -100,
			},
			wantErr: true,
		},
		{
			name: "exceeds max allowed",
			config: &RegularConfig{
				MaxTokens: MaxAllowedTokens + 1,
			},
			wantErr: true,
		},
		{
			name: "empty excluded tool",
			config: &RegularConfig{
				ExcludedTools: []string{""},
			},
			wantErr: true,
		},
		{
			name: "empty string in excluded tools",
			config: &RegularConfig{
				ExcludedTools: []string{"shell", "", "git"},
			},
			wantErr: true,
		},
		{
			name: "short custom prompt",
			config: &RegularConfig{
				CustomSystemPrompt: "short",
			},
			wantErr: true,
		},
		{
			name: "valid max tokens at boundary",
			config: &RegularConfig{
				MaxTokens: MaxAllowedTokens,
			},
			wantErr: false,
		},
		{
			name: "valid prompt at minimum length",
			config: &RegularConfig{
				CustomSystemPrompt: strings.Repeat("a", MinPromptLength),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegularWithConfig(tt.config)
			err := r.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRegular_Validate_MultipleErrors tests multiple validation errors
func TestRegular_Validate_MultipleErrors(t *testing.T) {
	config := &RegularConfig{
		MaxTokens:          -100,
		ExcludedTools:      []string{""},
		CustomSystemPrompt: "short",
	}

	r := NewRegularWithConfig(config)
	err := r.Validate()

	if err == nil {
		t.Fatal("Validate() should return error for multiple violations")
	}

	// Check that error contains multiple issues
	errStr := err.Error()
	checks := []string{
		"negative",
		"empty",
		"short",
	}

	foundCount := 0
	for _, check := range checks {
		if strings.Contains(strings.ToLower(errStr), check) {
			foundCount++
		}
	}

	if foundCount < 2 {
		t.Errorf("Validate() should report multiple errors, got: %v", err)
	}
}

// TestRegular_ImplementsTaskInterface tests Task interface implementation
func TestRegular_ImplementsTaskInterface(t *testing.T) {
	var _ Task = (*Regular)(nil) // Compile-time check

	r := NewRegular()

	// Runtime checks
	if r.Name() == "" {
		t.Error("Name() should not be empty")
	}

	if r.SystemPrompt() == "" {
		t.Error("SystemPrompt() should not be empty")
	}

	if len(r.AllowedTools()) == 0 {
		t.Error("AllowedTools() should not be empty")
	}

	if r.MaxTokens() <= 0 {
		t.Error("MaxTokens() should be positive")
	}

	if err := r.Validate(); err != nil {
		t.Errorf("Validate() should not error for default: %v", err)
	}
}

// TestRegular_RegistryIntegration tests integration with Registry
func TestRegular_RegistryIntegration(t *testing.T) {
	registry := NewRegistry()

	// Register regular task
	regular := NewRegular()
	if err := registry.Register("regular", regular); err != nil {
		t.Fatalf("Failed to register regular task: %v", err)
	}

	// Retrieve and verify
	task, err := registry.Get("regular")
	if err != nil {
		t.Fatalf("Failed to get regular task: %v", err)
	}

	if task.Name() != "regular" {
		t.Errorf("Retrieved task name = %q, want %q", task.Name(), "regular")
	}

	// Set as default
	if err := registry.SetDefault("regular"); err != nil {
		t.Fatalf("Failed to set regular as default: %v", err)
	}

	defaultTask, err := registry.GetDefault()
	if err != nil {
		t.Fatalf("Failed to get default task: %v", err)
	}

	if defaultTask.Name() != "regular" {
		t.Errorf("Default task name = %q, want %q", defaultTask.Name(), "regular")
	}
}

// TestFilterTools tests the filterTools helper function
func TestFilterTools(t *testing.T) {
	tests := []struct {
		name     string
		tools    []string
		excluded []string
		want     []string
	}{
		{
			name:     "no exclusions",
			tools:    []string{"a", "b", "c"},
			excluded: []string{},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "exclude one",
			tools:    []string{"a", "b", "c"},
			excluded: []string{"b"},
			want:     []string{"a", "c"},
		},
		{
			name:     "exclude multiple",
			tools:    []string{"a", "b", "c", "d"},
			excluded: []string{"b", "d"},
			want:     []string{"a", "c"},
		},
		{
			name:     "exclude non-existent",
			tools:    []string{"a", "b", "c"},
			excluded: []string{"x", "y"},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "exclude all",
			tools:    []string{"a", "b"},
			excluded: []string{"a", "b"},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterTools(tt.tools, tt.excluded)

			if len(got) != len(tt.want) {
				t.Errorf("filterTools() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i, tool := range got {
				if tool != tt.want[i] {
					t.Errorf("filterTools()[%d] = %q, want %q", i, tool, tt.want[i])
				}
			}
		})
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkRegular_Name(b *testing.B) {
	r := NewRegular()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Name()
	}
}

func BenchmarkRegular_SystemPrompt(b *testing.B) {
	r := NewRegular()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.SystemPrompt()
	}
}

func BenchmarkRegular_AllowedTools(b *testing.B) {
	r := NewRegular()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.AllowedTools()
	}
}

func BenchmarkRegular_AllowedTools_WithExclusions(b *testing.B) {
	r := NewRegularWithConfig(&RegularConfig{
		ExcludedTools: []string{"shell", "write_file"},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.AllowedTools()
	}
}

func BenchmarkRegular_Validate(b *testing.B) {
	r := NewRegular()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Validate()
	}
}
