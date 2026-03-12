package blocks

import (
	"testing"
)

func TestBlockType_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		blockType BlockType
		expected  string
	}{
		{"execute", BlockTypeExecute, "EXECUTE"},
		{"plan", BlockTypePlan, "PLAN"},
		{"read", BlockTypeRead, "READ"},
		{"grep", BlockTypeGrep, "GREP"},
		{"apply_patch", BlockTypeApplyPatch, "APPLY_PATCH"},
		{"summary", BlockTypeSummary, "SUMMARY"},
		{"tool", BlockTypeTool, "TOOL"},
		{"testing", BlockTypeTesting, "TESTING"},
		{"notice", BlockTypeNotice, "NOTICE"},
		{"error", BlockTypeError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.blockType.String()
			if result != tt.expected {
				t.Errorf("BlockType.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBlockType_Valid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		blockType BlockType
		expected  bool
	}{
		{"execute", BlockTypeExecute, true},
		{"plan", BlockTypePlan, true},
		{"read", BlockTypeRead, true},
		{"grep", BlockTypeGrep, true},
		{"apply_patch", BlockTypeApplyPatch, true},
		{"summary", BlockTypeSummary, true},
		{"tool", BlockTypeTool, true},
		{"testing", BlockTypeTesting, true},
		{"notice", BlockTypeNotice, true},
		{"error", BlockTypeError, true},
		{"invalid", BlockType("INVALID"), false},
		{"empty", BlockType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.blockType.Valid()
			if result != tt.expected {
				t.Errorf("BlockType.Valid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFoldState_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		foldState FoldState
		expected  string
	}{
		{"expanded", FoldStateExpanded, "expanded"},
		{"collapsed", FoldStateCollapsed, "collapsed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.foldState.String()
			if result != tt.expected {
				t.Errorf("FoldState.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFoldState_Valid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		foldState FoldState
		expected  bool
	}{
		{"expanded", FoldStateExpanded, true},
		{"collapsed", FoldStateCollapsed, true},
		{"invalid", FoldState("invalid"), false},
		{"empty", FoldState(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.foldState.Valid()
			if result != tt.expected {
				t.Errorf("FoldState.Valid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSeverity_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{"info", SeverityInfo, "info"},
		{"warn", SeverityWarn, "warn"},
		{"error", SeverityError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.severity.String()
			if result != tt.expected {
				t.Errorf("Severity.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSeverity_Valid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		severity Severity
		expected bool
	}{
		{"info", SeverityInfo, true},
		{"warn", SeverityWarn, true},
		{"error", SeverityError, true},
		{"invalid", Severity("invalid"), false},
		{"empty", Severity(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.severity.Valid()
			if result != tt.expected {
				t.Errorf("Severity.Valid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBlockType_Constants(t *testing.T) {
	t.Parallel(
	// Test that all constants are properly defined.
	)

	if BlockTypeExecute != "EXECUTE" {
		t.Errorf("BlockTypeExecute = %v, want %v", BlockTypeExecute, "EXECUTE")
	}

	if BlockTypePlan != "PLAN" {
		t.Errorf("BlockTypePlan = %v, want %v", BlockTypePlan, "PLAN")
	}

	if BlockTypeRead != "READ" {
		t.Errorf("BlockTypeRead = %v, want %v", BlockTypeRead, "READ")
	}

	if BlockTypeGrep != "GREP" {
		t.Errorf("BlockTypeGrep = %v, want %v", BlockTypeGrep, "GREP")
	}

	if BlockTypeApplyPatch != "APPLY_PATCH" {
		t.Errorf("BlockTypeApplyPatch = %v, want %v", BlockTypeApplyPatch, "APPLY_PATCH")
	}

	if BlockTypeSummary != "SUMMARY" {
		t.Errorf("BlockTypeSummary = %v, want %v", BlockTypeSummary, "SUMMARY")
	}

	if BlockTypeTool != "TOOL" {
		t.Errorf("BlockTypeTool = %v, want %v", BlockTypeTool, "TOOL")
	}

	if BlockTypeTesting != "TESTING" {
		t.Errorf("BlockTypeTesting = %v, want %v", BlockTypeTesting, "TESTING")
	}

	if BlockTypeNotice != "NOTICE" {
		t.Errorf("BlockTypeNotice = %v, want %v", BlockTypeNotice, "NOTICE")
	}

	if BlockTypeError != "ERROR" {
		t.Errorf("BlockTypeError = %v, want %v", BlockTypeError, "ERROR")
	}
}

func TestFoldState_Constants(t *testing.T) {
	t.Parallel(
	// Test that all constants are properly defined.
	)

	if FoldStateExpanded != "expanded" {
		t.Errorf("FoldStateExpanded = %v, want %v", FoldStateExpanded, "expanded")
	}

	if FoldStateCollapsed != "collapsed" {
		t.Errorf("FoldStateCollapsed = %v, want %v", FoldStateCollapsed, "collapsed")
	}
}

func TestSeverity_Constants(t *testing.T) {
	t.Parallel(
	// Test that all constants are properly defined.
	)

	if SeverityInfo != "info" {
		t.Errorf("SeverityInfo = %v, want %v", SeverityInfo, "info")
	}

	if SeverityWarn != "warn" {
		t.Errorf("SeverityWarn = %v, want %v", SeverityWarn, "warn")
	}

	if SeverityError != "error" {
		t.Errorf("SeverityError = %v, want %v", SeverityError, "error")
	}
}

func TestBlockType_AllValid(t *testing.T) {
	t.Parallel(
	// Test that all defined block types are valid.
	)

	blockTypes := []BlockType{
		BlockTypeExecute,
		BlockTypePlan,
		BlockTypeRead,
		BlockTypeGrep,
		BlockTypeApplyPatch,
		BlockTypeSummary,
		BlockTypeTool,
		BlockTypeTesting,
		BlockTypeNotice,
		BlockTypeError,
	}

	for _, bt := range blockTypes {
		if !bt.Valid() {
			t.Errorf("BlockType %v should be valid", bt)
		}
	}
}

func TestFoldState_AllValid(t *testing.T) {
	t.Parallel(
	// Test that all defined fold states are valid.
	)

	foldStates := []FoldState{
		FoldStateExpanded,
		FoldStateCollapsed,
	}

	for _, fs := range foldStates {
		if !fs.Valid() {
			t.Errorf("FoldState %v should be valid", fs)
		}
	}
}

func TestSeverity_AllValid(t *testing.T) {
	t.Parallel(
	// Test that all defined severities are valid.
	)

	severities := []Severity{
		SeverityInfo,
		SeverityWarn,
		SeverityError,
	}

	for _, s := range severities {
		if !s.Valid() {
			t.Errorf("Severity %v should be valid", s)
		}
	}
}
