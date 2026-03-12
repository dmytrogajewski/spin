package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

// fakeUI implements ports.UI for mapper tests without terminal dependencies.
type fakeUI struct {
	blocks map[string]*blocks.Block
	lines  []string
}

// Ensure fakeUI implements ports.UI.
var _ ports.UI = (*fakeUI)(nil)

func newFakeUI() *fakeUI {
	return &fakeUI{blocks: make(map[string]*blocks.Block)}
}

func (f *fakeUI) Run(_ context.Context) error { return nil }
func (f *fakeUI) Stop() error                   { return nil }
func (f *fakeUI) PrintLine(line string) error {
	f.lines = append(f.lines, line)
	return nil
}
func (f *fakeUI) PrintChunks(_ context.Context, _ <-chan string) error { return nil }
func (f *fakeUI) SetStatus(_ string) error                                 { return nil }
func (f *fakeUI) SetMaxTokens(_ int64)                                {}
func (f *fakeUI) RequestInput() <-chan string {
	ch := make(chan string)
	close(ch)
	return ch
}
func (f *fakeUI) AppendBlock(block *blocks.Block) error {
	f.blocks[block.ID] = block
	return nil
}
func (f *fakeUI) UpdateBlock(blockID string, block *blocks.Block) error {
	f.blocks[blockID] = block

	return nil
}
func (f *fakeUI) DeleteBlock(blockID string) error {
	delete(f.blocks, blockID)
	return nil
}

// Test that an execute_command error is not duplicated in the block body.
func TestMapper_ExecuteError_NoDuplication(t *testing.T) {
	ui := newFakeUI()
	mapper := NewMapper(ui)

	// Simulate tool start for execute_command.
	args, _ := tools.FromMap(map[string]any{
		"operation": "execute_command",
		"command":   "mkdir tetris && cd tetris && cargo init .",
	})

	start := events.Event{
		Type: events.EventToolCallStart,
		Data: events.ToolCallStartData{
			ToolName:         "execute_command",
			ToolID:           "tool_exec_err_1",
			Parameters:       args,
			RequiresApproval: false,
		},
	}
	err := mapper.MapEvent(start)
	if err != nil {
		t.Fatalf("handle start failed: %v", err)
	}

	// Simulate completion with failure and error text.
	complete := events.Event{
		Type: events.EventToolCallComplete,
		Data: events.ToolCallCompleteData{
			ToolID:   "tool_exec_err_1",
			ToolName: "execute_command",
			Success:  false,
			Output:   "", // no stdout/stderr merged output.
			Error:    "execution failed: exit status 1",
		},
	}
	err = mapper.MapEvent(complete)
	if err != nil {
		t.Fatalf("handle complete failed: %v", err)
	}

	blk, ok := ui.blocks["tool_exec_err_1"]
	if !ok {
		t.Fatalf("block not found in UI registry")
	}

	// Body should contain the error exactly once.
	if blk.Body == "" {
		t.Fatalf("expected block body to contain error text")
	}

	if strings.Count(blk.Body, "execution failed: exit status 1") != 1 {
		t.Fatalf("error message duplicated in body: %q", blk.Body)
	}
}
