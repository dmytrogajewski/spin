# pkg/ansi - ANSI Escape Sequence Handling

**Package:** `github.com/dmytrogajewski/spin/pkg/ansi`
**Purpose:** Parse, strip, and generate ANSI escape sequences for terminal formatting
**Coverage:** 97.9%
**Status:** ✅ Complete

---

## Overview

`pkg/ansi` provides robust ANSI escape sequence handling for terminal text styling. It supports parsing, stripping, generating, and visual length calculation of ANSI-formatted text.

### Key Features

- **Strip ANSI codes** from text (regex-based, O(n))
- **Calculate visual length** excluding ANSI sequences (UTF-8 aware)
- **Fluent styling API** for colors and text styles
- **Parse ANSI text** into structured segments
- **High performance**: <1μs for typical operations
- **Zero allocations** for no-style cases

---

## Quick Start

```go
import "github.com/dmytrogajewski/spin/pkg/ansi"

// Strip ANSI codes
text := "\x1b[31mRed text\x1b[0m"
plain := ansi.Strip(text)  // "Red text"

// Calculate visual length
len := ansi.Length("\x1b[1mBold\x1b[0m")  // 4

// Style text
styled := ansi.New("Error").Red().Bold().String()
// Output: "\x1b[31m\x1b[1mError\x1b[0m"

// Parse into segments
segments := ansi.Parse("\x1b[31mRed\x1b[0m Normal")
// segments[0]: {Text: "Red", Foreground: "red"}
// segments[1]: {Text: " Normal"}
```

---

## API Reference

### Strip and Length

```go
// Strip removes all ANSI escape sequences
func Strip(text string) string

// Length returns visual length (UTF-8 aware)
func Length(text string) int
```

**Example:**
```go
text := "\x1b[1m\x1b[32mBold Green\x1b[0m"
plain := ansi.Strip(text)   // "Bold Green"
length := ansi.Length(text) // 10
```

**Performance:**
- `Strip`: ~179ns for plain text, ~17μs for 1KB with ANSI
- `Length`: ~237ns for UTF-8, ~16μs for 1KB

---

### Color and Style Constants

```go
const (
    Reset = "\x1b[0m"

    // Foreground colors (30-37)
    Black, Red, Green, Yellow
    Blue, Magenta, Cyan, White

    // Text styles (1-4)
    Bold, Dim, Italic, Underline
)
```

**Usage:**
```go
fmt.Printf("%sError:%s Something failed\n", ansi.Red, ansi.Reset)
// Output (styled): Error: Something failed
```

---

### Fluent Styling API

```go
type Style struct { ... }

// Create styled text
func New(text string) *Style

// Color methods
func (s *Style) Red() *Style
func (s *Style) Green() *Style
func (s *Style) Yellow() *Style
func (s *Style) Blue() *Style
func (s *Style) Magenta() *Style
func (s *Style) Cyan() *Style
func (s *Style) White() *Style
func (s *Style) Black() *Style

// Style methods
func (s *Style) Bold() *Style
func (s *Style) Dim() *Style
func (s *Style) Italic() *Style
func (s *Style) Underline() *Style

// Finalize
func (s *Style) String() string
```

**Examples:**
```go
// Single style
error := ansi.New("Error").Red().String()

// Combined styles
warning := ansi.New("Warning").Yellow().Bold().String()

// Multiple colors (terminal uses last)
multi := ansi.New("Text").Red().Green().Blue().String()

// No style (zero allocations)
plain := ansi.New("Plain").String()  // "Plain"
```

**Performance:**
- With styles: ~50ns, 2 allocs
- No styles: ~3ns, 0 allocs
- All styles: ~97ns, 3 allocs

---

### Parse ANSI Text

```go
type Segment struct {
    Text       string
    Foreground string  // "red", "green", etc.
    Background string  // Reserved for future
    Bold       bool
    Dim        bool
    Italic     bool
    Underline  bool
}

// Parse ANSI text into segments
func Parse(text string) []Segment
```

**Examples:**
```go
// Simple color
text := "\x1b[31mRed\x1b[0m"
segments := ansi.Parse(text)
// segments[0]: {Text: "Red", Foreground: "red"}

// Combined styles
text = "\x1b[1;31mBold Red\x1b[0m"
segments = ansi.Parse(text)
// segments[0]: {Text: "Bold Red", Foreground: "red", Bold: true}

// Multiple segments
text = "\x1b[31mRed\x1b[0m Normal \x1b[1mBold\x1b[0m"
segments = ansi.Parse(text)
// segments[0]: {Text: "Red", Foreground: "red"}
// segments[1]: {Text: " Normal"}
// segments[2]: {Text: "Bold", Bold: true}

// State inheritance
text = "\x1b[1mBold \x1b[31mBold Red\x1b[0m"
segments = ansi.Parse(text)
// segments[0]: {Text: "Bold ", Bold: true}
// segments[1]: {Text: "Bold Red", Foreground: "red", Bold: true}
```

**Supported SGR Codes:**
- `0`: Reset all
- `1`: Bold, `2`: Dim, `3`: Italic, `4`: Underline
- `30-37`: Foreground colors (black through white)

**Performance:**
- Simple: ~624ns, 40 allocs
- Long text: ~1.4μs, 86 allocs

---

## Use Cases

### Terminal Output Formatting

```go
func PrintStatus(status string, message string) {
    var color *ansi.Style
    switch status {
    case "error":
        color = ansi.New(status).Red().Bold()
    case "warning":
        color = ansi.New(status).Yellow()
    case "info":
        color = ansi.New(status).Cyan()
    default:
        color = ansi.New(status)
    }
    fmt.Printf("[%s] %s\n", color.String(), message)
}
```

### Log Processing

```go
// Parse log line and extract plain text
logLine := "\x1b[32m[2025-10-12]\x1b[0m \x1b[1mINFO\x1b[0m: Request completed"

// Strip for storage
plainLog := ansi.Strip(logLine)
saveToDatabase(plainLog)

// Parse for analysis
segments := ansi.Parse(logLine)
for _, seg := range segments {
    if seg.Bold {
        // Highlight bold segments in UI
        highlightText(seg.Text)
    }
}
```

### Text Alignment

```go
// Calculate visual width for proper alignment
func AlignText(text string, width int) string {
    visualLen := ansi.Length(text)
    padding := width - visualLen
    if padding > 0 {
        return text + strings.Repeat(" ", padding)
    }
    return text
}

// Usage
fmt.Println(AlignText("\x1b[31mError\x1b[0m", 20) + "| Failed")
fmt.Println(AlignText("\x1b[32mSuccess\x1b[0m", 20) + "| OK")
// Output (aligned):
// Error               | Failed
// Success             | OK
```

---

## Implementation Details

### Strip Implementation

Uses pre-compiled regex to match ANSI sequences:
- CSI sequences: `\x1b\[[0-9;]*[a-zA-Z]`
- DEC save/restore: `\x1b[78]`

Single-pass replacement with `ReplaceAllString`.

### Length Implementation

1. Strip ANSI codes with `Strip()`
2. Convert to `[]rune` for UTF-8 handling
3. Return rune count

**UTF-8 Support:**
```go
ansi.Length("\x1b[1m你好\x1b[0m")  // 2 (Chinese characters)
ansi.Length("\x1b[32m🔥🚀\x1b[0m")  // 2 (emojis)
```

### Style Implementation

- **Fluent API**: Returns `*Style` for chaining
- **Lazy evaluation**: Builds ANSI codes on `String()` call
- **Optimization**: Pre-allocates slice (4 codes typical)
- **Zero alloc**: No-style case returns text unchanged

### Parse Implementation

- **Byte-by-byte iteration**: Preserves UTF-8 encoding
- **State machine**: Tracks cumulative formatting
- **strings.Builder**: Efficient text accumulation
- **State inheritance**: Formatting persists across codes

---

## Performance Characteristics

| Operation | Small Input | Large Input (1KB) | Notes |
|-----------|-------------|-------------------|-------|
| Strip (no ANSI) | ~179ns | ~2μs | Zero-copy optimization |
| Strip (with ANSI) | ~240ns | ~17μs | Regex replacement |
| Length | ~237ns | ~16μs | Includes Strip + rune count |
| Parse (simple) | ~624ns | - | 40 allocations |
| Parse (complex) | ~1.4μs | - | 86 allocations |
| Style.String() | ~50ns | - | 2 allocations |
| Style (no-op) | ~3ns | - | 0 allocations! |

**Memory:**
- Strip: 1-2KB for 1KB input (regex overhead)
- Parse: ~600B-1.4KB per 40-char string
- Style: 88-240B depending on style count

---

## Testing

### Test Coverage

97.9% statement coverage across all functions.

```bash
go test ./pkg/ansi/... -cover
# output: coverage: 97.9% of statements
```

### Run Tests

```bash
# All tests
go test ./pkg/ansi/...

# With coverage
go test ./pkg/ansi/... -cover

# With race detector
go test -race ./pkg/ansi/...

# Verbose
go test -v ./pkg/ansi/...
```

### Benchmarks

```bash
# Run all benchmarks
go test ./pkg/ansi/... -bench=. -benchmem

# Specific benchmark
go test ./pkg/ansi/... -bench=BenchmarkStrip
```

**Sample Results:**
```
BenchmarkStrip-16                 66248    17499 ns/op    1553 B/op     9 allocs/op
BenchmarkStripNoANSI-16         6590954      179 ns/op    1053 B/op     3 allocs/op
BenchmarkLength-16                76042    15836 ns/op    1550 B/op     9 allocs/op
BenchmarkParse-16               1891617      624 ns/op     624 B/op    40 allocs/op
BenchmarkStyleString-16        23862610       50 ns/op      88 B/op     2 allocs/op
BenchmarkStyleStringNoStyle-16 374094145        3 ns/op       0 B/op     0 allocs/op
```

---

## Limitations

### Not Supported

- **256-color palette** (SGR 38;5;N and 48;5;N)
- **True color RGB** (SGR 38;2;R;G;B and 48;2;R;G;B)
- **Cursor positioning** (use `internal/ui/term` instead)
- **Background colors in parser** (reserved, not parsed yet)
- **Screen buffer control** (alt screen, etc.)

### Malformed Sequences

Best-effort parsing:
- Incomplete sequences (`\x1b[31`) are preserved
- Unknown SGR codes are silently ignored
- Invalid syntax doesn't cause panics

### Terminal Compatibility

Standard SGR codes work on:
- ✅ xterm, vt100, Linux console
- ✅ macOS Terminal, iTerm2
- ✅ Windows Terminal, Windows 10+ console
- ❌ Very old terminals (pre-ANSI)

---

## Future Enhancements

1. **256-color support**: Extended palette for modern terminals
2. **True color (RGB)**: 24-bit color support
3. **Background colors**: Full parsing and generation
4. **Wrap function**: Text wrapping respecting ANSI codes
5. **Truncate function**: Smart truncation preserving formatting
6. **More styles**: Strikethrough, blink, reverse video

---

## Migration from `internal/ui/term`

The `internal/ui/term/ansi.go` provides terminal control sequences (cursor positioning, clear line, etc.) which remain in place. This package (`pkg/ansi`) focuses on text formatting (colors, styles, parsing).

**Before:**
```go
import "spin/internal/ui/term"

// No color/style support, only terminal control
fmt.Print(term.ClearLine + "Text" + term.HideCursor)
```

**After:**
```go
import (
    "spin/internal/ui/term"        // Terminal control
    "github.com/dmytrogajewski/spin/pkg/ansi"  // Text styling
)

// Combine both
fmt.Print(term.ClearLine + ansi.New("Text").Red().String())
```

**Compatibility:** No breaking changes to `internal/ui/term`.

---

## Related Packages

- **[internal/ui/term](term.md)**: Terminal control (cursor, clear, etc.)
- **[internal/ui/blocks](ui-blocks.md)**: Block rendering system
- **[internal/ui/output](ui-output.md)**: Streaming output
- **[pkg/strutil](strutil.md)**: String manipulation utilities

---

## References

- [ANSI Escape Codes (Wikipedia)](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [SGR Parameters](https://en.wikipedia.org/wiki/ANSI_escape_code#SGR)
- [VT100 Codes](https://vt100.net/docs/vt100-ug/chapter3.html)
- [FRD-20251012024500-ansi.md](../../specs/frds/FRD-20251012024500-ansi.md)

---

**Created:** 2025-10-12
**Last Updated:** 2025-10-12
**Status:** ✅ Complete (Feature 1.3)
