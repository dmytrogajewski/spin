package cycle

import (
	"testing"
	"time"
)

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
