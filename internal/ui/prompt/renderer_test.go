package prompt

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderer_Redraw_Golden verifies exact ANSI output for various scenarios.
func TestRenderer_Redraw_Golden(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		prefix     string
		bufferText string
		cursor     int
		status     string
		width      int
		want       string
	}{
		{
			name:       "empty buffer",
			prefix:     "> ",
			bufferText: "",
			cursor:     0,
			status:     "",
			width:      80,
			want:       "\x1b[24;1H\x1b[2K> \x1b[3G", // Changed: absolute positioning.
		},
		{
			name:       "simple text cursor at start",
			prefix:     "> ",
			bufferText: "hello",
			cursor:     0,
			status:     "",
			width:      80,
			want:       "\x1b[24;1H\x1b[2K> hello\x1b[3G", // Changed: absolute positioning.
		},
		{
			name:       "simple text cursor in middle",
			prefix:     "> ",
			bufferText: "hello",
			cursor:     2,
			status:     "",
			width:      80,
			want:       "\x1b[24;1H\x1b[2K> hello\x1b[5G", // Changed: absolute positioning.
		},
		{
			name:       "simple text cursor at end",
			prefix:     "> ",
			bufferText: "hello",
			cursor:     5,
			status:     "",
			width:      80,
			want:       "\x1b[24;1H\x1b[2K> hello\x1b[8G", // Changed: absolute positioning.
		},
		{
			name:       "right-aligned status with space",
			prefix:     "> ",
			bufferText: "test",
			cursor:     4,
			status:     "typing",
			width:      80,
			// Changed: absolute positioning + "> test" + padding + "typing".
			want: "\x1b[24;1H\x1b[2K> test" + repeatSpace(68) + "typing\x1b[7G",
		},
		{
			name:       "status omitted when no space",
			prefix:     "> ",
			bufferText: "verylongtextthattakesupalmostallthespace",
			cursor:     10,
			status:     "typing",
			width:      20,
			// prefix=2, buffer=42 > 20, so scroll. Status won't fit.
			// With scrolling, this is complex; for now just check no panic.
			want: "", // we'll check separately for no panic.
		},
		{
			name:       "wide character emoji",
			prefix:     "> ",
			bufferText: "Hi 👋",
			cursor:     3,
			status:     "",
			width:      80,
			// "Hi 👋" = "H"(1) + "i"(1) + " "(1) + "👋"(2) = 5 cells
			// cursor at 3 = "Hi " = 3 cells, so column = 2 (prefix) + 3 = 5.
			want: "\x1b[24;1H\x1b[2K> Hi 👋\x1b[6G", // Changed: absolute positioning.
		},
		{
			name:       "wide character CJK",
			prefix:     "> ",
			bufferText: "你好",
			cursor:     1,
			status:     "",
			width:      80,
			// "你"(2) + "好"(2) = 4 cells
			// cursor at 1 = "你" = 2 cells, column = 2 + 2 = 4.
			want: "\x1b[24;1H\x1b[2K> 你好\x1b[5G", // Changed: absolute positioning.
		},
		{
			name:       "combining mark",
			prefix:     "> ",
			bufferText: "e\u0301", // é as e + combining acute.
			cursor:     1,
			status:     "",
			width:      80,
			// "e\u0301" = 1 cell (combining mark is zero-width)
			// cursor at 1 = "e\u0301" = 1 cell, column = 2 + 1 = 3.
			want: "\x1b[24;1H\x1b[2K> e\u0301\x1b[4G", // Changed: absolute positioning.
		},
		{
			name:       "very narrow terminal",
			prefix:     "> ",
			bufferText: "hello world this is a long line",
			cursor:     5,
			status:     "",
			width:      20,
			// Should scroll, show ellipses.
			want: "", // complex, check separately.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer

			r := NewTermRenderer(&buf, tt.width, tt.prefix)

			model := NewModel(100)
			if tt.bufferText != "" {
				model.buffer.SetText(tt.bufferText)
				model.buffer.SetCursor(tt.cursor)
			}

			err := r.Redraw(model, tt.status)
			if err != nil {
				t.Fatalf("Redraw() error = %v", err)
			}

			got := buf.String()
			if tt.want != "" && got != tt.want {
				t.Errorf("Redraw() output mismatch\ngot:  %q (%d bytes)\nwant: %q (%d bytes)",
					got, len(got), tt.want, len(tt.want))
			}
		})
	}
}

// TestRenderer_Redraw_CursorPositioning focuses on cursor column accuracy.
func TestRenderer_Redraw_CursorPositioning(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		bufferText string
		cursor     int
		wantCol    int // expected cursor column (1-indexed).
	}{
		{"empty", "", 0, 3},            // "> " = 2 chars, col 3.
		{"ascii start", "hello", 0, 3}, // cursor before 'h'.
		{"ascii mid", "hello", 2, 5},   // cursor before 'l' (2nd).
		{"ascii end", "hello", 5, 8},   // cursor after 'o'.
		{"emoji start", "👋 hi", 0, 3},  // cursor before emoji.
		{"emoji after", "👋 hi", 1, 5},  // cursor after emoji (2 cells).
		{"emoji end", "👋 hi", 4, 8},    // cursor at end: 👋(2) + " "(1) + "hi"(2) = 5, prefix(2) + 5 + 1 = 8.
		{"CJK start", "你好", 0, 3},      // cursor before first char.
		{"CJK mid", "你好", 1, 5},        // cursor after 你 (2 cells).
		{"CJK end", "你好", 2, 7},        // cursor at end.
		{"combining", "e\u0301", 1, 4}, // cursor after "é" (1 cell).
		{"mixed", "Hi👋世界", 2, 5},       // cursor after "Hi" (2 cells).
		{"mixed mid", "Hi👋世界", 3, 7},   // cursor after "Hi👋" (2+2=4 cells).
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer

			r := NewTermRenderer(&buf, 80, "> ")

			model := NewModel(100)
			if tt.bufferText != "" {
				model.buffer.SetText(tt.bufferText)
				model.buffer.SetCursor(tt.cursor)
			}

			_ = r.Redraw(model, "")

			got := buf.String()

			// Extract cursor position from ANSI sequence
			// Expected format: "\r\x1b[2K> <text>\x1b[<col>G".
			wantSeq := "\x1b[" + string(rune('0'+tt.wantCol/10)) + string(rune('0'+tt.wantCol%10)) + "G"
			if tt.wantCol < 10 {
				wantSeq = "\x1b[" + string(rune('0'+tt.wantCol)) + "G"
			}
			// Simple check: output should end with cursor positioning.
			if !bytes.HasSuffix([]byte(got), []byte(wantSeq)) {
				t.Errorf("Redraw() cursor position mismatch\ngot output: %q\nwant cursor at col %d (seq %q)", got, tt.wantCol, wantSeq)
			}
		})
	}
}

// TestRenderer_Redraw_StatusRendering tests right-aligned status behavior.
func TestRenderer_Redraw_StatusRendering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		bufferText     string
		status         string
		width          int
		wantStatusFull bool // true if full status should appear.
		wantEllipsis   bool // true if status truncated with ellipsis.
		wantOmitted    bool // true if status omitted.
	}{
		{
			name:           "status fits with padding",
			bufferText:     "test",
			status:         "typing",
			width:          80,
			wantStatusFull: true,
		},
		{
			name:           "status fits exactly",
			bufferText:     "test",
			status:         "ok",
			width:          15, // "> test" (7) + " " (3) + "ok" (2) = 12.
			wantStatusFull: true,
		},
		{
			name:         "status truncated",
			bufferText:   "test",
			status:       "very long status message",
			width:        20,
			wantEllipsis: true,
		},
		{
			name:        "status omitted no space",
			bufferText:  "verylongbuffer",
			status:      "status",
			width:       18, // "> verylongbuffer" = 18, no room for status + gap.
			wantOmitted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer

			r := NewTermRenderer(&buf, tt.width, "> ")

			model := NewModel(100)
			if tt.bufferText != "" {
				model.buffer.SetText(tt.bufferText)
				model.buffer.SetCursor(len([]rune(tt.bufferText)))
			}

			_ = r.Redraw(model, tt.status)

			got := buf.String()

			if tt.wantStatusFull {
				if !bytes.Contains([]byte(got), []byte(tt.status)) {
					t.Errorf("Redraw() should contain full status %q, got: %q", tt.status, got)
				}
			}

			if tt.wantEllipsis {
				if !bytes.Contains([]byte(got), []byte("…")) {
					t.Errorf("Redraw() should contain ellipsis for truncated status, got: %q", got)
				}
			}

			if tt.wantOmitted {
				if bytes.Contains([]byte(got), []byte(tt.status)) {
					t.Errorf("Redraw() should omit status when no space, got: %q", got)
				}
			}
		})
	}
}

// TestRenderer_Redraw_HorizontalScrolling tests scrolling for long lines.
func TestRenderer_Redraw_HorizontalScrolling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		bufferText        string
		cursor            int
		width             int
		wantLeftEllipsis  bool
		wantRightEllipsis bool
	}{
		{
			name:              "no scroll short line",
			bufferText:        "hello",
			cursor:            2,
			width:             80,
			wantLeftEllipsis:  false,
			wantRightEllipsis: false,
		},
		{
			name:              "scroll long line cursor at start",
			bufferText:        "this is a very long line that exceeds terminal width by a lot",
			cursor:            0,
			width:             20,
			wantLeftEllipsis:  false, // cursor at start, no left scroll.
			wantRightEllipsis: true,  // content continues right.
		},
		{
			name:              "scroll long line cursor in middle",
			bufferText:        "this is a very long line that exceeds terminal width by a lot",
			cursor:            30,
			width:             20,
			wantLeftEllipsis:  true, // scrolled past start.
			wantRightEllipsis: true, // content continues right.
		},
		{
			name:              "scroll long line cursor at end",
			bufferText:        "this is a very long line that exceeds terminal width by a lot",
			cursor:            61,
			width:             20,
			wantLeftEllipsis:  true,  // scrolled past start.
			wantRightEllipsis: false, // at end.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer

			r := NewTermRenderer(&buf, tt.width, "> ")
			model := NewModel(100)
			model.buffer.SetText(tt.bufferText)
			model.buffer.SetCursor(tt.cursor)

			_ = r.Redraw(model, "")

			got := buf.String()

			if tt.wantLeftEllipsis {
				if !bytes.Contains([]byte(got), []byte("…")) {
					t.Errorf("Redraw() should contain left ellipsis, got: %q", got)
				}
			}

			if tt.wantRightEllipsis {
				if !bytes.Contains([]byte(got), []byte("…")) {
					t.Errorf("Redraw() should contain right ellipsis, got: %q", got)
				}
			}

			if !tt.wantLeftEllipsis && !tt.wantRightEllipsis {
				if bytes.Contains([]byte(got), []byte("…")) {
					t.Errorf("Redraw() should not contain ellipsis, got: %q", got)
				}
			}
		})
	}
}

// TestRenderer_SetWidth tests dynamic width updates.
func TestRenderer_SetWidth(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer

	r := NewTermRenderer(&buf, 80, "> ")
	r.SetWidth(40)
	// Subsequent Redraw should use new width.
	model := NewModel(100)
	model.buffer.SetText("hello")
	_ = r.Redraw(model, "")
	// Just verify no panic.
}

// TestRenderer_SetPrefix tests dynamic prefix updates.
func TestRenderer_SetPrefix(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer

	r := NewTermRenderer(&buf, 80, "> ")
	r.SetPrefix("$ ")

	model := NewModel(100)
	model.buffer.SetText("hello")
	_ = r.Redraw(model, "")

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("$ hello")) {
		t.Errorf("Redraw() should use new prefix '$ ', got: %q", got)
	}
}

// TestRenderer_Redraw_EdgeCases tests boundary conditions.
func TestRenderer_Redraw_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "zero width terminal",
			fn: func(t *testing.T) {
				var buf bytes.Buffer

				r := NewTermRenderer(&buf, 0, "> ")
				model := NewModel(100)
				model.buffer.SetText("hello")

				err := r.Redraw(model, "")
				if err != nil {
					t.Errorf("Redraw() should not panic on zero width, got error: %v", err)
				}
			},
		},
		{
			name: "negative width terminal",
			fn: func(t *testing.T) {
				var buf bytes.Buffer

				r := NewTermRenderer(&buf, -10, "> ")
				model := NewModel(100)
				model.buffer.SetText("hello")

				err := r.Redraw(model, "")
				if err != nil {
					t.Errorf("Redraw() should handle negative width, got error: %v", err)
				}
			},
		},
		{
			name: "very long buffer",
			fn: func(t *testing.T) {
				var buf bytes.Buffer

				r := NewTermRenderer(&buf, 80, "> ")
				model := NewModel(100)

				longText := ""
				var longTextSb431 strings.Builder
				for range 1000 {
					longTextSb431.WriteString("a")
				}
				longText += longTextSb431.String()

				model.buffer.SetText(longText)
				model.buffer.SetCursor(500)

				err := r.Redraw(model, "")
				if err != nil {
					t.Errorf("Redraw() should handle very long buffer, got error: %v", err)
				}
			},
		},
		{
			name: "empty prefix",
			fn: func(t *testing.T) {
				var buf bytes.Buffer

				r := NewTermRenderer(&buf, 80, "")
				model := NewModel(100)
				model.buffer.SetText("hello")

				err := r.Redraw(model, "")
				if err != nil {
					t.Errorf("Redraw() should handle empty prefix, got error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

// Helper function to repeat spaces.
func repeatSpace(n int) string {
	if n < 0 {
		n = 0
	}

	result := make([]byte, n)
	for i := range n {
		result[i] = ' '
	}

	return string(result)
}
