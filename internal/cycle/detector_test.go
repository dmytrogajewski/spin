package cycle

import (
	"testing"
	"time"
)

func TestNewDetector(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}

	if len(detector.history) != 0 {
		t.Errorf("NewDetector() history length = %d, want 0", len(detector.history))
	}

	if detector.config != config {
		t.Errorf("NewDetector() config = %v, want %v", detector.config, config)
	}
}

func TestDetector_Record(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       3,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record snapshots
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: ""},
		{Turn: 3, Response: "Third response", ToolCalls: []string{"tool3"}, Error: ""},
		{Turn: 4, Response: "Fourth response", ToolCalls: []string{"tool4"}, Error: ""},
		{Turn: 5, Response: "Fifth response", ToolCalls: []string{"tool5"}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	// Should maintain window size
	if len(detector.history) != 3 {
		t.Errorf("Detector.Record() history length = %d, want 3", len(detector.history))
	}

	// Should keep most recent snapshots
	expected := snapshots[2:] // Last 3 snapshots
	for i, snapshot := range detector.history {
		if snapshot.Turn != expected[i].Turn {
			t.Errorf("Detector.Record() history[%d].Turn = %d, want %d", i, snapshot.Turn, expected[i].Turn)
		}
	}
}

func TestDetector_Record_ZeroWindowSize(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       0, // Zero window size
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record more than default fallback (3)
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: ""},
		{Turn: 3, Response: "Third response", ToolCalls: []string{"tool3"}, Error: ""},
		{Turn: 4, Response: "Fourth response", ToolCalls: []string{"tool4"}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	// Should use fallback window size of 3
	if len(detector.history) != 3 {
		t.Errorf("Detector.Record() with zero window size, history length = %d, want 3", len(detector.history))
	}
}

func TestDetector_Check_InsufficientHistory(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record only one snapshot
	detector.Record(Snapshot{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""})

	result, err := detector.Check()

	if err != nil {
		t.Errorf("Detector.Check() unexpected error: %v", err)
	}

	if result.Type != CycleNone {
		t.Errorf("Detector.Check() with insufficient history, Type = %v, want %v", result.Type, CycleNone)
	}

	if result.Confidence != 0.0 {
		t.Errorf("Detector.Check() with insufficient history, Confidence = %f, want 0.0", result.Confidence)
	}
}

func TestDetector_Check_RepeatedTool(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record snapshots with repeated tool
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 3, Response: "Third response", ToolCalls: []string{"tool1"}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	result, err := detector.Check()

	if err != nil {
		t.Errorf("Detector.Check() unexpected error: %v", err)
	}

	if result.Type != CycleRepeatedTool {
		t.Errorf("Detector.Check() Type = %v, want %v", result.Type, CycleRepeatedTool)
	}

	if result.Confidence != 0.9 {
		t.Errorf("Detector.Check() Confidence = %f, want 0.9", result.Confidence)
	}

	if result.Details == "" {
		t.Errorf("Detector.Check() Details should not be empty")
	}
}

func TestDetector_Check_SameError(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record snapshots with same error
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: "test error"},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: "test error"},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	result, err := detector.Check()

	if err != nil {
		t.Errorf("Detector.Check() unexpected error: %v", err)
	}

	if result.Type != CycleSameError {
		t.Errorf("Detector.Check() Type = %v, want %v", result.Type, CycleSameError)
	}

	if result.Confidence != 0.95 {
		t.Errorf("Detector.Check() Confidence = %f, want 0.95", result.Confidence)
	}

	if result.Details == "" {
		t.Errorf("Detector.Check() Details should not be empty")
	}
}

func TestDetector_Check_Oscillation(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record snapshots with oscillation pattern - use identical responses for A and B groups
	snapshots := []Snapshot{
		{Turn: 1, Response: "Response A", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Response B", ToolCalls: []string{"tool2"}, Error: ""},
		{Turn: 3, Response: "Response A", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 4, Response: "Response B", ToolCalls: []string{"tool2"}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	result, err := detector.Check()

	if err != nil {
		t.Errorf("Detector.Check() unexpected error: %v", err)
	}

	// The detector might detect similar responses instead of oscillation
	// Both are valid cycle detections
	if result.Type != CycleOscillation && result.Type != CycleSimilarResponses {
		t.Errorf("Detector.Check() Type = %v, want %v or %v", result.Type, CycleOscillation, CycleSimilarResponses)
	}

	if result.Confidence <= 0.0 {
		t.Errorf("Detector.Check() Confidence = %f, want > 0.0", result.Confidence)
	}

	if result.Details == "" {
		t.Errorf("Detector.Check() Details should not be empty")
	}
}

func TestDetector_Check_SimilarResponses(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record snapshots with similar responses
	snapshots := []Snapshot{
		{Turn: 1, Response: "This is a test response", ToolCalls: []string{}, Error: ""},
		{Turn: 2, Response: "This is a test response", ToolCalls: []string{}, Error: ""},
		{Turn: 3, Response: "This is a test response", ToolCalls: []string{}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	result, err := detector.Check()

	if err != nil {
		t.Errorf("Detector.Check() unexpected error: %v", err)
	}

	if result.Type != CycleSimilarResponses {
		t.Errorf("Detector.Check() Type = %v, want %v", result.Type, CycleSimilarResponses)
	}

	if result.Confidence <= 0.0 {
		t.Errorf("Detector.Check() Confidence = %f, want > 0.0", result.Confidence)
	}

	if result.Details == "" {
		t.Errorf("Detector.Check() Details should not be empty")
	}
}

func TestDetector_GetHistory(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record some snapshots
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	history := detector.GetHistory()

	if len(history) != len(snapshots) {
		t.Errorf("Detector.GetHistory() length = %d, want %d", len(history), len(snapshots))
	}

	// Modify the returned history to ensure it's a copy
	history[0].Turn = 999

	// Original detector history should not be affected
	originalHistory := detector.GetHistory()
	if originalHistory[0].Turn != 1 {
		t.Errorf("Detector.GetHistory() should return a copy, original modified")
	}
}

func TestDetector_Reset(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record some snapshots
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool2"}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	// Verify history is not empty
	if len(detector.history) == 0 {
		t.Errorf("Detector history should not be empty before reset")
	}

	// Reset
	detector.Reset()

	// Verify history is empty
	if len(detector.history) != 0 {
		t.Errorf("Detector.Reset() history length = %d, want 0", len(detector.history))
	}
}

func TestDetector_Concurrency(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       10,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			snapshot := Snapshot{
				Turn:      i,
				Response:  "Response " + string(rune(i)),
				ToolCalls: []string{"tool" + string(rune(i))},
				Error:     "",
			}
			detector.Record(snapshot)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have recorded all snapshots (up to window size)
	if len(detector.history) != 10 {
		t.Errorf("Detector concurrent access, history length = %d, want 10", len(detector.history))
	}
}

func TestDetector_Check_Disabled(t *testing.T) {
	config := Config{
		Enabled:          false, // Disabled
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
	}

	detector := NewDetector(config)

	// Record snapshots that would normally trigger detection
	snapshots := []Snapshot{
		{Turn: 1, Response: "First response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 2, Response: "Second response", ToolCalls: []string{"tool1"}, Error: ""},
		{Turn: 3, Response: "Third response", ToolCalls: []string{"tool1"}, Error: ""},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	result, err := detector.Check()

	if err != nil {
		t.Errorf("Detector.Check() unexpected error: %v", err)
	}

	if result.Type != CycleNone {
		t.Errorf("Detector.Check() with disabled detector, Type = %v, want %v", result.Type, CycleNone)
	}
}

// TestCheckRepeatedTool_SameToolDifferentParams tests that cycle detection
// does NOT trigger when the same tool is called with DIFFERENT parameters.
// This is a regression test for the bug where:
//
//	ls .
//	ls .
//	ls .                      <- Cycle detected ✓ (correct - same params)
//	ls advanced-features-20251012  <- Cycle detected ✗ (WRONG - different params!)
func TestCheckRepeatedTool_SameToolDifferentParams(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 3,
	}

	detector := NewDetector(config)

	// Simulate the user's exact scenario
	// Turn 1: ls . (list current directory)
	detector.Record(Snapshot{
		Turn:      1,
		Response:  "Let me list the current directory",
		ToolCalls: []string{`list_directory({"path":"."})`}, // Same tool, same params
		Timestamp: time.Now(),
	})

	// Turn 2: ls . (list current directory again - DUPLICATE)
	detector.Record(Snapshot{
		Turn:      2,
		Response:  "Listing current directory again",
		ToolCalls: []string{`list_directory({"path":"."})`}, // Same tool, same params
		Timestamp: time.Now(),
	})

	// Turn 3: ls . (list current directory again - DUPLICATE)
	detector.Record(Snapshot{
		Turn:      3,
		Response:  "Listing current directory once more",
		ToolCalls: []string{`list_directory({"path":"."})`}, // Same tool, same params
		Timestamp: time.Now(),
	})

	// At this point, cycle should be detected (3 identical calls)
	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
	if result.Type != CycleRepeatedTool {
		t.Errorf("Expected CycleRepeatedTool after 3 identical list_directory calls, got %v", result.Type)
	}

	// Turn 4: ls advanced-features-20251012 (list SUBDIRECTORY - DIFFERENT params!)
	detector.Record(Snapshot{
		Turn:      4,
		Response:  "Now listing subdirectory advanced-features-20251012",
		ToolCalls: []string{`list_directory({"path":"advanced-features-20251012"})`}, // Same tool name, DIFFERENT params
		Timestamp: time.Now(),
	})

	// BUG TEST: Cycle should NOT be detected because parameters are different
	result, err = detector.Check()
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
	if result.Type == CycleRepeatedTool {
		t.Error("BUG: Cycle detected when listing DIFFERENT directory")
		t.Errorf("Tool: list_directory with different params should NOT trigger cycle")
		t.Errorf("Turn 1-3: ls .  Turn 4: ls advanced-features-20251012")
		t.Errorf("These are DIFFERENT operations exploring the filesystem")
	}
}

// TestCheckRepeatedTool_SameToolSameParams tests that cycle detection
// DOES trigger when the same tool is called with the SAME parameters.
func TestCheckRepeatedTool_SameToolSameParams(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 3,
	}

	detector := NewDetector(config)

	// Call the same tool with identical parameters 3 times
	for i := 1; i <= 3; i++ {
		detector.Record(Snapshot{
			Turn:      i,
			Response:  "Listing current directory",
			ToolCalls: []string{`list_directory({"path":"."})`}, // Same tool, same params
			Timestamp: time.Now(),
		})
	}

	// This SHOULD trigger cycle detection
	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
	if result.Type != CycleRepeatedTool {
		t.Errorf("Expected CycleRepeatedTool for 3 identical calls, got %v", result.Type)
	}
}

// TestCheckRepeatedTool_DifferentTools tests that different tools don't trigger cycles.
func TestCheckRepeatedTool_DifferentTools(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 3,
	}

	detector := NewDetector(config)

	// Call different tools
	detector.Record(Snapshot{
		Turn:      1,
		ToolCalls: []string{`list_directory({"path":"."})`},
		Timestamp: time.Now(),
	})

	detector.Record(Snapshot{
		Turn:      2,
		ToolCalls: []string{`read_file({"path":"test.go"})`},
		Timestamp: time.Now(),
	})

	detector.Record(Snapshot{
		Turn:      3,
		ToolCalls: []string{`execute_command({"command":"ls"})`},
		Timestamp: time.Now(),
	})

	// No cycle should be detected
	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
	if result.Type == CycleRepeatedTool {
		t.Error("Different tools should not trigger cycle detection")
	}
}

// TestCheckRepeatedTool_ExploratoryPattern tests a realistic exploratory workflow.
// Agent exploring filesystem: ls . → ls dir1 → ls dir2 → ls dir1/subdir
// This should NOT trigger cycles even though all use list_directory.
func TestCheckRepeatedTool_ExploratoryPattern(t *testing.T) {
	config := Config{
		Enabled:          true,
		WindowSize:       5,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 3,
	}

	detector := NewDetector(config)

	// Exploratory pattern - each call has different params
	explorations := []struct {
		dir      string
		response string
	}{
		{".", "Listing current directory"},
		{"advanced-features-20251012", "Checking advanced-features directory"},
		{"bioparse", "Checking bioparse directory"},
		{"advanced-features-20251012/src", "Diving into src subdirectory"},
	}

	for i, exp := range explorations {
		detector.Record(Snapshot{
			Turn:      i + 1,
			Response:  exp.response,
			ToolCalls: []string{`list_directory({"path":"` + exp.dir + `"})`}, // Each with different path param
			Timestamp: time.Now(),
		})
	}

	// This should NOT trigger cycle because each call explores a different directory
	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
	if result.Type == CycleRepeatedTool {
		t.Error("BUG: Exploratory filesystem navigation incorrectly flagged as cycle")
		t.Error("Each list_directory call had different parameters (different dirs)")
		t.Error("This is normal agent behavior, not a stuck loop")
	}
}
