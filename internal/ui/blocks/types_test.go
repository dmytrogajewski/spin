package blocks

import "testing"

func TestBlockType_String(t *testing.T) {
	tests := []struct {
		name string
		bt   BlockType
		want string
	}{
		{"execute", BlockTypeExecute, "EXECUTE"},
		{"plan", BlockTypePlan, "PLAN"},
		{"read", BlockTypeRead, "READ"},
		{"grep", BlockTypeGrep, "GREP"},
		{"apply_patch", BlockTypeApplyPatch, "APPLY_PATCH"},
		{"summary", BlockTypeSummary, "SUMMARY"},
		{"testing", BlockTypeTesting, "TESTING"},
		{"notice", BlockTypeNotice, "NOTICE"},
		{"error", BlockTypeError, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bt.String(); got != tt.want {
				t.Errorf("BlockType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockType_Valid(t *testing.T) {
	tests := []struct {
		name string
		bt   BlockType
		want bool
	}{
		{"execute_valid", BlockTypeExecute, true},
		{"plan_valid", BlockTypePlan, true},
		{"invalid_empty", BlockType(""), false},
		{"invalid_unknown", BlockType("UNKNOWN"), false},
		{"invalid_lowercase", BlockType("execute"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bt.Valid(); got != tt.want {
				t.Errorf("BlockType.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFoldState_String(t *testing.T) {
	tests := []struct {
		name string
		fs   FoldState
		want string
	}{
		{"expanded", FoldStateExpanded, "expanded"},
		{"collapsed", FoldStateCollapsed, "collapsed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fs.String(); got != tt.want {
				t.Errorf("FoldState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFoldState_Valid(t *testing.T) {
	tests := []struct {
		name string
		fs   FoldState
		want bool
	}{
		{"expanded_valid", FoldStateExpanded, true},
		{"collapsed_valid", FoldStateCollapsed, true},
		{"invalid_empty", FoldState(""), false},
		{"invalid_unknown", FoldState("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fs.Valid(); got != tt.want {
				t.Errorf("FoldState.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name string
		s    Severity
		want string
	}{
		{"info", SeverityInfo, "info"},
		{"warn", SeverityWarn, "warn"},
		{"error", SeverityError, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("Severity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverity_Valid(t *testing.T) {
	tests := []struct {
		name string
		s    Severity
		want bool
	}{
		{"info_valid", SeverityInfo, true},
		{"warn_valid", SeverityWarn, true},
		{"error_valid", SeverityError, true},
		{"invalid_empty", Severity(""), false},
		{"invalid_unknown", Severity("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Valid(); got != tt.want {
				t.Errorf("Severity.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
