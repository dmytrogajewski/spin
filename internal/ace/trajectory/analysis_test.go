package trajectory

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

func TestContainsError(t *testing.T) {
	t.Run("detects lowercase error keyword", func(t *testing.T) {
		content := "command failed with error"
		if !containsError(content) {
			t.Error("expected true for content with 'error'")
		}
	})

	t.Run("detects failed keyword", func(t *testing.T) {
		content := "operation failed"
		if !containsError(content) {
			t.Error("expected true for content with 'failed'")
		}
	})

	t.Run("detects exception keyword", func(t *testing.T) {
		content := "NullPointerException occurred"
		if !containsError(content) {
			t.Error("expected true for content with 'exception'")
		}
	})

	t.Run("detects panic keyword", func(t *testing.T) {
		content := "panic: runtime error"
		if !containsError(content) {
			t.Error("expected true for content with 'panic'")
		}
	})

	t.Run("detects fatal keyword", func(t *testing.T) {
		content := "fatal: cannot continue"
		if !containsError(content) {
			t.Error("expected true for content with 'fatal'")
		}
	})

	t.Run("case insensitive detection", func(t *testing.T) {
		content := "ERROR occurred"
		if !containsError(content) {
			t.Error("expected true for uppercase 'ERROR'")
		}
	})

	t.Run("returns false when no error keywords", func(t *testing.T) {
		content := "everything is working fine"
		if containsError(content) {
			t.Error("expected false for content without error keywords")
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		content := ""
		if containsError(content) {
			t.Error("expected false for empty string")
		}
	})
}

func TestHasRecentError(t *testing.T) {
	t.Run("detects error in recent steps", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{StepNumber: 0, Content: "starting task"},
			{StepNumber: 1, Content: "error occurred"},
		})

		if !ctx.HasRecentError(2) {
			t.Error("expected true when error in last 2 steps")
		}
	})

	t.Run("returns false when error outside lookback window", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{StepNumber: 0, Content: "error at start"},
			{StepNumber: 1, Content: "step 2"},
			{StepNumber: 2, Content: "step 3"},
		})

		if ctx.HasRecentError(2) {
			t.Error("expected false when error outside last 2 steps")
		}
	})

	t.Run("returns false when no errors", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{StepNumber: 0, Content: "all good"},
			{StepNumber: 1, Content: "working fine"},
		})

		if ctx.HasRecentError(2) {
			t.Error("expected false when no errors")
		}
	})

	t.Run("handles empty trajectory", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")

		if ctx.HasRecentError(2) {
			t.Error("expected false for empty trajectory")
		}
	})

	t.Run("checks all steps when lookback is 0", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{StepNumber: 0, Content: "error at start"},
			{StepNumber: 1, Content: "step 2"},
			{StepNumber: 2, Content: "step 3"},
		})

		if !ctx.HasRecentError(0) {
			t.Error("expected true when lookback=0 checks all steps")
		}
	})

	t.Run("checks all steps when lookback exceeds length", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{StepNumber: 0, Content: "error here"},
		})

		if !ctx.HasRecentError(100) {
			t.Error("expected true when lookback > steps length")
		}
	})
}

func TestExtractErrorPatterns(t *testing.T) {
	t.Run("extracts error content from steps", func(t *testing.T) {
		steps := []generator.TrajectoryStep{
			{StepNumber: 0, Content: "starting"},
			{StepNumber: 1, Content: "error: file not found"},
		}

		patterns := ExtractErrorPatterns(steps, 0)
		if len(patterns) != 1 {
			t.Fatalf("expected 1 pattern, got %d", len(patterns))
		}
		if patterns[0] != "error: file not found" {
			t.Errorf("expected 'error: file not found', got %q", patterns[0])
		}
	})

	t.Run("returns empty slice when no errors", func(t *testing.T) {
		steps := []generator.TrajectoryStep{
			{StepNumber: 0, Content: "all good"},
		}

		patterns := ExtractErrorPatterns(steps, 0)
		if len(patterns) != 0 {
			t.Errorf("expected empty slice, got %d patterns", len(patterns))
		}
	})

	t.Run("respects lookback window", func(t *testing.T) {
		steps := []generator.TrajectoryStep{
			{StepNumber: 0, Content: "error at start"},
			{StepNumber: 1, Content: "step 2"},
			{StepNumber: 2, Content: "failed at end"},
		}

		patterns := ExtractErrorPatterns(steps, 2)
		if len(patterns) != 1 {
			t.Fatalf("expected 1 pattern (within lookback), got %d", len(patterns))
		}
		if patterns[0] != "failed at end" {
			t.Errorf("expected 'failed at end', got %q", patterns[0])
		}
	})
}

func TestExtractToolName(t *testing.T) {
	t.Run("extracts tool name from content", func(t *testing.T) {
		content := "Calling tool: bash"
		name := extractToolName(content)
		if name != "bash" {
			t.Errorf("expected 'bash', got %q", name)
		}
	})

	t.Run("handles Tool: prefix", func(t *testing.T) {
		content := "Tool: read /file"
		name := extractToolName(content)
		if name != "read" {
			t.Errorf("expected 'read', got %q", name)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		content := "TOOL: grep"
		name := extractToolName(content)
		if name != "grep" {
			t.Errorf("expected 'grep', got %q", name)
		}
	})

	t.Run("returns empty when no tool found", func(t *testing.T) {
		content := "just some text"
		name := extractToolName(content)
		if name != "" {
			t.Errorf("expected empty string, got %q", name)
		}
	})
}

func TestGetRecentTools(t *testing.T) {
	t.Run("extracts unique tool names from recent steps", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{StepNumber: 0, Type: "tool_call", Content: "Tool: bash"},
			{StepNumber: 1, Type: "tool_result", Content: "output"},
			{StepNumber: 2, Type: "tool_call", Content: "Tool: read"},
		})

		tools := ctx.GetRecentTools(0)
		if len(tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(tools))
		}
		if tools[0] != "bash" || tools[1] != "read" {
			t.Errorf("expected [bash read], got %v", tools)
		}
	})

	t.Run("deduplicates repeated tools", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{Content: "Tool: bash"},
			{Content: "Tool: bash"},
			{Content: "Tool: read"},
		})

		tools := ctx.GetRecentTools(0)
		if len(tools) != 2 {
			t.Fatalf("expected 2 unique tools, got %d: %v", len(tools), tools)
		}
	})

	t.Run("respects lookback window", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{Content: "Tool: old"},
			{Content: "Tool: grep"},
			{Content: "Tool: read"},
		})

		tools := ctx.GetRecentTools(2)
		if len(tools) != 2 {
			t.Fatalf("expected 2 tools in lookback, got %d: %v", len(tools), tools)
		}
	})

	t.Run("returns empty slice when no tools", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.AppendSteps([]generator.TrajectoryStep{
			{Content: "no tools here"},
		})

		tools := ctx.GetRecentTools(0)
		if len(tools) != 0 {
			t.Errorf("expected empty slice, got %v", tools)
		}
	})
}

func TestExtractConcepts(t *testing.T) {
	t.Run("extracts capitalized words", func(t *testing.T) {
		steps := []generator.TrajectoryStep{
			{Content: "Analyzing Dockerfile for optimization"},
			{Content: "Using BuildKit caching"},
		}

		concepts := ExtractConcepts(steps, 0)
		// Should extract: Dockerfile, BuildKit
		if len(concepts) < 2 {
			t.Fatalf("expected at least 2 concepts, got %d: %v", len(concepts), concepts)
		}

		hasDockerfile := false
		hasBuildKit := false
		for _, c := range concepts {
			if c == "Dockerfile" {
				hasDockerfile = true
			}
			if c == "BuildKit" {
				hasBuildKit = true
			}
		}

		if !hasDockerfile || !hasBuildKit {
			t.Errorf("expected Dockerfile and BuildKit in concepts, got %v", concepts)
		}
	})

	t.Run("returns empty for common words only", func(t *testing.T) {
		steps := []generator.TrajectoryStep{
			{Content: "the and or but"},
		}

		concepts := ExtractConcepts(steps, 0)
		if len(concepts) != 0 {
			t.Errorf("expected no concepts from common words, got %v", concepts)
		}
	})

	t.Run("respects lookback window", func(t *testing.T) {
		steps := []generator.TrajectoryStep{
			{Content: "OldConcept here"},
			{Content: "NewConcept here"},
		}

		concepts := ExtractConcepts(steps, 1)
		if len(concepts) != 1 {
			t.Fatalf("expected 1 concept in lookback, got %d: %v", len(concepts), concepts)
		}
		if concepts[0] != "NewConcept" {
			t.Errorf("expected NewConcept, got %q", concepts[0])
		}
	})

	t.Run("handles empty steps", func(t *testing.T) {
		steps := []generator.TrajectoryStep{}

		concepts := ExtractConcepts(steps, 0)
		if len(concepts) != 0 {
			t.Errorf("expected empty slice for empty steps, got %v", concepts)
		}
	})
}
