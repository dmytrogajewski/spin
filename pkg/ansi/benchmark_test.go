package ansi_test

import (
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/pkg/ansi"
)

func BenchmarkStrip(b *testing.B) {
	text := strings.Repeat("\x1b[31mtext\x1b[0m ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ansi.Strip(text)
	}
}

func BenchmarkStripNoANSI(b *testing.B) {
	text := strings.Repeat("text ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ansi.Strip(text)
	}
}

func BenchmarkLength(b *testing.B) {
	text := strings.Repeat("\x1b[1mbold\x1b[0m ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ansi.Length(text)
	}
}

func BenchmarkLengthUTF8(b *testing.B) {
	text := "\x1b[1m你好世界🔥🚀\x1b[0m"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ansi.Length(text)
	}
}

func BenchmarkParse(b *testing.B) {
	text := "\x1b[31mred\x1b[0m normal \x1b[1mbold\x1b[0m"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ansi.Parse(text)
	}
}

func BenchmarkParseLong(b *testing.B) {
	// Simulate a long log line with multiple styled segments
	text := "\x1b[32m[INFO]\x1b[0m Processing \x1b[1m/path/to/file.txt\x1b[0m: " +
		"\x1b[33mwarning\x1b[0m found, continuing with \x1b[32msuccess\x1b[0m"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ansi.Parse(text)
	}
}

func BenchmarkStyleString(b *testing.B) {
	var result string
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = ansi.New("text").Red().Bold().String()
	}
	_ = result
}

func BenchmarkStyleStringNoStyle(b *testing.B) {
	var result string
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = ansi.New("text").String()
	}
	_ = result
}

func BenchmarkStyleStringAllStyles(b *testing.B) {
	var result string
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = ansi.New("text").Red().Bold().Dim().Italic().Underline().String()
	}
	_ = result
}

// Benchmarks for realistic scenarios

func BenchmarkLogLineProcessing(b *testing.B) {
	// Simul realistic log processing: parse, extract text, measure length
	logLine := "\x1b[32m[2025-10-12 10:30:45]\x1b[0m \x1b[1m\x1b[34mINFO\x1b[0m: Request completed in 42ms"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		segments := ansi.Parse(logLine)
		plain := ansi.Strip(logLine)
		length := ansi.Length(plain)
		_ = segments
		_ = length
	}
}

func BenchmarkTerminalOutput(b *testing.B) {
	// Simulate generating styled terminal output
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		error := ansi.New("Error:").Red().Bold().String()
		warning := ansi.New("Warning:").Yellow().String()
		info := ansi.New("Info:").Cyan().String()
		_ = error
		_ = warning
		_ = info
	}
}
