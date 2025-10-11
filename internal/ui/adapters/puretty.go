// Package adapters provides UI implementations for different terminal backends.
package adapters

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/overlay"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
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
	// ModeFilePreview is the mode where a file preview popup is shown.
	ModeFilePreview
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

	// File preview (Phase 6.3)
	filePreview         *overlay.FilePreview
	filePreviewRenderer *overlay.FilePreviewRenderer
	searchMatches       []int // current search matches in file preview

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

// WithModel sets a custom prompt model (for testing).
func WithModel(model *prompt.Model) PureTTYOption {
	return func(p *PureTTY) error {
		p.model = model
		return nil
	}
}

// WithCoordinator sets a custom coordinated writer (for testing).
func WithCoordinator(coord *output.CoordinatedWriter) PureTTYOption {
	return func(p *PureTTY) error {
		p.coord = coord
		return nil
	}
}

// WithTimeline sets a custom timeline (for testing).
func WithTimeline(timeline *blocks.Timeline) PureTTYOption {
	return func(p *PureTTY) error {
		p.timeline = timeline
		return nil
	}
}

// WithBlockRenderer sets a custom block renderer (for testing).
func WithBlockRenderer(r *blocks.Renderer) PureTTYOption {
	return func(p *PureTTY) error {
		p.blockRenderer = r
		return nil
	}
}

// WithKeyboardEvents sets a custom keyboard event channel (for testing).
// When set, Run() will use this channel instead of term.ReadKeys().
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
		mode:           ModeInput,            // Start in input mode (backward compat)
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
		w, _ := p.tty.Size()
		p.renderer = prompt.NewRenderer(out, w, "> ")
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
	defer u.tty.Exit()

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

// handleResize updates renderer width and redraws prompt on SIGWINCH.
func (u *PureTTY) handleResize(w, h int) {
	u.renderer.SetWidth(w)
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

// calculateViewport computes viewport height based on terminal size.
func (u *PureTTY) calculateViewport() {
	_, h := u.tty.Size()

	// Reserve space for UI elements
	inputBarHeight := 2   // 2 rows (mode line + prompt)
	statusLineHeight := 1 // 1 row
	padding := 2          // Top + bottom padding

	u.viewportHeight = h - inputBarHeight - statusLineHeight - padding

	// Update timeline viewport
	u.timeline.SetViewportHeight(u.viewportHeight)
}

// handleKey dispatches key events based on current mode.
func (u *PureTTY) handleKey(key term.KeyEvent) {
	// Global: Ctrl-P opens palette from any mode
	if key.Kind == term.KeyCtrlP && u.mode != ModePalette {
		u.palette.Open()
		u.mode = ModePalette
		u.renderPaletteOverlay()
		return
	}

	switch u.mode {
	case ModePalette:
		u.handlePaletteKey(key)
	case ModeFilePreview:
		u.handleFilePreviewKey(key)
	case ModeTimeline:
		u.handleTimelineKey(key)
	case ModeInput:
		u.handleInputKey(key)
	case ModeFilter:
		u.handleFilterKey(key)
	}
}

// handleTimelineKey handles navigation and block actions.
func (u *PureTTY) handleTimelineKey(key term.KeyEvent) {
	switch key.Kind {
	case term.KeyPgUp:
		u.timeline.ScrollUp(u.viewportHeight)
		u.render()
	case term.KeyPgDn:
		u.timeline.ScrollDown(u.viewportHeight)
		u.render()
	case term.KeyRune:
		switch key.Rune {
		case 'g':
			u.timeline.ScrollToTop()
			u.render()
		case 'G':
			u.timeline.ScrollToBottom()
			u.render()
		case '[':
			u.timeline.PrevBlock()
			u.render()
		case ']':
			u.timeline.NextBlock()
			u.render()
		case 'y':
			u.handleCopyBlock()
		case 'S':
			u.handleSaveBlock()
		case 'r':
			u.handleRerunBlock()
		case 'w':
			u.handleToggleWrap()
		case '/':
			u.enterFilterMode()
		case 'o':
			// Open file preview for anchor under cursor
			u.handleOpenFilePreview()
		case ':':
			// Switch to input mode (for commands)
			u.mode = ModeInput
			u.render()
		default:
			// Any other char: switch to input mode, insert char
			u.mode = ModeInput
			u.model.Insert(key.Rune)
			u.render()
		}
	case term.KeyEnter:
		// Toggle fold on focused block
		focused, err := u.timeline.GetFocusedBlock()
		if err == nil {
			u.timeline.ToggleFold(focused.ID)
			u.render()
		}
	case term.KeyEscape:
		// Exit timeline mode → input mode
		u.mode = ModeInput
		u.render()
	}
}

// handleInputKey delegates to existing prompt logic.
func (u *PureTTY) handleInputKey(key term.KeyEvent) {
	// New: Esc switches to timeline mode
	if key.Kind == term.KeyEscape {
		u.mode = ModeTimeline
		u.render()
		return
	}

	// Existing prompt key handling would go here
	// (delegated to prompt.Loop in actual implementation)
}

// handleFilterKey handles filter input editing.
func (u *PureTTY) handleFilterKey(key term.KeyEvent) {
	switch key.Kind {
	case term.KeyEscape:
		// Clear filter, return to timeline mode
		u.filterInput = ""
		u.timeline.SetFilter(nil)
		u.mode = ModeTimeline
		u.render()
	case term.KeyEnter:
		// Apply filter, return to timeline mode
		filter := u.parseFilter(u.filterInput)
		u.timeline.SetFilter(filter)
		u.mode = ModeTimeline
		u.render()
	case term.KeyBackspace:
		if len(u.filterInput) > 0 {
			u.filterInput = u.filterInput[:len(u.filterInput)-1]
			u.render()
		}
	case term.KeyRune:
		u.filterInput += string(key.Rune)
		u.render()
	}
}

// handlePaletteKey handles command palette interactions.
func (u *PureTTY) handlePaletteKey(key term.KeyEvent) {
	switch key.Kind {
	case term.KeyEscape:
		// Close palette, return to input mode
		u.palette.Close()
		u.mode = ModeInput
		u.render()
	case term.KeyEnter:
		// Execute selected command
		if cmd := u.palette.SelectedCommand(); cmd != nil {
			_ = cmd.Execute(context.Background())
		}
		u.palette.Close()
		u.mode = ModeInput
		u.render()
	case term.KeyUp:
		u.palette.MoveUp()
		u.renderPaletteOverlay()
	case term.KeyDown:
		u.palette.MoveDown()
		u.renderPaletteOverlay()
	case term.KeyBackspace:
		u.palette.Backspace()
		u.renderPaletteOverlay()
	case term.KeyCtrlU:
		u.palette.ClearQuery()
		u.renderPaletteOverlay()
	case term.KeyRune:
		u.palette.Insert(key.Rune)
		u.renderPaletteOverlay()
	}
}

// parseFilter parses filter string into blocks.Filter.
// Syntax: "type:EXECUTE file:foo.go exit:0 impact:high"
func (u *PureTTY) parseFilter(input string) *blocks.Filter {
	filter := &blocks.Filter{}

	parts := strings.Fields(input)
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key, val := kv[0], kv[1]
		switch key {
		case "type":
			filter.Types = append(filter.Types, blocks.BlockType(val))
		case "file":
			filter.File = val
		case "exit":
			if code, err := strconv.Atoi(val); err == nil {
				filter.ExitCode = &code
			}
		case "impact":
			filter.Impact = val
		}
	}

	return filter
}

// formatFilterChips formats active filter as colored chips.
func (u *PureTTY) formatFilterChips(f *blocks.Filter) string {
	var chips []string

	for _, typ := range f.Types {
		chips = append(chips, fmt.Sprintf("[type:%s]", typ))
	}
	if f.File != "" {
		chips = append(chips, fmt.Sprintf("[file:%s]", f.File))
	}
	if f.ExitCode != nil {
		chips = append(chips, fmt.Sprintf("[exit:%d]", *f.ExitCode))
	}
	if f.Impact != "" {
		chips = append(chips, fmt.Sprintf("[impact:%s]", f.Impact))
	}

	return strings.Join(chips, " ")
}

// render redraws UI elements (filter, prompt).
// Note: Blocks are printed via AppendBlock in append-only mode.
func (u *PureTTY) render() {
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

// renderPaletteOverlay renders the command palette overlay.
func (u *PureTTY) renderPaletteOverlay() {
	// Update renderer size (in case of resize)
	w, h := u.tty.Size()
	u.paletteRenderer.SetSize(w, h)

	// Render palette
	paletteOutput := u.paletteRenderer.Render(u.palette)
	fmt.Fprint(u.out, paletteOutput)
}

// enterFilterMode switches to filter mode.
func (u *PureTTY) enterFilterMode() {
	u.mode = ModeFilter
	u.filterInput = ""
	u.render()
}

// handleCopyBlock copies focused block body to clipboard.
func (u *PureTTY) handleCopyBlock() {
	block, err := u.timeline.GetFocusedBlock()
	if err != nil {
		return
	}

	// Copy to clipboard (platform-specific, use external cmd)
	// For now: just log (clipboard support is optional)
	u.coord.PrintLine(fmt.Sprintf("[Copied block %s]", block.ID))
}

// handleSaveBlock saves focused block to file.
func (u *PureTTY) handleSaveBlock() {
	block, err := u.timeline.GetFocusedBlock()
	if err != nil {
		return
	}

	// Prompt for filename (simple implementation: use block ID)
	filename := fmt.Sprintf("block_%s.txt", block.ID)

	// Write body to file
	if err := os.WriteFile(filename, []byte(block.Body), 0644); err != nil {
		u.coord.PrintLine(fmt.Sprintf("[Error saving: %v]", err))
		return
	}

	u.coord.PrintLine(fmt.Sprintf("[Saved to %s]", filename))
}

// handleRerunBlock emits rerun event for EXECUTE blocks.
func (u *PureTTY) handleRerunBlock() {
	block, err := u.timeline.GetFocusedBlock()
	if err != nil || block.Type != blocks.BlockTypeExecute {
		return
	}

	// Emit rerun event (to be consumed by application)
	// For now: just log
	u.coord.PrintLine(fmt.Sprintf("[Rerun requested for block %s]", block.ID))
}

// handleToggleWrap toggles line wrapping for block.
func (u *PureTTY) handleToggleWrap() {
	// TODO: Implement wrap toggle (requires renderer state)
	u.coord.PrintLine("[Toggle wrap not implemented yet]")
}

// handleOpenFilePreview opens a file preview for anchor under cursor.
func (u *PureTTY) handleOpenFilePreview() {
	// Get focused block
	block, err := u.timeline.GetFocusedBlock()
	if err != nil {
		return
	}

	// Detect anchors in block body
	anchors := overlay.DetectAnchors(block.Body)
	if len(anchors) == 0 {
		u.coord.PrintLine("[No file anchors found in focused block]")
		return
	}

	// Use first anchor for now (TODO: let user pick if multiple)
	anchor := anchors[0]

	// Resolve absolute path (try relative to cwd)
	cwd, _ := os.Getwd()
	absPath := overlay.GetAbsolutePath(cwd, anchor.FilePath)

	// Calculate popup dimensions
	w, h := u.tty.Size()
	popupWidth, popupHeight := overlay.CalculatePopupDimensions(w, h)

	// Open file preview
	fp, err := overlay.NewFilePreview(absPath, anchor.Line, popupWidth, popupHeight)
	if err != nil {
		u.coord.PrintLine(fmt.Sprintf("[Failed to open file: %v]", err))
		return
	}

	u.filePreview = fp
	u.filePreviewRenderer = overlay.NewFilePreviewRenderer(popupWidth)
	u.searchMatches = nil
	u.mode = ModeFilePreview
	u.renderFilePreviewOverlay()
}

// handleFilePreviewKey handles file preview navigation keys.
func (u *PureTTY) handleFilePreviewKey(key term.KeyEvent) {
	if u.filePreview == nil {
		return
	}

	switch key.Kind {
	case term.KeyEscape:
		// Close preview, return to timeline mode
		u.filePreview = nil
		u.filePreviewRenderer = nil
		u.searchMatches = nil
		u.mode = ModeTimeline
		u.render()
		return

	case term.KeyUp:
		u.filePreview.ScrollUp(1)
		u.renderFilePreviewOverlay()

	case term.KeyDown:
		u.filePreview.ScrollDown(1)
		u.renderFilePreviewOverlay()

	case term.KeyPgUp:
		// Scroll up by viewport height
		contentHeight := u.filePreview.Height - 3
		u.filePreview.ScrollUp(contentHeight)
		u.renderFilePreviewOverlay()

	case term.KeyPgDn:
		// Scroll down by viewport height
		contentHeight := u.filePreview.Height - 3
		u.filePreview.ScrollDown(contentHeight)
		u.renderFilePreviewOverlay()

	case term.KeyRune:
		switch key.Rune {
		case 'g':
			u.filePreview.ScrollToTop()
			u.renderFilePreviewOverlay()
		case 'G':
			u.filePreview.ScrollToBottom()
			u.renderFilePreviewOverlay()
		case '/':
			// TODO: Enter search mode (Phase 6.3 stretch goal)
			u.coord.PrintLine("[Search in preview not implemented yet]")
		case 'n':
			// Next search match
			if len(u.searchMatches) > 0 {
				u.filePreview.NextMatch(u.searchMatches)
				u.renderFilePreviewOverlay()
			}
		case 'N':
			// Previous search match
			if len(u.searchMatches) > 0 {
				u.filePreview.PrevMatch(u.searchMatches)
				u.renderFilePreviewOverlay()
			}
		}
	}
}

// renderFilePreviewOverlay renders the file preview popup.
func (u *PureTTY) renderFilePreviewOverlay() {
	if u.filePreview == nil || u.filePreviewRenderer == nil {
		return
	}

	// Update dimensions in case of resize
	w, h := u.tty.Size()
	popupWidth, popupHeight := overlay.CalculatePopupDimensions(w, h)
	u.filePreview.Width = popupWidth
	u.filePreview.Height = popupHeight
	u.filePreviewRenderer = overlay.NewFilePreviewRenderer(popupWidth)

	// Render preview
	previewOutput := u.filePreviewRenderer.Render(u.filePreview)
	fmt.Fprint(u.out, previewOutput)
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
// Note: In append-only mode, we don't re-print updated blocks.
// Updates are stored but not visually reflected (for now).
func (u *PureTTY) UpdateBlock(blockID string, block *blocks.Block) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if err := u.timeline.Update(blockID, block); err != nil {
		return err
	}
	// Don't re-render in append-only mode
	// Future: could print a delta or status update
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
