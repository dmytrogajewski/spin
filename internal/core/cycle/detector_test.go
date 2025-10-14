package cycle

import (
	"strings"
	"testing"
	"time"
)

func TestNewDetector(t *testing.T) {
	config := DefaultConfig()
	detector := NewDetector(config)

	if detector == nil {
		t.Fatal("NewDetector returned nil")
	}

	if len(detector.history) != 0 {
		t.Errorf("Expected empty history, got %d snapshots", len(detector.history))
	}

	if detector.config.WindowSize != config.WindowSize {
		t.Errorf("Expected window size %d, got %d", config.WindowSize, detector.config.WindowSize)
	}
}

func TestDetector_Record(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	// Test recording snapshots
	snapshots := []Snapshot{
		{
			Turn:      1,
			Response:  "Hello world",
			ToolCalls: []string{"read_file"},
			Timestamp: time.Now(),
		},
		{
			Turn:      2,
			Response:  "Hello again",
			ToolCalls: []string{"write_file"},
			Timestamp: time.Now(),
		},
		{
			Turn:      3,
			Response:  "Third response",
			ToolCalls: []string{"list_dir"},
			Timestamp: time.Now(),
		},
	}

	for _, snapshot := range snapshots {
		detector.Record(snapshot)
	}

	history := detector.GetHistory()
	if len(history) != len(snapshots) {
		t.Errorf("Expected %d snapshots, got %d", len(snapshots), len(history))
	}

	// Verify snapshots are in order
	for i, snapshot := range snapshots {
		if history[i].Turn != snapshot.Turn {
			t.Errorf("Expected turn %d at index %d, got %d", snapshot.Turn, i, history[i].Turn)
		}
	}
}

func TestDetector_Record_RollingWindow(t *testing.T) {
	config := Config{
		WindowSize: 2,
		Enabled:    true,
	}
	detector := NewDetector(config)

	// Record more snapshots than window size
	for i := 1; i <= 5; i++ {
		detector.Record(Snapshot{
			Turn:      i,
			Response:  "Response",
			Timestamp: time.Now(),
		})
	}

	history := detector.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected window size 2, got %d snapshots", len(history))
	}

	// Should only keep the last 2 snapshots
	expectedTurns := []int{4, 5}
	for i, snapshot := range history {
		if snapshot.Turn != expectedTurns[i] {
			t.Errorf("Expected turn %d at index %d, got %d", expectedTurns[i], i, snapshot.Turn)
		}
	}
}

func TestDetector_Check_NoHistory(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Type != CycleNone {
		t.Errorf("Expected CycleNone, got %v", result.Type)
	}

	if result.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0, got %f", result.Confidence)
	}
}

func TestDetector_Check_MinimalHistory(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	// Record only one snapshot
	detector.Record(Snapshot{
		Turn:      1,
		Response:  "Single response",
		Timestamp: time.Now(),
	})

	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Type != CycleNone {
		t.Errorf("Expected CycleNone with minimal history, got %v", result.Type)
	}
}

func TestDetector_Check_SimilarResponses(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	// Record identical responses first to ensure basic detection works
	responses := []string{
		"I think we should try a different approach",
		"I think we should try a different approach",
		"I think we should try a different approach",
		"I think we should try a different approach",
	}

	for i, response := range responses {
		detector.Record(Snapshot{
			Turn:      i + 1,
			Response:  response,
			Timestamp: time.Now(),
		})
	}

	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Type != CycleSimilarResponses {
		t.Errorf("Expected CycleSimilarResponses, got %v", result.Type)
	}

	if result.Confidence <= 0.5 {
		t.Errorf("Expected high confidence, got %f", result.Confidence)
	}

	if !contains(result.Details, "similar") && !contains(result.Details, "consecutive") {
		t.Errorf("Expected details to mention similarity or consecutive, got: %s", result.Details)
	}
}

func TestDetector_Check_RepeatedTool(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	// Record repeated tool calls
	for i := 1; i <= 3; i++ {
		detector.Record(Snapshot{
			Turn:      i,
			Response:  "Response",
			ToolCalls: []string{"read_file"},
			Timestamp: time.Now(),
		})
	}

	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Type != CycleRepeatedTool {
		t.Errorf("Expected CycleRepeatedTool, got %v", result.Type)
	}

	if result.Confidence < 0.9 {
		t.Errorf("Expected high confidence for exact matches, got %f", result.Confidence)
	}

	if !contains(result.Details, "read_file") {
		t.Errorf("Expected details to mention the tool name, got: %s", result.Details)
	}
}

func TestDetector_Check_Oscillation(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	// Record oscillating pattern: A B A B with similar A responses and similar B responses
	responses := []string{
		"I think we should use the read_file tool to examine the code",
		"Let's use the list_dir tool to see what files are available",
		"I think we should use the read_file tool to examine the code again",
		"Let's use the list_dir tool to see what files are available once more",
	}

	for i, response := range responses {
		detector.Record(Snapshot{
			Turn:      i + 1,
			Response:  response,
			Timestamp: time.Now(),
		})
	}

	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Type != CycleOscillation {
		t.Errorf("Expected CycleOscillation, got %v", result.Type)
	}

	if result.Confidence < 0.5 {
		t.Errorf("Expected reasonable confidence for oscillation, got %f", result.Confidence)
	}

	if !contains(result.Details, "oscillation") {
		t.Errorf("Expected details to mention oscillation, got: %s", result.Details)
	}
}

func TestDetector_Check_SameError(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	// Record repeated errors
	errorMsg := "File not found: /missing/path"
	for i := 1; i <= 3; i++ {
		detector.Record(Snapshot{
			Turn:      i,
			Response:  "Response",
			Error:     errorMsg,
			Timestamp: time.Now(),
		})
	}

	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Type != CycleSameError {
		t.Errorf("Expected CycleSameError, got %v", result.Type)
	}

	if result.Confidence < 0.9 {
		t.Errorf("Expected high confidence for exact error matches, got %f", result.Confidence)
	}

	if !contains(result.Details, errorMsg) {
		t.Errorf("Expected details to mention the error, got: %s", result.Details)
	}
}

func TestDetector_Reset(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	// Add some history
	for i := 1; i <= 3; i++ {
		detector.Record(Snapshot{
			Turn:      i,
			Response:  "Response",
			Timestamp: time.Now(),
		})
	}

	if len(detector.GetHistory()) != 3 {
		t.Errorf("Expected 3 snapshots before reset")
	}

	detector.Reset()

	history := detector.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history after reset, got %d snapshots", len(history))
	}
}

func TestDetector_Check_Disabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}
	detector := NewDetector(config)

	// Add similar responses that would trigger detection
	for i := 1; i <= 3; i++ {
		detector.Record(Snapshot{
			Turn:      i,
			Response:  "Identical response",
			Timestamp: time.Now(),
		})
	}

	result, err := detector.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Even with identical responses, should return CycleNone when disabled
	if result.Type != CycleNone {
		t.Errorf("Expected CycleNone when detection is disabled, got %v", result.Type)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
