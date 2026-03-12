package term

import (
	"testing"
)

// TestANSIConstants verifies ANSI escape sequences match specification.
func TestANSIConstants(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

// BenchmarkANSIConstants ensures constants don't allocate.
func BenchmarkANSIConstants(b *testing.B) {
	b.ReportAllocs()

	for range b.N {
		_ = ClearLine
		_ = HideCursor
		_ = ShowCursor
	}
}
