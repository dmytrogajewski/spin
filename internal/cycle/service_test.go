package cycle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent implements the Event interface for testing.
type testEvent struct {
	eventType string
	timestamp time.Time
	data      EventData
}

func (e *testEvent) GetType() string         { return e.eventType }
func (e *testEvent) GetTimestamp() time.Time { return e.timestamp }
func (e *testEvent) GetData() any            { return e.data }

// Mock implementations for testing.

type mockTracker struct {
	snapshots []Snapshot
	result    Result
	err       error
}

func newMockTracker() *mockTracker {
	return &mockTracker{
		snapshots: []Snapshot{},
		result:    Result{Type: None},
	}
}

func (m *mockTracker) Record(snapshot Snapshot) {
	m.snapshots = append(m.snapshots, snapshot)
}

func (m *mockTracker) Check() (Result, error) {
	return m.result, m.err
}

func (m *mockTracker) GetHistory() []Snapshot {
	return m.snapshots
}

func (m *mockTracker) Reset() {
	m.snapshots = []Snapshot{}
}

type mockPatternAnalyzer struct {
	results []PatternResult
}

func newMockPatternAnalyzer() *mockPatternAnalyzer {
	return &mockPatternAnalyzer{
		results: []PatternResult{},
	}
}

func (m *mockPatternAnalyzer) AnalyzePatterns(_ []Snapshot) []PatternResult {
	return m.results
}

// TestEventData tests the EventData type.
func TestEventData(t *testing.T) {
	t.Parallel()

	t.Run("create and access event data", func(t *testing.T) {
		t.Parallel()

		data := EventData{
			"status":  "paused",
			"message": "Agent stuck in cycle",
		}

		assert.Equal(t, "paused", data["status"])
		assert.Equal(t, "Agent stuck in cycle", data["message"])
	})

	t.Run("event uses EventData", func(t *testing.T) {
		t.Parallel()

		evt := &testEvent{
			eventType: "turn_paused",
			data: EventData{
				"status":  "paused",
				"message": "Test message",
			},
		}

		// Verify data is strongly typed.
		data := evt.GetData()
		eventData, ok := data.(EventData)
		require.True(t, ok, "GetData() should return EventData")
		assert.Equal(t, "paused", eventData["status"])
		assert.Equal(t, "Test message", eventData["message"])
	})
}

func TestNewService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		tracker         Tracker
		patternDetector PatternAnalyzer
		wantNil         bool
	}{
		{
			name:            "with both detectors",
			tracker:         newMockTracker(),
			patternDetector: newMockPatternAnalyzer(),
			wantNil:         false,
		},
		{
			name:            "with nil tracker",
			tracker:         nil,
			patternDetector: newMockPatternAnalyzer(),
			wantNil:         false, // Service allows nil.
		},
		{
			name:            "with nil pattern detector",
			tracker:         newMockTracker(),
			patternDetector: nil,
			wantNil:         false, // Service allows nil.
		},
		{
			name:            "with both nil",
			tracker:         nil,
			patternDetector: nil,
			wantNil:         false, // Service allows nil.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewService(tt.tracker, tt.patternDetector)

			if tt.wantNil {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
			}
		})
	}
}

func TestService_RecordSnapshot(t *testing.T) {
	t.Parallel()

	tracker := newMockTracker()
	svc := NewService(tracker, nil)

	snapshot := Snapshot{
		Turn:      1,
		Response:  "Hello world",
		ToolCalls: []string{"read_file"},
		Error:     "",
	}

	// Should not panic.
	svc.RecordSnapshot(snapshot)

	// Verify snapshot was recorded.
	history := svc.GetHistory()
	require.Len(t, history, 1)
	assert.Equal(t, 1, history[0].Turn)
}

func TestService_RecordSnapshot_NilDetector(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, nil)

	snapshot := Snapshot{
		Turn:     1,
		Response: "Hello",
	}

	// Should not panic even with nil detector.
	svc.RecordSnapshot(snapshot)
}

func TestService_CheckCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		snapshots  []Snapshot
		wantCycle  Type
		wantDetect bool
	}{
		{
			name:       "no snapshots - no cycle",
			snapshots:  []Snapshot{},
			wantCycle:  None,
			wantDetect: false,
		},
		{
			name: "similar responses - cycle detected",
			snapshots: []Snapshot{
				{Turn: 1, Response: "I tried to read the file", ToolCalls: []string{"list_files"}},
				{Turn: 2, Response: "I tried to read the file", ToolCalls: []string{"read_file"}},
				{Turn: 3, Response: "I tried to read the file", ToolCalls: []string{"write_file"}},
			},
			wantCycle:  SimilarResponses,
			wantDetect: true,
		},
		{
			name: "repeated tool - cycle detected",
			snapshots: []Snapshot{
				{Turn: 1, Response: "Reading...", ToolCalls: []string{"read_file"}},
				{Turn: 2, Response: "Reading again...", ToolCalls: []string{"read_file"}},
				{Turn: 3, Response: "Still reading...", ToolCalls: []string{"read_file"}},
			},
			wantCycle:  RepeatedTool,
			wantDetect: true,
		},
		{
			name: "same error - cycle detected",
			snapshots: []Snapshot{
				{Turn: 1, Response: "Error", Error: "file not found"},
				{Turn: 2, Response: "Error", Error: "file not found"},
			},
			wantCycle:  SameError,
			wantDetect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tracker := newMockTracker()
			if tt.wantDetect {
				tracker.result = Result{
					Type:       tt.wantCycle,
					Confidence: 0.9,
				}
			}

			svc := NewService(tracker, nil)

			// Record snapshots.
			for _, snapshot := range tt.snapshots {
				svc.RecordSnapshot(snapshot)
			}

			// Check for cycle.
			result, err := svc.CheckCycle()
			require.NoError(t, err)

			if tt.wantDetect {
				assert.Equal(t, tt.wantCycle, result.Type)
				assert.Greater(t, result.Confidence, 0.0)
			} else {
				assert.Equal(t, None, result.Type)
			}
		})
	}
}

func TestService_CheckCycle_NilDetector(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, nil)

	result, err := svc.CheckCycle()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detector not configured")
	assert.Equal(t, None, result.Type)
}

func TestService_DetectPattern(t *testing.T) {
	t.Parallel()

	tracker := newMockTracker()
	patternDetector := newMockPatternAnalyzer()
	patternDetector.results = []PatternResult{
		{Type: PatternOscillatingTools, Confidence: 0.8, Details: "A-B pattern detected"},
	}
	svc := NewService(tracker, patternDetector)

	// Add some snapshots.
	snapshots := []Snapshot{
		{Turn: 1, Response: "Reading file A", ToolCalls: []string{"read_file"}},
		{Turn: 2, Response: "Writing to file B", ToolCalls: []string{"write_file"}},
		{Turn: 3, Response: "Reading file A", ToolCalls: []string{"read_file"}},
		{Turn: 4, Response: "Writing to file B", ToolCalls: []string{"write_file"}},
	}

	for _, s := range snapshots {
		svc.RecordSnapshot(s)
	}

	// Detect pattern.
	results, err := svc.DetectPattern()
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestService_DetectPattern_NilDetector(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, nil)

	results, err := svc.DetectPattern()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pattern detector not configured")
	assert.Nil(t, results)
}

func TestService_Reset(t *testing.T) {
	t.Parallel()

	tracker := newMockTracker()
	svc := NewService(tracker, nil)

	// Add some snapshots.
	svc.RecordSnapshot(Snapshot{Turn: 1, Response: "Test"})
	svc.RecordSnapshot(Snapshot{Turn: 2, Response: "Test"})

	history := svc.GetHistory()
	require.Len(t, history, 2)

	// Reset.
	svc.Reset()

	// History should be empty.
	history = svc.GetHistory()
	assert.Empty(t, history)
}

func TestService_Reset_NilDetector(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, nil)

	// Should not panic even with nil detector.
	svc.Reset()
}

func TestService_GetHistory(t *testing.T) {
	t.Parallel()

	tracker := newMockTracker()
	svc := NewService(tracker, nil)

	// Empty history.
	history := svc.GetHistory()
	assert.Empty(t, history)

	// Add snapshots.
	svc.RecordSnapshot(Snapshot{Turn: 1, Response: "First"})
	svc.RecordSnapshot(Snapshot{Turn: 2, Response: "Second"})

	// Get history.
	history = svc.GetHistory()
	assert.Len(t, history, 2)
	assert.Equal(t, "First", history[0].Response)
	assert.Equal(t, "Second", history[1].Response)
}

func TestService_GetHistory_NilDetector(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, nil)

	history := svc.GetHistory()
	assert.Empty(t, history)
}

// Benchmark tests.
func BenchmarkService_RecordSnapshot(b *testing.B) {
	tracker := newMockTracker()
	svc := NewService(tracker, nil)

	snapshot := Snapshot{
		Turn:      1,
		Response:  "Test response",
		ToolCalls: []string{"read_file"},
	}

	b.ResetTimer()

	for range b.N {
		svc.RecordSnapshot(snapshot)
	}
}

func BenchmarkService_CheckCycle(b *testing.B) {
	tracker := newMockTracker()
	svc := NewService(tracker, nil)

	// Pre-populate with some snapshots.
	for i := range 10 {
		svc.RecordSnapshot(Snapshot{
			Turn:      i,
			Response:  "Test response",
			ToolCalls: []string{"read_file"},
		})
	}

	b.ResetTimer()

	for range b.N {
		_, _ = svc.CheckCycle()
	}
}
