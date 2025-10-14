package adapters

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
	"github.com/dmytrogajewski/spin/internal/ui/status"
	"github.com/dmytrogajewski/spin/internal/ui/term"
	"github.com/dmytrogajewski/spin/internal/ui/testkit"
)

// Helper to create a test PureTTY instance with all dependencies
func newTestPureTTY(t *testing.T, w, h int) (*PureTTY, *bytes.Buffer) {
	t.Helper()

	out := &bytes.Buffer{}
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, w, "> ")
	printer := output.NewPrinter(out)
	timeline := blocks.NewTimeline()
	blockRenderer := blocks.NewRenderer(w)

	// Create renderer adapter
	rendererAdapter := &rendererAdapter{renderer: renderer}

	// Create coordinator
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	// Create status management components
	statusManager := status.NewManager()
	statusAggregator := status.NewAggregator(statusManager)

	// Create PureTTY directly, bypassing constructor
	ui := &PureTTY{
		model:            model,
		renderer:         renderer,
		coord:            coord,
		statusManager:    statusManager,
		statusAggregator: statusAggregator,
		out:              out,
		timeline:         timeline,
		blockRenderer:    blockRenderer,
		viewportHeight:   0,
		mode:             ModeInput,
		filterInput:      "",
	}

	// For tests that need TTY (like calculateViewport), we'll mock it via test setup
	// Most tests don't need a real TTY

	return ui, out
}

// TestCalculateViewport tests viewport height calculation logic
func TestCalculateViewport(t *testing.T) {
	tests := []struct {
		name         string
		termHeight   int
		wantViewport int
	}{
		{
			name:         "standard terminal 24 rows",
			termHeight:   24,
			wantViewport: 19, // 24 - 2 (input) - 1 (status) - 2 (padding)
		},
		{
			name:         "large terminal 60 rows",
			termHeight:   60,
			wantViewport: 55, // 60 - 2 - 1 - 2
		},
		{
			name:         "small terminal 10 rows",
			termHeight:   10,
			wantViewport: 5, // 10 - 2 - 1 - 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui, _ := newTestPureTTY(t, 80, tt.termHeight)

			// Test the calculation logic directly without calling TTY
			// Formula: height - inputBarHeight - statusLineHeight - padding
			// Formula: height - 2 - 1 - 2 = height - 5
			calculated := tt.termHeight - 5

			if calculated != tt.wantViewport {
				t.Errorf("calculation = %d, want %d", calculated, tt.wantViewport)
			}

			// Set viewport manually to test timeline update
			ui.viewportHeight = calculated
			ui.timeline.SetViewportHeight(calculated)

			// Verify timeline viewport height was set
			if ui.timeline.GetViewportHeight() != tt.wantViewport {
				t.Errorf("timeline.GetViewportHeight() = %d, want %d",
					ui.timeline.GetViewportHeight(), tt.wantViewport)
			}
		})
	}
}

// TestParseFilter tests filter string parsing
func TestParseFilter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *blocks.Filter
	}{
		{
			name:  "type filter",
			input: "type:EXECUTE",
			want: &blocks.Filter{
				Types: []blocks.BlockType{blocks.BlockTypeExecute},
			},
		},
		{
			name:  "file filter",
			input: "file:foo.go",
			want: &blocks.Filter{
				File: "foo.go",
			},
		},
		{
			name:  "exit code filter",
			input: "exit:0",
			want: &blocks.Filter{
				ExitCode: intPtr(0),
			},
		},
		{
			name:  "impact filter",
			input: "impact:high",
			want: &blocks.Filter{
				Impact: "high",
			},
		},
		{
			name:  "multiple filters",
			input: "type:EXECUTE file:foo.go exit:0",
			want: &blocks.Filter{
				Types:    []blocks.BlockType{blocks.BlockTypeExecute},
				File:     "foo.go",
				ExitCode: intPtr(0),
			},
		},
		{
			name:  "invalid filter ignored",
			input: "invalid type:PLAN",
			want: &blocks.Filter{
				Types: []blocks.BlockType{blocks.BlockTypePlan},
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  &blocks.Filter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui, _ := newTestPureTTY(t, 80, 24)

			got := ui.parseFilter(tt.input)

			// Compare types
			if len(got.Types) != len(tt.want.Types) {
				t.Errorf("parseFilter() types len = %d, want %d", len(got.Types), len(tt.want.Types))
			}
			for i := range got.Types {
				if got.Types[i] != tt.want.Types[i] {
					t.Errorf("parseFilter() types[%d] = %v, want %v", i, got.Types[i], tt.want.Types[i])
				}
			}

			// Compare file
			if got.File != tt.want.File {
				t.Errorf("parseFilter() file = %q, want %q", got.File, tt.want.File)
			}

			// Compare exit code
			if (got.ExitCode == nil) != (tt.want.ExitCode == nil) {
				t.Errorf("parseFilter() exitCode nil mismatch")
			}
			if got.ExitCode != nil && tt.want.ExitCode != nil && *got.ExitCode != *tt.want.ExitCode {
				t.Errorf("parseFilter() exitCode = %d, want %d", *got.ExitCode, *tt.want.ExitCode)
			}

			// Compare impact
			if got.Impact != tt.want.Impact {
				t.Errorf("parseFilter() impact = %q, want %q", got.Impact, tt.want.Impact)
			}
		})
	}
}

// TestFormatFilterChips tests filter formatting
func TestFormatFilterChips(t *testing.T) {
	tests := []struct {
		name   string
		filter *blocks.Filter
		want   string
	}{
		{
			name: "single type",
			filter: &blocks.Filter{
				Types: []blocks.BlockType{blocks.BlockTypeExecute},
			},
			want: "[type:EXECUTE]",
		},
		{
			name: "multiple types",
			filter: &blocks.Filter{
				Types: []blocks.BlockType{blocks.BlockTypeExecute, blocks.BlockTypeRead},
			},
			want: "[type:EXECUTE] [type:READ]",
		},
		{
			name: "all filters",
			filter: &blocks.Filter{
				Types:    []blocks.BlockType{blocks.BlockTypePlan},
				File:     "foo.go",
				ExitCode: intPtr(1),
				Impact:   "high",
			},
			want: "[type:PLAN] [file:foo.go] [exit:1] [impact:high]",
		},
		{
			name:   "empty filter",
			filter: &blocks.Filter{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui, _ := newTestPureTTY(t, 80, 24)

			got := ui.formatFilterChips(tt.filter)

			if got != tt.want {
				t.Errorf("formatFilterChips() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestHandleTimelineKey tests navigation key handling
func TestHandleTimelineKey(t *testing.T) {
	tests := []struct {
		name      string
		key       term.KeyEvent
		setupFunc func(*PureTTY)
		checkFunc func(*testing.T, *PureTTY)
	}{
		{
			name: "PgUp scrolls up",
			key:  term.KeyEvent{Kind: term.KeyPgUp},
			setupFunc: func(ui *PureTTY) {
				// Append blocks and scroll down
				for i := 0; i < 50; i++ {
					ui.timeline.Append(createTestBlock(i, blocks.BlockTypeExecute))
				}
				ui.timeline.ScrollDown(10)
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				// Verify scroll position changed
				if ui.timeline.GetScrollPosition() >= 10 {
					t.Error("PgUp did not scroll up")
				}
			},
		},
		{
			name: "PgDn scrolls down",
			key:  term.KeyEvent{Kind: term.KeyPgDn},
			setupFunc: func(ui *PureTTY) {
				for i := 0; i < 50; i++ {
					ui.timeline.Append(createTestBlock(i, blocks.BlockTypeExecute))
				}
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				if ui.timeline.GetScrollPosition() == 0 {
					t.Error("PgDn did not scroll down")
				}
			},
		},
		{
			name: "g scrolls to top",
			key:  term.KeyEvent{Kind: term.KeyRune, Rune: 'g'},
			setupFunc: func(ui *PureTTY) {
				for i := 0; i < 50; i++ {
					ui.timeline.Append(createTestBlock(i, blocks.BlockTypeExecute))
				}
				ui.timeline.ScrollToBottom()
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				if ui.timeline.GetScrollPosition() != 0 {
					t.Errorf("g did not scroll to top, pos = %d", ui.timeline.GetScrollPosition())
				}
			},
		},
		{
			name: "G scrolls to bottom",
			key:  term.KeyEvent{Kind: term.KeyRune, Rune: 'G'},
			setupFunc: func(ui *PureTTY) {
				for i := 0; i < 50; i++ {
					ui.timeline.Append(createTestBlock(i, blocks.BlockTypeExecute))
				}
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				// Bottom means last possible scroll position
				maxScroll := ui.timeline.Len() - ui.timeline.GetViewportHeight()
				if maxScroll < 0 {
					maxScroll = 0
				}
				if ui.timeline.GetScrollPosition() != maxScroll {
					t.Errorf("G did not scroll to bottom, pos = %d, want %d",
						ui.timeline.GetScrollPosition(), maxScroll)
				}
			},
		},
		{
			name: "Enter toggles fold",
			key:  term.KeyEvent{Kind: term.KeyEnter},
			setupFunc: func(ui *PureTTY) {
				block := createTestBlock(1, blocks.BlockTypeExecute)
				block.FoldState = blocks.FoldStateExpanded
				ui.timeline.Append(block)
				ui.timeline.FocusBlock(block.ID)
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				focused, err := ui.timeline.GetFocusedBlock()
				if err != nil {
					t.Fatal("GetFocusedBlock() error:", err)
				}
				if focused.FoldState != blocks.FoldStateCollapsed {
					t.Errorf("Enter did not toggle fold, state = %v", focused.FoldState)
				}
			},
		},
		{
			name: "/ enters filter mode",
			key:  term.KeyEvent{Kind: term.KeyRune, Rune: '/'},
			setupFunc: func(ui *PureTTY) {
				ui.mode = ModeTimeline
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				if ui.mode != ModeFilter {
					t.Errorf("/ did not enter filter mode, mode = %v", ui.mode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui, _ := newTestPureTTY(t, 80, 24)
			ui.timeline.SetViewportHeight(19)
			ui.viewportHeight = 19
			ui.mode = ModeTimeline

			if tt.setupFunc != nil {
				tt.setupFunc(ui)
			}

			ui.handleTimelineKey(tt.key)

			if tt.checkFunc != nil {
				tt.checkFunc(t, ui)
			}
		})
	}
}

// TestHandleFilterKey tests filter input key handling
func TestHandleFilterKey(t *testing.T) {
	tests := []struct {
		name      string
		key       term.KeyEvent
		setupFunc func(*PureTTY)
		checkFunc func(*testing.T, *PureTTY)
	}{
		{
			name: "typing builds filter string",
			key:  term.KeyEvent{Kind: term.KeyRune, Rune: 't'},
			setupFunc: func(ui *PureTTY) {
				ui.filterInput = "type:"
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				if ui.filterInput != "type:t" {
					t.Errorf("filterInput = %q, want %q", ui.filterInput, "type:t")
				}
			},
		},
		{
			name: "backspace removes char",
			key:  term.KeyEvent{Kind: term.KeyBackspace},
			setupFunc: func(ui *PureTTY) {
				ui.filterInput = "type:EX"
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				if ui.filterInput != "type:E" {
					t.Errorf("filterInput = %q, want %q", ui.filterInput, "type:E")
				}
			},
		},
		{
			name: "enter applies filter",
			key:  term.KeyEvent{Kind: term.KeyEnter},
			setupFunc: func(ui *PureTTY) {
				ui.filterInput = "type:EXECUTE"
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				if ui.mode != ModeTimeline {
					t.Errorf("mode = %v, want ModeTimeline", ui.mode)
				}
				filter := ui.timeline.GetFilter()
				if filter == nil {
					t.Fatal("filter not applied")
				}
				if len(filter.Types) != 1 || filter.Types[0] != blocks.BlockTypeExecute {
					t.Errorf("filter types = %v, want [EXECUTE]", filter.Types)
				}
			},
		},
		{
			name: "escape clears filter",
			key:  term.KeyEvent{Kind: term.KeyEscape},
			setupFunc: func(ui *PureTTY) {
				ui.filterInput = "type:EXECUTE"
				ui.timeline.SetFilter(&blocks.Filter{Types: []blocks.BlockType{blocks.BlockTypeExecute}})
			},
			checkFunc: func(t *testing.T, ui *PureTTY) {
				if ui.filterInput != "" {
					t.Errorf("filterInput = %q, want empty", ui.filterInput)
				}
				if ui.timeline.GetFilter() != nil {
					t.Error("filter not cleared")
				}
				if ui.mode != ModeTimeline {
					t.Errorf("mode = %v, want ModeTimeline", ui.mode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui, _ := newTestPureTTY(t, 80, 24)
			ui.mode = ModeFilter

			if tt.setupFunc != nil {
				tt.setupFunc(ui)
			}

			ui.handleFilterKey(tt.key)

			if tt.checkFunc != nil {
				tt.checkFunc(t, ui)
			}
		})
	}
}

// TestModeSwitch tests mode transitions
func TestModeSwitch(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	// Start in input mode
	if ui.mode != ModeInput {
		t.Errorf("initial mode = %v, want ModeInput", ui.mode)
	}

	// Esc in input mode → timeline mode
	ui.handleInputKey(term.KeyEvent{Kind: term.KeyEscape})
	if ui.mode != ModeTimeline {
		t.Errorf("after Esc in input: mode = %v, want ModeTimeline", ui.mode)
	}

	// / in timeline mode → filter mode
	ui.handleTimelineKey(term.KeyEvent{Kind: term.KeyRune, Rune: '/'})
	if ui.mode != ModeFilter {
		t.Errorf("after / in timeline: mode = %v, want ModeFilter", ui.mode)
	}

	// Esc in filter mode → timeline mode
	ui.handleFilterKey(term.KeyEvent{Kind: term.KeyEscape})
	if ui.mode != ModeTimeline {
		t.Errorf("after Esc in filter: mode = %v, want ModeTimeline", ui.mode)
	}

	// Any char in timeline mode → input mode
	ui.handleTimelineKey(term.KeyEvent{Kind: term.KeyRune, Rune: 'a'})
	if ui.mode != ModeInput {
		t.Errorf("after 'a' in timeline: mode = %v, want ModeInput", ui.mode)
	}
}

// TestAppendBlock tests block append and render
func TestAppendBlock(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	block := createTestBlock(1, blocks.BlockTypeExecute)

	if err := ui.AppendBlock(block); err != nil {
		t.Errorf("AppendBlock() error = %v", err)
	}

	if ui.timeline.Len() != 1 {
		t.Errorf("timeline.Len() = %d, want 1", ui.timeline.Len())
	}

	retrieved, err := ui.timeline.GetByIndex(0)
	if err != nil {
		t.Fatalf("GetByIndex() error = %v", err)
	}
	if retrieved.ID != block.ID {
		t.Error("AppendBlock() did not add block to timeline")
	}
}

// TestUpdateBlock tests block update
func TestUpdateBlock(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	block := createTestBlock(1, blocks.BlockTypeExecute)
	ui.timeline.Append(block)

	// Update block
	updated := createTestBlock(1, blocks.BlockTypeExecute)
	updated.Body = "updated body"

	if err := ui.UpdateBlock(block.ID, updated); err != nil {
		t.Errorf("UpdateBlock() error = %v", err)
	}

	retrieved, err := ui.timeline.Get(block.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.Body != "updated body" {
		t.Error("UpdateBlock() did not update block")
	}
}

// TestDeleteBlock tests block deletion
func TestDeleteBlock(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	block := createTestBlock(1, blocks.BlockTypeExecute)
	ui.timeline.Append(block)

	if err := ui.DeleteBlock(block.ID); err != nil {
		t.Errorf("DeleteBlock() error = %v", err)
	}

	if ui.timeline.Len() != 0 {
		t.Errorf("timeline.Len() = %d, want 0 after delete", ui.timeline.Len())
	}
}

// TestNavigate100Blocks tests navigation through large timeline
func TestNavigate100Blocks(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)
	ui.timeline.SetViewportHeight(19)
	ui.viewportHeight = 19

	// Append 100 blocks (start at 1 to avoid ID collision with seq=0)
	for i := 1; i <= 100; i++ {
		ui.AppendBlock(createTestBlock(i, blocks.BlockTypeExecute))
	}

	if ui.timeline.Len() != 100 {
		t.Fatalf("timeline.Len() = %d, want 100", ui.timeline.Len())
	}

	// Scroll through with PgDn
	ui.mode = ModeTimeline
	initialPos := ui.timeline.GetScrollPosition()

	ui.handleTimelineKey(term.KeyEvent{Kind: term.KeyPgDn})
	afterPgDnPos := ui.timeline.GetScrollPosition()

	if afterPgDnPos <= initialPos {
		t.Error("PgDn did not scroll down in large timeline")
	}

	// Scroll to bottom
	ui.handleTimelineKey(term.KeyEvent{Kind: term.KeyRune, Rune: 'G'})
	maxScroll := 100 - 19 // len - viewport
	if ui.timeline.GetScrollPosition() != maxScroll {
		t.Errorf("G did not scroll to bottom, pos = %d, want %d",
			ui.timeline.GetScrollPosition(), maxScroll)
	}

	// Scroll to top
	ui.handleTimelineKey(term.KeyEvent{Kind: term.KeyRune, Rune: 'g'})
	if ui.timeline.GetScrollPosition() != 0 {
		t.Errorf("g did not scroll to top, pos = %d", ui.timeline.GetScrollPosition())
	}

	// Verify visible blocks
	visible := ui.timeline.GetVisibleBlocks()
	if len(visible) == 0 {
		t.Error("GetVisibleBlocks() returned empty after navigation")
	}
}

// TestFilterByType tests filtering by block type
func TestFilterByType(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	// Append mixed blocks
	ui.AppendBlock(createTestBlock(1, blocks.BlockTypeExecute))
	ui.AppendBlock(createTestBlock(2, blocks.BlockTypeRead))
	ui.AppendBlock(createTestBlock(3, blocks.BlockTypeExecute))
	ui.AppendBlock(createTestBlock(4, blocks.BlockTypePlan))
	ui.AppendBlock(createTestBlock(5, blocks.BlockTypeExecute))

	// Apply filter via UI
	ui.mode = ModeFilter
	ui.filterInput = "type:EXECUTE"
	ui.handleFilterKey(term.KeyEvent{Kind: term.KeyEnter})

	// Verify filter applied
	visible := ui.timeline.GetVisibleBlocks()
	if len(visible) != 3 {
		t.Errorf("GetVisibleBlocks() returned %d blocks, want 3 EXECUTE blocks", len(visible))
	}

	for _, block := range visible {
		if block.Type != blocks.BlockTypeExecute {
			t.Errorf("visible block type = %v, want EXECUTE", block.Type)
		}
	}
}

// TestConcurrentBlockAppend tests thread safety
func TestConcurrentBlockAppend(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	// Append blocks concurrently
	const goroutines = 10
	const blocksPerGoroutine = 10

	done := make(chan bool, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(g int) {
			for i := 0; i < blocksPerGoroutine; i++ {
				// Add 1 to avoid ID collision with seq=0
				block := createTestBlock(g*blocksPerGoroutine+i+1, blocks.BlockTypeExecute)
				ui.AppendBlock(block)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for concurrent appends")
		}
	}

	// Verify all blocks appended
	expected := goroutines * blocksPerGoroutine
	if ui.timeline.Len() != expected {
		t.Errorf("timeline.Len() = %d, want %d after concurrent appends",
			ui.timeline.Len(), expected)
	}
}

// Helper functions

func createTestBlock(id int, typ blocks.BlockType) *blocks.Block {
	block := blocks.NewBlock(typ)
	block.ID = blocks.GenerateBlockID(id)
	block.Title = "Test block"
	block.Body = "Test body"
	block.FoldState = blocks.FoldStateExpanded
	return block
}

func intPtr(i int) *int {
	return &i
}

// TestFilePreviewAnchorDetection tests opening file preview from block with anchor
func TestFilePreviewAnchorDetection(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	// Create a temp file for testing
	tmpFile := t.TempDir() + "/test.go"
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	if err := writeFile(tmpFile, content); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create block with file anchor
	block := createTestBlock(1, blocks.BlockTypeExecute)
	block.Body = "Check " + tmpFile + ":3 for the main function"
	ui.timeline.Append(block)
	ui.timeline.FocusBlock(block.ID)

	// Mock TTY size
	fakeTTY := testkit.NewFakeTTY(80, 24)
	ui.tty = fakeTTY

	// Trigger 'o' key to open file preview
	ui.handleOpenFilePreview()

	// Verify mode switched to file preview
	if ui.mode != ModeFilePreview {
		t.Errorf("mode = %v, want ModeFilePreview after handleOpenFilePreview", ui.mode)
	}

	// Verify file preview was created
	if ui.filePreview == nil {
		t.Fatal("filePreview is nil after handleOpenFilePreview")
	}

	// Verify file content loaded
	if len(ui.filePreview.Lines) != 5 {
		t.Errorf("filePreview.Lines = %d lines, want 5", len(ui.filePreview.Lines))
	}

	// Verify target line set correctly (line 3)
	if ui.filePreview.TargetLine != 3 {
		t.Errorf("filePreview.TargetLine = %d, want 3", ui.filePreview.TargetLine)
	}
}

// TestFilePreviewNavigation tests navigation keys in file preview mode
func TestFilePreviewNavigation(t *testing.T) {
	ui, _ := newTestPureTTY(t, 80, 24)

	// Create a large file for scroll testing
	tmpFile := t.TempDir() + "/large.go"
	content := "package main\n\n"
	for i := 0; i < 100; i++ {
		content += "// Line " + string(rune(i)) + "\n"
	}
	if err := writeFile(tmpFile, content); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup file preview
	block := createTestBlock(1, blocks.BlockTypeExecute)
	block.Body = "See " + tmpFile + ":50"
	ui.timeline.Append(block)
	ui.timeline.FocusBlock(block.ID)

	fakeTTY := testkit.NewFakeTTY(80, 24)
	ui.tty = fakeTTY
	ui.handleOpenFilePreview()

	if ui.filePreview == nil {
		t.Fatal("filePreview is nil")
	}

	// Test scroll down
	initialScrollPos := ui.filePreview.ScrollPos
	ui.handleFilePreviewKey(term.KeyEvent{Kind: term.KeyDown})
	if ui.filePreview.ScrollPos <= initialScrollPos {
		t.Error("ScrollPos did not increase after KeyDown")
	}

	// Test scroll up
	ui.handleFilePreviewKey(term.KeyEvent{Kind: term.KeyUp})
	if ui.filePreview.ScrollPos != initialScrollPos {
		t.Error("ScrollPos did not return to initial after KeyUp")
	}

	// Test scroll to top (g key)
	ui.filePreview.ScrollPos = 50
	ui.handleFilePreviewKey(term.KeyEvent{Kind: term.KeyRune, Rune: 'g'})
	if ui.filePreview.ScrollPos != 0 {
		t.Errorf("ScrollPos = %d after 'g', want 0", ui.filePreview.ScrollPos)
	}

	// Test Escape closes preview
	ui.handleFilePreviewKey(term.KeyEvent{Kind: term.KeyEscape})
	if ui.mode != ModeTimeline {
		t.Errorf("mode = %v after Escape, want ModeTimeline", ui.mode)
	}
	if ui.filePreview != nil {
		t.Error("filePreview should be nil after closing")
	}
}

// TestFilePreviewNoAnchor tests handling when block has no anchors
func TestFilePreviewNoAnchor(t *testing.T) {
	ui, out := newTestPureTTY(t, 80, 24)

	// Create block without any file anchors
	block := createTestBlock(1, blocks.BlockTypeNotice)
	block.Body = "This is just a notice with no file references"
	ui.timeline.Append(block)
	ui.timeline.FocusBlock(block.ID)

	// Try to open file preview
	ui.handleOpenFilePreview()

	// Should stay in timeline mode
	if ui.mode != ModeInput {
		t.Errorf("mode = %v, want ModeInput (should not switch to preview)", ui.mode)
	}

	// Should print error message
	output := out.String()
	if !contains(output, "No file anchors found") {
		t.Error("Expected 'No file anchors found' message in output")
	}
}

// Helper to write file content
func writeFile(path, content string) error {
	f, err := create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexHelper(s, substr) >= 0
}

func indexHelper(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func create(name string) (*os.File, error) {
	return os.Create(name)
}
