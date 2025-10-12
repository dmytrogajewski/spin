package task

import (
	"strings"
	"testing"
)

// TestNewCompact tests the default constructor
func TestNewCompact(t *testing.T) {
	c := NewCompact()
	if c == nil {
		t.Fatal("NewCompact() returned nil")
	}

	// Should have nil config for defaults
	if c.config != nil {
		t.Error("NewCompact() should have nil config for defaults")
	}
}

// TestNewCompactWithConfig tests constructor with custom config
func TestNewCompactWithConfig(t *testing.T) {
	config := &CompactConfig{
		MaxTokens:       8192,
		AdditionalTools: []string{"get_context"},
	}

	c := NewCompactWithConfig(config)
	if c == nil {
		t.Fatal("NewCompactWithConfig() returned nil")
	}

	if c.config != config {
		t.Error("NewCompactWithConfig() did not set config")
	}
}

// TestNewCompactWithConfig_Nil tests constructor with nil config
func TestNewCompactWithConfig_Nil(t *testing.T) {
	c := NewCompactWithConfig(nil)
	if c == nil {
		t.Fatal("NewCompactWithConfig(nil) returned nil")
	}

	if c.config != nil {
		t.Error("NewCompactWithConfig(nil) should have nil config")
	}
}

// TestCompact_Name tests the Name() method
func TestCompact_Name(t *testing.T) {
	tests := []struct {
		name   string
		config *CompactConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &CompactConfig{
				MaxTokens: 8192,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompactWithConfig(tt.config)
			if got := c.Name(); got != "compact" {
				t.Errorf("Name() = %q, want %q", got, "compact")
			}
		})
	}
}

// TestCompact_SystemPrompt tests the SystemPrompt() method
func TestCompact_SystemPrompt(t *testing.T) {
	tests := []struct {
		name         string
		config       *CompactConfig
		wantContains string
		minLength    int
		maxLength    int
	}{
		{
			name:         "default prompt",
			config:       nil,
			wantContains: "Compact Mode",
			minLength:    100,
			maxLength:    1000, // Should be concise
		},
		{
			name: "custom prompt",
			config: &CompactConfig{
				CustomSystemPrompt: "Custom compact prompt for testing purposes that is long enough",
			},
			wantContains: "Custom compact prompt",
			minLength:    50,
			maxLength:    10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompactWithConfig(tt.config)
			prompt := c.SystemPrompt()

			if prompt == "" {
				t.Error("SystemPrompt() returned empty string")
			}

			if len(prompt) < tt.minLength {
				t.Errorf("SystemPrompt() length = %d, want >= %d", len(prompt), tt.minLength)
			}

			if len(prompt) > tt.maxLength {
				t.Errorf("SystemPrompt() length = %d, want <= %d (should be concise)", len(prompt), tt.maxLength)
			}

			if !strings.Contains(prompt, tt.wantContains) {
				t.Errorf("SystemPrompt() should contain %q", tt.wantContains)
			}
		})
	}
}

// TestCompact_SystemPrompt_Concise tests prompt is concise
func TestCompact_SystemPrompt_Concise(t *testing.T) {
	compact := NewCompact()
	regular := NewRegular()

	compactPrompt := compact.SystemPrompt()
	regularPrompt := regular.SystemPrompt()

	// Compact prompt should be shorter than Regular
	if len(compactPrompt) >= len(regularPrompt) {
		t.Errorf("Compact prompt (%d chars) should be shorter than Regular (%d chars)",
			len(compactPrompt), len(regularPrompt))
	}
}

// TestCompact_SystemPrompt_DefaultContent tests default prompt content
func TestCompact_SystemPrompt_DefaultContent(t *testing.T) {
	c := NewCompact()
	prompt := c.SystemPrompt()

	requiredPhrases := []string{
		"Compact Mode",
		"quick",
		"concise",
		"Limited token budget",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("Default compact prompt should contain %q", phrase)
		}
	}
}

// TestCompact_AllowedTools tests the AllowedTools() method
func TestCompact_AllowedTools(t *testing.T) {
	tests := []struct {
		name         string
		config       *CompactConfig
		wantContains []string
		wantExcludes []string
		exactCount   int
	}{
		{
			name:   "default minimal set",
			config: nil,
			wantContains: []string{
				"read_file",
				"get_context",
				"file_search",
			},
			wantExcludes: []string{
				"write_file",
				"execute_command",
				"git_context",
			},
			exactCount: 3,
		},
		{
			name: "with additional tools",
			config: &CompactConfig{
				AdditionalTools: []string{"get_context"},
			},
			wantContains: []string{
				"read_file",
				"get_context",
				"file_search",
				"get_context",
			},
			wantExcludes: []string{
				"write_file",
				"execute_command",
			},
			exactCount: 4,
		},
		{
			name: "multiple additional tools",
			config: &CompactConfig{
				AdditionalTools: []string{"get_context", "search_files"},
			},
			wantContains: []string{
				"read_file",
				"get_context",
				"search_files",
			},
			exactCount: 5,
		},
		{
			name: "empty additional tools",
			config: &CompactConfig{
				AdditionalTools: []string{},
			},
			wantContains: []string{
				"read_file",
				"get_context",
				"file_search",
			},
			exactCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompactWithConfig(tt.config)
			tools := c.AllowedTools()

			if len(tools) == 0 {
				t.Error("AllowedTools() returned empty list")
			}

			if tt.exactCount > 0 && len(tools) != tt.exactCount {
				t.Errorf("AllowedTools() count = %d, want exactly %d", len(tools), tt.exactCount)
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
					t.Errorf("AllowedTools() should not contain %q in compact mode", exclude)
				}
			}
		})
	}
}

// TestCompact_AllowedTools_Minimal tests minimal tool set
func TestCompact_AllowedTools_Minimal(t *testing.T) {
	c := NewCompact()
	tools := c.AllowedTools()

	// Should have exactly 3 tools in default config
	if len(tools) != 3 {
		t.Errorf("Default Compact mode should have exactly 3 tools, got %d: %v", len(tools), tools)
	}

	// Essential tools should be present
	essentialTools := []string{"read_file", "get_context", "file_search"}
	for _, tool := range essentialTools {
		if !contains(tools, tool) {
			t.Errorf("Compact mode missing essential tool: %q", tool)
		}
	}
}

// TestCompact_AllowedTools_NoWriteOperations tests write operations are excluded
func TestCompact_AllowedTools_NoWriteOperations(t *testing.T) {
	c := NewCompact()
	tools := c.AllowedTools()

	writeTools := []string{
		"write_file",
		"execute_command",
		"git_add",
		"git_commit",
	}

	for _, writeTool := range writeTools {
		if contains(tools, writeTool) {
			t.Errorf("Compact mode should not allow write tool: %q", writeTool)
		}
	}
}

// TestCompact_MaxTokens tests the MaxTokens() method
func TestCompact_MaxTokens(t *testing.T) {
	tests := []struct {
		name   string
		config *CompactConfig
		want   int
	}{
		{
			name:   "default",
			config: nil,
			want:   DefaultCompactMaxTokens,
		},
		{
			name: "custom",
			config: &CompactConfig{
				MaxTokens: 8192,
			},
			want: 8192,
		},
		{
			name: "zero uses default",
			config: &CompactConfig{
				MaxTokens: 0,
			},
			want: DefaultCompactMaxTokens,
		},
		{
			name: "custom small",
			config: &CompactConfig{
				MaxTokens: 2048,
			},
			want: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompactWithConfig(tt.config)
			if got := c.MaxTokens(); got != tt.want {
				t.Errorf("MaxTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestCompact_MaxTokens_SmallestBudget tests Compact has smallest budget
func TestCompact_MaxTokens_SmallestBudget(t *testing.T) {
	compact := NewCompact()
	regular := NewRegular()
	review := NewReview()

	if compact.MaxTokens() >= review.MaxTokens() {
		t.Errorf("Compact MaxTokens (%d) should be smaller than Review (%d)",
			compact.MaxTokens(), review.MaxTokens())
	}

	if compact.MaxTokens() >= regular.MaxTokens() {
		t.Errorf("Compact MaxTokens (%d) should be smaller than Regular (%d)",
			compact.MaxTokens(), regular.MaxTokens())
	}

	// Compact should be significantly smaller (at least 50% smaller than Review)
	if compact.MaxTokens() > review.MaxTokens()/2 {
		t.Errorf("Compact MaxTokens (%d) should be at most 50%% of Review (%d)",
			compact.MaxTokens(), review.MaxTokens())
	}
}

// TestCompact_Validate tests the Validate() method
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
			name:    "default config",
			config:  &CompactConfig{},
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: &CompactConfig{
				MaxTokens:          8192,
				AdditionalTools:    []string{"get_context"},
				CustomSystemPrompt: "This is a valid custom compact prompt for testing purposes that meets length",
			},
			wantErr: false,
		},
		{
			name: "negative max tokens",
			config: &CompactConfig{
				MaxTokens: -100,
			},
			wantErr: true,
		},
		{
			name: "exceeds max allowed",
			config: &CompactConfig{
				MaxTokens: MaxAllowedTokens + 1,
			},
			wantErr: true,
		},
		{
			name: "empty additional tool",
			config: &CompactConfig{
				AdditionalTools: []string{""},
			},
			wantErr: true,
		},
		{
			name: "empty string in additional tools",
			config: &CompactConfig{
				AdditionalTools: []string{"get_context", "", "search_files"},
			},
			wantErr: true,
		},
		{
			name: "short custom prompt",
			config: &CompactConfig{
				CustomSystemPrompt: "short",
			},
			wantErr: true,
		},
		{
			name: "valid max tokens at boundary",
			config: &CompactConfig{
				MaxTokens: MaxAllowedTokens,
			},
			wantErr: false,
		},
		{
			name: "valid prompt at minimum length",
			config: &CompactConfig{
				CustomSystemPrompt: strings.Repeat("a", MinPromptLength),
			},
			wantErr: false,
		},
		{
			name: "valid with multiple additional tools",
			config: &CompactConfig{
				AdditionalTools: []string{
					"get_context",
					"search_files",
					"git_context",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompactWithConfig(tt.config)
			err := c.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCompact_Validate_MultipleErrors tests multiple validation errors
func TestCompact_Validate_MultipleErrors(t *testing.T) {
	config := &CompactConfig{
		MaxTokens:          -100,
		AdditionalTools:    []string{""},
		CustomSystemPrompt: "short",
	}

	c := NewCompactWithConfig(config)
	err := c.Validate()

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

// TestCompact_ImplementsTaskInterface tests Task interface implementation
func TestCompact_ImplementsTaskInterface(t *testing.T) {
	var _ Task = (*Compact)(nil) // Compile-time check

	c := NewCompact()

	// Runtime checks
	if c.Name() == "" {
		t.Error("Name() should not be empty")
	}

	if c.SystemPrompt() == "" {
		t.Error("SystemPrompt() should not be empty")
	}

	if len(c.AllowedTools()) == 0 {
		t.Error("AllowedTools() should not be empty")
	}

	if c.MaxTokens() <= 0 {
		t.Error("MaxTokens() should be positive")
	}

	if err := c.Validate(); err != nil {
		t.Errorf("Validate() should not error for default: %v", err)
	}
}

// TestCompact_RegistryIntegration tests integration with Registry
func TestCompact_RegistryIntegration(t *testing.T) {
	registry := NewRegistry()

	// Register compact task
	compact := NewCompact()
	if err := registry.Register("compact", compact); err != nil {
		t.Fatalf("Failed to register compact task: %v", err)
	}

	// Retrieve and verify
	task, err := registry.Get("compact")
	if err != nil {
		t.Fatalf("Failed to get compact task: %v", err)
	}

	if task.Name() != "compact" {
		t.Errorf("Retrieved task name = %q, want %q", task.Name(), "compact")
	}

	// Verify it has minimal tools
	tools := task.AllowedTools()
	if len(tools) > 5 {
		t.Errorf("Compact task should have minimal tools, got %d", len(tools))
	}
}

// TestCompact_MostConstrained tests Compact is most constrained mode
func TestCompact_MostConstrained(t *testing.T) {
	compact := NewCompact()
	regular := NewRegular()
	review := NewReview()

	// Smallest token budget
	if compact.MaxTokens() >= review.MaxTokens() {
		t.Error("Compact should have smallest token budget")
	}
	if compact.MaxTokens() >= regular.MaxTokens() {
		t.Error("Compact should have smallest token budget")
	}

	// Fewest tools
	compactTools := compact.AllowedTools()
	reviewTools := review.AllowedTools()
	regularTools := regular.AllowedTools()

	if len(compactTools) >= len(reviewTools) {
		t.Errorf("Compact tools (%d) should be fewer than Review (%d)",
			len(compactTools), len(reviewTools))
	}
	if len(compactTools) >= len(regularTools) {
		t.Errorf("Compact tools (%d) should be fewer than Regular (%d)",
			len(compactTools), len(regularTools))
	}

	// Different name
	if compact.Name() == regular.Name() || compact.Name() == review.Name() {
		t.Error("Compact should have unique name")
	}
}

// TestCompact_AdditionalTools tests additional tools configuration
func TestCompact_AdditionalTools(t *testing.T) {
	tests := []struct {
		name            string
		additionalTools []string
		wantValid       bool
		expectedCount   int
	}{
		{
			name:            "no additional tools",
			additionalTools: nil,
			wantValid:       true,
			expectedCount:   3,
		},
		{
			name:            "one additional tool",
			additionalTools: []string{"get_context"},
			wantValid:       true,
			expectedCount:   4,
		},
		{
			name:            "multiple additional tools",
			additionalTools: []string{"get_context", "search_files", "git_context"},
			wantValid:       true,
			expectedCount:   6,
		},
		{
			name:            "empty tool name",
			additionalTools: []string{""},
			wantValid:       false,
			expectedCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CompactConfig{
				AdditionalTools: tt.additionalTools,
			}
			c := NewCompactWithConfig(config)
			err := c.Validate()

			if tt.wantValid && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
			if !tt.wantValid && err == nil {
				t.Error("Validate() should return error for invalid additional tools")
			}

			if tt.wantValid && tt.expectedCount > 0 {
				tools := c.AllowedTools()
				if len(tools) != tt.expectedCount {
					t.Errorf("AllowedTools() count = %d, want %d", len(tools), tt.expectedCount)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkCompact_Name(b *testing.B) {
	c := NewCompact()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Name()
	}
}

func BenchmarkCompact_SystemPrompt(b *testing.B) {
	c := NewCompact()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.SystemPrompt()
	}
}

func BenchmarkCompact_AllowedTools(b *testing.B) {
	c := NewCompact()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.AllowedTools()
	}
}

func BenchmarkCompact_AllowedTools_WithAdditional(b *testing.B) {
	c := NewCompactWithConfig(&CompactConfig{
		AdditionalTools: []string{"get_context", "search_files"},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.AllowedTools()
	}
}

func BenchmarkCompact_Validate(b *testing.B) {
	c := NewCompact()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Validate()
	}
}
