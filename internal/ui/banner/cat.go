// Package banner renders the spin welcome mascot in the terminal.
package banner

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Pixel kinds in the converted sy.png grid.
const (
	pixelEmpty = '.'
	pixelFur   = 'W'
	pixelKey   = 'K'
	pixelEye   = 'E'
	pixelDark  = 'D'
	pixelGlow  = 'G'
)

const (
	furR  = 254
	furG  = 254
	furB  = 254
	keyR  = 254
	keyG  = 196
	keyB  = 48
	glowR = 255
	glowG = 230
	glowB = 120
	eyeR  = 16
	eyeG  = 16
	eyeB  = 22
	dimR  = 130
	dimG  = 130
	dimB  = 142

	halfBlockUpper = "▀"
	halfBlockLower = "▄"
	fullBlock      = "█"

	csiReset          = "\x1b[0m"
	csiBold           = "\x1b[1m"
	csiCursorUpPrefix = "\r\x1b["
	csiCursorUpSuffix = "A"
	csiCursorSave     = "\x1b7"
	csiCursorRestore  = "\x1b8"
	csiRowHomePrefix  = "\x1b["
	csiRowHomeSuffix  = ";1H"
	csiFGPrefix       = "\x1b[38;2;"
	csiBGPrefix       = "\x1b[48;2;"
	csiColorSep       = ";"
	csiSGREnd         = "m"

	defaultFrameDelay    = 90 * time.Millisecond
	defaultLoops         = 1
	defaultBlinkInterval = 2800 * time.Millisecond
	defaultBlinkHold     = 130 * time.Millisecond
	defaultBaseRow       = 1
	halfBlockPair        = 2
	oddRowPad            = 1

	leftMargin = "  "
	noteGap    = "    "
	// Render rows (half-block pairs) where the wordmark appears.
	noteTitleRow = 8
	noteTagRow   = 9

	// Tail region moved by the wag frame (grid coordinates).
	wagRowStart = 24
	wagRowEnd   = 30
	wagColStart = 33
)

// catGrid is sy.png quantized to a 44x42 terminal cell grid.
// Border-connected black is background; enclosed black keeps detail:
// E = blinkable eyes, D = static dark detail (inner ears, nose, key-ring hole).
// Source: ~/sources/sy/assets/sy.png.
var catGrid = []string{
	"....WWW.................WWW.................",
	"...WWWWW...............WWWWW................",
	"...WWDWWWW...........WWWWWDW................",
	"...WWDWWWWW..........WWWWDDW................",
	"...WWDDDWWWW.......WWWWWDDDW................",
	"...WWDDDDWWWW.....WWWWWDDDDW................",
	"...WWDDDDWWWWWWWWWWWWWDDDDDW................",
	"...WWDDDWWWWWWWWWWWWWWWDDDDW................",
	"...WWDDWWWWWWWWWWWWWWWWWDDDW................",
	"...WWDWWWWWWWWWWWWWWWWWWWWDW................",
	"...WWWWWWWWWWWWWWWWWWWWWWWWW................",
	"...WWWWWWWWWWWWWWWWWWWWWWWWW................",
	"...WWWWWEEEWWWWWWWWEEEWWWWWWW...............",
	"..WWWWWEEEEEWWWWWWEEEEEWWWWWW...............",
	"..WWWWWEEEEEWWWWWWEEEEEWWWWWW...............",
	".WWWWWWEEEEEWWWWWWEEEEEWWWWWWWW.............",
	".WWWWWWWEEEWWWWWWWWEEEWWWWWWWWW.............",
	"..WWWWWWWWWWWDDDWWWWWWWWWWWWW...............",
	"..WWWWWWWWWWWDDDWWWWWWWWWWWWW...............",
	"...WWWKKKKWWWWWWWWWWWW.KKWW.................",
	".....KKKKKKDDDDDDDKKKKKKK...................",
	".....KKDDKKKKKKKKKKKKKKKKK..................",
	".....KKDDKKKKKKKKKKKKKKKK...................",
	".....KKDDKK.........KKKKK...................",
	"......KKKK..........KKKKK..........WWWWW....",
	"............WWWWWW................WWWWWWW...",
	"...........WWWWWWWW...............WWWWWWW...",
	"........WWWWWWWWWWW...WWWW........WWWWWWWW..",
	"......WWWWWWWWWWWWWWWWWWWWW........WWWWWWWW.",
	"......WWWWWWWWWWWWWWWWWWWWWWW.........WWWWW.",
	".....WWWWWWWWWWWWWWWWWWWWWWWWW.........WWWW.",
	"....WWWWWWWWWWWWWWWWWWWWWWWWWW.........WWWWW",
	"....WWWWWWWWWWWWWWWWWWWWWWWWWWW........WWWWW",
	"....WWWWWWWWWWWWWWWWWWWWWWWWWWW........WWWWW",
	".....WWWWWWWWWWWWWWWWWWWWW.WWWWW......WWWWW.",
	"......WWWWWW.WWWWWWWWWWWWW.WWWWW.....WWWWWW.",
	"......WWWWWW......WWWWWWWW.WWWWW...WWWWWWWW.",
	"......WWWWWW......WWWWWW..WWWWWWWWWWWWWWWW..",
	".......WWWWW......WWWWWW..WWWWWWWWWWWWWWW...",
	".WWWW..WWWWWW.....WWWWW..WWWWWWWW.WWWWWW....",
	"WWWWW..WWWWWW....WWWWW...WWWWWWWW.WWWW......",
	"WWW.....WWWWW....WWW......WWWWWWW...........",
}

// PlayOptions controls welcome animation timing.
type PlayOptions struct {
	Delay time.Duration
	Sleep func(time.Duration)
	Loops int
}

// RawWriter serializes raw ANSI sequences with other terminal output.
type RawWriter interface {
	WriteRaw(s string) error
}

// BlinkOptions controls the idle blink loop.
type BlinkOptions struct {
	// BaseRow is the 1-based terminal row of the banner's first line.
	BaseRow int
	// Interval is the pause between blinks.
	Interval time.Duration
	// Hold is how long the eyes stay closed.
	Hold time.Duration
	// Active, when set, is checked before each blink; returning false stops the loop.
	Active func() bool
}

// Height is the number of terminal lines one rendered frame occupies.
func Height() int {
	return renderHeight(catGrid)
}

// FrameCount is the number of distinct animation frames.
func FrameCount() int {
	return len(buildFrames())
}

// RenderFrame returns one colored half-block frame (no trailing newline on last line).
func RenderFrame(index int) string {
	frames := buildFrames()
	if index < 0 || index >= len(frames) {
		index = 0
	}

	return renderGrid(frames[index])
}

// Play writes an animated colored cat, leaving the last frame on screen.
func Play(w io.Writer, opts PlayOptions) error {
	if w == nil {
		return nil
	}

	delay, sleep, loops := resolvePlayOptions(opts)
	frames := buildFrames()
	height := renderHeight(frames[0])

	return playFrames(w, frames, height, delay, sleep, loops)
}

// Blink alternates the mascot's eyes in place until ctx is canceled.
// The eyes are always left open on exit. The banner must sit at
// opts.BaseRow with no scrolling in between (stop the loop before any
// transcript output).
func Blink(ctx context.Context, w RawWriter, opts BlinkOptions) error {
	interval, hold, baseRow := resolveBlinkOptions(opts)

	for {
		if !sleepCtx(ctx, interval) {
			return nil
		}

		if opts.Active != nil && !opts.Active() {
			return nil
		}

		if err := w.WriteRaw(EyeOverlay(baseRow, true)); err != nil {
			return err
		}

		held := sleepCtx(ctx, hold)

		if err := w.WriteRaw(EyeOverlay(baseRow, false)); err != nil {
			return err
		}

		if !held {
			return nil
		}
	}
}

func resolveBlinkOptions(opts BlinkOptions) (interval, hold time.Duration, baseRow int) {
	interval = opts.Interval
	if interval <= 0 {
		interval = defaultBlinkInterval
	}

	hold = opts.Hold
	if hold <= 0 {
		hold = defaultBlinkHold
	}

	baseRow = opts.BaseRow
	if baseRow <= 0 {
		baseRow = defaultBaseRow
	}

	return interval, hold, baseRow
}

// sleepCtx waits for d and reports false if ctx was canceled first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// EyeOverlay returns a self-contained ANSI sequence that redraws only the
// eye rows of a banner anchored at baseRow, preserving the cursor position.
func EyeOverlay(baseRow int, closed bool) string {
	grid := catGrid
	if closed {
		blink := cloneGrid(catGrid)
		replacePixels(blink, pixelEye, pixelFur)
		grid = blink
	}

	start, end := eyePairRange()

	var b strings.Builder

	b.WriteString(csiCursorSave)

	for pair := start; pair <= end; pair++ {
		b.WriteString(csiRowHomePrefix + strconv.Itoa(baseRow+pair) + csiRowHomeSuffix)
		b.WriteString(leftMargin)
		writeHalfRow(&b, grid, pair)
		b.WriteString(csiReset)
	}

	b.WriteString(csiCursorRestore)

	return b.String()
}

// eyePairRange returns the inclusive render-row range containing eye pixels.
func eyePairRange() (start, end int) {
	start, end = -1, -1

	for i, row := range catGrid {
		if !strings.ContainsRune(row, pixelEye) {
			continue
		}

		pair := i / halfBlockPair
		if start == -1 {
			start = pair
		}

		end = pair
	}

	return start, end
}

func resolvePlayOptions(opts PlayOptions) (delay time.Duration, sleep func(time.Duration), loops int) {
	delay = opts.Delay
	if delay <= 0 {
		delay = defaultFrameDelay
	}

	sleep = opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	loops = opts.Loops
	if loops <= 0 {
		loops = defaultLoops
	}

	return delay, sleep, loops
}

func playFrames(
	w io.Writer,
	frames [][]string,
	height int,
	delay time.Duration,
	sleep func(time.Duration),
	loops int,
) error {
	for loop := range loops {
		for i, grid := range frames {
			if err := writeFrame(w, grid, height, i > 0 || loop > 0); err != nil {
				return err
			}

			lastLoop := loop == loops-1

			lastFrame := i == len(frames)-1
			if delay > 0 && (!lastLoop || !lastFrame) {
				sleep(delay)
			}
		}
	}

	return nil
}

func writeFrame(w io.Writer, grid []string, height int, rewind bool) error {
	if rewind {
		up := csiCursorUpPrefix + strconv.Itoa(height) + csiCursorUpSuffix
		if err := writeBanner(w, up); err != nil {
			return err
		}
	}

	return writeBanner(w, renderGrid(grid)+"\n")
}

func writeBanner(w io.Writer, s string) error {
	if _, err := io.WriteString(w, s); err != nil {
		return fmt.Errorf("banner write: %w", err)
	}

	return nil
}

func buildFrames() [][]string {
	idle := cloneGrid(catGrid)
	blink := cloneGrid(catGrid)
	replacePixels(blink, pixelEye, pixelFur)

	wag := cloneGrid(catGrid)
	wagTail(wag)

	glow := cloneGrid(catGrid)
	replacePixels(glow, pixelKey, pixelGlow)

	return [][]string{idle, blink, idle, wag, glow, idle}
}

func cloneGrid(src []string) []string {
	out := make([]string, len(src))
	copy(out, src)

	return out
}

func replacePixels(grid []string, from, to byte) {
	for i, row := range grid {
		b := []byte(row)
		for j, c := range b {
			if c == from {
				b[j] = to
			}
		}

		grid[i] = string(b)
	}
}

// wagTail shifts the free tail segment one cell left, leaving the body intact.
func wagTail(grid []string) {
	for i := wagRowStart; i <= wagRowEnd && i < len(grid); i++ {
		b := []byte(grid[i])
		for j := wagColStart; j < len(b)-1; j++ {
			b[j] = b[j+1]
		}

		b[len(b)-1] = pixelEmpty
		grid[i] = string(b)
	}
}

func renderHeight(grid []string) int {
	return (len(grid) + oddRowPad) / halfBlockPair
}

func renderGrid(grid []string) string {
	var b strings.Builder

	rows := renderHeight(grid)
	for pair := range rows {
		if pair > 0 {
			b.WriteByte('\n')
		}

		b.WriteString(leftMargin)
		writeHalfRow(&b, grid, pair)
		b.WriteString(csiReset)
		b.WriteString(sideNote(pair))
	}

	return b.String()
}

// sideNote returns the wordmark text shown to the right of the cat.
func sideNote(pair int) string {
	switch pair {
	case noteTitleRow:
		return noteGap + csiBold + fgRGB(furR, furG, furB) + "spin" + csiReset
	case noteTagRow:
		return noteGap + fgRGB(dimR, dimG, dimB) + "AI coding agent" + csiReset
	default:
		return ""
	}
}

func writeHalfRow(b *strings.Builder, grid []string, pair int) {
	upper := grid[pair*halfBlockPair]

	var lower string
	if pair*halfBlockPair+1 < len(grid) {
		lower = grid[pair*halfBlockPair+1]
	}

	width := contentWidth(upper, lower)
	for x := range width {
		b.WriteString(halfBlock(cellAt(upper, x), cellAt(lower, x)))
	}
}

// contentWidth is the last occupied column + 1 so the wordmark can sit
// against the cat instead of after a run of empty grid padding.
func contentWidth(upper, lower string) int {
	width := max(len(upper), len(lower))
	for width > 0 {
		if cellAt(upper, width-1) != pixelEmpty || cellAt(lower, width-1) != pixelEmpty {
			return width
		}

		width--
	}

	return 0
}

func cellAt(row string, x int) byte {
	if x < 0 || x >= len(row) {
		return pixelEmpty
	}

	return row[x]
}

func halfBlock(upper, lower byte) string {
	uOn, ur, ug, ub := pixelRGB(upper)
	lOn, lr, lg, lb := pixelRGB(lower)

	switch {
	case !uOn && !lOn:
		return " "
	case uOn && lOn && ur == lr && ug == lg && ub == lb:
		return fgRGB(ur, ug, ub) + fullBlock
	case uOn && lOn:
		return fgRGB(ur, ug, ub) + bgRGB(lr, lg, lb) + halfBlockUpper + csiReset
	case uOn:
		return fgRGB(ur, ug, ub) + halfBlockUpper
	default:
		return fgRGB(lr, lg, lb) + halfBlockLower
	}
}

func pixelRGB(c byte) (on bool, r, g, b int) {
	switch c {
	case pixelFur:
		return true, furR, furG, furB
	case pixelKey:
		return true, keyR, keyG, keyB
	case pixelGlow:
		return true, glowR, glowG, glowB
	case pixelEye, pixelDark:
		return true, eyeR, eyeG, eyeB
	default:
		return false, 0, 0, 0
	}
}

func fgRGB(r, g, b int) string {
	return csiFGPrefix + joinRGB(r, g, b) + csiSGREnd
}

func bgRGB(r, g, b int) string {
	return csiBGPrefix + joinRGB(r, g, b) + csiSGREnd
}

func joinRGB(r, g, b int) string {
	return strconv.Itoa(r) + csiColorSep + strconv.Itoa(g) + csiColorSep + strconv.Itoa(b)
}
