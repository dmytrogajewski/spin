// Package adapters provides UI implementations for different terminal backends.
package adapters

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/overlay"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
	"github.com/dmytrogajewski/spin/internal/ui/status"
	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// UIMode represents the current UI mode.
type UIMode int

const (
	// ModeInput is the mode where keys go to the prompt.
	ModeInput UIMode = iota
	// ModeTimeline is the mode where keys navigate blocks.
	ModeTimeline
	// ModeFilter is the mode where keys edit the filter string.
	ModeFilter
	// ModePalette is the mode where keys control the command palette.
	ModePalette
	// ModeApproval is the mode where an approval dialog is shown.
	ModeApproval
)

// PureTTY implements ports.UI using native terminal control without alt-screen buffer.
// It follows Factory Droid principles: append-only transcript, single-line prompt redraw.
type PureTTY struct {
	tty      term.TerminalController
	model    *prompt.Model
	renderer *prompt.Renderer
	coord    *output.CoordinatedWriter
	out      io.Writer

	// Internal prompt loop channel (consumed by Run)
	promptInputs <-chan string

	// External input channel (for RequestInput callers)
	externalInputs chan string

	// Timeline and block rendering (Phase 6.1)
	timeline       *blocks.Timeline
	blockRenderer  *blocks.Renderer
	viewportHeight int
	mode           UIMode
	filterInput    string

	// Command palette (Phase 6.2)
	palette         *overlay.Palette
	paletteRegistry *overlay.CommandRegistry
	paletteRenderer *overlay.PaletteRenderer

	// File preview (Phase 6.3) - REMOVED
	searchMatches []int // current search matches in file preview

	// Approval dialog (Feature 5)
	approvalDialog *overlay.ApprovalDialog

	// Status management (Phase 1)
	statusManager    *status.Manager
	statusAggregator *status.Aggregator
	statusRenderer   *status.Renderer
	lastStatusText   string // Track last status to avoid unnecessary updates

	// Testing support
	keyboardEvents <-chan term.KeyEvent // If set, use this instead of ReadKeys

	mu      sync.Mutex
	running bool
	stopped bool
	cancel  context.CancelFunc
}

// PureTTYOption configures PureTTY behavior.
type PureTTYOption func(*PureTTY) error

// WithTTY sets a custom TTY implementation (for testing).
func WithTTY(tty term.TerminalController) PureTTYOption {
	return func(p *PureTTY) error {
		p.tty = tty
		return nil
	}
}

// NewPureTTY creates a new PureTTY adapter.
// Defaults: stdin/stdout TTY, 100-entry history, "> " prefix.
func NewPureTTY(out io.Writer, opts ...PureTTYOption) (*PureTTY, error) {
	p := &PureTTY{
		out:            out,
		mode:           ModeInput,              // Start in input mode (backward compat)
		externalInputs: make(chan string, 100), // Buffered channel for RequestInput() callers
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}

	// Create defaults if not provided
	if p.tty == nil {
		// Use real TTY (stdin/stdout)
		tty, err := term.New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
		if err != nil {
			return nil, fmt.Errorf("create TTY: %w", err)
		}
		p.tty = tty
	}

	if p.model == nil {
		p.model = prompt.NewModel(100) // 100-entry history
	}

	// Create renderer if not provided
	if p.renderer == nil {
		w, h := p.tty.Size()
		p.renderer = prompt.NewRenderer(out, w, "> ")
		p.renderer.SetHeight(h)
	}

	if p.coord == nil {
		// Create printer
		printer := output.NewPrinter(out)

		// Create adapter that wraps renderer to match output.PromptRenderer interface
		rendererAdapter := &rendererAdapter{renderer: p.renderer}

		// Create coordinator
		p.coord = output.NewCoordinatedWriter(printer, rendererAdapter, p.model)
	}

	// Create timeline if not provided
	if p.timeline == nil {
		p.timeline = blocks.NewTimeline()
	}

	// Create block renderer if not provided
	if p.blockRenderer == nil {
		w, _ := p.tty.Size()
		p.blockRenderer = blocks.NewRenderer(w)
	}

	// Create command palette (Phase 6.2)
	if p.paletteRegistry == nil {
		p.paletteRegistry = overlay.NewCommandRegistry()
		p.registerDefaultCommands()
	}
	if p.palette == nil {
		p.palette = overlay.NewPalette(p.paletteRegistry)
	}
	if p.paletteRenderer == nil {
		w, h := p.tty.Size()
		p.paletteRenderer = overlay.NewPaletteRenderer(w, h)
	}

	// Create status management components (Phase 1)
	if p.statusManager == nil {
		p.statusManager = status.NewManager()
	}
	if p.statusAggregator == nil {
		p.statusAggregator = status.NewAggregator(p.statusManager)
	}
	if p.statusRenderer == nil {
		w, h := p.tty.Size()
		p.statusRenderer = status.NewRenderer(p.out, w, h)
	}

	// Connect scroll manager to coordinator
	if p.coord != nil && p.statusRenderer != nil {
		p.coord.SetScrollManager(p.statusRenderer)
	}

	return p, nil
}

// Run starts the UI event loop and blocks until context cancel or quit.
func (u *PureTTY) Run(ctx context.Context) error {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return fmt.Errorf("already running")
	}
	u.running = true
	u.mu.Unlock()

	// Create cancelable context for internal goroutines
	ctx, cancel := context.WithCancel(ctx)
	u.mu.Lock()
	u.cancel = cancel
	u.mu.Unlock()

	// Ensure cleanup on exit
	defer func() {
		u.mu.Lock()
		u.running = false
		u.stopped = true
		u.mu.Unlock()
		cancel()
	}()

	// Enter raw mode
	if err := u.tty.Enter(); err != nil {
		return fmt.Errorf("enter raw mode: %w", err)
	}
	defer func() {
		// Reset scrolling region before exiting
		fmt.Fprint(u.out, "\x1b[r") // Reset scroll region to full screen
		u.tty.Exit()
	}()

	// Start keyboard reader (or use injected events for testing)
	var keys <-chan term.KeyEvent
	if u.keyboardEvents != nil {
		// Use injected keyboard events (for testing)
		keys = u.keyboardEvents
	} else {
		// Use real keyboard reader
		var err error
		keys, err = term.ReadKeys(ctx, os.Stdin, nil)
		if err != nil {
			return fmt.Errorf("start keyboard reader: %w", err)
		}
	}

	// Start prompt loop
	inputs := u.startPromptLoop(ctx, keys)
	u.mu.Lock()
	u.promptInputs = inputs
	u.mu.Unlock()

	// Setup SIGWINCH handler
	u.tty.OnResize(func(w, h int) {
		u.handleResize(w, h)
	})

	// Initial prompt draw
	u.coord.RedrawPrompt()

	// Initialize status bar with "Ready" message
	if u.statusManager != nil {
		u.statusManager.SetStatus("Ready")
		u.updateStatusBar()
	}

	// Ensure external inputs channel is closed on exit
	defer close(u.externalInputs)

	// Event loop
	for {
		select {
		case line, ok := <-inputs:
			if !ok {
				// Prompt loop closed (Ctrl-C, Ctrl-D, or context cancel)
				// Check if it was context cancel
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					return nil
				}
			}
			// Handle line internally
			u.handleSubmittedLine(line)

			// Forward to external consumers (buffered, will block if full)
			u.externalInputs <- line

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Stop gracefully shuts down the UI.
func (u *PureTTY) Stop() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.stopped {
		return nil // Already stopped
	}

	// Cancel context to stop goroutines
	if u.cancel != nil {
		u.cancel()
	}

	return nil
}

// PrintLine prints a line to the transcript with newline.
func (u *PureTTY) PrintLine(line string) error {
	return u.coord.PrintLine(line)
}

// PrintChunks streams chunks to the transcript.
func (u *PureTTY) PrintChunks(ctx context.Context, chunks <-chan string) error {
	return u.coord.PrintChunks(ctx, chunks)
}

// SetStatus sets transient right-aligned status text in prompt.
func (u *PureTTY) SetStatus(text string) error {
	return u.coord.SetStatus(text)
}

// SetTaskMode sets the task mode for display in status bar.
func (u *PureTTY) SetTaskMode(mode string) {
	if u.statusManager != nil {
		u.statusManager.SetTaskMode(mode)
		u.updateStatusBar()
	}
}

// SetConversationID sets the conversation/session ID for display in status bar.
func (u *PureTTY) SetConversationID(id string) {
	if u.statusManager != nil {
		u.statusManager.SetConversationID(id)
		u.updateStatusBar()
	}
}

// SetProviderInfo sets the LLM provider and model information for display in status bar.
func (u *PureTTY) SetProviderInfo(provider, model string) {
	if u.statusManager != nil {
		u.statusManager.SetProvider(provider, model)
		u.statusManager.SetConnected(true)
		u.updateStatusBar()
	}
}

// SetMaxTokens sets the maximum token limit for context percentage calculation.
func (u *PureTTY) SetMaxTokens(maxTokens int64) {
	if u.statusManager != nil {
		u.statusManager.SetMaxTokens(maxTokens)
		u.updateStatusBar()
	}
}

// SetTokenCount sets the current token count for context percentage calculation.
func (u *PureTTY) SetTokenCount(tokenCount int64) {
	if u.statusManager != nil {
		// Set token count directly (this is the cumulative total)
		u.statusManager.UpdateMetrics(func(m *status.Metrics) {
			m.TokenCount = tokenCount
			if m.MaxTokens > 0 {
				m.TokenUsage = float64(tokenCount) / float64(m.MaxTokens) * 100
			}
		})
		u.updateStatusBar()
	}
}

// ProcessEvent processes a core.Event and updates the status manager.
// This method is called by the event mapper to update status information.
func (u *PureTTY) ProcessEvent(event *core.Event) {
	if u.statusAggregator == nil {
		return
	}

	u.statusAggregator.ProcessEvent(event)

	if u.shouldUpdateStatusBar(event.Type) {
		u.updateStatusBar()
	}
}

// shouldUpdateStatusBar determines if the status bar should be updated for the given event type.
func (u *PureTTY) shouldUpdateStatusBar(eventType core.EventType) bool {
	switch eventType {
	case core.EventTurnStart,
		core.EventToolCallStart,
		core.EventToolCallComplete,
		core.EventContentDelta,
		core.EventContentComplete,
		core.EventTurnComplete:
		return true
	default:
		return false
	}
}

// updateStatusBar updates the sticky status bar with the current status from StatusManager.
// Only updates if the status text has actually changed to avoid unnecessary redraws.
func (u *PureTTY) updateStatusBar() {
	if u.statusManager == nil || u.statusRenderer == nil {
		return
	}

	// Get terminal width for adaptive formatting
	w, _ := u.tty.Size()

	// Get formatted status from manager (adaptive based on width)
	newStatusText := u.statusManager.FormatAdaptive(w)

	// Only update if status actually changed
	u.mu.Lock()
	lastStatusText := u.lastStatusText
	u.mu.Unlock()

	if newStatusText != lastStatusText {
		// Update sticky status bar
		_ = u.statusRenderer.Render(newStatusText)

		// Remember the last status text
		u.mu.Lock()
		u.lastStatusText = newStatusText
		u.mu.Unlock()
	}
}

// RequestInput returns a channel that emits user-submitted lines.
func (u *PureTTY) RequestInput() <-chan string {
	// Return external inputs channel that receives forwarded messages from Run()
	return u.externalInputs
}

// startPromptLoop starts the prompt input loop in a background goroutine.
func (u *PureTTY) startPromptLoop(ctx context.Context, keys <-chan term.KeyEvent) <-chan string {
	loop := prompt.NewLoop(u.model, u.renderer, keys)
	return loop.Run(ctx)
}

// handleResize updates renderer dimensions and redraws prompt on SIGWINCH.
func (u *PureTTY) handleResize(w, h int) {
	// Update prompt renderer dimensions
	u.renderer.SetSize(w, h)

	// Update status renderer dimensions
	if u.statusRenderer != nil {
		u.statusRenderer.SetSize(w, h)
		// Redraw status bar with new dimensions
		u.updateStatusBar()
	}

	// Redraw prompt
	u.coord.RedrawPrompt()
}

// handleSubmittedLine echoes user input to transcript.
func (u *PureTTY) handleSubmittedLine(line string) {
	// Echo user input with prompt prefix
	u.coord.PrintLine("> " + line)
}

// rendererAdapter adapts prompt.Renderer to output.PromptRenderer interface.
type rendererAdapter struct {
	renderer *prompt.Renderer
}

func (a *rendererAdapter) Redraw(model output.PromptModel, status string) error {
	// Cast model back to *prompt.Model (safe because we control the type)
	promptModel := model.(*prompt.Model)
	return a.renderer.Redraw(promptModel, status)
}

// formatFilterChips formats active filter as colored chips.
func (u *PureTTY) formatFilterChips(f *blocks.Filter) string {
	var chips []string

	chips = append(chips, u.formatTypeChips(f.Types)...)
	chips = append(chips, u.formatFileChip(f.File)...)
	chips = append(chips, u.formatExitCodeChip(f.ExitCode)...)
	chips = append(chips, u.formatImpactChip(f.Impact)...)

	return strings.Join(chips, " ")
}

// formatTypeChips formats type filter chips.
func (u *PureTTY) formatTypeChips(types []blocks.BlockType) []string {
	var chips []string
	for _, typ := range types {
		chips = append(chips, fmt.Sprintf("[type:%s]", typ))
	}
	return chips
}

// formatFileChip formats file filter chip.
func (u *PureTTY) formatFileChip(file string) []string {
	if file == "" {
		return nil
	}
	return []string{fmt.Sprintf("[file:%s]", file)}
}

// formatExitCodeChip formats exit code filter chip.
func (u *PureTTY) formatExitCodeChip(exitCode *int) []string {
	if exitCode == nil {
		return nil
	}
	return []string{fmt.Sprintf("[exit:%d]", *exitCode)}
}

// formatImpactChip formats impact filter chip.
func (u *PureTTY) formatImpactChip(impact string) []string {
	if impact == "" {
		return nil
	}
	return []string{fmt.Sprintf("[impact:%s]", impact)}
}

// render redraws UI elements (filter, prompt).
// Note: Blocks are printed via AppendBlock in append-only mode.
func (u *PureTTY) render() {
	// Render approval dialog if active
	if u.mode == ModeApproval && u.approvalDialog != nil {
		u.renderApprovalOverlay()
		return
	}

	// Render filter UI if active
	if u.mode == ModeFilter || u.timeline.GetFilter() != nil {
		u.renderFilterUI()
	}

	// Render prompt (via coordinator)
	u.coord.RedrawPrompt()
}

// renderFilterUI renders filter input or active filter chips.
func (u *PureTTY) renderFilterUI() {
	if u.mode == ModeFilter {
		// Show filter input line
		fmt.Fprintf(u.out, "\r%s/ %s%s\r\n",
			term.ClearLine,
			u.filterInput,
			term.ShowCursor,
		)
	} else if f := u.timeline.GetFilter(); f != nil {
		// Show active filter chips
		chips := u.formatFilterChips(f)
		fmt.Fprintf(u.out, "\r%sFilter: %s%s\r\n",
			term.ClearLine,
			chips,
			term.HideCursor,
		)
	}
}

// ShowApprovalDialog displays an approval dialog for the given request.
func (u *PureTTY) ShowApprovalDialog(req core.ApprovalRequest) core.ApprovalResponse {
	// Create approval dialog
	u.approvalDialog = overlay.NewApprovalDialog(req, 60*time.Second)
	u.mode = ModeApproval

	// Render the dialog
	u.renderApprovalOverlay()

	// Wait for user response
	ctx := context.Background()
	response := u.approvalDialog.Show(ctx)

	// Clean up
	u.approvalDialog = nil
	u.mode = ModeInput

	return response
}

// renderApprovalOverlay renders the approval dialog overlay.
func (u *PureTTY) renderApprovalOverlay() {
	if u.approvalDialog == nil {
		return
	}

	// Get terminal dimensions
	w, h := u.tty.Size()

	// Render the dialog
	output := u.approvalDialog.Render(w, h)
	if output != "" {
		fmt.Fprint(u.out, output)
	}
}

// AppendBlock appends a new block to timeline and prints it.
func (u *PureTTY) AppendBlock(block *blocks.Block) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Handle duplicate block IDs by appending a suffix
	// This handles the case where LLM reuses tool IDs for different tool calls
	originalID := block.ID
	suffix := 1
	for {
		if err := u.timeline.Append(block); err != nil {
			// Check if error is due to duplicate ID
			if err == blocks.ErrDuplicateID {
				// Make ID unique by appending -1, -2, etc.
				block.ID = fmt.Sprintf("%s-%d", originalID, suffix)
				suffix++
				continue // Retry with new ID
			}
			// Other error, return it
			return err
		}
		// Successfully appended
		break
	}

	// Render only the new block (append-only UI)
	rendered, err := u.blockRenderer.Render(block)
	if err != nil {
		return err
	}

	// Print via coordinator to maintain prompt integrity
	u.coord.PrintLine(strings.ReplaceAll(rendered, "\n", "\r\n"))
	return nil
}

// UpdateBlock updates an existing block in the timeline.
// In append-only mode, prints the completion status line for completed blocks.
//
// IMPORTANT: This method MUST print the completion status line when tools complete.
// Simply updating the timeline internal state is not enough - the user needs to SEE
// the completion status. This is tested in TestToolCallFormatting_ListDirectory.
func (u *PureTTY) UpdateBlock(blockID string, block *blocks.Block) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if err := u.timeline.Update(blockID, block); err != nil {
		return err
	}

	// Print completion status line for tool blocks that have completed
	// This shows the "↳ Exit code: 0..." line without re-rendering the entire block
	statusLine := u.blockRenderer.RenderCompletionStatus(block)
	if statusLine != "" {
		// Move cursor up one line (to overwrite the prompt), clear the line, write status, then redraw prompt
		// Sequence: ESC[1A (up), ESC[2K (clear line), write status, newline, redraw prompt
		fmt.Fprint(u.out, "\x1b[1A\x1b[2K")                                    // Up + clear line
		fmt.Fprint(u.out, strings.ReplaceAll(statusLine, "\n", "\r\n")+"\r\n") // Write status
		u.renderer.Redraw(u.model, "")                                         // Redraw prompt
		// Move cursor back to scrolling region after redrawing prompt
		if u.statusRenderer != nil {
			_ = u.statusRenderer.MoveToScrollRegion()
		}
	}

	return nil
}

// DeleteBlock deletes a block and re-renders.
func (u *PureTTY) DeleteBlock(blockID string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if err := u.timeline.Delete(blockID); err != nil {
		return err
	}
	u.render()
	return nil
}

// SetMode switches UI mode (for testing or external control).
func (u *PureTTY) SetMode(mode UIMode) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.mode = mode
	u.render()
}

// registerDefaultCommands registers built-in command palette commands.
func (u *PureTTY) registerDefaultCommands() {
	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Run...",
		"Execute shell command",
		"Edit",
		'▶',
		func(ctx context.Context, args ...interface{}) error {
			// TODO: Implement run command
			return nil
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Search in repo...",
		"Grep/search files",
		"Tools",
		'🔍',
		func(ctx context.Context, args ...interface{}) error {
			// TODO: Implement search command
			return nil
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Open recent file...",
		"File picker",
		"File",
		'📄',
		func(ctx context.Context, args ...interface{}) error {
			// TODO: Implement file picker
			return nil
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"New plan...",
		"Create plan block",
		"Edit",
		'📋',
		func(ctx context.Context, args ...interface{}) error {
			// TODO: Implement new plan
			return nil
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Toggle mode...",
		"Switch Auto/Manual",
		"System",
		'🔄',
		func(ctx context.Context, args ...interface{}) error {
			// TODO: Implement toggle mode
			return nil
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Change theme...",
		"Switch Dark/Light",
		"System",
		'🎨',
		func(ctx context.Context, args ...interface{}) error {
			// TODO: Implement theme change
			return nil
		},
	))
}

// Verify PureTTY implements ports.UI
var _ ports.UI = (*PureTTY)(nil)
var _ output.PromptRenderer = (*rendererAdapter)(nil)
