package task

import (
	"testing"
)

func TestNewPlanning(t *testing.T) {
	planning := NewPlanning()

	if planning == nil {
		t.Fatal("NewPlanning() returned nil")
	}

	if planning.config != nil {
		t.Errorf("NewPlanning() config = %v, want nil", planning.config)
	}
}

func TestPlanning_Name(t *testing.T) {
	planning := NewPlanning()

	name := planning.Name()

	if name != "planning" {
		t.Errorf("Planning.Name() = %s, want 'planning'", name)
	}
}

func TestPlanning_SystemPrompt(t *testing.T) {
	tests := []struct {
		name   string
		config *PlanningConfig
		want   string
	}{
		{
			name:   "default prompt",
			config: nil,
			want:   planningSystemPrompt,
		},
		{
			name: "custom prompt",
			config: &PlanningConfig{
				CustomPrompt: "Custom planning prompt",
			},
			want: "Custom planning prompt",
		},
		{
			name: "empty custom prompt",
			config: &PlanningConfig{
				CustomPrompt: "",
			},
			want: planningSystemPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planning := &Planning{config: tt.config}

			prompt := planning.SystemPrompt()

			if prompt != tt.want {
				t.Errorf("Planning.SystemPrompt() = %s, want %s", prompt, tt.want)
			}
		})
	}
}

func TestPlanning_AllowedTools(t *testing.T) {
	planning := NewPlanning()
	tools := planning.AllowedTools()

	expectedTools := []string{"get_context", "file_search", "git_context"}

	if len(tools) != len(expectedTools) {
		t.Errorf("Planning.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("Planning.AllowedTools() [%d] = %s, want %s", i, tool, expectedTools[i])
		}
	}
}

func TestPlanning_MaxTokens(t *testing.T) {
	planning := NewPlanning()
	maxTokens := planning.MaxTokens()

	if maxTokens != PlanningMaxTokens {
		t.Errorf("Planning.MaxTokens() = %d, want %d", maxTokens, PlanningMaxTokens)
	}
}

func TestPlanning_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *PlanningConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &PlanningConfig{
				MaxSteps:     10,
				MinSteps:     3,
				CustomPrompt: "Custom prompt with sufficient length that meets minimum requirements",
			},
			wantErr: false,
		},
		{
			name: "negative max steps",
			config: &PlanningConfig{
				MaxSteps: -1,
			},
			wantErr: true,
		},
		{
			name: "max steps less than min steps",
			config: &PlanningConfig{
				MaxSteps: 5,
				MinSteps: 10,
			},
			wantErr: true,
		},
		{
			name: "negative min steps",
			config: &PlanningConfig{
				MinSteps: -1,
			},
			wantErr: true,
		},
		{
			name: "custom prompt too short",
			config: &PlanningConfig{
				CustomPrompt: "short",
			},
			wantErr: true,
		},
		{
			name: "multiple validation errors",
			config: &PlanningConfig{
				MaxSteps:     -1,
				MinSteps:     -1,
				CustomPrompt: "short",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planning := &Planning{config: tt.config}

			err := planning.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Planning.Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Planning.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPlanning_Validate_ErrorMessages(t *testing.T) {
	// Test specific error messages
	planning := &Planning{
		config: &PlanningConfig{
			MaxSteps:     -1,
			MinSteps:     -1,
			CustomPrompt: "short",
		},
	}

	err := planning.Validate()

	if err == nil {
		t.Fatal("Planning.Validate() expected error, got nil")
	}

	// Check that multiple errors are joined
	errStr := err.Error()
	if !containsPlanning(errStr, "max steps cannot be negative: -1") {
		t.Errorf("Planning.Validate() error should contain max steps error")
	}

	if !containsPlanning(errStr, "min steps cannot be negative: -1") {
		t.Errorf("Planning.Validate() error should contain min steps error")
	}

	if !containsPlanning(errStr, "custom prompt too short") {
		t.Errorf("Planning.Validate() error should contain prompt length error")
	}
}

func TestPlanning_GetMinSteps(t *testing.T) {
	tests := []struct {
		name   string
		config *PlanningConfig
		want   int
	}{
		{
			name:   "nil config",
			config: nil,
			want:   PlanningMinSteps,
		},
		{
			name: "zero min steps",
			config: &PlanningConfig{
				MinSteps: 0,
			},
			want: PlanningMinSteps,
		},
		{
			name: "custom min steps",
			config: &PlanningConfig{
				MinSteps: 5,
			},
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planning := &Planning{config: tt.config}

			minSteps := planning.getMinSteps()

			if minSteps != tt.want {
				t.Errorf("Planning.getMinSteps() = %d, want %d", minSteps, tt.want)
			}
		})
	}
}

func TestPlanning_DefaultPrompt(t *testing.T) {
	planning := NewPlanning()
	prompt := planning.SystemPrompt()

	// Check that default prompt contains key elements
	expectedElements := []string{
		"task planning assistant",
		"decompose",
		"executable steps",
		"dependencies",
		"duration",
		"JSON",
		"steps",
	}

	for _, element := range expectedElements {
		if !containsPlanning(prompt, element) {
			t.Errorf("Planning default prompt should contain '%s'", element)
		}
	}
}

func TestPlanning_DefaultTools(t *testing.T) {
	planning := NewPlanning()
	tools := planning.AllowedTools()

	expectedTools := []string{"get_context", "file_search", "git_context"}

	if len(tools) != len(expectedTools) {
		t.Errorf("Planning.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("Planning.AllowedTools() [%d] = %s, want %s", i, tool, expectedTools[i])
		}
	}
}

func TestPlanning_DefaultMaxTokens(t *testing.T) {
	planning := NewPlanning()
	maxTokens := planning.MaxTokens()

	if maxTokens != PlanningMaxTokens {
		t.Errorf("Planning.MaxTokens() = %d, want %d", maxTokens, PlanningMaxTokens)
	}
}

func TestPlanning_Concurrency(t *testing.T) {
	planning := NewPlanning()

	// Test concurrent access to methods
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			// These methods should be safe for concurrent access
			planning.Name()
			planning.SystemPrompt()
			planning.AllowedTools()
			planning.MaxTokens()
			planning.Validate()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPlanning_Constants(t *testing.T) {
	// Test that constants are properly defined
	if PlanningMaxTokens <= 0 {
		t.Errorf("PlanningMaxTokens = %d, want > 0", PlanningMaxTokens)
	}

	if PlanningMinSteps <= 0 {
		t.Errorf("PlanningMinSteps = %d, want > 0", PlanningMinSteps)
	}

	if PlanningMaxSteps <= 0 {
		t.Errorf("PlanningMaxSteps = %d, want > 0", PlanningMaxSteps)
	}

	if PlanningMinEstimate <= 0 {
		t.Errorf("PlanningMinEstimate = %d, want > 0", PlanningMinEstimate)
	}

	// Test relationships
	if PlanningMinSteps > PlanningMaxSteps {
		t.Errorf("PlanningMinSteps (%d) > PlanningMaxSteps (%d)", PlanningMinSteps, PlanningMaxSteps)
	}
}

// Helper function to check if a string contains a substring
func containsPlanning(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && containsPlanning(s[1:], substr)
}
