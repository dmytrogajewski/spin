package blocks

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewBlock(t *testing.T) {
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
		{"error", BlockTypeError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBlock(tt.blockType)
			if b == nil {
				t.Fatal("NewBlock() returned nil")
			}
			if b.Type != tt.blockType {
				t.Errorf("NewBlock() Type = %v, want %v", b.Type, tt.blockType)
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
			if b.Meta == nil {
				t.Error("NewBlock() Meta is nil")
			}
			if b.Timestamp <= 0 {
				t.Error("NewBlock() Timestamp <= 0")
			}
		})
	}
}

func TestBlock_Validate(t *testing.T) {
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
		{"valid", func(b *Block) {}, false},
		{"empty_id", func(b *Block) { b.ID = "" }, true},
		{"invalid_type", func(b *Block) { b.Type = BlockType("INVALID") }, true},
		{"invalid_fold_state", func(b *Block) { b.FoldState = FoldState("invalid") }, true},
		{"invalid_severity", func(b *Block) { b.Severity = Severity("invalid") }, true},
		{"zero_timestamp", func(b *Block) { b.Timestamp = 0 }, true},
		{"negative_timestamp", func(b *Block) { b.Timestamp = -1 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := validBlock()
			tt.modify(b)
			err := b.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Block.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock_GetMeta(t *testing.T) {
	b := NewBlock(BlockTypeExecute)
	b.SetMeta("command", "ls -la")
	b.SetMeta("exit_code", 0)

	val, ok := b.GetMeta("command")
	if !ok {
		t.Error("GetMeta('command') returned ok=false")
	}
	if val != "ls -la" {
		t.Errorf("GetMeta('command') = %v, want 'ls -la'", val)
	}

	val, ok = b.GetMeta("exit_code")
	if !ok {
		t.Error("GetMeta('exit_code') returned ok=false")
	}
	if val != 0 {
		t.Errorf("GetMeta('exit_code') = %v, want 0", val)
	}

	_, ok = b.GetMeta("nonexistent")
	if ok {
		t.Error("GetMeta('nonexistent') returned ok=true")
	}
}

func TestBlock_SetMeta(t *testing.T) {
	b := NewBlock(BlockTypeExecute)
	b.SetMeta("key1", "value1")
	b.SetMeta("key2", 123)

	if len(b.Meta) != 2 {
		t.Errorf("Meta length = %d, want 2", len(b.Meta))
	}

	if b.Meta["key1"] != "value1" {
		t.Errorf("Meta['key1'] = %v, want 'value1'", b.Meta["key1"])
	}
	if b.Meta["key2"] != 123 {
		t.Errorf("Meta['key2'] = %v, want 123", b.Meta["key2"])
	}
}

func TestGenerateBlockID(t *testing.T) {
	// Test uniqueness
	id1 := GenerateBlockID(1)
	time.Sleep(1 * time.Millisecond)
	id2 := GenerateBlockID(2)

	if id1 == id2 {
		t.Errorf("GenerateBlockID produced duplicate IDs: %s", id1)
	}

	// Test format
	if len(id1) < 10 {
		t.Errorf("GenerateBlockID produced short ID: %s", id1)
	}

	// Test prefix
	if id1[:4] != "blk_" {
		t.Errorf("GenerateBlockID ID doesn't start with 'blk_': %s", id1)
	}
}

func TestBlock_JSON_Roundtrip(t *testing.T) {
	original := NewBlock(BlockTypeExecute)
	original.Title = "Test Command"
	original.Body = "command output here"
	original.SetMeta("command", "go test")
	original.SetMeta("exit_code", 0)
	original.FoldState = FoldStateCollapsed
	original.Severity = SeverityWarn

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal
	var restored Block
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Compare
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

	// Check metadata
	cmd, ok := restored.GetMeta("command")
	if !ok || cmd != "go test" {
		t.Errorf("Restored Meta['command'] = %v, want 'go test'", cmd)
	}
}

func TestBlock_JSON_Format(t *testing.T) {
	b := NewBlock(BlockTypeExecute)
	b.ID = "blk_1738950123_07"
	b.Title = "Run tests"
	b.Body = "test output"
	b.SetMeta("command", "go test")
	b.SetMeta("exit_code", 0)
	b.Timestamp = 1738950123456

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal to check structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Check required fields exist
	requiredFields := []string{"id", "type", "meta", "body", "fold_state", "severity", "timestamp"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("JSON missing required field: %s", field)
		}
	}
}
