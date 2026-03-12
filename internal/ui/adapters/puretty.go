// Package adapters provides UI implementations for different terminal backends.
package adapters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/overlay"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
	"github.com/dmytrogajewski/spin/internal/ui/status"
	"github.com/dmytrogajewski/spin/internal/ui/term"
)

var ErrAlreadyRunning = errors.New("already running")

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
	renderer *prompt.TermRenderer
	coord    *output.CoordinatedWriter
	out      io.Writer

	// Internal prompt loop channel (consumed by Run).
	promptInputs <-chan string

	// External input channel (for RequestInput callers).
	externalInputs chan string

	// Timeline and block rendering (Phase 6.1).
	timeline       *blocks.Timeline
	blockRenderer  *blocks.Renderer
	viewportHeight int
	mode           UIMode
	filterInput    string

	// Command palette (Phase 6.2).
	palette         *overlay.Palette
	paletteRegistry *overlay.CommandRegistry
	paletteRenderer *overlay.PaletteRenderer

	// Approval dialog (Feature 5).
	approvalDialog *overlay.ApprovalDialog

	// Status management (Phase 1).
	statusManager    *status.Manager
	statusAggregator *status.Aggregator
	statusRenderer   *status.Renderer
	lastStatusText   string // Track last status to avoid unnecessary updates.

	// Testing support.
	keyboardEvents <-chan term.KeyEvent // If set, use this instead of ReadKeys.

	// Exec mode disables prompt/status rendering and scrolling regions.
	// Used by non-interactive `spin exec` to avoid cursor positioning issues.
	execMode bool

	// Approval TTL hints (from config) used for key preview in approval status.
	sessionPolicyTTL time.Duration
	globalPolicyTTL  time.Duration

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

// WithExecMode puts the adapter into non-interactive exec mode:
// - Disables prompt redraws
// - Disables sticky status bar and scrolling regions
// - Prints output as plain lines (append-only).
func WithExecMode() PureTTYOption {
	return func(p *PureTTY) error {
		p.execMode = true

		return nil
	}
}

// WithModel sets a custom prompt model (for testing).

// WithKeyboardEvents sets a custom keyboard event channel (for testing).
func WithKeyboardEvents(events <-chan term.KeyEvent) PureTTYOption {
	return func(p *PureTTY) error {
		p.keyboardEvents = events

		return nil
	}
}

// NewPureTTY creates a new PureTTY adapter.
// Defaults: stdin/stdout TTY, 100-entry history, "> " prefix.
func NewPureTTY(out io.Writer, opts ...PureTTYOption) (*PureTTY, error) {
	p := &PureTTY{
		out:            out,
		mode:           ModeInput,              // Start in input mode (backward compat).
		externalInputs: make(chan string, 100), // Buffered channel for RequestInput() callers.
	}

	// Apply options.
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}

	if err := p.initCoreDeps(out); err != nil {
		return nil, err
	}

	p.initRendering(out)
	p.initPalette()
	p.initStatus(out)

	return p, nil
}

// initCoreDeps initializes the TTY, model, renderer, and coordinator.
func (p *PureTTY) initCoreDeps(out io.Writer) error {
	if p.tty == nil {
		tty, err := term.New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
		if err != nil {
			return fmt.Errorf("create TTY: %w", err)
		}
		p.tty = tty
	}

	if p.model == nil {
		p.model = prompt.NewModel(100)
	}

	if p.renderer == nil {
		w, h := p.tty.Size()
		p.renderer = prompt.NewTermRenderer(out, w, "> ")
		p.renderer.SetHeight(h)
	}

	if p.coord == nil {
		printer := output.NewPrinter(out)
		rendererAdapter := &rendererAdapter{renderer: p.renderer, noPrompt: p.execMode}
		p.coord = output.NewCoordinatedWriter(printer, rendererAdapter, p.model)
	}

	return nil
}

// initRendering initializes timeline, block renderer, and palette.
func (p *PureTTY) initRendering(out io.Writer) {
	if p.timeline == nil {
		p.timeline = blocks.NewTimeline()
	}

	if p.blockRenderer == nil {
		w, _ := p.tty.Size()
		p.blockRenderer = blocks.NewRenderer(w)
	}
}

// initPalette initializes the command palette.
func (p *PureTTY) initPalette() {
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
}

// initStatus initializes status management components.
func (p *PureTTY) initStatus(out io.Writer) {
	if p.statusManager == nil {
		p.statusManager = status.NewManager()
	}

	if p.statusAggregator == nil {
		p.statusAggregator = status.NewAggregator(p.statusManager)
	}

	if !p.execMode && p.statusRenderer == nil {
		w, h := p.tty.Size()
		p.statusRenderer = status.NewRenderer(out, w, h)
	}

	if p.coord != nil && p.statusRenderer != nil {
		p.coord.SetScrollManager(p.statusRenderer)
	}

	if p.statusManager != nil && !p.execMode {
		p.statusManager.SetSpinnerCallback(func() {
			p.updateStatusBar()
		})
	}
}

// Run starts the UI event loop and blocks until context cancel or quit.
func (u *PureTTY) Run(ctx context.Context) error {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()

		return ErrAlreadyRunning
	}

	u.running = true
	u.mu.Unlock()

	// Create cancelable context for internal goroutines.
	ctx, cancel := context.WithCancel(ctx)

	u.mu.Lock()
	u.cancel = cancel
	u.mu.Unlock()

	// Ensure cleanup on exit.
	defer func() {
		u.mu.Lock()
		u.running = false
		u.stopped = true
		u.mu.Unlock()
		cancel()
	}()

	// Enter raw mode.
	err := u.tty.Enter()
	if err != nil {
		return fmt.Errorf("enter raw mode: %w", err)
	}

	defer func() {
		// Reset scrolling region before exiting.
		fmt.Fprint(u.out, "\x1b[r") // Reset scroll region to full screen.
		_ = u.tty.Exit()
	}()

	// Start keyboard and prompt loop.
	inputs, err2 := u.startEventLoop(ctx)
	if err2 != nil {
		return err2
	}

	// Ensure external inputs channel is closed on exit.
	defer close(u.externalInputs)

	return u.runMainLoop(ctx, inputs)
}

// Stop gracefully shuts down the UI.
func (u *PureTTY) Stop() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.stopped {
		return nil // Already stopped.
	}

	// Stop spinner animation.
	if u.statusManager != nil {
		u.statusManager.StopSpinner()
	}

	// Cancel context to stop goroutines.
	if u.cancel != nil {
		u.cancel()
	}

	return nil
}

// startEventLoop initializes keyboard routing, prompt loop, resize handler, and initial draw.
func (u *PureTTY) startEventLoop(ctx context.Context) (<-chan string, error) {
	rawKeys, err := u.resolveKeyboardSource(ctx)
	if err != nil {
		return nil, err
	}

	routedKeys := make(chan term.KeyEvent)
	go u.routeKeyboardEvents(ctx, rawKeys, routedKeys)

	inputs := u.startPromptLoop(ctx, routedKeys)
	u.mu.Lock()
	u.promptInputs = inputs
	u.mu.Unlock()

	u.tty.OnResize(func(w, h int) {
		u.handleResize(w, h)
	})

	if !u.execMode {
		_ = u.coord.RedrawPrompt()
	}

	if u.statusManager != nil {
		u.statusManager.SetStatus("Ready")
		u.updateStatusBar()
	}

	return inputs, nil
}

// resolveKeyboardSource returns injected or real keyboard events.
func (u *PureTTY) resolveKeyboardSource(ctx context.Context) (<-chan term.KeyEvent, error) {
	if u.keyboardEvents != nil {
		return u.keyboardEvents, nil
	}

	rawKeys, err := term.ReadKeys(ctx, os.Stdin, nil)
	if err != nil {
		return nil, fmt.Errorf("start keyboard reader: %w", err)
	}

	return rawKeys, nil
}

// runMainLoop processes input lines and context cancellation.
func (u *PureTTY) runMainLoop(ctx context.Context, inputs <-chan string) error {
	for {
		select {
		case line, ok := <-inputs:
			if !ok {
				select {
				case <-ctx.Done():
					return fmt.Errorf("prompt loop: %w", ctx.Err())
				default:
					return nil
				}
			}

			u.handleSubmittedLine(line)
			u.externalInputs <- line

		case <-ctx.Done():
			return fmt.Errorf("prompt loop context: %w", ctx.Err())
		}
	}
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
		// Set token count directly (this is the cumulative total).
		u.statusManager.UpdateMetrics(func(m *status.Metrics) {
			m.TokenCount = tokenCount
			if m.MaxTokens > 0 {
				m.TokenUsage = float64(tokenCount) / float64(m.MaxTokens) * 100
			}
		})
		u.updateStatusBar()
	}
}

// IsExecMode returns true if the UI is in exec mode (non-interactive).
func (u *PureTTY) IsExecMode() bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.execMode
}

// ProcessEvent processes an events.Event and updates the status manager.
// This method is called by the event mapper to update status information.
func (u *PureTTY) ProcessEvent(event *events.Event) {
	if u.statusAggregator == nil {
		return
	}

	u.statusAggregator.ProcessEvent(event)

	if u.shouldUpdateStatusBar(event.Type) {
		u.updateStatusBar()
	}
}

// shouldUpdateStatusBar determines if the status bar should be updated for the given event type.
func (u *PureTTY) shouldUpdateStatusBar(eventType events.EventType) bool {
	switch eventType {
	case events.EventTurnStart,
		events.EventToolCallStart,
		events.EventToolCallComplete,
		events.EventContentDelta,
		events.EventContentComplete,
		events.EventTurnComplete:
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

	// Skip status updates when in approval mode
	// The approval dialog manages its own status bar display.
	u.mu.Lock()
	mode := u.mode
	u.mu.Unlock()

	if mode == ModeApproval {
		return
	}

	// Get terminal width for adaptive formatting.
	w, _ := u.tty.Size()

	// Get formatted status from manager (adaptive based on width).
	newStatusText := u.statusManager.FormatAdaptive(w)

	// Only update if status actually changed.
	u.mu.Lock()
	lastStatusText := u.lastStatusText
	u.mu.Unlock()

	if newStatusText != lastStatusText {
		// Update sticky status bar.
		_ = u.statusRenderer.Render(newStatusText)

		// Remember the last status text.
		u.mu.Lock()
		u.lastStatusText = newStatusText
		u.mu.Unlock()
	}
}

// RequestInput returns a channel that emits user-submitted lines.
func (u *PureTTY) RequestInput() <-chan string {
	// Return external inputs channel that receives forwarded messages from Run().
	return u.externalInputs
}

// routeKeyboardEvents routes keyboard events to the appropriate handler based on current mode.
// When in ModeApproval, routes keys to the approval dialog.
// Otherwise, forwards keys to the prompt loop.
func (u *PureTTY) routeKeyboardEvents(ctx context.Context, rawKeys <-chan term.KeyEvent, promptKeys chan<- term.KeyEvent) {
	defer close(promptKeys)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-rawKeys:
			if !ok {
				return
			}

			// Check current mode.
			u.mu.Lock()
			mode := u.mode
			dialog := u.approvalDialog
			u.mu.Unlock()

			// Route based on mode.
			if mode == ModeApproval && dialog != nil {
				// Route to approval dialog.
				u.handleApprovalKey(event, dialog)
			} else {
				// Route to prompt loop.
				select {
				case promptKeys <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// handleApprovalKey handles keyboard input for the approval dialog.
func (u *PureTTY) handleApprovalKey(event term.KeyEvent, dialog *overlay.ApprovalDialog) {
	// Convert KeyEvent to string for HandleKey.
	var keyStr string

	switch event.Kind {
	case term.KeyRune:
		keyStr = string(event.Rune)
	case term.KeyEscape:
		keyStr = "\x1b"
	case term.KeyEnter:
		keyStr = "\r"
	case term.KeyLeft:
		keyStr = "\x1b[D" // Left arrow.
	case term.KeyRight:
		keyStr = "\x1b[C" // Right arrow.
	case term.KeyUp:
		keyStr = "\x1b[A" // Up arrow.
	case term.KeyDown:
		keyStr = "\x1b[B" // Down arrow.
	default:
		return
	}

	// Pass key to dialog.
	dialog.HandleKey(keyStr)
}

// startPromptLoop starts the prompt input loop in a background goroutine.
func (u *PureTTY) startPromptLoop(ctx context.Context, keys <-chan term.KeyEvent) <-chan string {
	loop := prompt.NewLoop(u.model, u.renderer, keys)

	return loop.Run(ctx)
}

// handleResize updates renderer dimensions and redraws prompt on SIGWINCH.
func (u *PureTTY) handleResize(w, h int) {
	// Update prompt renderer dimensions.
	if u.renderer != nil {
		u.renderer.SetSize(w, h)
	}

	// Update status renderer dimensions.
	if u.statusRenderer != nil {
		u.statusRenderer.SetSize(w, h)
		// Redraw status bar with new dimensions.
		u.updateStatusBar()
	}

	// Redraw prompt unless in exec mode.
	if !u.execMode {
		_ = u.coord.RedrawPrompt()
	}
}

// handleSubmittedLine echoes user input to transcript.
func (u *PureTTY) handleSubmittedLine(line string) {
	// Echo user input with prompt prefix.
	_ = u.coord.PrintLine("> " + line)
}

// rendererAdapter adapts prompt.TermRenderer to output.PromptRenderer interface.
type rendererAdapter struct {
	renderer *prompt.TermRenderer
	noPrompt bool
}

// Redraw implements the Redraw operation.
func (a *rendererAdapter) Redraw(model output.PromptModel, status string) error {
	// Cast model back to *prompt.Model (safe because we control the type).
	promptModel, ok := model.(*prompt.Model)
	if !ok {
		return nil
	}

	if a == nil || a.noPrompt || a.renderer == nil {
		return nil
	}

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
	// Render approval status if active.
	if u.mode == ModeApproval && u.approvalDialog != nil {
		// Approval status is already shown in status bar, just update it.
		u.updateStatusBar()

		return
	}

	// Render filter UI if active.
	if u.mode == ModeFilter || u.timeline.GetFilter() != nil {
		u.renderFilterUI()
	}

	// Render prompt (via coordinator) unless in exec mode.
	if !u.execMode {
		_ = u.coord.RedrawPrompt()
	}
}

// renderFilterUI renders filter input or active filter chips.
func (u *PureTTY) renderFilterUI() {
	if u.mode == ModeFilter {
		// Show filter input line.
		fmt.Fprintf(u.out, "\r%s/ %s%s\r\n",
			term.ClearLine,
			u.filterInput,
			term.ShowCursor,
		)
	} else if f := u.timeline.GetFilter(); f != nil {
		// Show active filter chips.
		chips := u.formatFilterChips(f)
		fmt.Fprintf(u.out, "\r%sFilter: %s%s\r\n",
			term.ClearLine,
			chips,
			term.HideCursor,
		)
	}
}

// ShowApprovalDialog displays an approval dialog for the given request.
func (u *PureTTY) ShowApprovalDialog(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
	// Set approval mode.
	u.mode = ModeApproval

	// Create approval dialog for key handling.
	u.approvalDialog = overlay.NewApprovalDialog(req)

	// Show approval prompt in status bar.
	u.showApprovalStatus(req)

	// Wait for user response (respect context cancellation).
	response := u.approvalDialog.Show(ctx)

	// Clean up.
	u.approvalDialog = nil
	u.mode = ModeInput

	// Clear approval status and show result.
	u.clearApprovalStatus()
	u.displayApprovalResult(req, response)

	return response
}

// showApprovalStatus displays the approval prompt in the status bar.
func (u *PureTTY) showApprovalStatus(req security.ApprovalRequest) {
	if u.statusRenderer == nil {
		return
	}

	// Create approval prompt text.
	command := req.Command.Raw
	if len(command) > 50 {
		command = command[:47] + "..."
	}

	// Compute normalized key preview (matches PolicyStore semantics).
	keyPreview := ""

	if req.Command != nil {
		key := security.NewPolicyKey(req.Command.Program, req.Command.Args, req.WorkDir)

		args := strings.Join(key.Args, " ")
		if args != "" {
			keyPreview = fmt.Sprintf("%s %s (wd=%s)", key.Program, args, key.WorkDir)
		} else {
			keyPreview = fmt.Sprintf("%s (wd=%s)", key.Program, key.WorkDir)
		}
	}

	ttlPreview := u.formatApprovalTTLPreview()

	// Show scope-aware options: A=once, S=session, G=global, D=deny.
	if keyPreview != "" {
		approvalText := fmt.Sprintf("Executing: \"%s\" | Key: %s | %s | [A] once  [S] session  [G] global  [D] deny", command, keyPreview, ttlPreview)
		_ = u.statusRenderer.Render(approvalText)

		return
	}

	approvalText := fmt.Sprintf("Executing: \"%s\" | %s | [A] once  [S] session  [G] global  [D] deny", command, ttlPreview)

	// Render in status bar.
	_ = u.statusRenderer.Render(approvalText)
}

// SetApprovalPolicyTTLs configures TTL hints for approval persistence scopes.
func (u *PureTTY) SetApprovalPolicyTTLs(sessionTTL, globalTTL time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.sessionPolicyTTL = sessionTTL
	u.globalPolicyTTL = globalTTL
}

// formatApprovalTTLPreview returns a compact human-readable TTL hint string.
func (u *PureTTY) formatApprovalTTLPreview() string {
	u.mu.Lock()
	sessionTTL := u.sessionPolicyTTL
	globalTTL := u.globalPolicyTTL
	u.mu.Unlock()

	var parts []string
	if sessionTTL > 0 {
		parts = append(parts, fmt.Sprintf("session=%s", sessionTTL))
	}

	if globalTTL > 0 {
		parts = append(parts, fmt.Sprintf("global=%s", globalTTL))
	}

	if len(parts) == 0 {
		return "TTLs: disabled"
	}

	return "TTLs: " + strings.Join(parts, ", ")
}

// clearApprovalStatus clears the approval status from the status bar.
func (u *PureTTY) clearApprovalStatus() {
	if u.statusRenderer == nil {
		return
	}

	// Clear the status bar.
	_ = u.statusRenderer.Clear()
}

// displayApprovalResult displays a message showing the approval decision.
func (u *PureTTY) displayApprovalResult(req security.ApprovalRequest, resp security.ApprovalResponse) {
	var (
		message      string
		statusSymbol string
	)

	if resp.Approved {
		// Green checkmark for approved.
		statusSymbol = "\033[32m✓\033[0m"
		message = fmt.Sprintf("%s Command approved: %s", statusSymbol, req.Command.Raw)
	} else {
		// Red X for denied/canceled.
		statusSymbol = "\033[31m✗\033[0m"

		reason := resp.Reason
		if reason == "" {
			reason = "denied"
		}

		message = fmt.Sprintf("%s Command %s: %s", statusSymbol, reason, req.Command.Raw)
	}

	// Print the result message.
	_ = u.PrintLine(message)
	_ = u.PrintLine("") // Empty line for spacing.
}

// AppendBlock appends a new block to timeline and prints it.
func (u *PureTTY) AppendBlock(block *blocks.Block) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Handle duplicate block IDs by appending a suffix
	// This handles the case where LLM reuses tool IDs for different tool calls.
	originalID := block.ID
	suffix := 1

	for {
		err := u.timeline.Append(block)
		if err != nil {
			// Check if error is due to duplicate ID.
			if errors.Is(err, blocks.ErrDuplicateID) {
				// Make ID unique by appending -1, -2, etc.
				block.ID = fmt.Sprintf("%s-%d", originalID, suffix)
				suffix++

				continue // Retry with new ID.
			}
			// Other error, return it.
			return err
		}
		// Successfully appended.
		break
	}

	// Render only the new block (append-only UI).
	rendered, err := u.blockRenderer.Render(block)
	if err != nil {
		return err
	}

	// Print via coordinator to maintain prompt integrity.
	_ = u.coord.PrintLine(rendered)

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

	// Get the existing block to preserve CompletionPrinted flag.
	existingBlock, _ := u.timeline.Get(blockID)
	if existingBlock != nil {
		// Preserve the CompletionPrinted flag from the existing block.
		block.CompletionPrinted = existingBlock.CompletionPrinted
	}

	err := u.timeline.Update(blockID, block)
	if err != nil {
		return err
	}

	// Print completion status line for tool blocks that have completed.
	// Only print if we haven't already printed it (prevents duplicate "Tool completed" messages).
	statusLine := u.blockRenderer.RenderCompletionStatus(block)
	if statusLine != "" && !block.CompletionPrinted {
		block.CompletionPrinted = true
		_ = u.timeline.Update(blockID, block)
		u.printCompletionStatus(block, statusLine)
	}

	return nil
}

// printCompletionStatus prints a block's completion status line and optional body.
func (u *PureTTY) printCompletionStatus(block *blocks.Block, statusLine string) {
	if u.execMode {
		u.printCompletionExecMode(block, statusLine)
	} else {
		u.printCompletionInteractive(block, statusLine)
	}
}

// printCompletionExecMode prints completion status in exec (non-interactive) mode.
func (u *PureTTY) printCompletionExecMode(block *blocks.Block, statusLine string) {
	_ = u.coord.PrintLine(strings.ReplaceAll(statusLine, "\n", "\r\n"))
	u.printBlockBody(block)
}

// printCompletionInteractive prints completion status in interactive mode.
func (u *PureTTY) printCompletionInteractive(block *blocks.Block, statusLine string) {
	fmt.Fprint(u.out, "\x1b[1A\x1b[2K")
	fmt.Fprint(u.out, strings.ReplaceAll(statusLine, "\n", "\r\n")+"\r\n")
	u.printBlockBody(block)
	_ = u.renderer.Redraw(u.model, "")

	if u.statusRenderer != nil {
		_ = u.statusRenderer.MoveToScrollRegion()
	}
}

// printBlockBody renders and prints a block's body if it has content.
func (u *PureTTY) printBlockBody(block *blocks.Block) {
	if block.Body == "" {
		return
	}

	body, renderErr := u.blockRenderer.RenderBody(block)
	if renderErr == nil {
		fmt.Fprint(u.out, strings.ReplaceAll(body, "\n", "\r\n"))
	}
}

// DeleteBlock deletes a block and re-renders.
func (u *PureTTY) DeleteBlock(blockID string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	err := u.timeline.Delete(blockID)
	if err != nil {
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
// Currently no commands are registered by default to follow "Implement, or stop" principle.
// Commands can be added via u.paletteRegistry.Register() when fully implemented.
func (u *PureTTY) registerDefaultCommands() {
	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Run...",
		"Execute shell command",
		"Edit",
		'▶',
		func(ctx context.Context) error {
			return u.executeRunCommand(ctx)
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Search in repo...",
		"Grep/search files",
		"Tools",
		'🔍',
		func(ctx context.Context) error {
			return u.executeSearchCommand(ctx)
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Open recent file...",
		"File picker",
		"File",
		'📄',
		func(ctx context.Context) error {
			return u.executeFilePickerCommand(ctx)
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"New plan...",
		"Create plan block",
		"Edit",
		'📋',
		func(ctx context.Context) error {
			return u.executeNewPlanCommand(ctx)
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Toggle mode...",
		"Switch Auto/Manual",
		"System",
		'🔄',
		func(ctx context.Context) error {
			return u.executeToggleModeCommand(ctx)
		},
	))

	u.paletteRegistry.Register(overlay.NewSimpleCommand(
		"Change theme...",
		"Switch Dark/Light",
		"System",
		'🎨',
		func(ctx context.Context) error {
			return u.executeChangeThemeCommand(ctx)
		},
	))
}

// executeRunCommand implements the "Run..." command.
func (u *PureTTY) executeRunCommand(_ context.Context) error {
	u.showStatusMessage("Type a command at the prompt and press Enter to execute")

	return nil
}

// executeSearchCommand implements the "Search in repo..." command.
func (u *PureTTY) executeSearchCommand(_ context.Context) error {
	u.showStatusMessage("Try: grep <pattern> or use file search at the prompt")

	return nil
}

// executeFilePickerCommand implements the "Open recent file..." command.
func (u *PureTTY) executeFilePickerCommand(_ context.Context) error {
	u.showStatusMessage("File picker: Type file path at prompt or use 'ls' command")

	return nil
}

// executeNewPlanCommand implements the "New plan..." command.
func (u *PureTTY) executeNewPlanCommand(_ context.Context) error {
	// Create a new plan block.
	block := &blocks.Block{
		ID:        fmt.Sprintf("plan_%d", time.Now().UnixMilli()),
		Type:      blocks.BlockTypePlan,
		Title:     "New Plan",
		Meta:      nil, // Set when plan data is available.
		Body:      "- Add your first step here\n- Add your second step here\n- Add your third step here",
		FoldState: blocks.FoldStateExpanded,
		Severity:  blocks.SeverityInfo,
		Timestamp: time.Now().UnixMilli(),
	}

	err := u.timeline.Append(block)
	if err != nil {
		u.showStatusMessage(fmt.Sprintf("Failed to create plan: %v", err))

		return err
	}

	u.showStatusMessage("Created new plan block in timeline")
	u.render()

	return nil
}

// executeToggleModeCommand implements the "Toggle mode..." command.
func (u *PureTTY) executeToggleModeCommand(_ context.Context) error {
	u.showStatusMessage("Mode toggle: Use agent flags (--auto/--manual) or configuration")

	return nil
}

// executeChangeThemeCommand implements the "Change theme..." command.
func (u *PureTTY) executeChangeThemeCommand(_ context.Context) error {
	u.showStatusMessage("Theme switching: Not implemented")

	return nil
}

// showStatusMessage displays a status message in the status bar.
func (u *PureTTY) showStatusMessage(msg string) {
	if u.statusRenderer == nil {
		return
	}

	_ = u.statusRenderer.Render(msg)
}

// Verify PureTTY implements ports.UI.
var _ ports.UI = (*PureTTY)(nil)
var _ output.PromptRenderer = (*rendererAdapter)(nil)
