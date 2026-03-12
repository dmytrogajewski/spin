package blocks

import (
	"errors"
	"testing"
)

// ========== Block Management Tests ==========.

func TestTimeline_Append(t *testing.T) {
	timeline := NewTimeline()

	block := NewBlock(BlockTypeExecute)
	block.ID = "blk_1"
	block.Title = "Test block"

	err := timeline.Append(block)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if timeline.Len() != 1 {
		t.Errorf("Expected length 1, got %d", timeline.Len())
	}

	retrieved, err := timeline.Get("blk_1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != "blk_1" {
		t.Errorf("Expected ID blk_1, got %s", retrieved.ID)
	}
}

func TestTimeline_AppendMultiple(t *testing.T) {
	timeline := NewTimeline()

	for i := range 5 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if timeline.Len() != 5 {
		t.Errorf("Expected length 5, got %d", timeline.Len())
	}

	// Verify order.
	for i := range 5 {
		block, err := timeline.GetByIndex(i)
		if err != nil {
			t.Fatalf("GetByIndex(%d) failed: %v", i, err)
		}

		expectedID := string(rune('a' + i))
		if block.ID != expectedID {
			t.Errorf("Index %d: expected ID %s, got %s", i, expectedID, block.ID)
		}
	}
}

func TestTimeline_AppendDuplicateID(t *testing.T) {
	timeline := NewTimeline()

	block1 := NewBlock(BlockTypeExecute)
	block1.ID = "duplicate"
	if err := timeline.Append(block1); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	block2 := NewBlock(BlockTypePlan)
	block2.ID = "duplicate"
	err := timeline.Append(block2)

	if !errors.Is(err, ErrDuplicateID) {
		t.Errorf("Expected ErrDuplicateID, got %v", err)
	}

	if timeline.Len() != 1 {
		t.Errorf("Expected length 1 after duplicate append, got %d", timeline.Len())
	}
}

func TestTimeline_Update(t *testing.T) {
	timeline := NewTimeline()

	block := NewBlock(BlockTypeExecute)
	block.ID = "blk_1"
	block.Title = "Original"
	if err := timeline.Append(block); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	updated := NewBlock(BlockTypeExecute)
	updated.ID = "blk_1"
	updated.Title = "Updated"

	err := timeline.Update("blk_1", updated)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, err := timeline.Get("blk_1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Title != "Updated" {
		t.Errorf("Expected title 'Updated', got '%s'", retrieved.Title)
	}
}

func TestTimeline_UpdateNonExistent(t *testing.T) {
	timeline := NewTimeline()

	block := NewBlock(BlockTypeExecute)
	block.ID = "nonexistent"

	err := timeline.Update("nonexistent", block)
	if !errors.Is(err, ErrBlockNotFound) {
		t.Errorf("Expected ErrBlockNotFound, got %v", err)
	}
}

func TestTimeline_Delete(t *testing.T) {
	timeline := NewTimeline()

	for i := range 3 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	err := timeline.Delete("b")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if timeline.Len() != 2 {
		t.Errorf("Expected length 2, got %d", timeline.Len())
	}

	// Verify remaining blocks.
	block0, err := timeline.GetByIndex(0)
	if err != nil {
		t.Fatalf("GetByIndex(0) failed: %v", err)
	}
	if block0.ID != "a" {
		t.Errorf("Index 0: expected 'a', got '%s'", block0.ID)
	}

	block1, err := timeline.GetByIndex(1)
	if err != nil {
		t.Fatalf("GetByIndex(1) failed: %v", err)
	}
	if block1.ID != "c" {
		t.Errorf("Index 1: expected 'c', got '%s'", block1.ID)
	}
}

func TestTimeline_DeleteNonExistent(t *testing.T) {
	timeline := NewTimeline()

	err := timeline.Delete("nonexistent")
	if !errors.Is(err, ErrBlockNotFound) {
		t.Errorf("Expected ErrBlockNotFound, got %v", err)
	}
}

// ========== Viewport Tests ==========.

func TestTimeline_Viewport_Basic(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	// Add 10 blocks.
	for i := range 10 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	viewport := timeline.GetViewport()
	if viewport.Start != 0 || viewport.End != 5 || viewport.Height != 5 {
		t.Errorf("Expected viewport {0, 5, 5}, got {%d, %d, %d}",
			viewport.Start, viewport.End, viewport.Height)
	}

	visible := timeline.GetVisibleBlocks()
	if len(visible) != 5 {
		t.Errorf("Expected 5 visible blocks, got %d", len(visible))
	}

	if visible[0].ID != "a" || visible[4].ID != "e" {
		t.Errorf("Expected visible range [a..e], got [%s..%s]",
			visible[0].ID, visible[4].ID)
	}
}

func TestTimeline_Viewport_AtBottom(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	for i := range 10 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.ScrollToBottom()

	viewport := timeline.GetViewport()
	if viewport.Start != 5 || viewport.End != 10 {
		t.Errorf("Expected viewport {5, 10, 5}, got {%d, %d, %d}",
			viewport.Start, viewport.End, viewport.Height)
	}

	visible := timeline.GetVisibleBlocks()
	if len(visible) != 5 {
		t.Errorf("Expected 5 visible blocks, got %d", len(visible))
	}

	if visible[0].ID != "f" || visible[4].ID != "j" {
		t.Errorf("Expected visible range [f..j], got [%s..%s]",
			visible[0].ID, visible[4].ID)
	}
}

func TestTimeline_Viewport_LargerThanBlocks(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(10)

	// Only 3 blocks.
	for i := range 3 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	viewport := timeline.GetViewport()
	if viewport.Start != 0 || viewport.End != 3 {
		t.Errorf("Expected viewport {0, 3, 10}, got {%d, %d, %d}",
			viewport.Start, viewport.End, viewport.Height)
	}

	visible := timeline.GetVisibleBlocks()
	if len(visible) != 3 {
		t.Errorf("Expected 3 visible blocks, got %d", len(visible))
	}
}

func TestTimeline_Viewport_Empty(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	viewport := timeline.GetViewport()
	if viewport.Start != 0 || viewport.End != 0 {
		t.Errorf("Expected viewport {0, 0, 5}, got {%d, %d, %d}",
			viewport.Start, viewport.End, viewport.Height)
	}

	visible := timeline.GetVisibleBlocks()
	if len(visible) != 0 {
		t.Errorf("Expected 0 visible blocks, got %d", len(visible))
	}
}

// ========== Navigation Tests ==========.

func TestTimeline_ScrollDown(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	for range 10 {
		block := NewBlock(BlockTypeExecute)
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.ScrollDown(3)

	viewport := timeline.GetViewport()
	if viewport.Start != 3 {
		t.Errorf("Expected scrollPos 3, got %d", viewport.Start)
	}
}

func TestTimeline_ScrollDownClamping(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	for i := range 10 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.ScrollDown(100) // Way past end.

	viewport := timeline.GetViewport()
	// Max scrollPos = 10 - 5 = 5.
	if viewport.Start != 5 {
		t.Errorf("Expected scrollPos clamped to 5, got %d", viewport.Start)
	}
}

func TestTimeline_ScrollUp(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	for i := range 10 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.ScrollDown(5)
	timeline.ScrollUp(2)

	viewport := timeline.GetViewport()
	if viewport.Start != 3 {
		t.Errorf("Expected scrollPos 3, got %d", viewport.Start)
	}
}

func TestTimeline_ScrollUpClamping(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	for i := range 10 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.ScrollDown(2)
	timeline.ScrollUp(100) // Way past top.

	viewport := timeline.GetViewport()
	if viewport.Start != 0 {
		t.Errorf("Expected scrollPos clamped to 0, got %d", viewport.Start)
	}
}

func TestTimeline_ScrollToTopBottom(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	for i := range 10 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.ScrollToBottom()

	viewport := timeline.GetViewport()
	if viewport.Start != 5 {
		t.Errorf("ScrollToBottom: expected 5, got %d", viewport.Start)
	}

	timeline.ScrollToTop()

	viewport = timeline.GetViewport()
	if viewport.Start != 0 {
		t.Errorf("ScrollToTop: expected 0, got %d", viewport.Start)
	}
}

func TestTimeline_ScrollToBlock(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	// Add 15 blocks so scrollPos 6 is valid (maxScroll = 15-5 = 10).
	for i := range 15 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	err := timeline.ScrollToBlock("g") // Index 6.
	if err != nil {
		t.Fatalf("ScrollToBlock failed: %v", err)
	}

	viewport := timeline.GetViewport()
	// Should position block at top of viewport.
	if viewport.Start != 6 {
		t.Errorf("Expected scrollPos 6, got %d", viewport.Start)
	}
}

func TestTimeline_ScrollToBlockNonExistent(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(5)

	block := NewBlock(BlockTypeExecute)
	block.ID = "exists"
	if err := timeline.Append(block); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	err := timeline.ScrollToBlock("nonexistent")
	if !errors.Is(err, ErrBlockNotFound) {
		t.Errorf("Expected ErrBlockNotFound, got %v", err)
	}
}

func TestTimeline_FocusBlock(t *testing.T) {
	timeline := NewTimeline()

	for i := range 5 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	err := timeline.FocusBlock("c")
	if err != nil {
		t.Fatalf("FocusBlock failed: %v", err)
	}

	focused, err := timeline.GetFocusedBlock()
	if err != nil {
		t.Fatalf("GetFocusedBlock failed: %v", err)
	}

	if focused.ID != "c" {
		t.Errorf("Expected focused block 'c', got '%s'", focused.ID)
	}
}

func TestTimeline_NextPrevBlock(t *testing.T) {
	timeline := NewTimeline()

	for i := range 5 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if err := timeline.FocusBlock("b"); err != nil {
		t.Fatalf("FocusBlock failed: %v", err)
	}

	if err := timeline.NextBlock(); err != nil {
		t.Fatalf("NextBlock failed: %v", err)
	}

	focused, err := timeline.GetFocusedBlock()
	if err != nil {
		t.Fatalf("GetFocusedBlock failed: %v", err)
	}
	if focused.ID != "c" {
		t.Errorf("NextBlock: expected 'c', got '%s'", focused.ID)
	}

	err = timeline.PrevBlock()
	if err != nil {
		t.Fatalf("PrevBlock failed: %v", err)
	}

	focused, err = timeline.GetFocusedBlock()
	if err != nil {
		t.Fatalf("GetFocusedBlock failed: %v", err)
	}
	if focused.ID != "b" {
		t.Errorf("PrevBlock: expected 'b', got '%s'", focused.ID)
	}
}

func TestTimeline_NextBlockClamping(t *testing.T) {
	timeline := NewTimeline()

	for i := range 3 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if err := timeline.FocusBlock("c"); err != nil {
		t.Fatalf("FocusBlock failed: %v", err)
	}

	if err := timeline.NextBlock(); err != nil {
		t.Fatalf("NextBlock failed: %v", err)
	}

	focused, err := timeline.GetFocusedBlock()
	if err != nil {
		t.Fatalf("GetFocusedBlock failed: %v", err)
	}
	if focused.ID != "c" {
		t.Errorf("NextBlock at end: expected 'c', got '%s'", focused.ID)
	}
}

func TestTimeline_PrevBlockClamping(t *testing.T) {
	timeline := NewTimeline()

	for i := range 3 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if err := timeline.FocusBlock("a"); err != nil {
		t.Fatalf("FocusBlock failed: %v", err)
	}

	if err := timeline.PrevBlock(); err != nil {
		t.Fatalf("PrevBlock failed: %v", err)
	}

	focused, err := timeline.GetFocusedBlock()
	if err != nil {
		t.Fatalf("GetFocusedBlock failed: %v", err)
	}
	if focused.ID != "a" {
		t.Errorf("PrevBlock at start: expected 'a', got '%s'", focused.ID)
	}
}

// ========== Filtering Tests ==========.

func TestTimeline_FilterByType(t *testing.T) {
	timeline := NewTimeline()

	types := []BlockType{BlockTypeExecute, BlockTypePlan, BlockTypeRead, BlockTypeExecute}
	for i, typ := range types {
		block := NewBlock(typ)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	filter := &Filter{
		Types: []BlockType{BlockTypeExecute},
	}
	timeline.SetFilter(filter)

	filtered := timeline.GetVisibleBlocks()
	if len(filtered) != 2 {
		t.Errorf("Expected 2 EXECUTE blocks, got %d", len(filtered))
	}

	if filtered[0].ID != "a" || filtered[1].ID != "d" {
		t.Errorf("Expected filtered IDs [a, d], got [%s, %s]",
			filtered[0].ID, filtered[1].ID)
	}
}

func TestTimeline_FilterByMultipleTypes(t *testing.T) {
	timeline := NewTimeline()

	types := []BlockType{BlockTypeExecute, BlockTypePlan, BlockTypeRead, BlockTypeError}
	for i, typ := range types {
		block := NewBlock(typ)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	filter := &Filter{
		Types: []BlockType{BlockTypeExecute, BlockTypeError},
	}
	timeline.SetFilter(filter)

	filtered := timeline.GetVisibleBlocks()
	if len(filtered) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(filtered))
	}

	if filtered[0].Type != BlockTypeExecute || filtered[1].Type != BlockTypeError {
		t.Errorf("Unexpected filtered types")
	}
}

func TestTimeline_FilterByFile(t *testing.T) {
	timeline := NewTimeline()

	files := []string{"main.go", "test.go", "util.go"}
	for i, file := range files {
		block := NewBlock(BlockTypeRead)
		block.ID = string(rune('a' + i))
		meta := &ReadMeta{File: file}
		if err := SetReadMeta(block, meta); err != nil {
			t.Fatalf("SetReadMeta failed: %v", err)
		}
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	filter := &Filter{
		File: "test",
	}
	timeline.SetFilter(filter)

	filtered := timeline.GetVisibleBlocks()
	if len(filtered) != 1 {
		t.Errorf("Expected 1 block, got %d", len(filtered))
	}

	if filtered[0].ID != "b" {
		t.Errorf("Expected ID 'b', got '%s'", filtered[0].ID)
	}
}

func TestTimeline_FilterByExitCode(t *testing.T) {
	timeline := NewTimeline()

	exitCodes := []int{0, 1, 0, 2}
	for i, code := range exitCodes {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		meta := &ExecuteMeta{
			Command:  "test",
			CWD:      "./",
			Impact:   "low",
			ExitCode: &code,
		}
		if err := SetExecuteMeta(block, meta); err != nil {
			t.Fatalf("SetExecuteMeta failed: %v", err)
		}
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	exitCode1 := 1
	filter := &Filter{
		ExitCode: &exitCode1,
	}
	timeline.SetFilter(filter)

	filtered := timeline.GetVisibleBlocks()
	if len(filtered) != 1 {
		t.Errorf("Expected 1 block with exit code 1, got %d", len(filtered))
	}

	if filtered[0].ID != "b" {
		t.Errorf("Expected ID 'b', got '%s'", filtered[0].ID)
	}
}

func TestTimeline_FilterByImpact(t *testing.T) {
	timeline := NewTimeline()

	impacts := []string{"low", "medium", "high", "medium"}
	for i, impact := range impacts {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		meta := &ExecuteMeta{
			Command: "test",
			CWD:     "./",
			Impact:  impact,
		}
		if err := SetExecuteMeta(block, meta); err != nil {
			t.Fatalf("SetExecuteMeta failed: %v", err)
		}
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	filter := &Filter{
		Impact: "medium",
	}
	timeline.SetFilter(filter)

	filtered := timeline.GetVisibleBlocks()
	if len(filtered) != 2 {
		t.Errorf("Expected 2 blocks with impact=medium, got %d", len(filtered))
	}

	if filtered[0].ID != "b" || filtered[1].ID != "d" {
		t.Errorf("Expected IDs [b, d], got [%s, %s]",
			filtered[0].ID, filtered[1].ID)
	}
}

func TestTimeline_FilterCombined(t *testing.T) {
	timeline := NewTimeline()

	// Block 0: EXECUTE, main.go, exit 0, impact low.
	block0 := NewBlock(BlockTypeExecute)
	block0.ID = "a"
	exitCode0 := 0
	meta0 := &ExecuteMeta{
		Command:  "test",
		CWD:      "./",
		Impact:   "low",
		ExitCode: &exitCode0,
	}
	if err := SetExecuteMeta(block0, meta0); err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}
	if err := timeline.Append(block0); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Block 1: EXECUTE, test.go, exit 1, impact high.
	block1 := NewBlock(BlockTypeExecute)
	block1.ID = "b"
	exitCode1 := 1
	meta1 := &ExecuteMeta{
		Command:  "test",
		CWD:      "./",
		Impact:   "high",
		ExitCode: &exitCode1,
	}
	if err := SetExecuteMeta(block1, meta1); err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}
	if err := timeline.Append(block1); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Block 2: PLAN.
	block2 := NewBlock(BlockTypePlan)
	block2.ID = "c"
	if err := timeline.Append(block2); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Block 3: EXECUTE, util.go, exit 1, impact high.
	block3 := NewBlock(BlockTypeExecute)
	block3.ID = "d"
	exitCode3 := 1
	meta3 := &ExecuteMeta{
		Command:  "test",
		CWD:      "./",
		Impact:   "high",
		ExitCode: &exitCode3,
	}
	if err := SetExecuteMeta(block3, meta3); err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}
	if err := timeline.Append(block3); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Filter: type=EXECUTE AND exitCode=1 AND impact=high.
	exitCode := 1
	filter := &Filter{
		Types:    []BlockType{BlockTypeExecute},
		ExitCode: &exitCode,
		Impact:   "high",
	}
	timeline.SetFilter(filter)

	filtered := timeline.GetVisibleBlocks()
	if len(filtered) != 2 {
		t.Errorf("Expected 2 blocks matching all criteria, got %d", len(filtered))
	}

	// Should match blocks b and d.
	if filtered[0].ID != "b" || filtered[1].ID != "d" {
		t.Errorf("Expected IDs [b, d], got [%s, %s]",
			filtered[0].ID, filtered[1].ID)
	}
}

func TestTimeline_ClearFilter(t *testing.T) {
	timeline := NewTimeline()

	for i := range 5 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	filter := &Filter{
		Types: []BlockType{BlockTypePlan},
	}
	timeline.SetFilter(filter)

	filtered := timeline.GetVisibleBlocks()
	if len(filtered) != 0 {
		t.Errorf("Expected 0 PLAN blocks, got %d", len(filtered))
	}

	timeline.ClearFilter()

	all := timeline.GetVisibleBlocks()
	if len(all) != 5 {
		t.Errorf("Expected 5 blocks after clearing filter, got %d", len(all))
	}
}

func TestTimeline_GetFilter(t *testing.T) {
	timeline := NewTimeline()

	if timeline.GetFilter() != nil {
		t.Error("Expected nil filter initially")
	}

	filter := &Filter{
		Types: []BlockType{BlockTypeExecute},
	}
	timeline.SetFilter(filter)

	retrieved := timeline.GetFilter()
	if retrieved == nil {
		t.Fatal("Expected non-nil filter")
	}

	if len(retrieved.Types) != 1 || retrieved.Types[0] != BlockTypeExecute {
		t.Error("Retrieved filter doesn't match")
	}
}

// ========== Collapse/Expand Tests ==========.

func TestTimeline_ToggleFold(t *testing.T) {
	timeline := NewTimeline()

	block := NewBlock(BlockTypeExecute)
	block.ID = "blk_1"
	block.FoldState = FoldStateExpanded
	if err := timeline.Append(block); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	err := timeline.ToggleFold("blk_1")
	if err != nil {
		t.Fatalf("ToggleFold failed: %v", err)
	}

	retrieved, err := timeline.Get("blk_1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.FoldState != FoldStateCollapsed {
		t.Errorf("Expected FoldStateCollapsed, got %v", retrieved.FoldState)
	}

	// Toggle again.
	err = timeline.ToggleFold("blk_1")
	if err != nil {
		t.Fatalf("ToggleFold failed: %v", err)
	}

	retrieved, err = timeline.Get("blk_1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.FoldState != FoldStateExpanded {
		t.Errorf("Expected FoldStateExpanded after second toggle, got %v", retrieved.FoldState)
	}
}

func TestTimeline_ExpandAll(t *testing.T) {
	timeline := NewTimeline()

	for i := range 3 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		block.FoldState = FoldStateCollapsed
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.ExpandAll()

	for i := range 3 {
		block, err := timeline.GetByIndex(i)
		if err != nil {
			t.Fatalf("GetByIndex(%d) failed: %v", i, err)
		}
		if block.FoldState != FoldStateExpanded {
			t.Errorf("Block %d: expected FoldStateExpanded, got %v", i, block.FoldState)
		}
	}
}

func TestTimeline_CollapseAll(t *testing.T) {
	timeline := NewTimeline()

	for i := range 3 {
		block := NewBlock(BlockTypeExecute)
		block.ID = string(rune('a' + i))
		block.FoldState = FoldStateExpanded
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	timeline.CollapseAll()

	for i := range 3 {
		block, err := timeline.GetByIndex(i)
		if err != nil {
			t.Fatalf("GetByIndex(%d) failed: %v", i, err)
		}
		if block.FoldState != FoldStateCollapsed {
			t.Errorf("Block %d: expected FoldStateCollapsed, got %v", i, block.FoldState)
		}
	}
}

// ========== Edge Cases ==========.

func TestTimeline_EmptyOperations(t *testing.T) {
	timeline := NewTimeline()

	if timeline.Len() != 0 {
		t.Error("Empty timeline should have length 0")
	}

	_, err := timeline.Get("nonexistent")
	if !errors.Is(err, ErrBlockNotFound) {
		t.Errorf("Get on empty timeline: expected ErrBlockNotFound, got %v", err)
	}

	_, err = timeline.GetByIndex(0)
	if !errors.Is(err, ErrInvalidIndex) {
		t.Errorf("GetByIndex on empty timeline: expected ErrInvalidIndex, got %v", err)
	}

	visible := timeline.GetVisibleBlocks()
	if len(visible) != 0 {
		t.Errorf("GetVisibleBlocks on empty timeline: expected 0, got %d", len(visible))
	}

	err = timeline.NextBlock()
	if !errors.Is(err, ErrNoFocusedBlock) {
		t.Errorf("NextBlock on empty timeline: expected ErrNoFocusedBlock, got %v", err)
	}
}

func TestTimeline_SingleBlock(t *testing.T) {
	timeline := NewTimeline()

	block := NewBlock(BlockTypeExecute)
	block.ID = "only"
	if err := timeline.Append(block); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if err := timeline.FocusBlock("only"); err != nil {
		t.Fatalf("FocusBlock failed: %v", err)
	}

	// NextBlock should stay at same block.
	if err := timeline.NextBlock(); err != nil {
		t.Fatalf("NextBlock failed: %v", err)
	}

	focused, err := timeline.GetFocusedBlock()
	if err != nil {
		t.Fatalf("GetFocusedBlock failed: %v", err)
	}
	if focused.ID != "only" {
		t.Errorf("NextBlock on single block: expected 'only', got '%s'", focused.ID)
	}

	// PrevBlock should stay at same block.
	err = timeline.PrevBlock()
	if err != nil {
		t.Fatalf("PrevBlock failed: %v", err)
	}

	focused, err = timeline.GetFocusedBlock()
	if err != nil {
		t.Fatalf("GetFocusedBlock failed: %v", err)
	}
	if focused.ID != "only" {
		t.Errorf("PrevBlock on single block: expected 'only', got '%s'", focused.ID)
	}
}

func TestTimeline_LargeTimeline(t *testing.T) {
	timeline := NewTimeline()
	timeline.SetViewportHeight(20)

	// Add 1000 blocks.
	for i := range 1000 {
		block := NewBlock(BlockTypeExecute)
		block.ID = GenerateBlockID(i + 1)
		if err := timeline.Append(block); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if timeline.Len() != 1000 {
		t.Errorf("Expected 1000 blocks, got %d", timeline.Len())
	}

	// Test viewport calculation.
	visible := timeline.GetVisibleBlocks()
	if len(visible) != 20 {
		t.Errorf("Expected 20 visible blocks, got %d", len(visible))
	}

	// Test scroll to bottom.
	timeline.ScrollToBottom()

	viewport := timeline.GetViewport()
	if viewport.Start != 980 {
		t.Errorf("Expected scrollPos 980, got %d", viewport.Start)
	}

	visible = timeline.GetVisibleBlocks()
	if len(visible) != 20 {
		t.Errorf("Expected 20 visible blocks at bottom, got %d", len(visible))
	}
}
