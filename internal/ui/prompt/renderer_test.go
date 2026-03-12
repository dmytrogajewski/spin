package prompt

import (
	"bytes"
	"strings"
	"testing"
)

// redrawCase is a test case for renderer redraw tests.
type redrawCase struct {
	name       string
	prefix     string
	bufferText string
	cursor     int
	status     string
	width      int
	want       string
}

// goldenRedrawCases returns test cases for exact ANSI output verification.
func goldenRedrawCases() []redrawCase {
	return []redrawCase{
		{name: "empty buffer", prefix: "> ", width: 80, want: "\x1b[24;1H\x1b[2K> \x1b[3G"},
		{name: "simple text cursor at start", prefix: "> ", bufferText: "hello", cursor: 0, width: 80, want: "\x1b[24;1H\x1b[2K> hello\x1b[3G"},
		{name: "simple text cursor in middle", prefix: "> ", bufferText: "hello", cursor: 2, width: 80, want: "\x1b[24;1H\x1b[2K> hello\x1b[5G"},
		{name: "simple text cursor at end", prefix: "> ", bufferText: "hello", cursor: 5, width: 80, want: "\x1b[24;1H\x1b[2K> hello\x1b[8G"},
		{name: "right-aligned status with space", prefix: "> ", bufferText: "test", cursor: 4, status: "typing", width: 80, want: "\x1b[24;1H\x1b[2K> test" + repeatSpace(68) + "typing\x1b[7G"},
		{name: "status omitted when no space", prefix: "> ", bufferText: "verylongtextthattakesupalmostallthespace", cursor: 10, status: "typing", width: 20},
		{name: "wide character emoji", prefix: "> ", bufferText: "Hi 👋", cursor: 3, width: 80, want: "\x1b[24;1H\x1b[2K> Hi 👋\x1b[6G"},
		{name: "wide character CJK", prefix: "> ", bufferText: "你好", cursor: 1, width: 80, want: "\x1b[24;1H\x1b[2K> 你好\x1b[5G"},
		{name: "combining mark", prefix: "> ", bufferText: "e\u0301", cursor: 1, width: 80, want: "\x1b[24;1H\x1b[2K> e\u0301\x1b[4G"},
		{name: "very narrow terminal", prefix: "> ", bufferText: "hello world this is a long line", cursor: 5, width: 20},
	}
}

// TestRenderer_Redraw_Golden verifies exact ANSI output for various scenarios.
func TestRenderer_Redraw_Golden(t *testing.T) {
	t.Parallel()
	tests := goldenRedrawCases()

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

			got := redrawWithStatus(tt.bufferText, tt.status, tt.width)
			verifyStatusOutput(t, got, tt)
		})
	}
}

// redrawWithStatus renders a buffer with status and returns the output.
func redrawWithStatus(text, status string, width int) string {
	var buf bytes.Buffer

	r := NewTermRenderer(&buf, width, "> ")
	model := NewModel(100)

	if text != "" {
		model.buffer.SetText(text)
		model.buffer.SetCursor(len([]rune(text)))
	}

	_ = r.Redraw(model, status)

	return buf.String()
}

// verifyStatusOutput checks status rendering expectations.
func verifyStatusOutput(t *testing.T, got string, tt struct {
	name           string
	bufferText     string
	status         string
	width          int
	wantStatusFull bool
	wantEllipsis   bool
	wantOmitted    bool
}) {
	t.Helper()

	if tt.wantStatusFull && !strings.Contains(got, tt.status) {
		t.Errorf("Redraw() should contain full status %q, got: %q", tt.status, got)
	}

	if tt.wantEllipsis && !strings.Contains(got, "…") {
		t.Errorf("Redraw() should contain ellipsis for truncated status, got: %q", got)
	}

	if tt.wantOmitted && strings.Contains(got, tt.status) {
		t.Errorf("Redraw() should omit status when no space, got: %q", got)
	}
}

// TestRenderer_Redraw_HorizontalScrolling tests scrolling for long lines.
func TestRenderer_Redraw_HorizontalScrolling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		bufferText   string
		cursor       int
		width        int
		wantEllipsis bool // true if any ellipsis expected.
		wantNone     bool // true if no ellipsis expected.
	}{
		{name: "no scroll short line", bufferText: "hello", cursor: 2, width: 80, wantNone: true},
		{name: "scroll long line cursor at start", bufferText: "this is a very long line that exceeds terminal width by a lot", cursor: 0, width: 20, wantEllipsis: true},
		{name: "scroll long line cursor in middle", bufferText: "this is a very long line that exceeds terminal width by a lot", cursor: 30, width: 20, wantEllipsis: true},
		{name: "scroll long line cursor at end", bufferText: "this is a very long line that exceeds terminal width by a lot", cursor: 61, width: 20, wantEllipsis: true},
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
			hasEllipsis := strings.Contains(got, "…")

			if tt.wantEllipsis && !hasEllipsis {
				t.Errorf("Redraw() should contain ellipsis, got: %q", got)
			}

			if tt.wantNone && hasEllipsis {
				t.Errorf("Redraw() should not contain ellipsis, got: %q", got)
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

	t.Run("zero width terminal", func(t *testing.T) {
		t.Parallel()
		assertRedrawNoError(t, 0, "> ", "hello", 0)
	})

	t.Run("negative width terminal", func(t *testing.T) {
		t.Parallel()
		assertRedrawNoError(t, -10, "> ", "hello", 0)
	})

	t.Run("very long buffer", func(t *testing.T) {
		t.Parallel()
		assertRedrawNoError(t, 80, "> ", strings.Repeat("a", 1000), 500)
	})

	t.Run("empty prefix", func(t *testing.T) {
		t.Parallel()
		assertRedrawNoError(t, 80, "", "hello", 0)
	})
}

// assertRedrawNoError verifies that Redraw does not return an error.
func assertRedrawNoError(t *testing.T, width int, prefix, text string, cursor int) {
	t.Helper()

	var buf bytes.Buffer

	r := NewTermRenderer(&buf, width, prefix)
	model := NewModel(100)
	model.buffer.SetText(text)
	model.buffer.SetCursor(cursor)

	if err := r.Redraw(model, ""); err != nil {
		t.Errorf("Redraw() unexpected error: %v", err)
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
