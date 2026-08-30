package banner

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const eyeRowProbe = 13

func TestCatGrid_UniformWidth(t *testing.T) {
	t.Parallel()

	want := len(catGrid[0])
	for i, row := range catGrid {
		if len(row) != want {
			t.Fatalf("row %d width = %d, want %d", i, len(row), want)
		}
	}
}

func TestRenderFrame_HasColorAndCatShape(t *testing.T) {
	t.Parallel()

	got := RenderFrame(0)
	if !strings.Contains(got, "38;2;") {
		t.Fatalf("expected 24-bit color in frame, got %q", truncate(got))
	}

	if !strings.Contains(got, fullBlock) && !strings.Contains(got, halfBlockUpper) {
		t.Fatal("expected block glyphs in cat frame")
	}

	if strings.Count(got, "\n")+1 != renderHeight(catGrid) {
		t.Fatalf("frame height = %d, want %d", strings.Count(got, "\n")+1, renderHeight(catGrid))
	}

	if !strings.Contains(got, joinRGB(eyeR, eyeG, eyeB)) {
		t.Fatal("idle frame should paint eye color")
	}

	if !strings.Contains(got, joinRGB(keyR, keyG, keyB)) {
		t.Fatal("idle frame should paint key color")
	}

	if !strings.Contains(got, "spin") {
		t.Fatal("frame should include the wordmark")
	}
}

func TestBuildFrames_BlinkAndWagDiffer(t *testing.T) {
	t.Parallel()

	frames := buildFrames()
	if len(frames) < 4 {
		t.Fatalf("frames = %d, want at least 4", len(frames))
	}

	if frames[0][eyeRowProbe] == frames[1][eyeRowProbe] {
		t.Fatal("blink frame should change the eye row")
	}

	if strings.ContainsRune(frames[1][eyeRowProbe], pixelEye) {
		t.Fatal("blink frame should close both eyes")
	}

	if frames[0][wagRowStart] == frames[3][wagRowStart] {
		t.Fatal("wag frame should move tail pixels")
	}

	bodyProbe := frames[3][wagRowStart][:wagColStart]
	if bodyProbe != frames[0][wagRowStart][:wagColStart] {
		t.Fatal("wag frame must not disturb the body")
	}
}

func TestPlay_RedrawsFrames(t *testing.T) {
	t.Parallel()

	var b strings.Builder

	slept := 0

	err := Play(&b, PlayOptions{
		Delay: time.Millisecond,
		Sleep: func(_ time.Duration) { slept++ },
		Loops: 1,
	})
	if err != nil {
		t.Fatalf("Play: %v", err)
	}

	out := b.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatal("Play should write ANSI sequences")
	}

	if slept == 0 {
		t.Fatal("Play should sleep between frames")
	}
}

func TestFrameCount(t *testing.T) {
	t.Parallel()

	if FrameCount() != 6 {
		t.Fatalf("FrameCount() = %d, want 6", FrameCount())
	}
}

func TestEyeOverlay_ClosedHidesEyes(t *testing.T) {
	t.Parallel()

	open := EyeOverlay(1, false)
	if !strings.Contains(open, joinRGB(eyeR, eyeG, eyeB)) {
		t.Fatal("open overlay should paint eye color")
	}

	// The nose (dark detail) shares the eye color and stays visible,
	// so the closed overlay has strictly fewer dark cells, not zero.
	closed := EyeOverlay(1, true)

	openDark := strings.Count(open, joinRGB(eyeR, eyeG, eyeB))

	closedDark := strings.Count(closed, joinRGB(eyeR, eyeG, eyeB))
	if closedDark >= openDark {
		t.Fatalf("closed overlay dark cells = %d, want fewer than open %d", closedDark, openDark)
	}
}

func TestEyeOverlay_AnchorsAtBaseRowAndRestoresCursor(t *testing.T) {
	t.Parallel()

	got := EyeOverlay(1, false)

	start, end := eyePairRange()
	if start < 0 || end < start {
		t.Fatalf("eyePairRange() = %d, %d; want a valid range", start, end)
	}

	for pair := start; pair <= end; pair++ {
		cup := csiRowHomePrefix + strconv.Itoa(1+pair) + csiRowHomeSuffix
		if !strings.Contains(got, cup) {
			t.Fatalf("overlay missing cursor positioning %q", cup)
		}
	}

	if !strings.HasPrefix(got, csiCursorSave) || !strings.HasSuffix(got, csiCursorRestore) {
		t.Fatal("overlay must save and restore the cursor")
	}

	if strings.Contains(got, "\n") {
		t.Fatal("overlay must not contain newlines (would scroll)")
	}

	if strings.Contains(got, "spin") {
		t.Fatal("overlay must not repaint the wordmark (ghosts a second spin when rows drift)")
	}
}

func TestRenderFrame_WordmarkHugsCat(t *testing.T) {
	t.Parallel()

	for line := range strings.SplitSeq(stripCSI(RenderFrame(0)), "\n") {
		idx := strings.Index(line, "spin")
		if idx < 0 {
			continue
		}

		prefix := strings.TrimRight(line[:idx], " ")
		gap := idx - len(prefix)

		if gap > len(noteGap) {
			t.Fatalf("wordmark gap = %d spaces, want <= %d (line %q)", gap, len(noteGap), line)
		}

		return
	}

	t.Fatal("wordmark line not found")
}

// blinkRecorder records overlay writes and cancels the context after
// the first write so the loop winds down deterministically.
type blinkRecorder struct {
	mu     sync.Mutex
	writes []string
	cancel context.CancelFunc
}

func (r *blinkRecorder) WriteRaw(s string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.writes = append(r.writes, s)
	if len(r.writes) == 1 {
		r.cancel()
	}

	return nil
}

func TestBlink_ReopensEyesOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &blinkRecorder{cancel: cancel}

	err := Blink(ctx, rec, BlinkOptions{
		BaseRow:  1,
		Interval: time.Millisecond,
		Hold:     time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Blink: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()

	const wantWrites = 2
	if len(rec.writes) != wantWrites {
		t.Fatalf("writes = %d, want %d (close then reopen)", len(rec.writes), wantWrites)
	}

	if rec.writes[0] != EyeOverlay(1, true) {
		t.Fatal("first write should close the eyes")
	}

	if rec.writes[1] != EyeOverlay(1, false) {
		t.Fatal("final write should reopen the eyes")
	}
}

func TestBlink_StopsWhenInactive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rec := &blinkRecorder{cancel: cancel}

	err := Blink(ctx, rec, BlinkOptions{
		Interval: time.Millisecond,
		Hold:     time.Millisecond,
		Active:   func() bool { return false },
	})
	if err != nil {
		t.Fatalf("Blink: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()

	if len(rec.writes) != 0 {
		t.Fatalf("writes = %d, want 0 when inactive", len(rec.writes))
	}
}

func stripCSI(s string) string {
	var b strings.Builder

	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			i++
			if i < len(s) && s[i] == '[' {
				i++
				for i < len(s) && (s[i] < '@' || s[i] > '~') {
					i++
				}

				if i < len(s) {
					i++
				}

				continue
			}

			continue
		}

		b.WriteByte(s[i])
		i++
	}

	return b.String()
}

func truncate(s string) string {
	if len(s) > 80 {
		return s[:80]
	}

	return s
}
