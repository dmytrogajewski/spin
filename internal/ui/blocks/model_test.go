package blocks

import (
	"encoding/json"
	"testing"
	"time"
)

const (
	testGoTestCmd = "go test"
)

func TestNewBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		blockType BlockType
	}{
		{"execute", BlockTypeExecute},
		{"plan", BlockTypePlan},
		{"read", BlockTypeRead},
		{"grep", BlockTypeGrep},
		{"apply_patch", BlockTypeApplyPatch},
		{"summary", BlockTypeSummary},
		{"testing", BlockTypeTesting},
		{"notice", BlockTypeNotice},
		{"skill", BlockTypeSkill},
		{"task", BlockTypeTask},
		{"subagent", BlockTypeSubagent},
		{"hook", BlockTypeHook},
		{"compact", BlockTypeCompact},
		{"error", BlockTypeError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := NewBlock(tt.blockType)
			if b == nil {
				t.Fatal("NewBlock() returned nil")
			}

			verifyNewBlockDefaults(t, b, tt.blockType)
		})
	}
}

// verifyNewBlockDefaults checks all default field values of a newly created block.
func verifyNewBlockDefaults(t *testing.T, b *Block, expectedType BlockType) {
	t.Helper()

	if b.Type != expectedType {
		t.Errorf("NewBlock() Type = %v, want %v", b.Type, expectedType)
	}

	if b.ID == "" {
		t.Error("NewBlock() ID is empty")
	}

	if b.FoldState != FoldStateExpanded {
		t.Errorf("NewBlock() FoldState = %v, want %v", b.FoldState, FoldStateExpanded)
	}

	if b.Severity != SeverityInfo {
		t.Errorf("NewBlock() Severity = %v, want %v", b.Severity, SeverityInfo)
	}

	if b.Meta != nil {
		t.Error("NewBlock() Meta should be nil by default")
	}

	if b.Timestamp <= 0 {
		t.Error("NewBlock() Timestamp <= 0")
	}
}

func TestBlock_Validate(t *testing.T) {
	t.Parallel()

	validBlock := func() *Block {
		b := NewBlock(BlockTypeExecute)
		b.Body = "test output"

		return b
	}

	tests := []struct {
		name    string
		modify  func(*Block)
		wantErr bool
	}{
		{"valid", func(_ *Block) {}, false},
		{"empty_id", func(b *Block) { b.ID = "" }, true},
		{"invalid_type", func(b *Block) { b.Type = BlockType("INVALID") }, true},
		{"invalid_fold_state", func(b *Block) { b.FoldState = FoldState("invalid") }, true},
		{"invalid_severity", func(b *Block) { b.Severity = Severity("invalid") }, true},
		{"zero_timestamp", func(b *Block) { b.Timestamp = 0 }, true},
		{"negative_timestamp", func(b *Block) { b.Timestamp = -1 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := validBlock()
			tt.modify(b)

			err := b.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Block.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBlock_GetMeta and TestBlock_SetMeta removed - replaced by type-safe accessors
// See TestBlock_TypeSafeMetadata for the new approach.

func TestGenerateBlockID(t *testing.T) {
	t.Parallel(
	// Test uniqueness.
	)

	id1 := GenerateBlockID(1)

	time.Sleep(1 * time.Millisecond)

	id2 := GenerateBlockID(2)

	if id1 == id2 {
		t.Errorf("GenerateBlockID produced duplicate IDs: %s", id1)
	}

	// Test format.
	if len(id1) < 10 {
		t.Errorf("GenerateBlockID produced short ID: %s", id1)
	}

	// Test prefix.
	if id1[:4] != "blk_" {
		t.Errorf("GenerateBlockID ID doesn't start with 'blk_': %s", id1)
	}
}

func TestBlock_JSON_Roundtrip(t *testing.T) {
	t.Parallel()

	original := NewBlock(BlockTypeExecute)
	original.Title = "Test Command"
	original.Body = "command output here"

	// Use type-safe metadata.
	exitCode := 0

	meta := &ExecuteMeta{
		Command:  testGoTestCmd,
		CWD:      "/tmp",
		Impact:   "low",
		ExitCode: &exitCode,
	}

	err := original.SetExecuteMeta(meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta() error = %v", err)
	}

	original.FoldState = FoldStateCollapsed
	original.Severity = SeverityWarn

	// Marshal.
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal.
	var restored Block

	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Compare.
	if restored.ID != original.ID {
		t.Errorf("Restored ID = %v, want %v", restored.ID, original.ID)
	}

	if restored.Type != original.Type {
		t.Errorf("Restored Type = %v, want %v", restored.Type, original.Type)
	}

	if restored.Title != original.Title {
		t.Errorf("Restored Title = %v, want %v", restored.Title, original.Title)
	}

	if restored.Body != original.Body {
		t.Errorf("Restored Body = %v, want %v", restored.Body, original.Body)
	}

	if restored.FoldState != original.FoldState {
		t.Errorf("Restored FoldState = %v, want %v", restored.FoldState, original.FoldState)
	}

	if restored.Severity != original.Severity {
		t.Errorf("Restored Severity = %v, want %v", restored.Severity, original.Severity)
	}

	if restored.Timestamp != original.Timestamp {
		t.Errorf("Restored Timestamp = %v, want %v", restored.Timestamp, original.Timestamp)
	}

	// Check metadata using type-safe accessor.
	restoredMeta, err := restored.GetExecuteMeta()
	if err != nil {
		t.Fatalf("GetExecuteMeta() error = %v", err)
	}

	if restoredMeta.Command != testGoTestCmd {
		t.Errorf("Restored Meta.Command = %v, want 'go test'", restoredMeta.Command)
	}
}

func TestBlock_JSON_Format(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeExecute)
	b.ID = "blk_1738950123_07"
	b.Title = "Run tests"
	b.Body = "test output"

	// Use type-safe metadata.
	exitCode := 0

	meta := &ExecuteMeta{
		Command:  testGoTestCmd,
		CWD:      "/tmp",
		Impact:   "low",
		ExitCode: &exitCode,
	}

	err := b.SetExecuteMeta(meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta() error = %v", err)
	}

	b.Timestamp = 1738950123456

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal to check structure.
	var raw map[string]any

	err = json.Unmarshal(data, &raw)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Check required fields exist (meta is now optional with omitempty).
	requiredFields := []string{"id", "type", "body", "fold_state", "severity", "timestamp"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("JSON missing required field: %s", field)
		}
	}

	// Check meta exists when set.
	if _, ok := raw["meta"]; !ok {
		t.Error("JSON missing 'meta' field when metadata is set")
	}
}

func TestBlock_TypeSafeMetadata_Execute(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeExecute)
	meta := &ExecuteMeta{Command: testGoTestCmd, CWD: "/tmp", TimeoutSec: 30, Impact: "low"}

	err := b.SetExecuteMeta(meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta() error = %v", err)
	}

	retrieved, err := b.GetExecuteMeta()
	if err != nil {
		t.Fatalf("GetExecuteMeta() error = %v", err)
	}

	if retrieved.Command != testGoTestCmd {
		t.Errorf("Command = %v, want 'go test'", retrieved.Command)
	}

	if retrieved.CWD != "/tmp" {
		t.Errorf("CWD = %v, want '/tmp'", retrieved.CWD)
	}
}

func TestBlock_TypeSafeMetadata_Read(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeRead)
	meta := &ReadMeta{File: "main.go", Offset: 10, Limit: 50}

	err := b.SetReadMeta(meta)
	if err != nil {
		t.Fatalf("SetReadMeta() error = %v", err)
	}

	retrieved, err := b.GetReadMeta()
	if err != nil {
		t.Fatalf("GetReadMeta() error = %v", err)
	}

	if retrieved.File != "main.go" {
		t.Errorf("File = %v, want 'main.go'", retrieved.File)
	}
}

func TestBlock_TypeSafeMetadata_Tool(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeTool)
	meta := &ToolMeta{ToolName: "execute_command", Params: map[string]any{"command": "ls -la"}}

	err := b.SetToolMeta(meta)
	if err != nil {
		t.Fatalf("SetToolMeta() error = %v", err)
	}

	retrieved, err := b.GetToolMeta()
	if err != nil {
		t.Fatalf("GetToolMeta() error = %v", err)
	}

	if retrieved.ToolName != "execute_command" {
		t.Errorf("ToolName = %v, want 'execute_command'", retrieved.ToolName)
	}
}

// TestBlock_MetadataValidation tests that invalid metadata is rejected.
func TestBlock_MetadataValidation(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeExecute)

	// Invalid ExecuteMeta (empty command).
	meta := &ExecuteMeta{
		CWD:    "/tmp",
		Impact: "low",
	}

	err := b.SetExecuteMeta(meta)
	if err == nil {
		t.Error("SetExecuteMeta() should reject empty command")
	}
}

// TestBlock_MetadataJSONRoundtrip tests JSON serialization with [json.RawMessage].
func TestBlock_MetadataJSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := NewBlock(BlockTypeExecute)
	original.Title = "Test Command"
	original.Body = "output"

	exitCode := 0

	meta := &ExecuteMeta{
		Command:  testGoTestCmd,
		CWD:      "/tmp",
		Impact:   "low",
		ExitCode: &exitCode,
	}

	err := original.SetExecuteMeta(meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta() error = %v", err)
	}

	// Marshal to JSON.
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal from JSON.
	var restored Block

	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Get metadata from restored block.
	restoredMeta, err := restored.GetExecuteMeta()
	if err != nil {
		t.Fatalf("GetExecuteMeta() error = %v", err)
	}

	if restoredMeta.Command != testGoTestCmd {
		t.Errorf("Command = %v, want 'go test'", restoredMeta.Command)
	}

	if restoredMeta.CWD != "/tmp" {
		t.Errorf("CWD = %v, want '/tmp'", restoredMeta.CWD)
	}

	if restoredMeta.ExitCode == nil || *restoredMeta.ExitCode != 0 {
		t.Errorf("ExitCode = %v, want 0", restoredMeta.ExitCode)
	}
}
