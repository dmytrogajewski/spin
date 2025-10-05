package task

import (
	"strings"
	"testing"
)

// TestNewPlanning tests the default constructor
func TestNewPlanning(t *testing.T) {
	p := NewPlanning()
	if p == nil {
		t.Fatal("NewPlanning() returned nil")
	}

	if p.config != nil {
		t.Error("NewPlanning() should have nil config for defaults")
	}
}

// TestNewPlanningWithConfig tests constructor with custom config
func TestNewPlanningWithConfig(t *testing.T) {
	config := &PlanningConfig{
		MaxSteps: 50,
		MinSteps: 3,
	}

	p := NewPlanningWithConfig(config)
	if p == nil {
		t.Fatal("NewPlanningWithConfig() returned nil")
	}

	if p.config != config {
		t.Error("NewPlanningWithConfig() did not set config")
	}
}

// TestNewPlanningWithConfig_Nil tests constructor with nil config
func TestNewPlanningWithConfig_Nil(t *testing.T) {
	p := NewPlanningWithConfig(nil)
	if p == nil {
		t.Fatal("NewPlanningWithConfig(nil) returned nil")
	}

	if p.config != nil {
		t.Error("NewPlanningWithConfig(nil) should have nil config")
	}
}

// TestPlanning_Name tests the Name() method
func TestPlanning_Name(t *testing.T) {
	tests := []struct {
		name   string
		config *PlanningConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &PlanningConfig{
				MaxSteps: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlanningWithConfig(tt.config)
			if got := p.Name(); got != "planning" {
				t.Errorf("Name() = %q, want %q", got, "planning")
			}
		})
	}
}

// TestPlanning_SystemPrompt tests the SystemPrompt() method
func TestPlanning_SystemPrompt(t *testing.T) {
	tests := []struct {
		name         string
		config       *PlanningConfig
		wantContains string
		minLength    int
	}{
		{
			name:         "default prompt",
			config:       nil,
			wantContains: "task planning assistant",
			minLength:    500,
		},
		{
			name: "custom prompt",
			config: &PlanningConfig{
				CustomPrompt: "Custom planning prompt for testing purposes that is long enough to meet minimum",
			},
			wantContains: "Custom planning prompt",
			minLength:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlanningWithConfig(tt.config)
			prompt := p.SystemPrompt()

			if len(prompt) < tt.minLength {
				t.Errorf("SystemPrompt() length = %d, want >= %d", len(prompt), tt.minLength)
			}

			if !strings.Contains(prompt, tt.wantContains) {
				t.Errorf("SystemPrompt() does not contain %q", tt.wantContains)
			}
		})
	}
}

// TestPlanning_AllowedTools tests the AllowedTools() method
func TestPlanning_AllowedTools(t *testing.T) {
	p := NewPlanning()
	tools := p.AllowedTools()

	// Planning mode should have limited tools
	expectedTools := []string{"get_context", "read_file", "list_dir"}

	if len(tools) != len(expectedTools) {
		t.Errorf("AllowedTools() returned %d tools, want %d", len(tools), len(expectedTools))
	}

	// Check all expected tools are present
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("AllowedTools() missing expected tool %q", expected)
		}
	}

	// Should NOT include execution tools
	dangerousTools := []string{"shell", "write_file", "git_commit"}
	for _, dangerous := range dangerousTools {
		if toolMap[dangerous] {
			t.Errorf("AllowedTools() should not include %q in planning mode", dangerous)
		}
	}
}

// TestPlanning_MaxTokens tests the MaxTokens() method
func TestPlanning_MaxTokens(t *testing.T) {
	tests := []struct {
		name   string
		config *PlanningConfig
		want   int
	}{
		{
			name:   "default config",
			config: nil,
			want:   PlanningMaxTokens,
		},
		{
			name:   "custom config (ignored - always uses constant)",
			config: &PlanningConfig{MaxSteps: 50},
			want:   PlanningMaxTokens,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlanningWithConfig(tt.config)
			if got := p.MaxTokens(); got != tt.want {
				t.Errorf("MaxTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestPlanning_Validate tests the Validate() method
func TestPlanning_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *PlanningConfig
		wantErr bool
	}{
		{
			name:    "nil config is valid",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &PlanningConfig{
				MaxSteps: 50,
				MinSteps: 3,
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
			name: "negative min steps",
			config: &PlanningConfig{
				MinSteps: -1,
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
			name: "custom prompt too short",
			config: &PlanningConfig{
				CustomPrompt: "short",
			},
			wantErr: true,
		},
		{
			name: "custom prompt valid",
			config: &PlanningConfig{
				CustomPrompt: "This is a valid custom planning prompt that meets the minimum length requirement for testing",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlanningWithConfig(tt.config)
			err := p.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPlanning_DefaultValues tests that default values are correct
func TestPlanning_DefaultValues(t *testing.T) {
	p := NewPlanning()

	if p.Name() != "planning" {
		t.Errorf("Default Name() = %q, want %q", p.Name(), "planning")
	}

	if p.MaxTokens() != PlanningMaxTokens {
		t.Errorf("Default MaxTokens() = %d, want %d", p.MaxTokens(), PlanningMaxTokens)
	}

	tools := p.AllowedTools()
	if len(tools) == 0 {
		t.Error("Default AllowedTools() should not be empty")
	}

	prompt := p.SystemPrompt()
	if len(prompt) == 0 {
		t.Error("Default SystemPrompt() should not be empty")
	}

	if err := p.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

// TestPlanning_PromptFormat tests that the planning prompt has required elements
func TestPlanning_PromptFormat(t *testing.T) {
	p := NewPlanning()
	prompt := p.SystemPrompt()

	requiredElements := []string{
		"JSON",
		"steps",
		"id",
		"description",
		"action",
		"depends_on",
		"estimated_minutes",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(prompt, elem) {
			t.Errorf("SystemPrompt() missing required element %q", elem)
		}
	}
}

func TestPlanning_getMaxSteps(t *testing.T) {
	tests := []struct {
		name   string
		config *PlanningConfig
		want   int
	}{
		{
			name:   "nil config uses default",
			config: nil,
			want:   100,
		},
		{
			name:   "config with MaxSteps set",
			config: &PlanningConfig{MaxSteps: 50},
			want:   50,
		},
		{
			name:   "config with zero MaxSteps uses default",
			config: &PlanningConfig{MaxSteps: 0},
			want:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Planning{config: tt.config}
			got := p.getMaxSteps()
			if got != tt.want {
				t.Errorf("getMaxSteps() = %d, want %d", got, tt.want)
			}
		})
	}
}
