package curator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// TestNewPromptBuilder tests creating a new prompt builder.
func TestNewPromptBuilder(t *testing.T) {
	builder := NewPromptBuilder()
	require.NotNil(t, builder)
}

// TestBuildCurationPrompt tests building a curation prompt.
func TestBuildCurationPrompt(t *testing.T) {
	builder := NewPromptBuilder()

	req := CurationRequest{
		TaskContext:     "Fix null pointer exception",
		CurrentPlaybook: "[B001] Always check for nil\n",
		Reflection:      "The code failed because nil check was missing",
	}

	prompt := builder.BuildCurationPrompt(req)

	require.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Fix null pointer exception")
	assert.Contains(t, prompt, "Always check for nil")
	assert.Contains(t, prompt, "nil check was missing")
	assert.Contains(t, prompt, "reasoning")
	assert.Contains(t, prompt, "operations")
}

// TestBuildRefinementPrompt tests building a refinement prompt.
func TestBuildRefinementPrompt(t *testing.T) {
	builder := NewPromptBuilder()

	stats := PlaybookStats{
		TotalBullets:     100,
		AvgHelpfulCount:  5.2,
		AvgHarmfulCount:  1.3,
		LowUtilityCount:  10,
		HighUtilityCount: 20,
	}

	prompt := builder.BuildRefinementPrompt("[B001] Check nil\n[B002] Validate input\n", stats)

	require.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "100")
	assert.Contains(t, prompt, "5.2")
	assert.Contains(t, prompt, "reasoning")
	assert.Contains(t, prompt, "operations")
}

// TestFormatReflectionForCurator tests formatting insights for curator.
func TestFormatReflectionForCurator(t *testing.T) {
	insight := &reflector.Insight{
		Content:    "Always validate user input before processing",
		Confidence: 0.95,
		Category:   reflector.CategorySuccessPattern,
		Evidence:   []string{"prevented SQL injection", "caught invalid data"},
	}

	formatted := FormatReflectionForCurator(insight)

	require.NotEmpty(t, formatted)
	assert.Contains(t, formatted, "Always validate user input")
	assert.Contains(t, formatted, "0.95")
	assert.Contains(t, formatted, "success_pattern")
	assert.Contains(t, formatted, "prevented SQL injection")
}
