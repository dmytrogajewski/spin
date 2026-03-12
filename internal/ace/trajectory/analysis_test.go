package trajectory

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

func TestContainsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"detects lowercase error keyword", "command failed with error", true},
		{"detects failed keyword", "operation failed", true},
		{"detects exception keyword", "NullPointerException occurred", true},
		{"detects panic keyword", "panic: runtime error", true},
		{"detects fatal keyword", "fatal: cannot continue", true},
		{"case insensitive detection", "ERROR occurred", true},
		{"returns false when no error keywords", "everything is working fine", false},
		{"handles empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := containsError(tt.content); got != tt.want {
				t.Errorf("containsError(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

// hasRecentErrorCase defines a test case for HasRecentError.
type hasRecentErrorCase struct {
	name     string
	steps    []generator.TrajectoryStep
	lookback int
	want     bool
}

// hasRecentErrorCases returns test cases for HasRecentError.
func hasRecentErrorCases() []hasRecentErrorCase {
	return []hasRecentErrorCase{
		{
			name:     "detects error in recent steps",
			steps:    []generator.TrajectoryStep{{StepNumber: 0, Content: "starting task"}, {StepNumber: 1, Content: "error occurred"}},
			lookback: 2, want: true,
		},
		{
			name:     "returns false when error outside lookback window",
			steps: []generator.TrajectoryStep{
				{StepNumber: 0, Content: "error at start"},
				{StepNumber: 1, Content: "step 2"},
				{StepNumber: 2, Content: "step 3"},
			},
			lookback: 2, want: false,
		},
		{
			name: "returns false when no errors",
			steps: []generator.TrajectoryStep{
				{StepNumber: 0, Content: "all good"},
				{StepNumber: 1, Content: "working fine"},
			},
			lookback: 2, want: false,
		},
		{name: "handles empty trajectory", lookback: 2, want: false},
		{
			name: "checks all steps when lookback is 0",
			steps: []generator.TrajectoryStep{
				{StepNumber: 0, Content: "error at start"},
				{StepNumber: 1, Content: "step 2"},
				{StepNumber: 2, Content: "step 3"},
			},
			lookback: 0, want: true,
		},
		{
			name:     "checks all steps when lookback exceeds length",
			steps:    []generator.TrajectoryStep{{StepNumber: 0, Content: "error here"}},
			lookback: 100, want: true,
		},
	}
}

func TestHasRecentError(t *testing.T) {
	t.Parallel()

	for _, tt := range hasRecentErrorCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := NewContext("test query")
			if len(tt.steps) > 0 {
				ctx.AppendSteps(tt.steps)
			}

			if got := ctx.HasRecentError(tt.lookback); got != tt.want {
				t.Errorf("HasRecentError(%d) = %v, want %v", tt.lookback, got, tt.want)
			}
		})
	}
}

func TestExtractErrorPatterns(t *testing.T) {
	t.Parallel()

	t.Run("extracts error content from steps", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		steps := []generator.TrajectoryStep{
			{StepNumber: 0, Content: "all good"},
		}

		patterns := ExtractErrorPatterns(steps, 0)
		if len(patterns) != 0 {
			t.Errorf("expected empty slice, got %d patterns", len(patterns))
		}
	})

	t.Run("respects lookback window", func(t *testing.T) {
		t.Parallel()

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
	t.Parallel()

	t.Run("extracts tool name from content", func(t *testing.T) {
		t.Parallel()

		content := "Calling tool: bash"

		name := extractToolName(content)
		if name != "bash" {
			t.Errorf("expected 'bash', got %q", name)
		}
	})

	t.Run("handles Tool: prefix", func(t *testing.T) {
		t.Parallel()

		content := "Tool: read /file"

		name := extractToolName(content)
		if name != "read" {
			t.Errorf("expected 'read', got %q", name)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		t.Parallel()

		content := "TOOL: grep"

		name := extractToolName(content)
		if name != "grep" {
			t.Errorf("expected 'grep', got %q", name)
		}
	})

	t.Run("returns empty when no tool found", func(t *testing.T) {
		t.Parallel()

		content := "just some text"

		name := extractToolName(content)
		if name != "" {
			t.Errorf("expected empty string, got %q", name)
		}
	})
}

func TestGetRecentTools(t *testing.T) {
	t.Parallel()

	t.Run("extracts unique tool names from recent steps", func(t *testing.T) {
		t.Parallel()

		ctx := NewContext("test")
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
		t.Parallel()

		ctx := NewContext("test")
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
		t.Parallel()

		ctx := NewContext("test")
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
		t.Parallel()

		ctx := NewContext("test")
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
	t.Parallel()

	t.Run("extracts capitalized words", func(t *testing.T) {
		t.Parallel()

		steps := []generator.TrajectoryStep{
			{Content: "Analyzing Dockerfile for optimization"},
			{Content: "Using BuildKit caching"},
		}

		concepts := ExtractConcepts(steps, 0)
		requireConceptsContain(t, concepts, "Dockerfile", "BuildKit")
	})

	t.Run("returns empty for common words only", func(t *testing.T) {
		t.Parallel()

		concepts := ExtractConcepts([]generator.TrajectoryStep{{Content: "the and or but"}}, 0)
		if len(concepts) != 0 {
			t.Errorf("expected no concepts from common words, got %v", concepts)
		}
	})

	t.Run("respects lookback window", func(t *testing.T) {
		t.Parallel()

		steps := []generator.TrajectoryStep{
			{Content: "OldConcept here"},
			{Content: "NewConcept here"},
		}

		concepts := ExtractConcepts(steps, 1)
		if len(concepts) != 1 || concepts[0] != "NewConcept" {
			t.Errorf("expected [NewConcept], got %v", concepts)
		}
	})

	t.Run("handles empty steps", func(t *testing.T) {
		t.Parallel()

		concepts := ExtractConcepts([]generator.TrajectoryStep{}, 0)
		if len(concepts) != 0 {
			t.Errorf("expected empty slice, got %v", concepts)
		}
	})
}

// requireConceptsContain checks that all expected concepts are present.
func requireConceptsContain(t *testing.T, concepts []string, expected ...string) {
	t.Helper()

	conceptSet := make(map[string]bool, len(concepts))
	for _, c := range concepts {
		conceptSet[c] = true
	}

	for _, exp := range expected {
		if !conceptSet[exp] {
			t.Errorf("expected concept %q not found in %v", exp, concepts)
		}
	}
}
