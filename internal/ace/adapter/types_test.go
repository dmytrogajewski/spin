package adapter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecutionSignal_Creation(t *testing.T) {
	t.Parallel()

	now := time.Now()

	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Context:    "TestFoo failed with panic",
		Outcome:    OutcomeFailure,
		Details: map[string]string{
			"test_name": "TestFoo",
			"error_msg": "panic: nil pointer",
		},
		Timestamp: now,
		SessionID: "session-123",
	}

	assert.Equal(t, SignalTypeTest, signal.SignalType)
	assert.Equal(t, "TestFoo failed with panic", signal.Context)
	assert.Equal(t, OutcomeFailure, signal.Outcome)
	assert.Equal(t, "TestFoo", signal.Details["test_name"])
	assert.Equal(t, "panic: nil pointer", signal.Details["error_msg"])
	assert.Equal(t, now, signal.Timestamp)
	assert.Equal(t, "session-123", signal.SessionID)
}

func TestSession_Creation(t *testing.T) {
	t.Parallel()

	now := time.Now()

	session := Session{
		ID:          "session-123",
		StartTime:   now,
		SignalCount: 0,
		UpdateCount: 0,
	}

	assert.Equal(t, "session-123", session.ID)
	assert.Equal(t, now, session.StartTime)
	assert.Equal(t, 0, session.SignalCount)
	assert.Equal(t, 0, session.UpdateCount)
	assert.Nil(t, session.LastSignal)
	assert.Empty(t, session.RecentSignals)
}

func TestSession_AddSignal(t *testing.T) {
	t.Parallel()

	session := Session{
		ID:            "session-123",
		StartTime:     time.Now(),
		RecentSignals: []*ExecutionSignal{},
	}

	signal1 := &ExecutionSignal{
		SignalType: SignalTypeTest,
		Context:    "Test 1",
		Outcome:    OutcomeFailure,
		Timestamp:  time.Now(),
	}

	session.AddSignal(signal1)

	assert.Equal(t, 1, session.SignalCount)
	assert.Equal(t, signal1, session.LastSignal)
	assert.Len(t, session.RecentSignals, 1)
	assert.Equal(t, signal1, session.RecentSignals[0])
}

func TestSession_SlidingWindow(t *testing.T) {
	t.Parallel()

	session := Session{
		ID:            "session-123",
		StartTime:     time.Now(),
		RecentSignals: []*ExecutionSignal{},
	}

	// Add 15 signals (more than max of 10).
	signals := make([]*ExecutionSignal, 15)
	for i := range 15 {
		signals[i] = &ExecutionSignal{
			SignalType: SignalTypeTest,
			Context:    "Test signal",
			Outcome:    OutcomeNeutral,
			Timestamp:  time.Now(),
		}
		session.AddSignal(signals[i])
	}

	// Should have all 15 signals counted.
	assert.Equal(t, 15, session.SignalCount)

	// 15 is under maxRecentSignals (50), so all are kept.
	assert.Len(t, session.RecentSignals, 15)

	// Last signal should be the 15th.
	assert.Same(t, signals[14], session.LastSignal)

	// First signal in recent should be the 1st (index 0).
	assert.Same(t, signals[0], session.RecentSignals[0])

	// Last signal in recent should be the 15th (index 14).
	assert.Same(t, signals[14], session.RecentSignals[14])
}

func TestAdaptationResult_Creation(t *testing.T) {
	t.Parallel()

	result := AdaptationResult{
		Action:              ActionReflect,
		BulletsAdded:        3,
		BulletsUpdated:      1,
		LatencyMs:           45,
		Reason:              "Test failure detected",
		RefinementTriggered: false,
	}

	assert.Equal(t, ActionReflect, result.Action)
	assert.Equal(t, 3, result.BulletsAdded)
	assert.Equal(t, 1, result.BulletsUpdated)
	assert.Equal(t, int64(45), result.LatencyMs)
	assert.Equal(t, "Test failure detected", result.Reason)
	assert.False(t, result.RefinementTriggered)
}

func TestAdaptationAction_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ActionSkip, AdaptationAction("skip"))
	assert.Equal(t, ActionReflect, AdaptationAction("reflect"))
	assert.Equal(t, ActionQuickAdd, AdaptationAction("quick_add"))
	assert.Equal(t, ActionUpdate, AdaptationAction("update"))
}
