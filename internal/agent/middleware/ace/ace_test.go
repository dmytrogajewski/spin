package ace

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/events"
)

// TestACEMiddleware_EmitRetrievalEvent tests ACE retrieval event emission.
func TestACEMiddleware_EmitRetrievalEvent(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	_, eventCh, _ := emitter.Subscribe()

	middleware := New(nil, nil, emitter, slog.Default())

	// Create trajectory context with known metrics.
	ctx := trajectory.NewContext("install nodejs")
	ctx.CurrentTurn = 5

	// Add some bullets to cache via RecordRetrieval (which updates stats).
	testBullets := []*bullet.Bullet{
		{ID: "b1", Content: "test bullet 1"},
		{ID: "b2", Content: "test bullet 2"},
	}
	event := trajectory.RetrievalEvent{
		Turn:         5,
		Trigger:      trajectory.TriggerError,
		Query:        "install nodejs error",
		BulletsAdded: []string{"b1", "b2"},
	}
	ctx.RecordRetrieval(event, testBullets)

	// Call EmitRetrievalEvent directly on middleware.
	middleware.EmitRetrievalEvent(ctx, trajectory.TriggerError, "install nodejs error", testBullets, 5)

	// Verify event was emitted.
	select {
	case emittedEvent := <-eventCh:
		assert.Equal(t, events.EventACERetrieval, emittedEvent.Type)

		data, ok := emittedEvent.ACERetrievalData()
		require.True(t, ok, "Expected ACERetrievalData")

		assert.Equal(t, 5, data.Turn)
		assert.Equal(t, "error", data.Trigger)
		assert.Equal(t, "install nodejs error", data.Query)
		assert.Equal(t, 2, data.BulletsRetrieved)
		assert.Equal(t, 2, data.CacheSize)

		expectedHitRate := computeExpectedHitRate(ctx.CacheHits, ctx.CacheMisses)
		assert.InDelta(t, expectedHitRate, data.CacheHitRate, 1e-9)

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ACE retrieval event")
	}
}

// computeExpectedHitRate calculates the expected cache hit rate.
func computeExpectedHitRate(hits, misses int) float64 {
	total := hits + misses
	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total)
}
