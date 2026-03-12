package trajectory

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

func TestNewContext_SessionID(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test query")
	if ctx.SessionID == "" {
		t.Error("expected non-empty session ID, got empty string")
	}
}

func TestNewContext_Query(t *testing.T) {
	t.Parallel()

	query := "debug file upload"
	ctx := NewContext(query)
	assertFieldEqual(t, "Query", ctx.Query, query)
}

func TestNewContext_StartTime(t *testing.T) {
	t.Parallel()

	before := time.Now()
	ctx := NewContext("test")
	after := time.Now()

	if ctx.StartTime.IsZero() {
		t.Error("expected non-zero start time")
	}

	if ctx.StartTime.Before(before) || ctx.StartTime.After(after) {
		t.Errorf("start time %v not between %v and %v", ctx.StartTime, before, after)
	}
}

func TestNewContext_EmptyCollections(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	assertNonNilEmpty(t, "Steps", ctx.Steps != nil, len(ctx.Steps))
	assertNonNilEmpty(t, "RetrievalEvents", ctx.RetrievalEvents != nil, len(ctx.RetrievalEvents))
	assertNonNilEmpty(t, "BulletCache", ctx.BulletCache != nil, len(ctx.BulletCache))
	assertIntFieldEqual(t, "CurrentTurn", ctx.CurrentTurn, 0)
}

// assertNonNilEmpty checks a collection is non-nil and empty.
func assertNonNilEmpty(t *testing.T, name string, nonNil bool, length int) {
	t.Helper()

	if !nonNil {
		t.Errorf("expected non-nil %s", name)
	}

	if length != 0 {
		t.Errorf("expected empty %s, got %d items", name, length)
	}
}

func TestAppendSteps_Single(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Type: "reasoning", Content: "test"},
	})

	assertIntFieldEqual(t, "step count", len(ctx.Steps), 1)
	assertFieldEqual(t, "content", ctx.Steps[0].Content, "test")
}

func TestAppendSteps_Multiple(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Type: "reasoning", Content: "step1"},
		{StepNumber: 1, Type: "tool_call", Content: "step2"},
	})

	assertIntFieldEqual(t, "step count", len(ctx.Steps), 2)
}

func TestAppendSteps_PreservesOrder(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.AppendSteps([]generator.TrajectoryStep{{StepNumber: 0, Content: "first"}})
	ctx.AppendSteps([]generator.TrajectoryStep{{StepNumber: 1, Content: "second"}})

	if len(ctx.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(ctx.Steps))
	}

	assertFieldEqual(t, "first step", ctx.Steps[0].Content, "first")
	assertFieldEqual(t, "second step", ctx.Steps[1].Content, "second")
}

func TestAppendSteps_NilAndEmpty(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.AppendSteps(nil)
	assertIntFieldEqual(t, "steps after nil", len(ctx.Steps), 0)

	ctx.AppendSteps([]generator.TrajectoryStep{})
	assertIntFieldEqual(t, "steps after empty", len(ctx.Steps), 0)
}

func TestRecordRetrieval_First(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	event := RetrievalEvent{
		Turn: 0, Trigger: TriggerInitial, Query: "test query",
		BulletsAdded: []string{"B1", "B2"}, Timestamp: time.Now(),
	}
	ctx.RecordRetrieval(event, []*bullet.Bullet{{ID: "B1", Content: "bullet 1"}, {ID: "B2", Content: "bullet 2"}})

	assertIntFieldEqual(t, "RetrievalEvents", len(ctx.RetrievalEvents), 1)
	assertIntFieldEqual(t, "LastRetrievalTurn", ctx.LastRetrievalTurn, 0)
	assertIntFieldEqual(t, "TotalRetrievals", ctx.TotalRetrievals, 1)
	assertIntFieldEqual(t, "BulletCache", len(ctx.BulletCache), 2)
}

func TestRecordRetrieval_CacheMisses(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}, {ID: "B2"}})

	assertIntFieldEqual(t, "CacheMisses", ctx.CacheMisses, 2)
	assertIntFieldEqual(t, "CacheHits", ctx.CacheHits, 0)
}

func TestRecordRetrieval_CacheHits(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}})
	ctx.RecordRetrieval(RetrievalEvent{Turn: 5}, []*bullet.Bullet{{ID: "B1"}})

	assertIntFieldEqual(t, "CacheMisses", ctx.CacheMisses, 1)
	assertIntFieldEqual(t, "CacheHits", ctx.CacheHits, 1)
}

func TestRecordRetrieval_AccessCount(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}})
	ctx.RecordRetrieval(RetrievalEvent{Turn: 5}, []*bullet.Bullet{{ID: "B1"}})

	assertIntFieldEqual(t, "AccessCount", ctx.BulletCache["B1"].AccessCount, 2)
}

func TestRecordRetrieval_MixedNewAndCached(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}})
	ctx.RecordRetrieval(RetrievalEvent{Turn: 5}, []*bullet.Bullet{{ID: "B1"}, {ID: "B2"}})

	assertIntFieldEqual(t, "CacheMisses", ctx.CacheMisses, 2)
	assertIntFieldEqual(t, "CacheHits", ctx.CacheHits, 1)
	assertIntFieldEqual(t, "BulletCache", len(ctx.BulletCache), 2)
}

func TestGetActiveBullets_WithinTTL(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.CurrentTurn = 5
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}})

	assertIntFieldEqual(t, "active bullets", len(ctx.GetActiveBullets()), 1)
}

func TestGetActiveBullets_BeyondTTL(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.CurrentTurn = 15
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}})

	assertIntFieldEqual(t, "active bullets", len(ctx.GetActiveBullets()), 0)
}

func TestGetActiveBullets_MixedFreshExpired(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}})
	ctx.CurrentTurn = 15
	ctx.RecordRetrieval(RetrievalEvent{Turn: 15}, []*bullet.Bullet{{ID: "B2"}})

	active := ctx.GetActiveBullets()
	assertIntFieldEqual(t, "active bullets", len(active), 1)
	assertFieldEqual(t, "active bullet ID", active[0].ID, "B2")
}

func TestGetActiveBullets_DeterministicOrder(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B3"}, {ID: "B1"}, {ID: "B2"}})

	active := ctx.GetActiveBullets()
	if len(active) != 3 {
		t.Fatalf("expected 3 bullets, got %d", len(active))
	}

	if active[0].ID != "B1" || active[1].ID != "B2" || active[2].ID != "B3" {
		t.Errorf("expected sorted order [B1, B2, B3], got [%s, %s, %s]",
			active[0].ID, active[1].ID, active[2].ID)
	}
}

func TestGetActiveBullets_UpdatesLastAccessed(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}})
	ctx.CurrentTurn = 5
	ctx.GetActiveBullets()

	assertIntFieldEqual(t, "LastAccessed", ctx.BulletCache["B1"].LastAccessed, 5)
}

func TestToTrajectory_EmptyContext(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test query")
	traj := ctx.ToTrajectory()

	assertFieldEqual(t, "ID", traj.ID, ctx.SessionID)
	assertFieldEqual(t, "Query", traj.Query, "test query")
	assertIntFieldEqual(t, "Steps count", len(traj.Steps), 0)
	assertIntFieldEqual(t, "RetrievedBullets count", len(traj.RetrievedBullets), 0)

	events := extractRetrievalEvents(t, traj)
	assertIntFieldEqual(t, "RetrievalEvents count", len(events), 0)
}

func TestToTrajectory_Steps(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Content: "step1"},
		{StepNumber: 1, Content: "step2"},
	})

	traj := ctx.ToTrajectory()
	assertIntFieldEqual(t, "Steps count", len(traj.Steps), 2)
}

func TestToTrajectory_Bullets(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0}, []*bullet.Bullet{{ID: "B1"}, {ID: "B2"}})

	traj := ctx.ToTrajectory()
	assertIntFieldEqual(t, "RetrievedBullets count", len(traj.RetrievedBullets), 2)
}

func TestToTrajectory_RetrievalEvents(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0, Trigger: TriggerInitial, Query: "test"}, nil)

	traj := ctx.ToTrajectory()
	if traj.Metadata.RetrievalEvents == nil {
		t.Error("expected non-nil retrieval events")
	}

	events := extractRetrievalEvents(t, traj)
	assertIntFieldEqual(t, "RetrievalEvents count", len(events), 1)
}

func TestToTrajectory_MultipleRetrievalEvents(t *testing.T) {
	t.Parallel()

	ctx := NewContext("test")
	ctx.RecordRetrieval(RetrievalEvent{Turn: 0, Trigger: TriggerInitial, Query: "initial query"}, nil)
	ctx.RecordRetrieval(RetrievalEvent{Turn: 5, Trigger: TriggerError, Query: "error recovery query"}, nil)
	ctx.RecordRetrieval(RetrievalEvent{Turn: 10, Trigger: TriggerToolChange, Query: "tool change query"}, nil)

	traj := ctx.ToTrajectory()
	events := extractRetrievalEvents(t, traj)

	if len(events) != 3 {
		t.Fatalf("expected 3 retrieval events, got %d", len(events))
	}

	verifyRetrievalEvent(t, events[0], 0, TriggerInitial)
	verifyRetrievalEvent(t, events[1], 5, TriggerError)
	verifyRetrievalEvent(t, events[2], 10, TriggerToolChange)
}

func TestToTrajectory_Metadata(t *testing.T) {
	t.Parallel()

	t.Run("sets success flag", func(t *testing.T) {
		t.Parallel()

		ctx := NewContext("test")
		ctx.Success = true
		if !ctx.ToTrajectory().Success {
			t.Error("expected Success true, got false")
		}
	})

	t.Run("calculates turns", func(t *testing.T) {
		t.Parallel()

		ctx := NewContext("test")
		ctx.CurrentTurn = 5
		assertIntFieldEqual(t, "Turns", ctx.ToTrajectory().Metadata.Turns, 6)
	})

	t.Run("calculates duration", func(t *testing.T) {
		t.Parallel()

		ctx := NewContext("test")
		time.Sleep(10 * time.Millisecond)
		if ctx.ToTrajectory().Metadata.Duration < 10*time.Millisecond {
			t.Errorf("expected duration >= 10ms, got %v", ctx.ToTrajectory().Metadata.Duration)
		}
	})
}

// extractRetrievalEvents extracts []RetrievalEvent from a trajectory.
func extractRetrievalEvents(t *testing.T, traj *generator.Trajectory) []RetrievalEvent {
	t.Helper()

	events, ok := traj.Metadata.RetrievalEvents.([]RetrievalEvent)
	if !ok {
		t.Fatalf("expected []RetrievalEvent type, got %T", traj.Metadata.RetrievalEvents)
	}

	return events
}

// verifyRetrievalEvent checks a retrieval event's turn and trigger.
func verifyRetrievalEvent(t *testing.T, event RetrievalEvent, expectedTurn int, expectedTrigger TriggerType) {
	t.Helper()

	if event.Turn != expectedTurn || event.Trigger != expectedTrigger {
		t.Errorf("expected turn=%d, trigger=%s, got turn=%d, trigger=%s",
			expectedTurn, expectedTrigger, event.Turn, event.Trigger)
	}
}

// assertFieldEqual checks string equality with a descriptive error.
func assertFieldEqual(t *testing.T, name, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("expected %s %q, got %q", name, want, got)
	}
}

// assertIntFieldEqual checks int equality with a descriptive error.
func assertIntFieldEqual(t *testing.T, name string, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("expected %s %d, got %d", name, want, got)
	}
}
