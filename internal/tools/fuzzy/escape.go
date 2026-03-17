package fuzzy

import "strings"

// escapeReplacements maps literal escape sequences to their actual characters.
var escapeReplacements = []struct {
	from string
	to   string
}{
	{`\\`, "\x00ESCAPE_BACKSLASH\x00"}, // Placeholder to avoid double-replacement.
	{`\n`, "\n"},
	{`\t`, "\t"},
	{`\"`, `"`},
}

const backslashPlaceholder = "\x00ESCAPE_BACKSLASH\x00"

// normalizeEscapes replaces literal escape sequences with actual characters.
func normalizeEscapes(str string) string {
	result := str
	for _, repl := range escapeReplacements {
		result = strings.ReplaceAll(result, repl.from, repl.to)
	}

	return strings.ReplaceAll(result, backslashPlaceholder, `\`)
}

// EscapeFind normalizes escape sequences and finds matches.
func EscapeFind(fileContent, oldContent string) []MatchResult {
	normalizedOld := normalizeEscapes(oldContent)

	// Try finding normalized old in original file content.
	matches := ExactFind(fileContent, normalizedOld)
	if len(matches) > 0 {
		return matches
	}

	// Also try normalizing file content.
	normalizedFile := normalizeEscapes(fileContent)

	return findByNormalized(fileContent, normalizedFile, normalizedOld)
}
