package term

import (
	"testing"
)

// TestANSIConstants verifies ANSI escape sequences match specification.
func TestANSIConstants(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ClearLine", ClearLine, "\x1b[2K"},
		{"HideCursor", HideCursor, "\x1b[?25l"},
		{"ShowCursor", ShowCursor, "\x1b[?25h"},
		{"SaveCursor", SaveCursor, "\x1b7"},
		{"RestoreCursor", RestoreCursor, "\x1b8"},
		{"CarriageRet", CarriageRet, "\r"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

// TestMoveCursorToCol golden tests for column positioning.
func TestMoveCursorToCol(t *testing.T) {
	tests := []struct {
		col  int
		want string
	}{
		{1, "\x1b[1G"},
		{10, "\x1b[10G"},
		{80, "\x1b[80G"},
		{200, "\x1b[200G"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := MoveCursorToCol(tt.col)
			if got != tt.want {
				t.Errorf("MoveCursorToCol(%d) = %q, want %q", tt.col, got, tt.want)
			}
		})
	}
}

// BenchmarkMoveCursorToCol ensures zero allocations in hot path.
func BenchmarkMoveCursorToCol(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = MoveCursorToCol(42)
	}
}

// BenchmarkANSIConstants ensures constants don't allocate.
func BenchmarkANSIConstants(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ClearLine
		_ = HideCursor
		_ = ShowCursor
	}
}
