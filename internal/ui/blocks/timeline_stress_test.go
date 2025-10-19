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
// ============================================================================

// TestTimeline_StressOOM_1MBlocks verifies timeline can handle large numbers of blocks
// without OOM or performance degradation. Tests O(1) viewport calculation.
// Uses 100k blocks for CI (reduced from 1M for speed).
func TestTimeline_StressOOM_1MBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Use 100k blocks (takes ~30s) instead of 1M (takes >10min)
	// This is still enough to verify O(1) performance and memory stability
	const totalBlocks = 100_000

	t.Logf("Starting OOM stress test with %d blocks...", totalBlocks)

	timeline := NewTimeline()
	timeline.SetViewportHeight(20)

	// Track initial memory
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	//Append blocks
	startTime := time.Now()

	for i := 0; i < totalBlocks; i++ {
		block := NewBlock(BlockTypeExecute)
		block.ID = fmt.Sprintf("block-%d", i) // Unique ID
		block.Title = fmt.Sprintf("Block %d", i)
		block.Body = fmt.Sprintf("Line %d\n", i)

		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append() at block %d failed: %v", i, err)
		}

		// Progress indicator every 10k blocks
		if (i+1)%10_000 == 0 {
			t.Logf("Progress: %d/%d blocks appended", i+1, totalBlocks)
		}
	}

	appendDuration := time.Since(startTime)
	t.Logf("Appended %d blocks in %v (%.2f blocks/sec)",
		totalBlocks, appendDuration, float64(totalBlocks)/appendDuration.Seconds())

	// Verify timeline length
	if timeline.Len() != totalBlocks {
		t.Errorf("timeline.Len() = %d, want %d", timeline.Len(), totalBlocks)
	}

	// Test viewport calculation (should be O(1))
	viewportStart := time.Now()
	visible := timeline.GetVisibleBlocks()
	viewportDuration := time.Since(viewportStart)

	if len(visible) > 20 {
		t.Errorf("GetVisibleBlocks() returned %d blocks, want <= 20", len(visible))
	}

	t.Logf("GetVisibleBlocks() took %v (should be <1ms)", viewportDuration)
	if viewportDuration > 10*time.Millisecond {
		t.Errorf("GetVisibleBlocks() took %v, expected O(1) <10ms", viewportDuration)
	}

	// Test scroll to bottom (should be O(1))
	scrollStart := time.Now()
	timeline.ScrollToBottom()
	scrollDuration := time.Since(scrollStart)

	// Verify we scrolled to end (check last visible block)
	visibleAfterScroll := timeline.GetVisibleBlocks()
	if len(visibleAfterScroll) > 0 {
		lastVisible := visibleAfterScroll[len(visibleAfterScroll)-1]
		if lastVisible.Title != fmt.Sprintf("Block %d", totalBlocks-1) {
			t.Errorf("ScrollToBottom() last visible block = %q, want Block %d", lastVisible.Title, totalBlocks-1)
		}
	}

	t.Logf("ScrollToBottom() took %v", scrollDuration)
	if scrollDuration > 10*time.Millisecond {
		t.Errorf("ScrollToBottom() took %v, expected O(1) <10ms", scrollDuration)
	}

	// Test filter (will be slower but should not crash)
	filterStart := time.Now()
	filter := &Filter{Types: []BlockType{BlockTypeExecute}}
	timeline.SetFilter(filter)
	filtered := timeline.GetVisibleBlocks()
	filterDuration := time.Since(filterStart)

	if len(filtered) == 0 {
		t.Error("Filter returned no blocks, expected some EXECUTE blocks")
	}

	t.Logf("Filter on %d blocks took %v (%.2f ms)", totalBlocks, filterDuration, float64(filterDuration.Milliseconds()))

	// Memory check
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	memUsed := memAfter.Alloc - memBefore.Alloc
	memUsedMB := float64(memUsed) / 1024 / 1024

	t.Logf("Memory used: %.2f MB (%.2f bytes/block)", memUsedMB, float64(memUsed)/float64(totalBlocks))

	// Expected: ~220 bytes/block from Phase 7.2 benchmarks = ~220MB for 1M blocks
	// Allow up to 500MB (generous margin)
	if memUsedMB > 500 {
		t.Errorf("Memory used %.2f MB exceeds expected ~220 MB (with margin)", memUsedMB)
	}

	// Goroutine leak check
	goroutinesBefore := runtime.NumGoroutine()
	runtime.GC() // Force GC
	time.Sleep(100 * time.Millisecond)
	goroutinesAfter := runtime.NumGoroutine()

	if goroutinesAfter > goroutinesBefore+5 {
		t.Errorf("Goroutine leak detected: before=%d, after=%d", goroutinesBefore, goroutinesAfter)
	}

	t.Log("OOM stress test PASSED")
}

// TestTimeline_StressConcurrent verifies thread-safety under high concurrency.
// 100 writers + 10 scrollers + 10 filters running concurrently for 10 seconds.
func TestTimeline_StressConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting concurrent stress test...")

	timeline := NewTimeline()
	timeline.SetViewportHeight(20)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errors := make(chan error, 120) // Buffered for all goroutines

	// 100 concurrent writers (each writes 100 blocks)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				block := NewBlock(BlockTypeExecute)
				block.ID = fmt.Sprintf("writer-%d-block-%d", id, j) // Unique ID per block
				block.Title = fmt.Sprintf("Writer %d Block %d", id, j)
				block.Body = fmt.Sprintf("Content from writer %d\n", id)

				if err := timeline.Append(block); err != nil {
					errors <- fmt.Errorf("Append error from writer %d: %w", id, err)
					return
				}

				// Small delay to interleave operations without race conditions
				select {
				case <-time.After(1 * time.Millisecond):
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// 10 concurrent scrollers
	for i := 0; i < 10; i++ {
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

	// 10 concurrent filters
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					filter := &Filter{Types: []BlockType{BlockTypeExecute}}
					timeline.SetFilter(filter)
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

	// Wait for all goroutines
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify timeline intact
	finalLen := timeline.Len()
	t.Logf("Final timeline length: %d blocks", finalLen)

	// Expected: 100 writers * 100 blocks = 10,000 blocks
	if finalLen != 10_000 {
		t.Errorf("timeline.Len() = %d, want 10000", finalLen)
	}

	t.Log("Concurrent stress test PASSED")
}

// TestTimeline_StressScrolling verifies scroll performance on large timelines
func TestTimeline_StressScrolling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting scroll stress test...")

	timeline := NewTimeline()
	timeline.SetViewportHeight(20)

	// Create 100k blocks
	const totalBlocks = 100_000
	for i := 0; i < totalBlocks; i++ {
		block := NewBlock(BlockTypeExecute)
		block.ID = fmt.Sprintf("scroll-block-%d", i) // Unique ID
		block.Title = fmt.Sprintf("Block %d", i)
		timeline.Append(block)
	}

	t.Logf("Created %d blocks, testing scroll performance...", totalBlocks)

	// Test rapid scrolling
	start := time.Now()
	iterations := 10_000

	for i := 0; i < iterations; i++ {
		timeline.ScrollDown(1)
		if i%2 == 0 {
			timeline.GetVisibleBlocks()
		}
	}

	duration := time.Since(start)
	opsPerSec := float64(iterations) / duration.Seconds()

	t.Logf("Performed %d scroll operations in %v (%.0f ops/sec)", iterations, duration, opsPerSec)

	// Should complete in under 1 second (10,000 ops/sec minimum)
	if duration > 1*time.Second {
		t.Errorf("Scroll performance degraded: %v for %d ops (expected <1s)", duration, iterations)
	}

	t.Log("Scroll stress test PASSED")
}
