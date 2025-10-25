package term

import (
	"bytes"
	"context"
	"io"
	"time"
	"unicode/utf8"
)

// KeyKind identifies the type of key event.
type KeyKind int

const (
	KeyRune KeyKind = iota
	KeyEnter
	KeyTab
	KeyBackspace
	KeyDelete
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDn
	KeyEscape
	KeyCtrlC
	KeyCtrlD
	KeyCtrlU // kill line left
	KeyCtrlK // kill line right
	KeyCtrlW // delete word left
	KeyCtrlL // redraw
	KeyCtrlP // command palette
	KeyPaste // bracketed paste
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyUnknown // fallback for unrecognized sequences
)

// String returns a human-readable name for the key kind.
func (k KeyKind) String() string {
	names := map[KeyKind]string{
		KeyRune:      "Rune",
		KeyEnter:     "Enter",
		KeyTab:       "Tab",
		KeyBackspace: "Backspace",
		KeyDelete:    "Delete",
		KeyUp:        "Up",
		KeyDown:      "Down",
		KeyLeft:      "Left",
		KeyRight:     "Right",
		KeyHome:      "Home",
		KeyEnd:       "End",
		KeyPgUp:      "PgUp",
		KeyPgDn:      "PgDn",
		KeyEscape:    "Escape",
		KeyCtrlC:     "Ctrl-C",
		KeyCtrlD:     "Ctrl-D",
		KeyCtrlU:     "Ctrl-U",
		KeyCtrlK:     "Ctrl-K",
		KeyCtrlW:     "Ctrl-W",
		KeyCtrlL:     "Ctrl-L",
		KeyCtrlP:     "Ctrl-P",
		KeyPaste:     "Paste",
		KeyF1:        "F1",
		KeyF2:        "F2",
		KeyF3:        "F3",
		KeyF4:        "F4",
		KeyF5:        "F5",
		KeyF6:        "F6",
		KeyF7:        "F7",
		KeyF8:        "F8",
		KeyF9:        "F9",
		KeyF10:       "F10",
		KeyF11:       "F11",
		KeyF12:       "F12",
		KeyUnknown:   "Unknown",
	}
	if name, ok := names[k]; ok {
		return name
	}
	return "Unknown"
}

// KeyEvent represents a single keyboard event.
type KeyEvent struct {
	Kind  KeyKind
	Rune  rune   // valid when Kind == KeyRune
	Paste []byte // valid when Kind == KeyPaste
	Raw   []byte // raw input bytes (for debugging)
}

// KeyReaderConfig configures the key reader.
type KeyReaderConfig struct {
	EscTimeout time.Duration // timeout for disambiguating ESC (default 100ms)
}

const (
	defaultEscTimeout = 100 * time.Millisecond
	maxPasteSize      = 1024 * 1024 // 1MB limit for bracketed paste
)

// ReadKeys reads keyboard events from r and sends them to the returned channel.
// It stops when ctx is canceled or r returns EOF.
// The channel is closed when the function exits.
//
// GOROUTINE LIFECYCLE:
// - ReadKeys() spawns one long-lived goroutine that:
//   - Reads bytes from stdin in a blocking loop
//   - Parses escape sequences and UTF-8 characters
//   - Sends KeyEvents to the returned channel
//   - Lives until context cancellation or EOF
//   - Automatically closes the channel on exit
//
// - parseEscapeSequence() spawns short-lived goroutines to:
//   - Read next byte with timeout for escape sequence detection
//   - Lives only during escape sequence parsing (~50ms)
//
// CONCURRENCY:
// - Safe to call ReadKeys() multiple times with different readers
// - Each call has its own independent goroutine and channel
// - Context cancellation ensures clean shutdown
func ReadKeys(ctx context.Context, r io.Reader, cfg *KeyReaderConfig) (<-chan KeyEvent, error) {
	if cfg == nil {
		cfg = &KeyReaderConfig{EscTimeout: defaultEscTimeout}
	}
	if cfg.EscTimeout == 0 {
		cfg.EscTimeout = defaultEscTimeout
	}

	ch := make(chan KeyEvent, 16) // buffered for burst input
	go func() {
		defer close(ch)
		parser := &keyParser{
			r:          r,
			cfg:        cfg,
			escTimeout: cfg.EscTimeout,
		}
		parser.run(ctx, ch)
	}()

	return ch, nil
}

// keyParser maintains state for parsing key sequences.
type keyParser struct {
	r          io.Reader
	cfg        *KeyReaderConfig
	escTimeout time.Duration
	buf        [1]byte // single-byte read buffer
}

// run is the main parsing loop.
func (p *keyParser) run(ctx context.Context, ch chan<- KeyEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := p.r.Read(p.buf[:])
		if err != nil {
			return // EOF or error
		}
		if n == 0 {
			continue
		}

		b := p.buf[0]
		event := p.parseByte(ctx, b)

		select {
		case ch <- event:
		case <-ctx.Done():
			return
		}
	}
}

// parseByte parses a single byte and returns the corresponding key event.
// It may read additional bytes for escape sequences.
func (p *keyParser) parseByte(ctx context.Context, b byte) KeyEvent {
	if event := p.parseControlChar(b); event.Kind != KeyUnknown {
		return event
	}

	if b == 0x1b {
		return p.parseEscapeSequence(ctx)
	}

	// UTF-8 or ASCII printable
	return p.parseRune(ctx, b)
}

// parseControlChar parses control characters.
func (p *keyParser) parseControlChar(b byte) KeyEvent {
	switch b {
	case 0x03: // Ctrl-C
		return KeyEvent{Kind: KeyCtrlC, Raw: []byte{b}}
	case 0x04: // Ctrl-D
		return KeyEvent{Kind: KeyCtrlD, Raw: []byte{b}}
	case 0x08, 0x7f: // Backspace
		return KeyEvent{Kind: KeyBackspace, Raw: []byte{b}}
	case '\t': // Tab
		return KeyEvent{Kind: KeyTab, Raw: []byte{b}}
	case '\n', '\r': // Enter
		return KeyEvent{Kind: KeyEnter, Raw: []byte{b}}
	case 0x0b: // Ctrl-K
		return KeyEvent{Kind: KeyCtrlK, Raw: []byte{b}}
	case 0x0c: // Ctrl-L
		return KeyEvent{Kind: KeyCtrlL, Raw: []byte{b}}
	case 0x10: // Ctrl-P
		return KeyEvent{Kind: KeyCtrlP, Raw: []byte{b}}
	case 0x15: // Ctrl-U
		return KeyEvent{Kind: KeyCtrlU, Raw: []byte{b}}
	case 0x17: // Ctrl-W
		return KeyEvent{Kind: KeyCtrlW, Raw: []byte{b}}
	default:
		return KeyEvent{Kind: KeyUnknown, Raw: []byte{b}}
	}
}

// parseEscapeSequence handles ESC sequences with timeout.
func (p *keyParser) parseEscapeSequence(ctx context.Context) KeyEvent {
	raw := []byte{0x1b}

	// Try to read next byte with timeout
	timer := time.NewTimer(p.escTimeout)
	defer timer.Stop()

	nextByte := make(chan byte, 1)
	errChan := make(chan error, 1)

	go func() {
		n, err := p.r.Read(p.buf[:])
		if err != nil {
			errChan <- err
			return
		}
		if n > 0 {
			nextByte <- p.buf[0]
		}
	}()

	select {
	case b := <-nextByte:
		raw = append(raw, b)
		return p.parseEscapeSeq(ctx, b, raw)
	case <-errChan:
		return KeyEvent{Kind: KeyEscape, Raw: raw}
	case <-timer.C:
		return KeyEvent{Kind: KeyEscape, Raw: raw}
	case <-ctx.Done():
		return KeyEvent{Kind: KeyEscape, Raw: raw}
	}
}

// parseEscapeSeq parses the byte following ESC.
func (p *keyParser) parseEscapeSeq(ctx context.Context, b byte, raw []byte) KeyEvent {
	switch b {
	case '[':
		return p.parseCSI(ctx, raw)
	case 'O':
		return p.parseSS3(ctx, raw)
	default:
		// Unknown escape sequence
		return KeyEvent{Kind: KeyUnknown, Raw: raw}
	}
}

// parseCSI handles Control Sequence Introducer sequences (ESC [).
func (p *keyParser) parseCSI(ctx context.Context, raw []byte) KeyEvent {
	seq, err := p.readSequence(ctx, 8) // read up to 8 more bytes
	if err != nil {
		return KeyEvent{Kind: KeyUnknown, Raw: raw}
	}

	raw = append(raw, seq...)

	// Check for bracketed paste mode
	if bytes.Equal(seq, []byte("200~")) {
		return p.parseBracketedPaste(ctx, raw)
	}

	// Try single character sequences first
	if event, ok := p.parseSingleCharCSI(seq, raw); ok {
		return event
	}

	// Try tilde-terminated sequences
	if event, ok := p.parseTildeCSI(seq, raw); ok {
		return event
	}

	return KeyEvent{Kind: KeyUnknown, Raw: raw}
}

// parseSingleCharCSI parses single-character CSI sequences.
func (p *keyParser) parseSingleCharCSI(seq, raw []byte) (KeyEvent, bool) {
	if len(seq) != 1 {
		return KeyEvent{}, false
	}

	var kind KeyKind
	switch seq[0] {
	case 'A':
		kind = KeyUp
	case 'B':
		kind = KeyDown
	case 'C':
		kind = KeyRight
	case 'D':
		kind = KeyLeft
	case 'H':
		kind = KeyHome
	case 'F':
		kind = KeyEnd
	default:
		return KeyEvent{}, false
	}

	return KeyEvent{Kind: kind, Raw: raw}, true
}

// tildeCSIMap maps tilde-terminated CSI codes to key kinds.
var tildeCSIMap = map[string]KeyKind{
	"1":  KeyHome,
	"2":  KeyDelete,
	"3":  KeyDelete,
	"4":  KeyEnd,
	"5":  KeyPgUp,
	"6":  KeyPgDn,
	"15": KeyF5,
	"17": KeyF6,
	"18": KeyF7,
	"19": KeyF8,
	"20": KeyF9,
	"21": KeyF10,
	"23": KeyF11,
	"24": KeyF12,
}

// parseTildeCSI parses tilde-terminated CSI sequences (e.g., ESC[3~).
func (p *keyParser) parseTildeCSI(seq, raw []byte) (KeyEvent, bool) {
	if len(seq) < 2 || seq[len(seq)-1] != '~' {
		return KeyEvent{}, false
	}

	code := string(seq[:len(seq)-1])
	kind, ok := tildeCSIMap[code]
	if !ok {
		return KeyEvent{}, false
	}

	return KeyEvent{Kind: kind, Raw: raw}, true
}

// parseSS3 handles Single Shift 3 sequences (ESC O), typically function keys.
func (p *keyParser) parseSS3(ctx context.Context, raw []byte) KeyEvent {
	n, err := p.r.Read(p.buf[:])
	if err != nil || n == 0 {
		return KeyEvent{Kind: KeyUnknown, Raw: raw}
	}

	raw = append(raw, p.buf[0])

	switch p.buf[0] {
	case 'P':
		return KeyEvent{Kind: KeyF1, Raw: raw}
	case 'Q':
		return KeyEvent{Kind: KeyF2, Raw: raw}
	case 'R':
		return KeyEvent{Kind: KeyF3, Raw: raw}
	case 'S':
		return KeyEvent{Kind: KeyF4, Raw: raw}
	default:
		return KeyEvent{Kind: KeyUnknown, Raw: raw}
	}
}

// parseBracketedPaste handles bracketed paste mode (ESC[200~ ... ESC[201~).
func (p *keyParser) parseBracketedPaste(ctx context.Context, raw []byte) KeyEvent {
	var paste []byte
	endMarker := []byte{0x1b, '[', '2', '0', '1', '~'}
	buf := make([]byte, 0, 4096) // start with 4KB buffer

	for {
		select {
		case <-ctx.Done():
			return KeyEvent{Kind: KeyPaste, Paste: paste, Raw: raw}
		default:
		}

		n, err := p.r.Read(p.buf[:])
		if err != nil {
			// EOF or error - return what we have
			return KeyEvent{Kind: KeyPaste, Paste: paste, Raw: raw}
		}
		if n == 0 {
			continue
		}

		buf = append(buf, p.buf[0])

		// Check if we've hit the end marker
		if len(buf) >= len(endMarker) {
			if bytes.HasSuffix(buf, endMarker) {
				// Remove end marker from paste
				paste = buf[:len(buf)-len(endMarker)]
				return KeyEvent{Kind: KeyPaste, Paste: paste, Raw: raw}
			}
		}

		// Safety: limit paste size
		if len(buf) > maxPasteSize {
			return KeyEvent{Kind: KeyPaste, Paste: buf, Raw: raw}
		}
	}
}

// parseRune decodes a UTF-8 rune starting with the given byte.
func (p *keyParser) parseRune(ctx context.Context, b byte) KeyEvent {
	raw := []byte{b}

	// Single-byte ASCII
	if b < 0x80 {
		return KeyEvent{Kind: KeyRune, Rune: rune(b), Raw: raw}
	}

	// Multi-byte UTF-8
	expectedLen := p.getUTF8Length(b)
	if expectedLen == 0 {
		return KeyEvent{Kind: KeyUnknown, Raw: raw}
	}

	raw = p.readUTF8Continuation(raw, expectedLen)
	if len(raw) != expectedLen {
		return KeyEvent{Kind: KeyUnknown, Raw: raw}
	}

	return p.decodeUTF8Rune(raw)
}

// getUTF8Length determines the expected length of a UTF-8 sequence.
func (p *keyParser) getUTF8Length(b byte) int {
	if b&0xe0 == 0xc0 {
		return 2
	}
	if b&0xf0 == 0xe0 {
		return 3
	}
	if b&0xf8 == 0xf0 {
		return 4
	}
	return 0 // Invalid UTF-8 start byte
}

// readUTF8Continuation reads continuation bytes for UTF-8 sequence.
func (p *keyParser) readUTF8Continuation(raw []byte, expectedLen int) []byte {
	for i := 1; i < expectedLen; i++ {
		n, err := p.r.Read(p.buf[:])
		if err != nil || n == 0 {
			return raw // Incomplete UTF-8 sequence
		}
		raw = append(raw, p.buf[0])
	}
	return raw
}

// decodeUTF8Rune decodes a UTF-8 sequence into a rune.
func (p *keyParser) decodeUTF8Rune(raw []byte) KeyEvent {
	r, size := utf8.DecodeRune(raw)
	if r == utf8.RuneError && size == 1 {
		return KeyEvent{Kind: KeyUnknown, Raw: raw}
	}
	return KeyEvent{Kind: KeyRune, Rune: r, Raw: raw}
}

// readSequence reads up to maxLen bytes until a terminating character.
// Returns when it sees a letter, '~', or reaches maxLen.
func (p *keyParser) readSequence(ctx context.Context, maxLen int) ([]byte, error) {
	seq := make([]byte, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		n, err := p.r.Read(p.buf[:])
		if err != nil || n == 0 {
			return seq, err
		}

		b := p.buf[0]
		seq = append(seq, b)

		// Termination conditions
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
			return seq, nil
		}
	}
	return seq, nil
}
