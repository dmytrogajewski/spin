// Package main provides a TUI blocks rendering example.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

const (
	exampleTimeout    = 600
	exampleDurationMS = 51
	exampleLinesOut   = 6
	exampleTotalSteps = 6
	examplePending    = 3
	exampleCompleted  = 2
	exampleLimit      = 50
	exampleContext    = 2
	exampleLinesAdded = 3
)

func main() {
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TUI: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startSignalHandler(cancel)
	startTUI(ctx, ui, cancel)
	printHeader(ui)
	createBlocks(ui)
	printInstructions(ui)
	runInputLoop(ctx, ui, cancel)
}

// startSignalHandler listens for interrupt signals and cancels the context.
func startSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()
}

// startTUI runs the TUI adapter in a background goroutine.
func startTUI(ctx context.Context, ui *adapters.PureTTY, cancel context.CancelFunc) {
	go func() {
		runErr := ui.Run(ctx)
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", runErr)
			cancel()
		}
	}()
}

// printHeader prints the application header.
func printHeader(ui *adapters.PureTTY) {
	lines := []string{
		"╔══════════════════════════════════════════════════════════╗",
		"║   Spin TUI - Block Types Demo                           ║",
		"╚══════════════════════════════════════════════════════════╝",
		"",
		"This demo shows all 9 block types with realistic examples.",
		"",
	}
	printLines(ui, lines)
}

// printInstructions prints usage instructions.
func printInstructions(ui *adapters.PureTTY) {
	lines := []string{
		"",
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
		"",
		"🎯 Try these actions:",
		"",
		"Navigation:",
		"  PgUp / PgDn     Scroll timeline by page",
		"  g / G           Jump to top / bottom",
		"  [ / ]           Previous / next block",
		"",
		"Block actions:",
		"  Enter           Toggle fold/expand block",
		"  y               Copy block body",
		"  S               Save block to file",
		"  r               Rerun EXECUTE block",
		"",
		"Advanced:",
		"  Ctrl-P          Command palette",
		"  /               Filter timeline",
		"  zR / zM         Expand / collapse all",
		"",
		"Type 'quit' to exit, or press Ctrl-D",
		"",
	}
	printLines(ui, lines)
}

// check panics if err is non-nil. Used in example/demo programs only.
func check(err error) {
	if err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

// printLines prints multiple lines to the TUI.
func printLines(ui *adapters.PureTTY, lines []string) {
	for _, line := range lines {
		check(ui.PrintLine(line))
	}
}

// runInputLoop handles user input until the context is canceled.
func runInputLoop(ctx context.Context, ui *adapters.PureTTY, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			check(ui.Stop())

			_, _ = fmt.Fprintln(os.Stdout, "\nGoodbye!")

			return

		case line, ok := <-ui.RequestInput():
			if !ok {
				check(ui.Stop())

				return
			}

			handleInput(ui, line, cancel)
		}
	}
}

// handleInput processes a single line of user input.
func handleInput(ui *adapters.PureTTY, line string, cancel context.CancelFunc) {
	switch line {
	case "quit", "exit", "q":
		cancel()
	case "help", "h", "?":
		check(ui.PrintLine(""))
		check(ui.PrintLine("See navigation and block actions above."))
		check(ui.PrintLine(""))
	case "":
		// Ignore empty lines.
	default:
		check(ui.PrintLine(fmt.Sprintf("Echo: %s", line)))
	}
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int { return &i }

// int64Ptr returns a pointer to an int64.
func int64Ptr(i int64) *int64 { return &i }

// createBlocks creates sample blocks of all types.
func createBlocks(ui *adapters.PureTTY) {
	createExecuteBlock(ui)
	createPlanBlock(ui)
	createReadBlock(ui)
	createGrepBlock(ui)
	createPatchBlock(ui)
	createSummaryBlock(ui)
	createTestingBlock(ui)
	createNoticeBlock(ui)
	createErrorBlock(ui)

	check(ui.PrintLine(""))
	check(ui.PrintLine("✅ Created 9 blocks (one of each type)"))
}

// createExecuteBlock creates a sample EXECUTE block.
func createExecuteBlock(ui *adapters.PureTTY) {
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
		TimeoutSec: exampleTimeout,
		Impact:     "medium",
		ExitCode:   intPtr(0),
		DurationMS: int64Ptr(exampleDurationMS),
		LinesOut:   intPtr(exampleLinesOut),
	}

	if err := blocks.SetExecuteMeta(execBlock, execMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting execute meta: %v\n", err)
	}

	check(ui.AppendBlock(execBlock))
}

// createPlanBlock creates a sample PLAN block.
func createPlanBlock(ui *adapters.PureTTY) {
	planBlock := blocks.NewBlock(blocks.BlockTypePlan)
	planBlock.Title = "Implementation plan"
	planBlock.Body = `✓ Install dependencies (completed)
✓ Create main.go skeleton (completed)
◦ Write unit tests (in progress)
• Add integration tests (pending)
• Write documentation (pending)
• Deploy to staging (pending)`

	planMeta := &blocks.PlanMeta{
		Total:      exampleTotalSteps,
		Pending:    examplePending,
		InProgress: 1,
		Completed:  exampleCompleted,
	}

	if err := blocks.SetPlanMeta(planBlock, planMeta); err != nil {
		log.Printf("Failed to set plan metadata: %v", err)
	}

	check(ui.AppendBlock(planBlock))
}

// createReadBlock creates a sample READ block.
func createReadBlock(ui *adapters.PureTTY) {
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
		Limit:  exampleLimit,
	}

	if err := blocks.SetReadMeta(readBlock, readMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting read meta: %v\n", err)
	}

	check(ui.AppendBlock(readBlock))
}

// createGrepBlock creates a sample GREP block.
func createGrepBlock(ui *adapters.PureTTY) {
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
		Context: exampleContext,
	}

	if err := blocks.SetGrepMeta(grepBlock, grepMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting grep meta: %v\n", err)
	}

	check(ui.AppendBlock(grepBlock))
}

// createPatchBlock creates a sample APPLY_PATCH block.
func createPatchBlock(ui *adapters.PureTTY) {
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
		LinesAdded:   intPtr(exampleLinesAdded),
		LinesRemoved: intPtr(0),
	}

	if err := blocks.SetPatchMeta(patchBlock, patchMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting patch meta: %v\n", err)
	}

	check(ui.AppendBlock(patchBlock))
}

// createSummaryBlock creates a sample SUMMARY block.
func createSummaryBlock(ui *adapters.PureTTY) {
	summaryBlock := blocks.NewBlock(blocks.BlockTypeSummary)
	summaryBlock.Title = "Changes summary"
	summaryBlock.Body = `Added error handling to the process() function:

• Added validation check before doWork()
• Log fatal error if validation fails
• Ensures clean shutdown on validation errors

Files modified:
• main.go (+3 lines)
• No breaking changes`

	check(ui.AppendBlock(summaryBlock))
}

// createTestingBlock creates a sample TESTING block.
func createTestingBlock(ui *adapters.PureTTY) {
	testingBlock := blocks.NewBlock(blocks.BlockTypeTesting)
	testingBlock.Title = "Test plan"
	testingBlock.Body = `✓ go test -race ./internal/... (passed, 0.5s)
✓ go test -bench=. ./... (passed, 2.1s)
✗ integration tests (failed, 5.2s)
    Error: database connection timeout
    Re-run: make test-integration`

	check(ui.AppendBlock(testingBlock))
}

// createNoticeBlock creates a sample NOTICE block.
func createNoticeBlock(ui *adapters.PureTTY) {
	noticeBlock := blocks.NewBlock(blocks.BlockTypeNotice)
	noticeBlock.Title = "System notice"
	noticeBlock.Body = `Conversation history has been compressed to reduce context size.

Previous messages have been summarized. Full history available in:
~/.spin/sessions/20251011-103042.json`
	noticeBlock.Severity = blocks.SeverityInfo

	check(ui.AppendBlock(noticeBlock))
}

// createErrorBlock creates a sample ERROR block.
func createErrorBlock(ui *adapters.PureTTY) {
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

	check(ui.AppendBlock(errorBlock))
}
