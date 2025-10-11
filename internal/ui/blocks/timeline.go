package blocks

import (
	"errors"
	"strings"
	"sync"
)

var (
	// ErrBlockNotFound indicates the requested block ID does not exist in the timeline.
	ErrBlockNotFound = errors.New("block not found")
	// ErrDuplicateID indicates a block with the same ID already exists.
	ErrDuplicateID = errors.New("block ID already exists")
	// ErrInvalidIndex indicates the index is out of range.
	ErrInvalidIndex = errors.New("index out of range")
	// ErrNoFocusedBlock indicates no block is currently focused.
	ErrNoFocusedBlock = errors.New("no block focused")
)

// Viewport represents the currently visible range of blocks.
type Viewport struct {
	Start  int // First visible block index
	End    int // Last visible block index (exclusive)
	Height int // Viewport height in blocks
}

// Filter defines criteria for filtering blocks in the timeline.
type Filter struct {
	Types    []BlockType // Filter by block type(s) (empty = all)
	File     string      // Filter by file path (substring match, empty = all)
	ExitCode *int        // Filter by exit code (nil = all)
	Impact   string      // Filter by impact level (empty = all)
}

// Timeline manages an ordered collection of blocks with viewport and filtering support.
// Thread-safe: all operations are protected by RWMutex (Phase 8.2 fix).
type Timeline struct {
	mu        sync.RWMutex // Protects all fields below
	blocks    []*Block
	scrollPos int
	focusedID string
	filter    *Filter
	viewport  Viewport
}

// NewTimeline creates a new empty timeline.
func NewTimeline() *Timeline {
	return &Timeline{
		blocks:    make([]*Block, 0),
		scrollPos: 0,
		focusedID: "",
		filter:    nil,
		viewport:  Viewport{Start: 0, End: 0, Height: 0},
	}
}

// Append adds a block to the end of the timeline.
func (t *Timeline) Append(block *Block) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check for duplicate ID
	for _, b := range t.blocks {
		if b.ID == block.ID {
			return ErrDuplicateID
		}
	}

	t.blocks = append(t.blocks, block)
	t.updateViewport()
	return nil
}

// Update replaces an existing block with the same ID.
func (t *Timeline) Update(blockID string, block *Block) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, b := range t.blocks {
		if b.ID == blockID {
			t.blocks[i] = block
			return nil
		}
	}
	return ErrBlockNotFound
}

// Delete removes a block from the timeline.
func (t *Timeline) Delete(blockID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, b := range t.blocks {
		if b.ID == blockID {
			t.blocks = append(t.blocks[:i], t.blocks[i+1:]...)
			if t.focusedID == blockID {
				t.focusedID = ""
			}
			t.updateViewport()
			return nil
		}
	}
	return ErrBlockNotFound
}

// Get retrieves a block by ID.
func (t *Timeline) Get(blockID string) (*Block, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, b := range t.blocks {
		if b.ID == blockID {
			return b, nil
		}
	}
	return nil, ErrBlockNotFound
}

// GetByIndex retrieves a block by index.
func (t *Timeline) GetByIndex(index int) (*Block, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if index < 0 || index >= len(t.blocks) {
		return nil, ErrInvalidIndex
	}
	return t.blocks[index], nil
}

// Len returns the number of blocks in the timeline.
func (t *Timeline) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.blocks)
}

// SetViewportHeight sets the viewport height in blocks.
func (t *Timeline) SetViewportHeight(height int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.viewport.Height = height
	t.updateViewport()
}

// GetViewport returns the current viewport state.
func (t *Timeline) GetViewport() Viewport {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.viewport
}

// GetViewportHeight returns the viewport height.
func (t *Timeline) GetViewportHeight() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.viewport.Height
}

// GetScrollPosition returns the current scroll position.
func (t *Timeline) GetScrollPosition() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.scrollPos
}

// GetVisibleBlocks returns the blocks currently visible in the viewport,
// after applying any active filter. If viewport height is 0 or unset,
// returns all filtered blocks.
func (t *Timeline) GetVisibleBlocks() []*Block {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Apply filter first
	blocks := t.getFilteredBlocks()

	// If no viewport height set, return all filtered blocks
	if t.viewport.Height == 0 {
		return blocks
	}

	// Calculate viewport bounds
	start := t.scrollPos
	if start < 0 {
		start = 0
	}
	if start > len(blocks) {
		start = len(blocks)
	}

	end := start + t.viewport.Height
	if end > len(blocks) {
		end = len(blocks)
	}

	return blocks[start:end]
}

// ScrollUp scrolls the viewport up by the specified number of blocks.
func (t *Timeline) ScrollUp(lines int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.scrollPos -= lines
	if t.scrollPos < 0 {
		t.scrollPos = 0
	}
	t.updateViewport()
}

// ScrollDown scrolls the viewport down by the specified number of blocks.
func (t *Timeline) ScrollDown(lines int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.scrollPos += lines
	t.clampScrollPos()
	t.updateViewport()
}

// ScrollToTop scrolls to the beginning of the timeline.
func (t *Timeline) ScrollToTop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.scrollPos = 0
	t.updateViewport()
}

// ScrollToBottom scrolls to the end of the timeline.
func (t *Timeline) ScrollToBottom() {
	t.mu.Lock()
	defer t.mu.Unlock()

	blocks := t.getFilteredBlocks()
	maxScroll := len(blocks) - t.viewport.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	t.scrollPos = maxScroll
	t.updateViewport()
}

// ScrollToBlock scrolls to make the specified block visible at the top of the viewport.
func (t *Timeline) ScrollToBlock(blockID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	blocks := t.getFilteredBlocks()
	for i, b := range blocks {
		if b.ID == blockID {
			t.scrollPos = i
			t.clampScrollPos()
			t.updateViewport()
			return nil
		}
	}
	return ErrBlockNotFound
}

// FocusBlock sets the focused block.
func (t *Timeline) FocusBlock(blockID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Verify block exists (call without lock since we already hold it)
	found := false
	for _, b := range t.blocks {
		if b.ID == blockID {
			found = true
			break
		}
	}
	if !found {
		return ErrBlockNotFound
	}

	t.focusedID = blockID
	return nil
}

// GetFocusedBlock returns the currently focused block.
func (t *Timeline) GetFocusedBlock() (*Block, error) {
	t.mu.RLock()
	focusedID := t.focusedID
	t.mu.RUnlock()

	if focusedID == "" {
		return nil, ErrNoFocusedBlock
	}
	return t.Get(focusedID)
}

// NextBlock moves focus to the next block in the (filtered) timeline.
func (t *Timeline) NextBlock() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.focusedID == "" {
		return ErrNoFocusedBlock
	}

	blocks := t.getFilteredBlocks()
	if len(blocks) == 0 {
		return ErrNoFocusedBlock
	}

	// Find current focused block index
	currentIdx := -1
	for i, b := range blocks {
		if b.ID == t.focusedID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return ErrNoFocusedBlock
	}

	// Move to next, clamping at end
	if currentIdx < len(blocks)-1 {
		t.focusedID = blocks[currentIdx+1].ID
	}

	return nil
}

// PrevBlock moves focus to the previous block in the (filtered) timeline.
func (t *Timeline) PrevBlock() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.focusedID == "" {
		return ErrNoFocusedBlock
	}

	blocks := t.getFilteredBlocks()
	if len(blocks) == 0 {
		return ErrNoFocusedBlock
	}

	// Find current focused block index
	currentIdx := -1
	for i, b := range blocks {
		if b.ID == t.focusedID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return ErrNoFocusedBlock
	}

	// Move to previous, clamping at start
	if currentIdx > 0 {
		t.focusedID = blocks[currentIdx-1].ID
	}

	return nil
}

// ToggleFold toggles the fold state of a block.
func (t *Timeline) ToggleFold(blockID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Find block
	var block *Block
	for _, b := range t.blocks {
		if b.ID == blockID {
			block = b
			break
		}
	}
	if block == nil {
		return ErrBlockNotFound
	}

	if block.FoldState == FoldStateExpanded {
		block.FoldState = FoldStateCollapsed
	} else {
		block.FoldState = FoldStateExpanded
	}

	return nil
}

// ExpandAll expands all blocks in the timeline.
func (t *Timeline) ExpandAll() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, block := range t.blocks {
		block.FoldState = FoldStateExpanded
	}
}

// CollapseAll collapses all blocks in the timeline.
func (t *Timeline) CollapseAll() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, block := range t.blocks {
		block.FoldState = FoldStateCollapsed
	}
}

// SetFilter sets the active filter.
func (t *Timeline) SetFilter(filter *Filter) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.filter = filter
	// Reset scroll position when filter changes
	t.scrollPos = 0
	t.updateViewport()
}

// ClearFilter clears the active filter.
func (t *Timeline) ClearFilter() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.filter = nil
	t.scrollPos = 0
	t.updateViewport()
}

// GetFilter returns the active filter, or nil if no filter is set.
func (t *Timeline) GetFilter() *Filter {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.filter
}

// updateViewport recalculates viewport start/end based on current scroll position.
func (t *Timeline) updateViewport() {
	blocks := t.getFilteredBlocks()

	start := t.scrollPos
	if start < 0 {
		start = 0
	}
	if start > len(blocks) {
		start = len(blocks)
	}

	end := start + t.viewport.Height
	if end > len(blocks) {
		end = len(blocks)
	}

	t.viewport.Start = start
	t.viewport.End = end
}

// clampScrollPos ensures scroll position stays within valid range.
func (t *Timeline) clampScrollPos() {
	blocks := t.getFilteredBlocks()
	maxScroll := len(blocks) - t.viewport.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.scrollPos > maxScroll {
		t.scrollPos = maxScroll
	}
	if t.scrollPos < 0 {
		t.scrollPos = 0
	}
}

// getFilteredBlocks returns all blocks matching the current filter.
func (t *Timeline) getFilteredBlocks() []*Block {
	if t.filter == nil {
		return t.blocks
	}

	filtered := make([]*Block, 0, len(t.blocks))
	for _, block := range t.blocks {
		if t.matchesFilter(block) {
			filtered = append(filtered, block)
		}
	}
	return filtered
}

// matchesFilter checks if a block matches the current filter criteria.
func (t *Timeline) matchesFilter(block *Block) bool {
	if t.filter == nil {
		return true
	}

	// Check type filter
	if len(t.filter.Types) > 0 {
		found := false
		for _, filterType := range t.filter.Types {
			if block.Type == filterType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check file filter
	if t.filter.File != "" {
		file := extractFile(block)
		if !strings.Contains(file, t.filter.File) {
			return false
		}
	}

	// Check exit code filter
	if t.filter.ExitCode != nil {
		exitCode := extractExitCode(block)
		if exitCode == nil || *exitCode != *t.filter.ExitCode {
			return false
		}
	}

	// Check impact filter
	if t.filter.Impact != "" {
		impact := extractImpact(block)
		if impact != t.filter.Impact {
			return false
		}
	}

	return true
}

// extractFile extracts the file path from block metadata.
func extractFile(block *Block) string {
	switch block.Type {
	case BlockTypeRead:
		if meta, err := ParseReadMeta(block); err == nil {
			return meta.File
		}
	case BlockTypeApplyPatch:
		if meta, err := ParsePatchMeta(block); err == nil {
			return meta.File
		}
	}
	return ""
}

// extractExitCode extracts the exit code from block metadata.
func extractExitCode(block *Block) *int {
	if block.Type == BlockTypeExecute {
		if meta, err := ParseExecuteMeta(block); err == nil {
			return meta.ExitCode
		}
	}
	return nil
}

// extractImpact extracts the impact level from block metadata.
func extractImpact(block *Block) string {
	if block.Type == BlockTypeExecute {
		if meta, err := ParseExecuteMeta(block); err == nil {
			return meta.Impact
		}
	}
	return ""
}
