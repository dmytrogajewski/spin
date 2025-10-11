package blocks_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

// BenchmarkRendererRender_Execute measures EXECUTE block rendering performance.
func BenchmarkRendererRender_Execute(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	// Create realistic EXECUTE block with 500-line transcript
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.Title = "Run integration tests"
	block.Body = generateLargeTranscript(500)
	meta := &blocks.ExecuteMeta{
		Command:    "go test -race ./...",
		CWD:        "/home/user/project",
		Impact:     "high",
		TimeoutSec: 600,
		ExitCode:   ptr(0),
		DurationMS: ptr(int64(45200)),
		LinesOut:   ptr(500),
	}
	blocks.SetExecuteMeta(block, meta)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_, _ = renderer.Render(block)
	}
}

// BenchmarkRendererRender_Diff measures APPLY_PATCH block rendering performance.
func BenchmarkRendererRender_Diff(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	// Create realistic diff with 100 lines
	block := blocks.NewBlock(blocks.BlockTypeApplyPatch)
	block.Title = "Update authentication logic"
	block.Body = generateUnifiedDiff(100)
	meta := &blocks.PatchMeta{
		File:         "internal/auth/jwt.go",
		Succeeded:    true,
		LinesAdded:   ptr(42),
		LinesRemoved: ptr(18),
	}
	blocks.SetPatchMeta(block, meta)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_, _ = renderer.Render(block)
	}
}

// BenchmarkRendererRender_Code measures READ block rendering performance.
func BenchmarkRendererRender_Code(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	// Create realistic code block with 200 lines
	block := blocks.NewBlock(blocks.BlockTypeRead)
	block.Title = "View source file"
	block.Body = generateCodeSnippet(200)
	meta := &blocks.ReadMeta{
		File:   "internal/ui/prompt/model.go",
		Offset: 0,
		Limit:  200,
	}
	blocks.SetReadMeta(block, meta)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_, _ = renderer.Render(block)
	}
}

// BenchmarkRendererRender_Plan measures PLAN block rendering performance.
func BenchmarkRendererRender_Plan(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	// Create plan with 50 items
	block := blocks.NewBlock(blocks.BlockTypePlan)
	block.Title = "Implementation plan"
	block.Body = generatePlanList(50)
	meta := &blocks.PlanMeta{
		Total:      50,
		Pending:    10,
		InProgress: 5,
		Completed:  35,
	}
	blocks.SetPlanMeta(block, meta)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_, _ = renderer.Render(block)
	}
}

// BenchmarkRendererRender_Error measures ERROR block rendering performance.
func BenchmarkRendererRender_Error(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	// Create error block with stack trace
	block := blocks.NewBlock(blocks.BlockTypeError)
	block.Title = "Command execution failed"
	block.Body = generateErrorWithStackTrace(30)
	block.Severity = blocks.SeverityError

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_, _ = renderer.Render(block)
	}
}

// BenchmarkRendererRenderHeader measures header rendering performance.
func BenchmarkRendererRenderHeader(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.Title = "Run comprehensive test suite with race detector and coverage analysis"
	meta := &blocks.ExecuteMeta{
		Command:    "go test -race -coverprofile=coverage.out ./...",
		CWD:        "/home/user/very/long/project/path/with/many/nested/directories",
		Impact:     "high",
		TimeoutSec: 600,
	}
	blocks.SetExecuteMeta(block, meta)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_ = renderer.RenderHeader(block)
	}
}

// BenchmarkRendererRenderBody_Large measures large body rendering performance.
func BenchmarkRendererRenderBody_Large(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.Body = generateLargeTranscript(1000) // 1000 lines

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_, _ = renderer.RenderBody(block)
	}
}

// BenchmarkRendererRenderFooter measures footer rendering performance.
func BenchmarkRendererRenderFooter(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	block := blocks.NewBlock(blocks.BlockTypeExecute)
	meta := &blocks.ExecuteMeta{
		Command:    "go test ./...",
		ExitCode:   ptr(0),
		DurationMS: ptr(int64(45200)),
		LinesOut:   ptr(542),
	}
	blocks.SetExecuteMeta(block, meta)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_ = renderer.RenderFooter(block)
	}
}

// BenchmarkRenderViewport_40Blocks measures end-to-end viewport rendering.
func BenchmarkRenderViewport_40Blocks(b *testing.B) {
	renderer := blocks.NewRenderer(120)

	// Create timeline with mixed block types
	tl := blocks.NewTimeline()
	tl.SetViewportHeight(40)

	blockTypes := []blocks.BlockType{
		blocks.BlockTypeExecute,
		blocks.BlockTypePlan,
		blocks.BlockTypeRead,
		blocks.BlockTypeApplyPatch,
		blocks.BlockTypeGrep,
	}

	for i := 0; i < 100; i++ {
		blockType := blockTypes[i%len(blockTypes)]
		block := blocks.NewBlock(blockType)
		block.Title = fmt.Sprintf("Block %d", i)

		switch blockType {
		case blocks.BlockTypeExecute:
			block.Body = generateLargeTranscript(50)
			meta := &blocks.ExecuteMeta{
				Command:  "test command",
				CWD:      ".",
				Impact:   "medium",
				ExitCode: ptr(0),
			}
			blocks.SetExecuteMeta(block, meta)
		case blocks.BlockTypeApplyPatch:
			block.Body = generateUnifiedDiff(20)
		case blocks.BlockTypeRead:
			block.Body = generateCodeSnippet(30)
		}

		_ = tl.Append(block)
	}

	b.ResetTimer()

	// Benchmark: Get visible blocks + render all
	for i := 0; i < b.N; i++ {
		visibleBlocks := tl.GetVisibleBlocks()
		for _, block := range visibleBlocks {
			_, _ = renderer.Render(block)
		}
	}
}

// BenchmarkRendererSetWidth measures width update performance.
func BenchmarkRendererSetWidth(b *testing.B) {
	renderer := blocks.NewRenderer(80)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		renderer.SetWidth(120)
		renderer.SetWidth(80)
	}
}

// Helper: Generate large transcript
func generateLargeTranscript(lines int) string {
	var sb strings.Builder
	sb.Grow(lines * 80) // Pre-allocate

	for i := 0; i < lines; i++ {
		fmt.Fprintf(&sb, "=== RUN   TestFunction%d\n", i)
		fmt.Fprintf(&sb, "--- PASS: TestFunction%d (0.%02ds)\n", i, i%100)
	}

	return sb.String()
}

// Helper: Generate unified diff
func generateUnifiedDiff(lines int) string {
	var sb strings.Builder
	sb.Grow(lines * 100)

	sb.WriteString("@@ -42,6 +42,8 @@ func authenticate(token string) error {\n")
	for i := 0; i < lines; i++ {
		if i%5 == 0 {
			fmt.Fprintf(&sb, "+    log.Info(\"Processing line %d\")\n", i)
		} else if i%7 == 0 {
			fmt.Fprintf(&sb, "-    // Old comment line %d\n", i)
		} else {
			fmt.Fprintf(&sb, "     // Context line %d\n", i)
		}
	}

	return sb.String()
}

// Helper: Generate code snippet
func generateCodeSnippet(lines int) string {
	var sb strings.Builder
	sb.Grow(lines * 60)

	for i := 0; i < lines; i++ {
		fmt.Fprintf(&sb, "func example%d() {\n", i)
		sb.WriteString("    return nil\n")
		sb.WriteString("}\n")
		sb.WriteString("\n")
	}

	return sb.String()
}

// Helper: Generate plan list
func generatePlanList(items int) string {
	var sb strings.Builder
	sb.Grow(items * 50)

	for i := 0; i < items; i++ {
		status := "pending"
		if i < 35 {
			status = "completed"
		} else if i < 40 {
			status = "in_progress"
		}

		fmt.Fprintf(&sb, "✓ Task %d (%s)\n", i, status)
	}

	return sb.String()
}

// Helper: Generate error with stack trace
func generateErrorWithStackTrace(depth int) string {
	var sb strings.Builder

	sb.WriteString("Error: command execution failed with exit code 1\n")
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&sb, "    at function%d (file.go:%d)\n", i, i*10+42)
	}

	return sb.String()
}

