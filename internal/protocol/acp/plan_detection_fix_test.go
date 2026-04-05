package acp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDetectPlanFromOutput_ExitsPlanSection tests that detectPlanFromOutput
// properly exits plan sections when encountering non-plan content.
func TestDetectPlanFromOutput_ExitsPlanSection(t *testing.T) {
	t.Parallel()

	t.Run("Plan followed by regular text", func(t *testing.T) {
		t.Parallel()

		entries := detectPlanFromOutput(
			"Plan:\n1. First step\n2. Second step\n3. Third step\n\n" +
				"This is regular text that explains something.\nMore explanation here.")
		assert.Len(t, entries, 3, "Should only detect 3 plan entries, not the explanation text")
	})

	t.Run("Plan followed by code example", func(t *testing.T) {
		t.Parallel()

		entries := detectPlanFromOutput(
			"Steps:\n1. Create file\n2. Write code\n3. Test code\n\n" +
				"The code looks like this:\n1. func main() {\n2.     fmt.Println(\"hello\")\n3. }")
		assert.Len(t, entries, 3, "Should not treat code line numbers as plan entries")
	})

	t.Run("Multiple disconnected lists", func(t *testing.T) {
		t.Parallel()

		entries := detectPlanFromOutput("Plan:\n1. Item A\n2. Item B\n\nSome text in between.\n\nSteps:\n1. Item C\n2. Item D")
		assert.Len(t, entries, 4, "Should detect both lists with explicit headers")
	})

	t.Run("Plan with one non-plan line intermixed", func(t *testing.T) {
		t.Parallel()

		entries := detectPlanFromOutput("Steps:\n1. First step\n(this is a note)\n2. Second step\n3. Third step")
		assert.Len(t, entries, 3, "Should tolerate one non-plan line within plan")
	})

	t.Run("Plan with two non-plan lines exits", func(t *testing.T) {
		t.Parallel()

		entries := detectPlanFromOutput("Steps:\n1. First step\nSome text\nMore text\n2. This should not be detected")
		assert.Len(t, entries, 1, "Should exit after 2 consecutive non-plan lines")
	})

	t.Run("No plan pattern", func(t *testing.T) {
		t.Parallel()

		entries := detectPlanFromOutput("This is just regular text.\nIt has no numbered lists.\nOr bullet points.")
		assert.Empty(t, entries, "Should detect no plan entries")
	})
}

// TestDetectPlanFromOutput_RealWorldScenario tests a realistic scenario
// where an LLM describes a project with multiple sections.
func TestDetectPlanFromOutput_RealWorldScenario(t *testing.T) {
	t.Parallel()

	output := `I'll help you review this project. Here's my analysis:

## Project Structure

1. Model Architecture
2. Data Pipeline  
3. Inference System
4. Training Infrastructure

## Key Components

### 1. Model Architecture

Uses Llama-3.2-1B as the base language model with PEFT.
Implements LoRA adapters for parameter-efficient fine-tuning.

### 2. Data Pipeline

Point cloud data processed through GeneralNpzDataset.
Supports both in-distribution and out-of-distribution clouds.

Let me analyze the codebase in detail.`

	entries := detectPlanFromOutput(output)

	// Should detect the 4-item numbered list, but NOT treat
	// the "### 1. Model Architecture" or "### 2. Data Pipeline" headings as plan entries.
	assert.LessOrEqual(t, len(entries), 4, "Should detect at most 4 plan entries from the numbered list")

	if len(entries) > 4 {
		t.Logf("ERROR: Detected too many entries (%d). Entries:", len(entries))

		for i, entry := range entries {
			t.Logf("  %d: %s", i+1, entry.Content)
		}
	}
}
