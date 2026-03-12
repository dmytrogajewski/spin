package trajectory

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

func TestNewTrajectoryContext(t *testing.T) {
	t.Run("creates non-empty session ID", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")

		if ctx.SessionID == "" {
			t.Error("expected non-empty session ID, got empty string")
		}
	})

	t.Run("stores query", func(t *testing.T) {
		query := "debug file upload"
		ctx := NewTrajectoryContext(query)

		if ctx.Query != query {
			t.Errorf("expected query %q, got %q", query, ctx.Query)
		}
	})

	t.Run("sets start time", func(t *testing.T) {
		before := time.Now()
		ctx := NewTrajectoryContext("test")
		after := time.Now()

		if ctx.StartTime.IsZero() {
			t.Error("expected non-zero start time")
		}

		if ctx.StartTime.Before(before) || ctx.StartTime.After(after) {
			t.Errorf("start time %v not between %v and %v", ctx.StartTime, before, after)
		}
	})

	t.Run("initializes empty collections", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")

		if ctx.Steps == nil {
			t.Error("expected non-nil Steps slice")
		}

		if len(ctx.Steps) != 0 {
			t.Errorf("expected empty Steps, got %d items", len(ctx.Steps))
		}

		if ctx.RetrievalEvents == nil {
			t.Error("expected non-nil RetrievalEvents slice")
		}

		if len(ctx.RetrievalEvents) != 0 {
			t.Errorf("expected empty RetrievalEvents, got %d items", len(ctx.RetrievalEvents))
		}

		if ctx.BulletCache == nil {
			t.Error("expected non-nil BulletCache map")
		}

		if len(ctx.BulletCache) != 0 {
			t.Errorf("expected empty BulletCache, got %d items", len(ctx.BulletCache))
		}
	})

	t.Run("initializes turn to zero", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")

		if ctx.CurrentTurn != 0 {
			t.Errorf("expected CurrentTurn 0, got %d", ctx.CurrentTurn)
		}
	})
}

func TestAppendSteps(t *testing.T) {
	t.Run("appends single step", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		steps := []generator.TrajectoryStep{
			{StepNumber: 0, Type: "reasoning", Content: "test"},
		}

		ctx.AppendSteps(steps)

		if len(ctx.Steps) != 1 {
			t.Errorf("expected 1 step, got %d", len(ctx.Steps))
		}

		if ctx.Steps[0].Content != "test" {
			t.Errorf("expected content 'test', got %q", ctx.Steps[0].Content)
		}
	})

	t.Run("appends multiple steps", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		steps := []generator.TrajectoryStep{
			{StepNumber: 0, Type: "reasoning", Content: "step1"},
			{StepNumber: 1, Type: "tool_call", Content: "step2"},
		}

		ctx.AppendSteps(steps)

		if len(ctx.Steps) != 2 {
			t.Errorf("expected 2 steps, got %d", len(ctx.Steps))
		}
	})

	t.Run("preserves order", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		step1 := []generator.TrajectoryStep{{StepNumber: 0, Content: "first"}}
		step2 := []generator.TrajectoryStep{{StepNumber: 1, Content: "second"}}

		ctx.AppendSteps(step1)
		ctx.AppendSteps(step2)

		if len(ctx.Steps) != 2 {
			t.Fatalf("expected 2 steps, got %d", len(ctx.Steps))
		}

		if ctx.Steps[0].Content != "first" {
			t.Errorf("expected first step content 'first', got %q", ctx.Steps[0].Content)
		}

		if ctx.Steps[1].Content != "second" {
			t.Errorf("expected second step content 'second', got %q", ctx.Steps[1].Content)
		}
	})

	t.Run("handles nil steps", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.AppendSteps(nil)

		if len(ctx.Steps) != 0 {
			t.Errorf("expected 0 steps after nil append, got %d", len(ctx.Steps))
		}
	})

	t.Run("handles empty steps", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.AppendSteps([]generator.TrajectoryStep{})

		if len(ctx.Steps) != 0 {
			t.Errorf("expected 0 steps after empty append, got %d", len(ctx.Steps))
		}
	})
}

func TestRecordRetrieval(t *testing.T) {
	t.Run("records first retrieval", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event := RetrievalEvent{
			Turn:         0,
			Trigger:      TriggerInitial,
			Query:        "test query",
			BulletsAdded: []string{"B1", "B2"},
			Timestamp:    time.Now(),
		}
		bullets := []*bullet.Bullet{
			{ID: "B1", Content: "bullet 1"},
			{ID: "B2", Content: "bullet 2"},
		}

		ctx.RecordRetrieval(event, bullets)

		if len(ctx.RetrievalEvents) != 1 {
			t.Errorf("expected 1 event, got %d", len(ctx.RetrievalEvents))
		}

		if ctx.LastRetrievalTurn != 0 {
			t.Errorf("expected LastRetrievalTurn 0, got %d", ctx.LastRetrievalTurn)
		}

		if ctx.TotalRetrievals != 1 {
			t.Errorf("expected TotalRetrievals 1, got %d", ctx.TotalRetrievals)
		}

		if len(ctx.BulletCache) != 2 {
			t.Errorf("expected 2 cached bullets, got %d", len(ctx.BulletCache))
		}
	})

	t.Run("counts cache misses for new bullets", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event := RetrievalEvent{Turn: 0}
		bullets := []*bullet.Bullet{
			{ID: "B1"},
			{ID: "B2"},
		}

		ctx.RecordRetrieval(event, bullets)

		if ctx.CacheMisses != 2 {
			t.Errorf("expected 2 cache misses, got %d", ctx.CacheMisses)
		}

		if ctx.CacheHits != 0 {
			t.Errorf("expected 0 cache hits, got %d", ctx.CacheHits)
		}
	})

	t.Run("counts cache hits for duplicate bullets", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event1 := RetrievalEvent{Turn: 0}
		bullets1 := []*bullet.Bullet{{ID: "B1"}}

		ctx.RecordRetrieval(event1, bullets1)

		event2 := RetrievalEvent{Turn: 5}
		bullets2 := []*bullet.Bullet{{ID: "B1"}}

		ctx.RecordRetrieval(event2, bullets2)

		if ctx.CacheMisses != 1 {
			t.Errorf("expected 1 cache miss, got %d", ctx.CacheMisses)
		}

		if ctx.CacheHits != 1 {
			t.Errorf("expected 1 cache hit, got %d", ctx.CacheHits)
		}
	})

	t.Run("increments access count on cache hit", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event1 := RetrievalEvent{Turn: 0}
		bullets1 := []*bullet.Bullet{{ID: "B1"}}
		ctx.RecordRetrieval(event1, bullets1)

		event2 := RetrievalEvent{Turn: 5}
		bullets2 := []*bullet.Bullet{{ID: "B1"}}
		ctx.RecordRetrieval(event2, bullets2)

		cached := ctx.BulletCache["B1"]
		if cached.AccessCount != 2 {
			t.Errorf("expected AccessCount 2, got %d", cached.AccessCount)
		}
	})

	t.Run("handles mixed new and cached bullets", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event1 := RetrievalEvent{Turn: 0}
		bullets1 := []*bullet.Bullet{{ID: "B1"}}
		ctx.RecordRetrieval(event1, bullets1)

		event2 := RetrievalEvent{Turn: 5}
		bullets2 := []*bullet.Bullet{
			{ID: "B1"}, // cached.
			{ID: "B2"}, // new.
		}
		ctx.RecordRetrieval(event2, bullets2)

		if ctx.CacheMisses != 2 {
			t.Errorf("expected 2 cache misses, got %d", ctx.CacheMisses)
		}

		if ctx.CacheHits != 1 {
			t.Errorf("expected 1 cache hit, got %d", ctx.CacheHits)
		}

		if len(ctx.BulletCache) != 2 {
			t.Errorf("expected 2 cached bullets, got %d", len(ctx.BulletCache))
		}
	})
}

func TestGetActiveBullets(t *testing.T) {
	t.Run("returns bullets within TTL", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.CurrentTurn = 5
		event := RetrievalEvent{Turn: 0}
		bullets := []*bullet.Bullet{
			{ID: "B1"},
		}
		ctx.RecordRetrieval(event, bullets)

		active := ctx.GetActiveBullets()

		if len(active) != 1 {
			t.Errorf("expected 1 active bullet, got %d", len(active))
		}
	})

	t.Run("excludes bullets beyond TTL", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.CurrentTurn = 15 // 15 turns later.
		event := RetrievalEvent{Turn: 0}
		bullets := []*bullet.Bullet{{ID: "B1"}}
		ctx.RecordRetrieval(event, bullets)

		active := ctx.GetActiveBullets()

		if len(active) != 0 {
			t.Errorf("expected 0 active bullets (beyond TTL), got %d", len(active))
		}
	})

	t.Run("handles mixed fresh and expired bullets", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")

		// Add old bullet (will expire).
		event1 := RetrievalEvent{Turn: 0}
		bullets1 := []*bullet.Bullet{{ID: "B1"}}
		ctx.RecordRetrieval(event1, bullets1)

		ctx.CurrentTurn = 15

		// Add fresh bullet.
		event2 := RetrievalEvent{Turn: 15}
		bullets2 := []*bullet.Bullet{{ID: "B2"}}
		ctx.RecordRetrieval(event2, bullets2)

		active := ctx.GetActiveBullets()

		if len(active) != 1 {
			t.Errorf("expected 1 active bullet, got %d", len(active))
		}

		if active[0].ID != "B2" {
			t.Errorf("expected active bullet B2, got %s", active[0].ID)
		}
	})

	t.Run("returns bullets in deterministic order", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event := RetrievalEvent{Turn: 0}
		bullets := []*bullet.Bullet{
			{ID: "B3"},
			{ID: "B1"},
			{ID: "B2"},
		}
		ctx.RecordRetrieval(event, bullets)

		active := ctx.GetActiveBullets()

		if len(active) != 3 {
			t.Fatalf("expected 3 bullets, got %d", len(active))
		}

		if active[0].ID != "B1" || active[1].ID != "B2" || active[2].ID != "B3" {
			t.Errorf("expected sorted order [B1, B2, B3], got [%s, %s, %s]",
				active[0].ID, active[1].ID, active[2].ID)
		}
	})

	t.Run("updates last accessed time", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event := RetrievalEvent{Turn: 0}
		bullets := []*bullet.Bullet{{ID: "B1"}}
		ctx.RecordRetrieval(event, bullets)

		ctx.CurrentTurn = 5
		ctx.GetActiveBullets()

		cached := ctx.BulletCache["B1"]
		if cached.LastAccessed != 5 {
			t.Errorf("expected LastAccessed 5, got %d", cached.LastAccessed)
		}
	})
}

func TestToTrajectory(t *testing.T) {
	t.Run("converts empty context", func(t *testing.T) {
		ctx := NewTrajectoryContext("test query")

		traj := ctx.ToTrajectory()

		if traj.ID != ctx.SessionID {
			t.Errorf("expected ID %s, got %s", ctx.SessionID, traj.ID)
		}

		if traj.Query != "test query" {
			t.Errorf("expected query 'test query', got %q", traj.Query)
		}

		if len(traj.Steps) != 0 {
			t.Errorf("expected 0 steps, got %d", len(traj.Steps))
		}

		if len(traj.RetrievedBullets) != 0 {
			t.Errorf("expected 0 bullets, got %d", len(traj.RetrievedBullets))
		}

		// Verify empty RetrievalEvents.
		events, ok := traj.Metadata.RetrievalEvents.([]RetrievalEvent)
		if !ok {
			t.Fatalf("expected []RetrievalEvent type, got %T", traj.Metadata.RetrievalEvents)
		}

		if len(events) != 0 {
			t.Errorf("expected 0 retrieval events, got %d", len(events))
		}
	})

	t.Run("includes all steps", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		steps := []generator.TrajectoryStep{
			{StepNumber: 0, Content: "step1"},
			{StepNumber: 1, Content: "step2"},
		}
		ctx.AppendSteps(steps)

		traj := ctx.ToTrajectory()

		if len(traj.Steps) != 2 {
			t.Errorf("expected 2 steps, got %d", len(traj.Steps))
		}
	})

	t.Run("includes all cached bullets", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event := RetrievalEvent{Turn: 0}
		bullets := []*bullet.Bullet{
			{ID: "B1"},
			{ID: "B2"},
		}
		ctx.RecordRetrieval(event, bullets)

		traj := ctx.ToTrajectory()

		if len(traj.RetrievedBullets) != 2 {
			t.Errorf("expected 2 bullets, got %d", len(traj.RetrievedBullets))
		}
	})

	t.Run("includes retrieval events", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		event := RetrievalEvent{
			Turn:    0,
			Trigger: TriggerInitial,
			Query:   "test",
		}
		ctx.RecordRetrieval(event, nil)

		traj := ctx.ToTrajectory()

		if traj.Metadata.RetrievalEvents == nil {
			t.Error("expected non-nil retrieval events")
		}

		events, ok := traj.Metadata.RetrievalEvents.([]RetrievalEvent)
		if !ok {
			t.Fatalf("expected []RetrievalEvent, got %T", traj.Metadata.RetrievalEvents)
		}

		if len(events) != 1 {
			t.Errorf("expected 1 retrieval event, got %d", len(events))
		}
	})

	t.Run("preserves multiple retrieval events in order", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")

		// Record multiple events.
		event1 := RetrievalEvent{
			Turn:    0,
			Trigger: TriggerInitial,
			Query:   "initial query",
		}
		event2 := RetrievalEvent{
			Turn:    5,
			Trigger: TriggerError,
			Query:   "error recovery query",
		}
		event3 := RetrievalEvent{
			Turn:    10,
			Trigger: TriggerToolChange,
			Query:   "tool change query",
		}

		ctx.RecordRetrieval(event1, nil)
		ctx.RecordRetrieval(event2, nil)
		ctx.RecordRetrieval(event3, nil)

		traj := ctx.ToTrajectory()

		events, ok := traj.Metadata.RetrievalEvents.([]RetrievalEvent)
		if !ok {
			t.Fatalf("expected []RetrievalEvent, got %T", traj.Metadata.RetrievalEvents)
		}

		if len(events) != 3 {
			t.Fatalf("expected 3 retrieval events, got %d", len(events))
		}

		// Verify order preserved.
		if events[0].Turn != 0 || events[0].Trigger != TriggerInitial {
			t.Errorf("event 0: expected turn=0, trigger=initial, got turn=%d, trigger=%s",
				events[0].Turn, events[0].Trigger)
		}

		if events[1].Turn != 5 || events[1].Trigger != TriggerError {
			t.Errorf("event 1: expected turn=5, trigger=error, got turn=%d, trigger=%s",
				events[1].Turn, events[1].Trigger)
		}

		if events[2].Turn != 10 || events[2].Trigger != TriggerToolChange {
			t.Errorf("event 2: expected turn=10, trigger=tool_change, got turn=%d, trigger=%s",
				events[2].Turn, events[2].Trigger)
		}
	})

	t.Run("sets success flag", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.Success = true

		traj := ctx.ToTrajectory()

		if !traj.Success {
			t.Error("expected Success true, got false")
		}
	})

	t.Run("calculates turns", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")
		ctx.CurrentTurn = 5

		traj := ctx.ToTrajectory()

		if traj.Metadata.Turns != 6 {
			t.Errorf("expected Turns 6 (CurrentTurn+1), got %d", traj.Metadata.Turns)
		}
	})

	t.Run("calculates duration", func(t *testing.T) {
		ctx := NewTrajectoryContext("test")

		time.Sleep(10 * time.Millisecond)

		traj := ctx.ToTrajectory()

		if traj.Metadata.Duration < 10*time.Millisecond {
			t.Errorf("expected duration >= 10ms, got %v", traj.Metadata.Duration)
		}
	})
}
