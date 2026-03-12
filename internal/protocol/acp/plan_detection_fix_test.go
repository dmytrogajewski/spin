package acp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDetectPlanFromOutput_ExitsPlanSection tests that detectPlanFromOutput
// properly exits plan sections when encountering non-plan content.
func TestDetectPlanFromOutput_ExitsPlanSection(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		expectedCount int
		description   string
	}{
		{
			name: "Plan followed by regular text",
			output: `Plan:
1. First step
2. Second step
3. Third step

This is regular text that explains something.
More explanation here.`,
			expectedCount: 3,
			description:   "Should only detect 3 plan entries, not the explanation text",
		},
		{
			name: "Plan followed by code example",
			output: `Steps:
1. Create file
2. Write code
3. Test code

The code looks like this:
1. func main() {
2.     fmt.Println("hello")
3. }`,
			expectedCount: 3,
			description:   "Should not treat code line numbers as plan entries",
		},
		{
			name: "Multiple disconnected lists",
			output: `Plan:
1. Item A
2. Item B

Some text in between.

Steps:
1. Item C
2. Item D`,
			expectedCount: 4,
			description:   "Should detect both lists with explicit headers",
		},
		{
			name: "Plan with one non-plan line intermixed",
			output: `Steps:
1. First step
(this is a note)
2. Second step
3. Third step`,
			expectedCount: 3,
			description:   "Should tolerate one non-plan line within plan",
		},
		{
			name: "Plan with two non-plan lines exits",
			output: `Steps:
1. First step
Some text
More text
2. This should not be detected`,
			expectedCount: 1,
			description:   "Should exit after 2 consecutive non-plan lines",
		},
		{
			name: "No plan pattern",
			output: `This is just regular text.
It has no numbered lists.
Or bullet points.`,
			expectedCount: 0,
			description:   "Should detect no plan entries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := detectPlanFromOutput(tt.output)
			assert.Equal(t, tt.expectedCount, len(entries), tt.description)

			if len(entries) > 0 {
				t.Logf("Detected entries: %v", entries)
			}
		})
	}
}

// TestDetectPlanFromOutput_RealWorldScenario tests a realistic scenario
// where an LLM describes a project with multiple sections.
func TestDetectPlanFromOutput_RealWorldScenario(t *testing.T) {
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
