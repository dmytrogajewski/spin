package cycle

import (
	"testing"
)

func TestNewPatternDetector(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	if detector == nil {
		t.Fatal("NewPatternDetector() returned nil")
	}

	if detector.config != config {
		t.Errorf("NewPatternDetector() config = %v, want %v", detector.config, config)
	}
}

func TestPatternDetector_AnalyzePatterns(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	// Test with empty snapshots.
	results := detector.analyzeInternal([]Snapshot{})
	if len(results) != 0 {
		t.Errorf("PatternDetector.AnalyzePatterns() with empty snapshots, got %d results, want 0", len(results))
	}

	// Test with insufficient snapshots.
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
	}

	results = detector.analyzeInternal(snapshots)
	if len(results) != 0 {
		t.Errorf("PatternDetector.AnalyzePatterns() with insufficient snapshots, got %d results, want 0", len(results))
	}
}

func TestPatternDetector_AnalyzePatterns_RepeatedPhrase(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	// Test with repeated phrases.
	snapshots := []Snapshot{
		{Turn: 1, Response: "This is a test response with repeated phrase", ToolCalls: []string{}, Error: ""},
		{Turn: 2, Response: "Another response with repeated phrase", ToolCalls: []string{}, Error: ""},
		{Turn: 3, Response: "Third response with repeated phrase", ToolCalls: []string{}, Error: ""},
	}

	results := detector.analyzeInternal(snapshots)

	// Should detect repeated phrase pattern.
	found := false

	for _, result := range results {
		if result.Type == PatternRepeatedPhrase {
			found = true

			if result.Confidence <= 0.0 {
				t.Errorf("PatternDetector.AnalyzePatterns() repeated phrase confidence = %f, want > 0.0", result.Confidence)
			}

			if result.Details == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() repeated phrase details should not be empty")
			}

			if result.Suggestion == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() repeated phrase suggestion should not be empty")
			}

			break
		}
	}

	if !found {
		t.Errorf("PatternDetector.AnalyzePatterns() should detect repeated phrase pattern")
	}
}

func TestPatternDetector_AnalyzePatterns_CircularReasoning(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	// Test with circular reasoning indicators.
	snapshots := []Snapshot{
		{Turn: 1, Response: "As I mentioned before, this is important", ToolCalls: []string{}, Error: ""},
		{Turn: 2, Response: "Going back to the previous point", ToolCalls: []string{}, Error: ""},
		{Turn: 3, Response: "As we saw earlier, this pattern continues", ToolCalls: []string{}, Error: ""},
	}

	results := detector.analyzeInternal(snapshots)

	// Should detect circular reasoning pattern.
	found := false

	for _, result := range results {
		if result.Type == PatternCircularReasoning {
			found = true

			if result.Confidence != 0.7 {
				t.Errorf("PatternDetector.AnalyzePatterns() circular reasoning confidence = %f, want 0.7", result.Confidence)
			}

			if result.Details == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() circular reasoning details should not be empty")
			}

			if result.Suggestion == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() circular reasoning suggestion should not be empty")
			}

			break
		}
	}

	if !found {
		t.Errorf("PatternDetector.AnalyzePatterns() should detect circular reasoning pattern")
	}
}

func TestPatternDetector_AnalyzePatterns_ToolStuck(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	// Test with stuck tool usage.
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 3, Response: "Third response", ToolCalls: []string{"tool1"}, Error: ""},
	}

	results := detector.analyzeInternal(snapshots)

	// Should detect tool stuck pattern.
	found := false

	for _, result := range results {
		if result.Type == PatternToolStuck {
			found = true

			if result.Confidence <= 0.0 {
				t.Errorf("PatternDetector.AnalyzePatterns() tool stuck confidence = %f, want > 0.0", result.Confidence)
			}

			if result.Details == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() tool stuck details should not be empty")
			}

			if result.Suggestion == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() tool stuck suggestion should not be empty")
			}

			break
		}
	}

	if !found {
		t.Errorf("PatternDetector.AnalyzePatterns() should detect tool stuck pattern")
	}
}

func TestPatternDetector_AnalyzePatterns_ErrorLoop(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	// Test with error loop.
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{}, Error: "test error"},
		{Turn: 2, Response: "Second response", ToolCalls: []string{}, Error: "test error"},
	}

	results := detector.analyzeInternal(snapshots)
	result := findPattern(results, PatternErrorLoop)

	if result == nil {
		t.Fatal("PatternDetector.AnalyzePatterns() should detect error loop pattern")
	}

	assertPatternResult(t, *result, "error loop")

	if len(result.AffectedTurns) == 0 {
		t.Errorf("PatternDetector.AnalyzePatterns() error loop affected turns should not be empty")
	}
}

func TestPatternDetector_AnalyzePatterns_OscillatingTools(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	// Test with oscillating tools.
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: ""},
		{Turn: 3, Response: "Third response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 4, Response: "Fourth response", ToolCalls: []string{"tool2"}, Error: ""},
	}

	results := detector.analyzeInternal(snapshots)

	// Should detect oscillating tools pattern.
	found := false

	for _, result := range results {
		if result.Type == PatternOscillatingTools {
			found = true

			if result.Confidence != 0.8 {
				t.Errorf("PatternDetector.AnalyzePatterns() oscillating tools confidence = %f, want 0.8", result.Confidence)
			}

			if result.Details == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() oscillating tools details should not be empty")
			}

			if result.Suggestion == "" {
				t.Errorf("PatternDetector.AnalyzePatterns() oscillating tools suggestion should not be empty")
			}

			break
		}
	}

	if !found {
		t.Errorf("PatternDetector.AnalyzePatterns() should detect oscillating tools pattern")
	}
}

func TestPatternType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern PatternType
		want    string
	}{
		{PatternNone, "none"},
		{PatternRepeatedPhrase, "repeated_phrase"},
		{PatternCircularReasoning, "circular_reasoning"},
		{PatternToolStuck, "tool_stuck"},
		{PatternErrorLoop, "error_loop"},
		{PatternOscillatingTools, "oscillating_tools"},
		{PatternType(999), "none"}, {PatternType(999), "none"}, // Unknown pattern type.
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got := tt.pattern.String()
			if got != tt.want {
				t.Errorf("PatternType.String() = %s, want %s", got, tt.want)
			}
		})
	}
}

// findPattern returns the first PatternResult matching the given type, or nil.
func findPattern(results []PatternResult, pt PatternType) *PatternResult {
	for i := range results {
		if results[i].Type == pt {
			return &results[i]
		}
	}

	return nil
}

// assertPatternResult validates common fields of a pattern result.
func assertPatternResult(t *testing.T, result PatternResult, label string) {
	t.Helper()

	if result.Confidence <= 0.0 {
		t.Errorf("PatternDetector.AnalyzePatterns() %s confidence = %f, want > 0.0", label, result.Confidence)
	}

	if result.Details == "" {
		t.Errorf("PatternDetector.AnalyzePatterns() %s details should not be empty", label)
	}

	if result.Suggestion == "" {
		t.Errorf("PatternDetector.AnalyzePatterns() %s suggestion should not be empty", label)
	}
}

func TestPatternDetector_DetectCircularReasoning(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	tests := []struct {
		name      string
		snapshots []Snapshot
		want      bool
	}{
		{
			name: "no circular reasoning",
			snapshots: []Snapshot{
				{Turn: 1, Response: "This is a normal response", ToolCalls: []string{}, Error: ""},
				{Turn: 2, Response: "Another normal response", ToolCalls: []string{}, Error: ""},
			},
			want: false,
		},
		{
			name: "circular reasoning detected",
			snapshots: []Snapshot{
				{Turn: 1, Response: "As I mentioned before", ToolCalls: []string{}, Error: ""},
				{Turn: 2, Response: "Going back to the previous point", ToolCalls: []string{}, Error: ""},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := detector.detectCircularReasoning(tt.snapshots)
			if got != tt.want {
				t.Errorf("PatternDetector.detectCircularReasoning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPatternDetector_DetectOscillatingTools(t *testing.T) {
	t.Parallel()

	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewPatternDetector(config)

	tests := []struct {
		name      string
		snapshots []Snapshot
		want      bool
	}{
		{
			name: "no oscillation",
			snapshots: []Snapshot{
				{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
				{Turn: 2, Response: "Second response", ToolCalls: []string{"tool1"}, Error: ""},
			},
			want: false,
		},
		{
			name: "oscillation detected",
			snapshots: []Snapshot{
				{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
				{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: ""},
				{Turn: 3, Response: "Third response", ToolCalls: []string{"tool1"}, Error: ""},
				{Turn: 4, Response: "Fourth response", ToolCalls: []string{"tool2"}, Error: ""},
			},
			want: true,
		},
		{
			name: "insufficient snapshots",
			snapshots: []Snapshot{
				{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
				{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: ""},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := detector.detectOscillatingTools(tt.snapshots)
			if got != tt.want {
				t.Errorf("PatternDetector.detectOscillatingTools() = %v, want %v", got, tt.want)
			}
		})
	}
}
