package delta

import (
	"testing"
	"time"
)

func TestDeltaHistory_Record(t *testing.T) {
	h := NewHistory()

	delta := NewContentUpdate("bullet-1", "content", Metadata{Source: "test"})
	h.Record(*delta)

	if h.Len() != 1 {
		t.Errorf("expected len 1, got %d", h.Len())
	}
}

func TestDeltaHistory_GetByBullet(t *testing.T) {
	h := NewHistory()

	// Add deltas for two different bullets.
	delta1 := NewContentUpdate("bullet-1", "content1", Metadata{Source: "test"})
	delta2 := NewIncrementHelpful("bullet-1", Metadata{Source: "test"})
	delta3 := NewContentUpdate("bullet-2", "content2", Metadata{Source: "test"})

	h.Record(*delta1)
	h.Record(*delta2)
	h.Record(*delta3)

	// Get deltas for bullet-1.
	deltas := h.GetByBullet("bullet-1")
	if len(deltas) != 2 {
		t.Errorf("expected 2 deltas for bullet-1, got %d", len(deltas))
	}

	// Get deltas for bullet-2.
	deltas = h.GetByBullet("bullet-2")
	if len(deltas) != 1 {
		t.Errorf("expected 1 delta for bullet-2, got %d", len(deltas))
	}

	// Get deltas for non-existent bullet.
	deltas = h.GetByBullet("bullet-999")
	if deltas != nil {
		t.Errorf("expected nil for non-existent bullet, got %v", deltas)
	}
}

func TestDeltaHistory_GetRecent(t *testing.T) {
	h := NewHistory()

	// Add 5 deltas.
	for range 5 {
		delta := NewContentUpdate("bullet-1", "content", Metadata{Source: "test"})
		h.Record(*delta)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps.
	}

	// Get 3 most recent.
	recent := h.GetRecent(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 recent deltas, got %d", len(recent))
	}

	// Get more than available.
	recent = h.GetRecent(10)
	if len(recent) != 5 {
		t.Errorf("expected 5 deltas when requesting 10, got %d", len(recent))
	}

	// Get with count 0.
	recent = h.GetRecent(0)
	if recent != nil {
		t.Errorf("expected nil for count 0, got %v", recent)
	}

	// Get with negative count.
	recent = h.GetRecent(-1)
	if recent != nil {
		t.Errorf("expected nil for negative count, got %v", recent)
	}
}

func TestDeltaHistory_GetSince(t *testing.T) {
	h := NewHistory()

	now := time.Now()
	past := now.Add(-10 * time.Second)
	future := now.Add(10 * time.Second)

	// Add deltas with known timestamps (using metadata to track).
	delta1 := NewContentUpdate("bullet-1", "content1", Metadata{Source: "test"})
	delta1.CreatedAt = past
	h.Record(*delta1)

	time.Sleep(1 * time.Millisecond)

	delta2 := NewContentUpdate("bullet-2", "content2", Metadata{Source: "test"})
	delta2.CreatedAt = now
	h.Record(*delta2)

	time.Sleep(1 * time.Millisecond)

	delta3 := NewContentUpdate("bullet-3", "content3", Metadata{Source: "test"})
	delta3.CreatedAt = future
	h.Record(*delta3)

	// Get deltas since now.
	deltas := h.GetSince(now)
	if len(deltas) != 2 { // now and future.
		t.Errorf("expected 2 deltas since now, got %d", len(deltas))
	}

	// Get deltas since past.
	deltas = h.GetSince(past)
	if len(deltas) != 3 {
		t.Errorf("expected 3 deltas since past, got %d", len(deltas))
	}

	// Get deltas since future.
	deltas = h.GetSince(future.Add(1 * time.Second))
	if len(deltas) != 0 {
		t.Errorf("expected 0 deltas since future, got %d", len(deltas))
	}
}

func TestDeltaHistory_Stats(t *testing.T) {
	h := NewHistory()

	// Empty history.
	stats := h.Stats()
	if stats.TotalDeltas != 0 {
		t.Errorf("expected TotalDeltas 0, got %d", stats.TotalDeltas)
	}

	if stats.UniqueBullets != 0 {
		t.Errorf("expected UniqueBullets 0, got %d", stats.UniqueBullets)
	}

	// Add deltas.
	h.Record(*NewContentUpdate("bullet-1", "content", Metadata{Source: "test"}))
	h.Record(*NewIncrementHelpful("bullet-1", Metadata{Source: "test"}))
	h.Record(*NewIncrementHarmful("bullet-2", Metadata{Source: "test"}))

	stats = h.Stats()
	if stats.TotalDeltas != 3 {
		t.Errorf("expected TotalDeltas 3, got %d", stats.TotalDeltas)
	}

	if stats.UniqueBullets != 2 {
		t.Errorf("expected UniqueBullets 2, got %d", stats.UniqueBullets)
	}

	if stats.DeltasByOperation[OpUpdateContent] != 1 {
		t.Errorf("expected 1 OpUpdateContent, got %d", stats.DeltasByOperation[OpUpdateContent])
	}

	if stats.DeltasByOperation[OpIncrementHelpful] != 1 {
		t.Errorf("expected 1 OpIncrementHelpful, got %d", stats.DeltasByOperation[OpIncrementHelpful])
	}

	if stats.DeltasByOperation[OpIncrementHarmful] != 1 {
		t.Errorf("expected 1 OpIncrementHarmful, got %d", stats.DeltasByOperation[OpIncrementHarmful])
	}

	if stats.OldestDelta.IsZero() {
		t.Error("expected non-zero OldestDelta")
	}

	if stats.NewestDelta.IsZero() {
		t.Error("expected non-zero NewestDelta")
	}

	if !stats.NewestDelta.After(stats.OldestDelta) && !stats.NewestDelta.Equal(stats.OldestDelta) {
		t.Error("expected NewestDelta >= OldestDelta")
	}
}

func TestDeltaHistory_Clear(t *testing.T) {
	h := NewHistory()

	// Add deltas.
	h.Record(*NewContentUpdate("bullet-1", "content", Metadata{Source: "test"}))
	h.Record(*NewIncrementHelpful("bullet-1", Metadata{Source: "test"}))

	if h.Len() != 2 {
		t.Errorf("expected len 2 before clear, got %d", h.Len())
	}

	// Clear.
	h.Clear()

	if h.Len() != 0 {
		t.Errorf("expected len 0 after clear, got %d", h.Len())
	}

	// Verify indices are also cleared.
	deltas := h.GetByBullet("bullet-1")
	if deltas != nil {
		t.Errorf("expected nil after clear, got %v", deltas)
	}
}

func TestDeltaHistory_Concurrency(t *testing.T) {
	h := NewHistory()

	// Concurrent writes.
	const (
		goroutines         = 10
		deltasPerGoroutine = 100
	)

	done := make(chan bool, goroutines)
	for g := range goroutines {
		go func(_ int) {
			for range deltasPerGoroutine {
				delta := NewContentUpdate("bullet-1", "content", Metadata{Source: "test"})
				h.Record(*delta)
			}

			done <- true
		}(g)
	}

	// Wait for all goroutines.
	for range goroutines {
		<-done
	}

	expected := goroutines * deltasPerGoroutine
	if h.Len() != expected {
		t.Errorf("expected %d deltas after concurrent writes, got %d", expected, h.Len())
	}

	// Concurrent reads while writing.
	stop := make(chan bool)
	errors := make(chan error, goroutines*2)

	// Start readers.
	for range goroutines {
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
					_ = h.GetByBullet("bullet-1")
					_ = h.GetRecent(10)
					_ = h.Stats()
				}
			}
		}()
	}

	// Start writers.
	for range goroutines {
		go func() {
			for range 10 {
				delta := NewContentUpdate("bullet-1", "content", Metadata{Source: "test"})
				h.Record(*delta)
			}

			errors <- nil
		}()
	}

	// Wait for writers.
	for range goroutines {
		err := <-errors
		if err != nil {
			t.Errorf("writer error: %v", err)
		}
	}

	// Stop readers.
	close(stop)
	time.Sleep(10 * time.Millisecond)
}
