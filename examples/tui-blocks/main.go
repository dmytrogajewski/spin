package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

func main() {
	// Create PureTTY adapter with block rendering
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TUI: %v\n", err)
		os.Exit(1)
	}

	// Set up context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Start TUI
	go func() {
		if err := ui.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			cancel()
		}
	}()

	// Print header
	ui.PrintLine("╔══════════════════════════════════════════════════════════╗")
	ui.PrintLine("║   Spin TUI - Block Types Demo                           ║")
	ui.PrintLine("╚══════════════════════════════════════════════════════════╝")
	ui.PrintLine("")
	ui.PrintLine("This demo shows all 9 block types with realistic examples.")
	ui.PrintLine("")

	// Create blocks and append to timeline
	createBlocks(ui)

	// Instructions
	ui.PrintLine("")
	ui.PrintLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	ui.PrintLine("")
	ui.PrintLine("🎯 Try these actions:")
	ui.PrintLine("")
	ui.PrintLine("Navigation:")
	ui.PrintLine("  PgUp / PgDn     Scroll timeline by page")
	ui.PrintLine("  g / G           Jump to top / bottom")
	ui.PrintLine("  [ / ]           Previous / next block")
	ui.PrintLine("")
	ui.PrintLine("Block actions:")
	ui.PrintLine("  Enter           Toggle fold/expand block")
	ui.PrintLine("  y               Copy block body")
	ui.PrintLine("  S               Save block to file")
	ui.PrintLine("  r               Rerun EXECUTE block")
	ui.PrintLine("")
	ui.PrintLine("Advanced:")
	ui.PrintLine("  Ctrl-P          Command palette")
	ui.PrintLine("  /               Filter timeline")
	ui.PrintLine("  zR / zM         Expand / collapse all")
	ui.PrintLine("")
	ui.PrintLine("Type 'quit' to exit, or press Ctrl-D")
	ui.PrintLine("")

	// Main loop
	for {
		select {
		case <-ctx.Done():
			ui.Stop()
			fmt.Println("\nGoodbye!")
			return

		case line, ok := <-ui.RequestInput():
			if !ok {
				ui.Stop()
				return
			}

			switch line {
			case "quit", "exit", "q":
				cancel()
			case "help", "h", "?":
				ui.PrintLine("")
				ui.PrintLine("See navigation and block actions above.")
				ui.PrintLine("")
			case "":
				// Ignore empty lines
			default:
				ui.PrintLine(fmt.Sprintf("Echo: %s", line))
			}
		}
	}
}

// createBlocks creates sample blocks of all types
func createBlocks(ui *adapters.PureTTY) {
	// Helper to create int pointer
	intPtr := func(i int) *int { return &i }
	int64Ptr := func(i int64) *int64 { return &i }

	// 1. EXECUTE block (success)
	execBlock := blocks.NewBlock(blocks.BlockTypeExecute)
	execBlock.Title = "Run tests"
	execBlock.Body = `=== RUN   TestCalculate
--- PASS: TestCalculate (0.00s)
=== RUN   TestValidate
--- PASS: TestValidate (0.05s)
PASS
ok      github.com/user/project    0.051s`

	execMeta := &blocks.ExecuteMeta{
		Command:    "go test -race ./...",
		CWD:        "./",
		TimeoutSec: 600,
		Impact:     "medium",
		ExitCode:   intPtr(0),
		DurationMS: int64Ptr(51),
		LinesOut:   intPtr(6),
	}
	if err := blocks.SetExecuteMeta(execBlock, execMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting execute meta: %v\n", err)
	}
	ui.AppendBlock(execBlock)

	// 2. PLAN block
	planBlock := blocks.NewBlock(blocks.BlockTypePlan)
	planBlock.Title = "Implementation plan"
	planBlock.Body = `✓ Install dependencies (completed)
✓ Create main.go skeleton (completed)
◦ Write unit tests (in progress)
• Add integration tests (pending)
• Write documentation (pending)
• Deploy to staging (pending)`

	planBlock.Meta = map[string]interface{}{
		"total":       6,
		"pending":     3,
		"in_progress": 1,
		"completed":   2,
	}
	ui.AppendBlock(planBlock)

	// 3. READ block
	readBlock := blocks.NewBlock(blocks.BlockTypeRead)
	readBlock.Title = "internal/tui/input.go"
	readBlock.Body = `package tui

import (
    "context"
    "io"
)

// Input handles user input with history and completion
type Input struct {
    buffer  []rune
    cursor  int
    history []string
}

// NewInput creates a new input handler
func NewInput() *Input {
    return &Input{
        buffer:  make([]rune, 0, 256),
        history: make([]string, 0, 100),
    }
}`

	readMeta := &blocks.ReadMeta{
		File:   "internal/tui/input.go",
		Offset: 0,
		Limit:  50,
	}
	if err := blocks.SetReadMeta(readBlock, readMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting read meta: %v\n", err)
	}
	ui.AppendBlock(readBlock)

	// 4. GREP block
	grepBlock := blocks.NewBlock(blocks.BlockTypeGrep)
	grepBlock.Title = "Search results"
	grepBlock.Body = `main.go:42:
40:  func process() {
41:      // TODO: Add error handling
42:      fmt.Println("processing")
43:  }

utils.go:18:
16:  func helper() {
17:      // TODO: Optimize this loop
18:      for i := 0; i < n; i++ {
19:  }`

	grepMeta := &blocks.GrepMeta{
		Pattern: "TODO",
		Mode:    "content",
		Context: 2,
	}
	if err := blocks.SetGrepMeta(grepBlock, grepMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting grep meta: %v\n", err)
	}
	ui.AppendBlock(grepBlock)

	// 5. APPLY_PATCH block (success)
	patchBlock := blocks.NewBlock(blocks.BlockTypeApplyPatch)
	patchBlock.Title = "Add error handling"
	patchBlock.Body = `@@ -15,6 +15,9 @@ func main() {
 func process() {
     fmt.Println("start")
+    if err := validate(); err != nil {
+        log.Fatal(err)
+    }
     doWork()
     fmt.Println("done")
 }`

	patchMeta := &blocks.PatchMeta{
		File:         "main.go",
		Succeeded:    true,
		LinesAdded:   intPtr(3),
		LinesRemoved: intPtr(0),
	}
	if err := blocks.SetPatchMeta(patchBlock, patchMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting patch meta: %v\n", err)
	}
	ui.AppendBlock(patchBlock)

	// 6. SUMMARY block
	summaryBlock := blocks.NewBlock(blocks.BlockTypeSummary)
	summaryBlock.Title = "Changes summary"
	summaryBlock.Body = `Added error handling to the process() function:

• Added validation check before doWork()
• Log fatal error if validation fails
• Ensures clean shutdown on validation errors

Files modified:
• main.go (+3 lines)
• No breaking changes`

	ui.AppendBlock(summaryBlock)

	// 7. TESTING block (mixed results)
	testingBlock := blocks.NewBlock(blocks.BlockTypeTesting)
	testingBlock.Title = "Test plan"
	testingBlock.Body = `✓ go test -race ./internal/... (passed, 0.5s)
✓ go test -bench=. ./... (passed, 2.1s)
✗ integration tests (failed, 5.2s)
    Error: database connection timeout
    Re-run: make test-integration`

	// TESTING block currently has no specific metadata type
	// Can use generic Meta map or leave empty
	ui.AppendBlock(testingBlock)

	// 8. NOTICE block
	noticeBlock := blocks.NewBlock(blocks.BlockTypeNotice)
	noticeBlock.Title = "System notice"
	noticeBlock.Body = `Conversation history has been compressed to reduce context size.

Previous messages have been summarized. Full history available in:
~/.spin/sessions/20251011-103042.json`
	noticeBlock.Severity = blocks.SeverityInfo

	ui.AppendBlock(noticeBlock)

	// 9. ERROR block
	errorBlock := blocks.NewBlock(blocks.BlockTypeError)
	errorBlock.Title = "Command failed"
	errorBlock.Body = `Error: exit status 1

Stack trace:
  at processFile (internal/core/exec.go:142)
  at runCommand (internal/core/agent.go:89)
  at handleTurn (internal/core/session.go:67)

Suggestion: Check file permissions and retry with sudo`
	errorBlock.Severity = blocks.SeverityError

	errorMeta := &blocks.ExecuteMeta{
		Command:  "make build",
		ExitCode: intPtr(1),
	}
	if err := blocks.SetExecuteMeta(errorBlock, errorMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting error meta: %v\n", err)
	}
	ui.AppendBlock(errorBlock)

	// Print separator
	ui.PrintLine("")
	ui.PrintLine("✅ Created 9 blocks (one of each type)")
}
