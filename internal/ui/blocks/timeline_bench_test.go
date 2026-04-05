package blocks_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

// BenchmarkTimelineAppend_10k measures append performance for 10k blocks.
func BenchmarkTimelineAppend_10k(b *testing.B) {
	for range b.N {
		b.StopTimer()

		tl := blocks.NewTimeline()

		b.StartTimer()

		for j := range 10000 {
			block := blocks.NewBlock(blocks.BlockTypeExecute)
			block.Title = fmt.Sprintf("Block %d", j)

			if err := tl.Append(block); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkTimelineGetVisibleBlocks_10k measures viewport calculation for 10k blocks.
func BenchmarkTimelineGetVisibleBlocks_10k(b *testing.B) {
	// Setup: Create timeline with 10k blocks.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		block.Body = generateMockTranscript(50) // 50 lines.
		meta := &blocks.ExecuteMeta{
			Command:    "go test ./...",
			CWD:        "/home/user/project",
			Impact:     "medium",
			ExitCode:   ptr(0),
			DurationMS: ptr(int64(4200)),
			LinesOut:   ptr(50),
		}

		if err := blocks.SetExecuteMeta(block, meta); err != nil {
			b.Fatal(err)
		}

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	// Scroll to middle to test realistic scenario.
	tl.ScrollDown(5000)

	// Reset timer after setup.
	b.ResetTimer()

	// Benchmark: GetVisibleBlocks.
	for range b.N {
		_ = tl.GetVisibleBlocks()
	}
}

// BenchmarkTimelineGetVisibleBlocks_100k measures viewport calculation for 100k blocks.
func BenchmarkTimelineGetVisibleBlocks_100k(b *testing.B) {
	// Setup: Create timeline with 100k blocks.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 100000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	// Scroll to middle.
	tl.ScrollDown(50000)

	// Reset timer after setup.
	b.ResetTimer()

	// Benchmark.
	for range b.N {
		_ = tl.GetVisibleBlocks()
	}
}

// BenchmarkTimelineScrollDown_10k measures scroll performance.
func BenchmarkTimelineScrollDown_10k(b *testing.B) {
	// Setup.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	// Benchmark: Scroll down operations.
	for range b.N {
		b.StopTimer()
		tl.ScrollToTop()
		b.StartTimer()

		for range 100 {
			tl.ScrollDown(10)
		}
	}
}

// BenchmarkTimelineScrollPgDn_10k measures page down performance.
func BenchmarkTimelineScrollPgDn_10k(b *testing.B) {
	// Setup.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	// Benchmark: Page down operations (40-row viewport).
	for range b.N {
		b.StopTimer()
		tl.ScrollToTop()
		b.StartTimer()

		for range 250 { // 250 pages * 40 rows = 10k blocks.
			tl.ScrollDown(40)
		}
	}
}

// BenchmarkTimelineScrollToBottom_10k measures jump to bottom performance.
func BenchmarkTimelineScrollToBottom_10k(b *testing.B) {
	// Setup.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	// Benchmark: Jump operations.
	for range b.N {
		tl.ScrollToTop()
		tl.ScrollToBottom()
	}
}

// BenchmarkTimelineFilter_10k measures filter application performance.
func BenchmarkTimelineFilter_10k(b *testing.B) {
	// Setup: Create timeline with mixed block types.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 10000 {
		var blockType blocks.BlockType
		if i%2 == 0 {
			blockType = blocks.BlockTypeExecute
		} else {
			blockType = blocks.BlockTypePlan
		}

		block := blocks.NewBlock(blockType)
		block.Title = fmt.Sprintf("Block %d", i)

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	filter := &blocks.Filter{
		Types: []blocks.BlockType{blocks.BlockTypeExecute},
	}

	b.ResetTimer()

	// Benchmark: Filter application + get visible.
	for range b.N {
		tl.SetFilter(filter)
		_ = tl.GetVisibleBlocks()
		tl.ClearFilter()
		_ = tl.GetVisibleBlocks()
	}
}

// BenchmarkTimelineFilter_ExitCode_10k measures exit code filtering performance.
func BenchmarkTimelineFilter_ExitCode_10k(b *testing.B) {
	// Setup: Create timeline with varied exit codes.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)

		exitCode := 0
		if i%10 == 0 {
			exitCode = 1 // 10% failures.
		}

		meta := &blocks.ExecuteMeta{
			Command:  "test command",
			ExitCode: ptr(exitCode),
		}

		if err := blocks.SetExecuteMeta(block, meta); err != nil {
			b.Fatal(err)
		}

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	exitCode := 1
	filter := &blocks.Filter{
		ExitCode: &exitCode,
	}

	b.ResetTimer()

	// Benchmark: Filter by exit code.
	for range b.N {
		tl.SetFilter(filter)
		_ = tl.GetVisibleBlocks()
		tl.ClearFilter()
	}
}

// BenchmarkTimelineNextBlock_10k measures focus navigation performance.
func BenchmarkTimelineNextBlock_10k(b *testing.B) {
	// Setup.
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	// Focus first block.
	firstBlock, err := tl.GetByIndex(0)
	if err != nil {
		b.Fatal(err)
	}

	err = tl.FocusBlock(firstBlock.ID)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Benchmark: Navigation through all blocks.
	for range b.N {
		b.StopTimer()

		err = tl.FocusBlock(firstBlock.ID)
		if err != nil {
			b.Fatal(err)
		}

		b.StartTimer()

		for range 10000 {
			err = tl.NextBlock()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkTimelineToggleFold_10k measures collapse/expand performance.
func BenchmarkTimelineToggleFold_10k(b *testing.B) {
	// Setup.
	tl := blocks.NewTimeline()

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		block.Body = generateMockTranscript(100) // Large body.

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	// Get middle block ID.
	middleBlock, err := tl.GetByIndex(5000)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Benchmark: Toggle fold state.
	for range b.N {
		err = tl.ToggleFold(middleBlock.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTimelineExpandAll_10k measures expand all performance.
func BenchmarkTimelineExpandAll_10k(b *testing.B) {
	// Setup.
	tl := blocks.NewTimeline()

	for i := range 10000 {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.Title = fmt.Sprintf("Block %d", i)
		block.FoldState = blocks.FoldStateCollapsed

		if err := tl.Append(block); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	// Benchmark: Expand all.
	for range b.N {
		tl.ExpandAll()

		// Reset for next iteration.
		b.StopTimer()
		tl.CollapseAll()
		b.StartTimer()
	}
}

// Helper: ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// Helper: Generate mock transcript.
func generateMockTranscript(lines int) string {
	result := ""

	var resultSb316 strings.Builder

	for i := range lines {
		fmt.Fprintf(&resultSb316, "Line %d: output from command execution\n", i)
	}

	result += resultSb316.String()

	return result
}
