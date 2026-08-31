package suggest

import "unicode"

// TokenAt returns the / or @ token touching cursor (rune index).
func TokenAt(text string, cursor int) Token {
	runes := []rune(text)
	cursor = clampCursor(cursor, len(runes))

	start, end := wordBounds(runes, cursor)
	if start == end {
		return Token{}
	}

	word := string(runes[start:end])

	kind, query := classifyWord(word)
	if kind == KindNone {
		return Token{}
	}

	return Token{Kind: kind, Query: query, Start: start, End: end}
}

func clampCursor(cursor, n int) int {
	if cursor < 0 {
		return 0
	}

	if cursor > n {
		return n
	}

	return cursor
}

func wordBounds(runes []rune, cursor int) (start, end int) {
	start = cursor
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}

	end = cursor
	for end < len(runes) && !unicode.IsSpace(runes[end]) {
		end++
	}

	return start, end
}

func classifyWord(word string) (kind Kind, query string) {
	if word == "" {
		return KindNone, ""
	}

	switch word[0] {
	case '/':
		return KindSlash, word[1:]
	case '@':
		return KindFile, word[1:]
	default:
		return KindNone, ""
	}
}

// Apply replaces the token range with item.Insert.
func Apply(text string, tok Token, item Item) (next string, cursor int) {
	runes := []rune(text)
	start := clampCursor(tok.Start, len(runes))
	end := max(clampCursor(tok.End, len(runes)), start)
	insert := []rune(item.Insert)
	out := make([]rune, 0, len(runes)-end+start+len(insert))
	out = append(out, runes[:start]...)
	out = append(out, insert...)
	out = append(out, runes[end:]...)

	return string(out), start + len(insert)
}
