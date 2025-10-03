package task

import (
	"strings"
	"testing"
)

// TestNewReview tests the default constructor
func TestNewReview(t *testing.T) {
	r := NewReview()
	if r == nil {
		t.Fatal("NewReview() returned nil")
	}

	// Should have nil config for defaults
	if r.config != nil {
		t.Error("NewReview() should have nil config for defaults")
	}
}

// TestNewReviewWithConfig tests constructor with custom config
func TestNewReviewWithConfig(t *testing.T) {
	config := &ReviewConfig{
		MaxTokens:     8192,
		TargetFiles:   []string{"*.go"},
		IncludeGitOps: false,
	}

	r := NewReviewWithConfig(config)
	if r == nil {
		t.Fatal("NewReviewWithConfig() returned nil")
	}

	if r.config != config {
		t.Error("NewReviewWithConfig() did not set config")
	}
}

// TestNewReviewWithConfig_Nil tests constructor with nil config
func TestNewReviewWithConfig_Nil(t *testing.T) {
	r := NewReviewWithConfig(nil)
	if r == nil {
		t.Fatal("NewReviewWithConfig(nil) returned nil")
	}

	if r.config != nil {
		t.Error("NewReviewWithConfig(nil) should have nil config")
	}
}

// TestReview_Name tests the Name() method
func TestReview_Name(t *testing.T) {
	tests := []struct {
		name   string
		config *ReviewConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &ReviewConfig{
				MaxTokens: 8192,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReviewWithConfig(tt.config)
			if got := r.Name(); got != "review" {
				t.Errorf("Name() = %q, want %q", got, "review")
			}
		})
	}
}

// TestReview_SystemPrompt tests the SystemPrompt() method
func TestReview_SystemPrompt(t *testing.T) {
	tests := []struct {
		name         string
		config       *ReviewConfig
		wantContains string
		minLength    int
	}{
		{
			name:         "default prompt",
			config:       nil,
			wantContains: "Review Mode",
			minLength:    200,
		},
		{
			name: "custom prompt",
			config: &ReviewConfig{
				CustomSystemPrompt: "Custom review prompt for testing purposes that is long enough to pass validation",
			},
			wantContains: "Custom review prompt",
			minLength:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReviewWithConfig(tt.config)
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

// TestReview_SystemPrompt_DefaultContent tests default prompt content
func TestReview_SystemPrompt_DefaultContent(t *testing.T) {
	r := NewReview()
	prompt := r.SystemPrompt()

	requiredPhrases := []string{
		"Review Mode",
		"read-only",
		"CANNOT modify",
		"Code Quality",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("Default review prompt should contain %q", phrase)
		}
	}
}

// TestReview_AllowedTools tests the AllowedTools() method
func TestReview_AllowedTools(t *testing.T) {
	tests := []struct {
		name         string
		config       *ReviewConfig
		wantContains []string
		wantExcludes []string
		minCount     int
	}{
		{
			name:   "default with git",
			config: nil,
			wantContains: []string{
				"read_file",
				"list_dir",
				"search_files",
				"search_code",
				"get_context",
				"git_status",
				"git_diff",
				"git_log",
			},
			wantExcludes: []string{
				"write_file",
				"shell",
				"git_add",
				"git_commit",
			},
			minCount: 8,
		},
		{
			name: "without git ops",
			config: &ReviewConfig{
				IncludeGitOps: false,
			},
			wantContains: []string{
				"read_file",
				"list_dir",
				"search_code",
			},
			wantExcludes: []string{
				"write_file",
				"shell",
				"git_status",
				"git_diff",
				"git_add",
			},
			minCount: 5,
		},
		{
			name: "explicitly enable git",
			config: &ReviewConfig{
				IncludeGitOps: true,
			},
			wantContains: []string{
				"read_file",
				"git_status",
				"git_diff",
			},
			wantExcludes: []string{
				"write_file",
				"shell",
			},
			minCount: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReviewWithConfig(tt.config)
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
					t.Errorf("AllowedTools() should not contain write tool %q", exclude)
				}
			}
		})
	}
}

// TestReview_AllowedTools_NoWriteOperations tests write operations are excluded
func TestReview_AllowedTools_NoWriteOperations(t *testing.T) {
	r := NewReview()
	tools := r.AllowedTools()

	writeTools := []string{
		"write_file",
		"shell",
		"git_add",
		"git_commit",
	}

	for _, writeTool := range writeTools {
		if contains(tools, writeTool) {
			t.Errorf("Review mode should not allow write tool: %q", writeTool)
		}
	}
}

// TestReview_AllowedTools_NoDuplicates tests no duplicate tools
func TestReview_AllowedTools_NoDuplicates(t *testing.T) {
	r := NewReview()
	tools := r.AllowedTools()

	seen := make(map[string]bool)
	for _, tool := range tools {
		if seen[tool] {
			t.Errorf("AllowedTools() contains duplicate: %q", tool)
		}
		seen[tool] = true
	}
}

// TestReview_MaxTokens tests the MaxTokens() method
func TestReview_MaxTokens(t *testing.T) {
	tests := []struct {
		name   string
		config *ReviewConfig
		want   int
	}{
		{
			name:   "default",
			config: nil,
			want:   DefaultReviewMaxTokens,
		},
		{
			name: "custom",
			config: &ReviewConfig{
				MaxTokens: 8192,
			},
			want: 8192,
		},
		{
			name: "zero uses default",
			config: &ReviewConfig{
				MaxTokens: 0,
			},
			want: DefaultReviewMaxTokens,
		},
		{
			name: "custom large",
			config: &ReviewConfig{
				MaxTokens: 20000,
			},
			want: 20000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReviewWithConfig(tt.config)
			if got := r.MaxTokens(); got != tt.want {
				t.Errorf("MaxTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestReview_MaxTokens_SmallerThanRegular tests Review has smaller default budget
func TestReview_MaxTokens_SmallerThanRegular(t *testing.T) {
	review := NewReview()
	regular := NewRegular()

	if review.MaxTokens() >= regular.MaxTokens() {
		t.Errorf("Review MaxTokens (%d) should be smaller than Regular (%d)",
			review.MaxTokens(), regular.MaxTokens())
	}
}

// TestReview_Validate tests the Validate() method
func TestReview_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ReviewConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "default config",
			config:  &ReviewConfig{},
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: &ReviewConfig{
				MaxTokens:          8192,
				TargetFiles:        []string{"*.go", "src/**/*.ts"},
				IncludeGitOps:      true,
				CustomSystemPrompt: "This is a valid custom review prompt for testing purposes that meets length requirement",
			},
			wantErr: false,
		},
		{
			name: "negative max tokens",
			config: &ReviewConfig{
				MaxTokens: -100,
			},
			wantErr: true,
		},
		{
			name: "exceeds max allowed",
			config: &ReviewConfig{
				MaxTokens: MaxAllowedTokens + 1,
			},
			wantErr: true,
		},
		{
			name: "empty target file pattern",
			config: &ReviewConfig{
				TargetFiles: []string{""},
			},
			wantErr: true,
		},
		{
			name: "empty string in target files",
			config: &ReviewConfig{
				TargetFiles: []string{"*.go", "", "*.ts"},
			},
			wantErr: true,
		},
		{
			name: "short custom prompt",
			config: &ReviewConfig{
				CustomSystemPrompt: "short",
			},
			wantErr: true,
		},
		{
			name: "valid max tokens at boundary",
			config: &ReviewConfig{
				MaxTokens: MaxAllowedTokens,
			},
			wantErr: false,
		},
		{
			name: "valid prompt at minimum length",
			config: &ReviewConfig{
				CustomSystemPrompt: strings.Repeat("a", MinPromptLength),
			},
			wantErr: false,
		},
		{
			name: "valid with multiple target files",
			config: &ReviewConfig{
				TargetFiles: []string{
					"*.go",
					"src/**/*.ts",
					"internal/**/*.go",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReviewWithConfig(tt.config)
			err := r.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestReview_Validate_MultipleErrors tests multiple validation errors
func TestReview_Validate_MultipleErrors(t *testing.T) {
	config := &ReviewConfig{
		MaxTokens:          -100,
		TargetFiles:        []string{""},
		CustomSystemPrompt: "short",
	}

	r := NewReviewWithConfig(config)
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

// TestReview_ImplementsTaskInterface tests Task interface implementation
func TestReview_ImplementsTaskInterface(t *testing.T) {
	var _ Task = (*Review)(nil) // Compile-time check

	r := NewReview()

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

// TestReview_RegistryIntegration tests integration with Registry
func TestReview_RegistryIntegration(t *testing.T) {
	registry := NewRegistry()

	// Register review task
	review := NewReview()
	if err := registry.Register("review", review); err != nil {
		t.Fatalf("Failed to register review task: %v", err)
	}

	// Retrieve and verify
	task, err := registry.Get("review")
	if err != nil {
		t.Fatalf("Failed to get review task: %v", err)
	}

	if task.Name() != "review" {
		t.Errorf("Retrieved task name = %q, want %q", task.Name(), "review")
	}

	// Verify it's read-only
	tools := task.AllowedTools()
	if contains(tools, "write_file") {
		t.Error("Review task should not allow write_file")
	}
	if contains(tools, "shell") {
		t.Error("Review task should not allow shell")
	}
}

// TestReview_DifferentFromRegular tests Review differs from Regular mode
func TestReview_DifferentFromRegular(t *testing.T) {
	review := NewReview()
	regular := NewRegular()

	// Different names
	if review.Name() == regular.Name() {
		t.Error("Review and Regular should have different names")
	}

	// Different token budgets
	if review.MaxTokens() >= regular.MaxTokens() {
		t.Error("Review should have smaller token budget than Regular")
	}

	// Different tool sets
	reviewTools := review.AllowedTools()
	regularTools := regular.AllowedTools()

	if len(reviewTools) >= len(regularTools) {
		t.Error("Review should have fewer tools than Regular")
	}

	// Review should not have write tools
	for _, tool := range reviewTools {
		if tool == "write_file" || tool == "shell" {
			t.Errorf("Review should not have write tool: %q", tool)
		}
	}
}

// TestReview_TargetFiles tests target file configuration
func TestReview_TargetFiles(t *testing.T) {
	tests := []struct {
		name        string
		targetFiles []string
		wantValid   bool
	}{
		{
			name:        "no target files",
			targetFiles: nil,
			wantValid:   true,
		},
		{
			name:        "single file pattern",
			targetFiles: []string{"*.go"},
			wantValid:   true,
		},
		{
			name: "multiple patterns",
			targetFiles: []string{
				"*.go",
				"src/**/*.ts",
				"internal/**/*.go",
			},
			wantValid: true,
		},
		{
			name:        "empty pattern",
			targetFiles: []string{""},
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ReviewConfig{
				TargetFiles: tt.targetFiles,
			}
			r := NewReviewWithConfig(config)
			err := r.Validate()

			if tt.wantValid && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
			if !tt.wantValid && err == nil {
				t.Error("Validate() should return error for invalid target files")
			}
		})
	}
}

// Benchmark tests
func BenchmarkReview_Name(b *testing.B) {
	r := NewReview()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Name()
	}
}

func BenchmarkReview_SystemPrompt(b *testing.B) {
	r := NewReview()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.SystemPrompt()
	}
}

func BenchmarkReview_AllowedTools(b *testing.B) {
	r := NewReview()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.AllowedTools()
	}
}

func BenchmarkReview_AllowedTools_NoGit(b *testing.B) {
	r := NewReviewWithConfig(&ReviewConfig{
		IncludeGitOps: false,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.AllowedTools()
	}
}

func BenchmarkReview_Validate(b *testing.B) {
	r := NewReview()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Validate()
	}
}
