package blocks_test

import (
	"fmt"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

// BenchmarkTimelineAppend_10k measures append performance for 10k blocks.
func BenchmarkTimelineAppend_10k(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tl := blocks.NewTimeline()
		b.StartTimer()

		for j := 0; j < 10000; j++ {
			block := blocks.NewBlock(blocks.BlockTypeExecute)
			block.Title = fmt.Sprintf("Block %d", j)
			_ = tl.Append(block)
		}
	}
}

// BenchmarkTimelineGetVisibleBlocks_10k measures viewport calculation for 10k blocks.
func BenchmarkTimelineGetVisibleBlocks_10k(b *testing.B) {
	// Setup: Create timeline with 10k blocks
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		block.Body = generateMockTranscript(50) // 50 lines
		meta := &blocks.ExecuteMeta{
			Command:    "go test ./...",
			CWD:        "/home/user/project",
			Impact:     "medium",
			ExitCode:   ptr(0),
			DurationMS: ptr(int64(4200)),
			LinesOut:   ptr(50),
		}
		blocks.SetExecuteMeta(block, meta)
		_ = tl.Append(block)
	}

	// Scroll to middle to test realistic scenario
	tl.ScrollDown(5000)

	// Reset timer after setup
	b.ResetTimer()

	// Benchmark: GetVisibleBlocks
	for i := 0; i < b.N; i++ {
		_ = tl.GetVisibleBlocks()
	}
}

// BenchmarkTimelineGetVisibleBlocks_100k measures viewport calculation for 100k blocks.
func BenchmarkTimelineGetVisibleBlocks_100k(b *testing.B) {
	// Setup: Create timeline with 100k blocks
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 100000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		_ = tl.Append(block)
	}

	// Scroll to middle
	tl.ScrollDown(50000)

	// Reset timer after setup
	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_ = tl.GetVisibleBlocks()
	}
}

// BenchmarkTimelineScrollDown_10k measures scroll performance.
func BenchmarkTimelineScrollDown_10k(b *testing.B) {
	// Setup
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		_ = tl.Append(block)
	}

	b.ResetTimer()

	// Benchmark: Scroll down operations
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tl.ScrollToTop()
		b.StartTimer()

		for j := 0; j < 100; j++ {
			tl.ScrollDown(10)
		}
	}
}

// BenchmarkTimelineScrollPgDn_10k measures page down performance.
func BenchmarkTimelineScrollPgDn_10k(b *testing.B) {
	// Setup
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		_ = tl.Append(block)
	}

	b.ResetTimer()

	// Benchmark: Page down operations (40-row viewport)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tl.ScrollToTop()
		b.StartTimer()

		for j := 0; j < 250; j++ { // 250 pages * 40 rows = 10k blocks
			tl.ScrollDown(40)
		}
	}
}

// BenchmarkTimelineScrollToBottom_10k measures jump to bottom performance.
func BenchmarkTimelineScrollToBottom_10k(b *testing.B) {
	// Setup
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		_ = tl.Append(block)
	}

	b.ResetTimer()

	// Benchmark: Jump operations
	for i := 0; i < b.N; i++ {
		tl.ScrollToTop()
		tl.ScrollToBottom()
	}
}

// BenchmarkTimelineFilter_10k measures filter application performance.
func BenchmarkTimelineFilter_10k(b *testing.B) {
	// Setup: Create timeline with mixed block types
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 10000; i++ {
		var blockType blocks.BlockType
		if i%2 == 0 {
			blockType = blocks.BlockTypeExecute
		} else {
			blockType = blocks.BlockTypePlan
		}

		block := blocks.NewBlock(blockType)
		block.Title = fmt.Sprintf("Block %d", i)
		_ = tl.Append(block)
	}

	filter := &blocks.Filter{
		Types: []blocks.BlockType{blocks.BlockTypeExecute},
	}

	b.ResetTimer()

	// Benchmark: Filter application + get visible
	for i := 0; i < b.N; i++ {
		tl.SetFilter(filter)
		_ = tl.GetVisibleBlocks()
		tl.ClearFilter()
		_ = tl.GetVisibleBlocks()
	}
}

// BenchmarkTimelineFilter_ExitCode_10k measures exit code filtering performance.
func BenchmarkTimelineFilter_ExitCode_10k(b *testing.B) {
	// Setup: Create timeline with varied exit codes
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)

		exitCode := 0
		if i%10 == 0 {
			exitCode = 1 // 10% failures
		}

		meta := &blocks.ExecuteMeta{
			Command:  "test command",
			ExitCode: ptr(exitCode),
		}
		blocks.SetExecuteMeta(block, meta)
		_ = tl.Append(block)
	}

	exitCode := 1
	filter := &blocks.Filter{
		ExitCode: &exitCode,
	}

	b.ResetTimer()

	// Benchmark: Filter by exit code
	for i := 0; i < b.N; i++ {
		tl.SetFilter(filter)
		_ = tl.GetVisibleBlocks()
		tl.ClearFilter()
	}
}

// BenchmarkTimelineNextBlock_10k measures focus navigation performance.
func BenchmarkTimelineNextBlock_10k(b *testing.B) {
	// Setup
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		_ = tl.Append(block)
	}

	// Focus first block
	firstBlock, _ := tl.GetByIndex(0)
	tl.FocusBlock(firstBlock.ID)

	b.ResetTimer()

	// Benchmark: Navigation through all blocks
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tl.FocusBlock(firstBlock.ID)
		b.StartTimer()

		for j := 0; j < 10000; j++ {
			tl.NextBlock()
		}
	}
}

// BenchmarkTimelineToggleFold_10k measures collapse/expand performance.
func BenchmarkTimelineToggleFold_10k(b *testing.B) {
	// Setup
	tl := blocks.NewTimeline()

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		block.Body = generateMockTranscript(100) // Large body
		_ = tl.Append(block)
	}

	// Get middle block ID
	middleBlock, _ := tl.GetByIndex(5000)

	b.ResetTimer()

	// Benchmark: Toggle fold state
	for i := 0; i < b.N; i++ {
		tl.ToggleFold(middleBlock.ID)
	}
}

// BenchmarkTimelineExpandAll_10k measures expand all performance.
func BenchmarkTimelineExpandAll_10k(b *testing.B) {
	// Setup
	tl := blocks.NewTimeline()

	for i := 0; i < 10000; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		block.FoldState = blocks.FoldStateCollapsed
		_ = tl.Append(block)
	}

	b.ResetTimer()

	// Benchmark: Expand all
	for i := 0; i < b.N; i++ {
		tl.ExpandAll()

		// Reset for next iteration
		b.StopTimer()
		tl.CollapseAll()
		b.StartTimer()
	}
}

// Helper: ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}

// Helper: Generate mock transcript
func generateMockTranscript(lines int) string {
	result := ""
	for i := 0; i < lines; i++ {
		result += fmt.Sprintf("Line %d: output from command execution\n", i)
	}
	return result
}
