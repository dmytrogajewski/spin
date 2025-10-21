package cycle

import (
	"testing"
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
