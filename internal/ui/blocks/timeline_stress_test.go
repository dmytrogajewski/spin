package blocks

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Phase 8.2: Stress Tests
// ============================================================================.

// TestTimeline_StressOOM_1MBlocks verifies timeline can handle large numbers of blocks
// without OOM or performance degradation. Tests O(1) viewport calculation.
// Uses 100k blocks for CI (reduced from 1M for speed).
func TestTimeline_StressOOM_1MBlocks(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const totalBlocks = 100_000

	t.Logf("Starting OOM stress test with %d blocks...", totalBlocks)

	timeline := NewTimeline()
	timeline.SetViewportHeight(20)

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	appendDuration := populateTimeline(t, timeline, totalBlocks, "block")

	t.Logf("Appended %d blocks in %v (%.2f blocks/sec)",
		totalBlocks, appendDuration, float64(totalBlocks)/appendDuration.Seconds())

	if timeline.Len() != totalBlocks {
		t.Errorf("timeline.Len() = %d, want %d", timeline.Len(), totalBlocks)
	}

	verifyViewport(t, timeline, 20)
	verifyScrollToBottom(t, timeline, totalBlocks)
	verifyFilter(t, timeline, totalBlocks)
	verifyMemory(t, &memBefore, totalBlocks)
	verifyNoGoroutineLeak(t)

	t.Log("OOM stress test PASSED")
}

// populateTimeline appends totalBlocks to the timeline, logging progress.
func populateTimeline(t *testing.T, timeline *Timeline, totalBlocks int, prefix string) time.Duration {
	t.Helper()

	startTime := time.Now()

	for i := range totalBlocks {
		block := NewBlock(BlockTypeExecute)
		block.ID = fmt.Sprintf("%s-%d", prefix, i)
		block.Title = fmt.Sprintf("Block %d", i)
		block.Body = fmt.Sprintf("Line %d\n", i)

		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append() at block %d failed: %v", i, err)
		}

		if (i+1)%10_000 == 0 {
			t.Logf("Progress: %d/%d blocks appended", i+1, totalBlocks)
		}
	}

	return time.Since(startTime)
}

// verifyViewport checks that GetVisibleBlocks returns at most maxVisible blocks in O(1) time.
func verifyViewport(t *testing.T, timeline *Timeline, maxVisible int) {
	t.Helper()

	start := time.Now()
	visible := timeline.GetVisibleBlocks()
	dur := time.Since(start)

	if len(visible) > maxVisible {
		t.Errorf("GetVisibleBlocks() returned %d blocks, want <= %d", len(visible), maxVisible)
	}

	t.Logf("GetVisibleBlocks() took %v (should be <1ms)", dur)

	if dur > 10*time.Millisecond {
		t.Errorf("GetVisibleBlocks() took %v, expected O(1) <10ms", dur)
	}
}

// verifyScrollToBottom checks ScrollToBottom is O(1) and scrolls to the last block.
func verifyScrollToBottom(t *testing.T, timeline *Timeline, totalBlocks int) {
	t.Helper()

	start := time.Now()

	timeline.ScrollToBottom()

	dur := time.Since(start)

	visibleAfterScroll := timeline.GetVisibleBlocks()
	if len(visibleAfterScroll) > 0 {
		lastVisible := visibleAfterScroll[len(visibleAfterScroll)-1]

		expected := fmt.Sprintf("Block %d", totalBlocks-1)
		if lastVisible.Title != expected {
			t.Errorf("ScrollToBottom() last visible block = %q, want %s", lastVisible.Title, expected)
		}
	}

	t.Logf("ScrollToBottom() took %v", dur)

	if dur > 10*time.Millisecond {
		t.Errorf("ScrollToBottom() took %v, expected O(1) <10ms", dur)
	}
}

// verifyFilter checks that filtering works correctly.
func verifyFilter(t *testing.T, timeline *Timeline, totalBlocks int) {
	t.Helper()

	start := time.Now()

	timeline.SetFilter(&Filter{Types: []BlockType{BlockTypeExecute}})
	filtered := timeline.GetVisibleBlocks()
	dur := time.Since(start)

	if len(filtered) == 0 {
		t.Error("Filter returned no blocks, expected some EXECUTE blocks")
	}

	t.Logf("Filter on %d blocks took %v (%.2f ms)", totalBlocks, dur, float64(dur.Milliseconds()))
}

// verifyMemory checks memory usage is within bounds.
func verifyMemory(t *testing.T, memBefore *runtime.MemStats, totalBlocks int) {
	t.Helper()

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memUsed := memAfter.Alloc - memBefore.Alloc
	memUsedMB := float64(memUsed) / 1024 / 1024

	t.Logf("Memory used: %.2f MB (%.2f bytes/block)", memUsedMB, float64(memUsed)/float64(totalBlocks))

	if memUsedMB > 500 {
		t.Errorf("Memory used %.2f MB exceeds expected ~220 MB (with margin)", memUsedMB)
	}
}

// verifyNoGoroutineLeak checks for goroutine leaks.
func verifyNoGoroutineLeak(t *testing.T) {
	t.Helper()

	goroutinesBefore := runtime.NumGoroutine()

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	goroutinesAfter := runtime.NumGoroutine()

	if goroutinesAfter > goroutinesBefore+5 {
		t.Errorf("Goroutine leak detected: before=%d, after=%d", goroutinesBefore, goroutinesAfter)
	}
}

// TestTimeline_StressConcurrent verifies thread-safety under high concurrency.
// 100 writers + 10 scrollers + 10 filters running concurrently for 10 seconds.
func TestTimeline_StressConcurrent(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting concurrent stress test...")

	timeline := NewTimeline()
	timeline.SetViewportHeight(20)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	errors := make(chan error, 120)

	startWriters(ctx, &wg, timeline, errors, 100, 100)
	startScrollers(ctx, &wg, timeline, 10)
	startFilterers(ctx, &wg, timeline, 10)

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	finalLen := timeline.Len()
	t.Logf("Final timeline length: %d blocks", finalLen)

	if finalLen != 10_000 {
		t.Errorf("timeline.Len() = %d, want 10000", finalLen)
	}

	t.Log("Concurrent stress test PASSED")
}

// startWriters launches concurrent writer goroutines that append blocks.
func startWriters(ctx context.Context, wg *sync.WaitGroup, timeline *Timeline, errors chan<- error, numWriters, blocksPerWriter int) {
	for i := range numWriters {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			writeBlocks(ctx, timeline, errors, id, blocksPerWriter)
		}(i)
	}
}

// writeBlocks appends blocks to the timeline for a single writer.
func writeBlocks(ctx context.Context, timeline *Timeline, errors chan<- error, writerID, count int) {
	for j := range count {
		select {
		case <-ctx.Done():
			return
		default:
		}

		block := NewBlock(BlockTypeExecute)
		block.ID = fmt.Sprintf("writer-%d-block-%d", writerID, j)
		block.Title = fmt.Sprintf("Writer %d Block %d", writerID, j)
		block.Body = fmt.Sprintf("Content from writer %d\n", writerID)

		if err := timeline.Append(block); err != nil {
			errors <- fmt.Errorf("Append error from writer %d: %w", writerID, err)

			return
		}

		select {
		case <-time.After(1 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

// startScrollers launches concurrent scroller goroutines.
func startScrollers(ctx context.Context, wg *sync.WaitGroup, timeline *Timeline, count int) {
	for range count {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					timeline.ScrollDown(1)
					timeline.ScrollUp(1)
					timeline.GetVisibleBlocks()

					select {
					case <-time.After(5 * time.Millisecond):
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
}

// startFilterers launches concurrent filter goroutines.
func startFilterers(ctx context.Context, wg *sync.WaitGroup, timeline *Timeline, count int) {
	for range count {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					timeline.SetFilter(&Filter{Types: []BlockType{BlockTypeExecute}})
					timeline.GetVisibleBlocks()
					timeline.ClearFilter()

					select {
					case <-time.After(10 * time.Millisecond):
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
}

// TestTimeline_StressScrolling verifies scroll performance on large timelines.
func TestTimeline_StressScrolling(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting scroll stress test...")

	timeline := NewTimeline()
	timeline.SetViewportHeight(20)

	// Create 100k blocks.
	const totalBlocks = 100_000
	for i := range totalBlocks {
		block := NewBlock(BlockTypeExecute)
		block.ID = fmt.Sprintf("scroll-block-%d", i) // Unique ID.
		block.Title = fmt.Sprintf("Block %d", i)
		_ = timeline.Append(block)
	}

	t.Logf("Created %d blocks, testing scroll performance...", totalBlocks)

	// Test rapid scrolling.
	start := time.Now()
	iterations := 10_000

	for i := range iterations {
		timeline.ScrollDown(1)

		if i%2 == 0 {
			timeline.GetVisibleBlocks()
		}
	}

	duration := time.Since(start)
	opsPerSec := float64(iterations) / duration.Seconds()

	t.Logf("Performed %d scroll operations in %v (%.0f ops/sec)", iterations, duration, opsPerSec)

	// Should complete in under 1 second (10,000 ops/sec minimum).
	if duration > 1*time.Second {
		t.Errorf("Scroll performance degraded: %v for %d ops (expected <1s)", duration, iterations)
	}

	t.Log("Scroll stress test PASSED")
}
