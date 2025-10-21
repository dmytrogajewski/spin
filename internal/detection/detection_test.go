package detection

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig returns a test configuration for cycle detection
func testConfig() cycle.Config {
	return cycle.Config{
		WindowSize:       3,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
		Enabled:          true,
	}
}

func TestNewDetectionService(t *testing.T) {
	tests := []struct {
		name            string
		cycleDetector   *cycle.Detector
		patternDetector *cycle.PatternDetector
		wantNil         bool
	}{
		{
			name:            "with both detectors",
			cycleDetector:   cycle.NewDetector(testConfig()),
			patternDetector: cycle.NewPatternDetector(testConfig()),
			wantNil:         false,
		},
		{
			name:            "with nil cycle detector",
			cycleDetector:   nil,
			patternDetector: cycle.NewPatternDetector(testConfig()),
			wantNil:         false, // Service allows nil
		},
		{
			name:            "with nil pattern detector",
			cycleDetector:   cycle.NewDetector(testConfig()),
			patternDetector: nil,
			wantNil:         false, // Service allows nil
		},
		{
			name:            "with both nil",
			cycleDetector:   nil,
			patternDetector: nil,
			wantNil:         false, // Service allows nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDetectionService(tt.cycleDetector, tt.patternDetector)

			if tt.wantNil {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
			}
		})
	}
}

func TestDetectionService_RecordSnapshot(t *testing.T) {
	detector := cycle.NewDetector(testConfig())
	svc := NewDetectionService(detector, nil)

	snapshot := cycle.Snapshot{
		Turn:      1,
		Response:  "Hello world",
		ToolCalls: []string{"read_file"},
		Error:     "",
	}

	// Should not panic
	svc.RecordSnapshot(snapshot)

	// Verify snapshot was recorded
	history := svc.GetHistory()
	require.Len(t, history, 1)
	assert.Equal(t, 1, history[0].Turn)
}

func TestDetectionService_RecordSnapshot_NilDetector(t *testing.T) {
	svc := NewDetectionService(nil, nil)

	snapshot := cycle.Snapshot{
		Turn:     1,
		Response: "Hello",
	}

	// Should not panic even with nil detector
	svc.RecordSnapshot(snapshot)
}

func TestDetectionService_CheckCycle(t *testing.T) {
	tests := []struct {
		name       string
		snapshots  []cycle.Snapshot
		wantCycle  cycle.CycleType
		wantDetect bool
	}{
		{
			name:       "no snapshots - no cycle",
			snapshots:  []cycle.Snapshot{},
			wantCycle:  cycle.CycleNone,
			wantDetect: false,
		},
		{
			name: "similar responses - cycle detected",
			snapshots: []cycle.Snapshot{
				{Turn: 1, Response: "I tried to read the file", ToolCalls: []string{"list_files"}},
				{Turn: 2, Response: "I tried to read the file", ToolCalls: []string{"read_file"}},
				{Turn: 3, Response: "I tried to read the file", ToolCalls: []string{"write_file"}},
			},
			wantCycle:  cycle.CycleSimilarResponses,
			wantDetect: true,
		},
		{
			name: "repeated tool - cycle detected",
			snapshots: []cycle.Snapshot{
				{Turn: 1, Response: "Reading...", ToolCalls: []string{"read_file"}},
				{Turn: 2, Response: "Reading again...", ToolCalls: []string{"read_file"}},
				{Turn: 3, Response: "Still reading...", ToolCalls: []string{"read_file"}},
			},
			wantCycle:  cycle.CycleRepeatedTool,
			wantDetect: true,
		},
		{
			name: "same error - cycle detected",
			snapshots: []cycle.Snapshot{
				{Turn: 1, Response: "Error", Error: "file not found"},
				{Turn: 2, Response: "Error", Error: "file not found"},
			},
			wantCycle:  cycle.CycleSameError,
			wantDetect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := cycle.Config{
				WindowSize:       3,
				SimilarityThresh: 0.8,
				ToolRepeatLimit:  3,
				ErrorRepeatLimit: 2,
				Enabled:          true,
			}
			detector := cycle.NewDetector(config)
			svc := NewDetectionService(detector, nil)

			// Record snapshots
			for _, snapshot := range tt.snapshots {
				svc.RecordSnapshot(snapshot)
			}

			// Check for cycle
			result, err := svc.CheckCycle()
			require.NoError(t, err)

			if tt.wantDetect {
				assert.Equal(t, tt.wantCycle, result.Type)
				assert.Greater(t, result.Confidence, 0.0)
			} else {
				assert.Equal(t, cycle.CycleNone, result.Type)
			}
		})
	}
}

func TestDetectionService_CheckCycle_NilDetector(t *testing.T) {
	svc := NewDetectionService(nil, nil)

	result, err := svc.CheckCycle()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detector not configured")
	assert.Equal(t, cycle.CycleNone, result.Type)
}

func TestDetectionService_DetectPattern(t *testing.T) {
	config := cycle.Config{
		WindowSize:       5,
		SimilarityThresh: 0.75,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
		Enabled:          true,
	}
	detector := cycle.NewDetector(config)
	patternDetector := cycle.NewPatternDetector(config)
	svc := NewDetectionService(detector, patternDetector)

	// Add some snapshots
	snapshots := []cycle.Snapshot{
		{Turn: 1, Response: "Reading file A", ToolCalls: []string{"read_file"}},
		{Turn: 2, Response: "Writing to file B", ToolCalls: []string{"write_file"}},
		{Turn: 3, Response: "Reading file A", ToolCalls: []string{"read_file"}},
		{Turn: 4, Response: "Writing to file B", ToolCalls: []string{"write_file"}},
	}

	for _, s := range snapshots {
		svc.RecordSnapshot(s)
	}

	// Detect pattern
	results, err := svc.DetectPattern()
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestDetectionService_DetectPattern_NilDetector(t *testing.T) {
	svc := NewDetectionService(nil, nil)

	results, err := svc.DetectPattern()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pattern detector not configured")
	assert.Nil(t, results)
}

func TestDetectionService_Reset(t *testing.T) {
	detector := cycle.NewDetector(testConfig())
	svc := NewDetectionService(detector, nil)

	// Add some snapshots
	svc.RecordSnapshot(cycle.Snapshot{Turn: 1, Response: "Test"})
	svc.RecordSnapshot(cycle.Snapshot{Turn: 2, Response: "Test"})

	history := svc.GetHistory()
	require.Len(t, history, 2)

	// Reset
	svc.Reset()

	// History should be empty
	history = svc.GetHistory()
	assert.Len(t, history, 0)
}

func TestDetectionService_Reset_NilDetector(t *testing.T) {
	svc := NewDetectionService(nil, nil)

	// Should not panic even with nil detector
	svc.Reset()
}

func TestDetectionService_GetHistory(t *testing.T) {
	detector := cycle.NewDetector(testConfig())
	svc := NewDetectionService(detector, nil)

	// Empty history
	history := svc.GetHistory()
	assert.Len(t, history, 0)

	// Add snapshots
	svc.RecordSnapshot(cycle.Snapshot{Turn: 1, Response: "First"})
	svc.RecordSnapshot(cycle.Snapshot{Turn: 2, Response: "Second"})

	// Get history
	history = svc.GetHistory()
	assert.Len(t, history, 2)
	assert.Equal(t, "First", history[0].Response)
	assert.Equal(t, "Second", history[1].Response)
}

func TestDetectionService_GetHistory_NilDetector(t *testing.T) {
	svc := NewDetectionService(nil, nil)

	history := svc.GetHistory()
	assert.Len(t, history, 0)
}

// Benchmark tests
func BenchmarkDetectionService_RecordSnapshot(b *testing.B) {
	detector := cycle.NewDetector(testConfig())
	svc := NewDetectionService(detector, nil)

	snapshot := cycle.Snapshot{
		Turn:      1,
		Response:  "Test response",
		ToolCalls: []string{"read_file"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.RecordSnapshot(snapshot)
	}
}

func BenchmarkDetectionService_CheckCycle(b *testing.B) {
	detector := cycle.NewDetector(testConfig())
	svc := NewDetectionService(detector, nil)

	// Pre-populate with some snapshots
	for i := 0; i < 10; i++ {
		svc.RecordSnapshot(cycle.Snapshot{
			Turn:      i,
			Response:  "Test response",
			ToolCalls: []string{"read_file"},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CheckCycle()
	}
}
