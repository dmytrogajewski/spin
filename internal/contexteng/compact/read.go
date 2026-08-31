package compact

import (
	"strings"
)

// Read levels (R8).
const (
	LevelNone       = "none"
	LevelMinimal    = "minimal"
	LevelAggressive = "aggressive"
)

const (
	flagLevelShort = "-l"
	flagLevelLong  = "--level="
)

func filterRead(cmd string, stdout, stderr []byte) (compactedStdout, compactedStderr []byte, err error) {
	return applyRead(readLevel(cmd), stdout), stderr, nil
}

func readLevel(cmd string) string {
	fields := strings.Fields(cmd)
	for idx, field := range fields {
		if field == flagLevelShort && idx+1 < len(fields) {
			return fields[idx+1]
		}

		if level, found := strings.CutPrefix(field, flagLevelLong); found {
			return level
		}
	}

	return LevelMinimal
}

func applyRead(level string, stdout []byte) []byte {
	switch level {
	case LevelNone:
		return stdout
	case LevelAggressive:
		return aggressiveRead(minimalRead(stdout))
	case LevelMinimal:
		return minimalRead(stdout)
	default:
		return minimalRead(stdout)
	}
}

func minimalRead(stdout []byte) []byte {
	text := stripBlockComments(string(stdout))
	lines := decodeLines([]byte(text))
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		kept = append(kept, stripLineComment(line))
	}

	return encodeLines(collapseBlank(kept))
}

func stripLineComment(line string) string {
	if cut, found := cutComment(line, "//"); found {
		return strings.TrimRight(cut, " \t")
	}

	if hashComment(line) {
		cut, _ := cutComment(line, "#")

		return strings.TrimRight(cut, " \t")
	}

	return strings.TrimRight(line, " \t")
}

func hashComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#!") {
		return false
	}

	_, found := cutComment(line, "#")

	return found
}

func cutComment(line, mark string) (string, bool) {
	idx := strings.Index(line, mark)
	if idx < 0 {
		return line, false
	}

	if mark == "//" && idx > 0 && line[idx-1] == ':' {
		return line, false
	}

	return line[:idx], true
}

func stripBlockComments(text string) string {
	var builder strings.Builder

	builder.Grow(len(text))

	idx := 0
	for idx < len(text) {
		if idx+1 < len(text) && text[idx] == '/' && text[idx+1] == '*' {
			end := strings.Index(text[idx+2:], "*/")
			if end < 0 {
				break
			}

			idx += end + len("/*") + len("*/")

			continue
		}

		builder.WriteByte(text[idx])
		idx++
	}

	return builder.String()
}

var signatureLeaders = []string{
	"package ", "import ", "func ", "type ", "const ", "var ",
	"class ", "def ", "async def ", "fn ", "pub ", "export ",
	"interface ",
}

func aggressiveRead(stdout []byte) []byte {
	lines := decodeLines(stdout)
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			kept = append(kept, "")

			continue
		}

		if isSignature(line) {
			kept = append(kept, trimSignature(line))
		}
	}

	return encodeLines(collapseBlank(kept))
}

func isSignature(line string) bool {
	trim := strings.TrimSpace(line)
	for _, lead := range signatureLeaders {
		if strings.HasPrefix(trim, lead) {
			return true
		}
	}

	return false
}

func trimSignature(line string) string {
	trim := strings.TrimRight(strings.TrimSpace(line), " \t")
	trim = strings.TrimSuffix(trim, "{")

	return strings.TrimRight(trim, " \t")
}
