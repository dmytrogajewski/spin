package lsp

import (
	"path/filepath"
	"strings"
)

// matchMode determines how a Matcher compares names.
type matchMode int

const (
	matchExact matchMode = iota
	matchPrefix
	matchWildcard
)

// Matcher matches symbol names against a pattern.
// Created via [ParseMatcher].
type Matcher struct {
	pattern string
	mode    matchMode
}

// Match returns true if the given symbol name matches the pattern.
func (m Matcher) Match(name string) bool {
	switch m.mode {
	case matchPrefix:
		return strings.HasPrefix(name, m.pattern)
	case matchWildcard:
		matched, _ := filepath.Match(m.pattern, name)

		return matched
	default:
		return name == m.pattern
	}
}

// ParseMatcher creates the appropriate matcher from a pattern string.
//   - Contains '*' or '?' → wildcard using [filepath.Match].
//   - Ends with '.' → prefix match (dot stripped).
//   - Otherwise → exact match.
func ParseMatcher(pattern string) Matcher {
	if strings.ContainsAny(pattern, "*?") {
		return Matcher{pattern: pattern, mode: matchWildcard}
	}

	if prefix, found := strings.CutSuffix(pattern, "."); found {
		return Matcher{pattern: prefix, mode: matchPrefix}
	}

	return Matcher{pattern: pattern, mode: matchExact}
}

// FilterSymbols returns symbols whose names match the given matcher.
func FilterSymbols(symbols []Symbol, matcher Matcher) []Symbol {
	var matched []Symbol

	for _, sym := range symbols {
		if matcher.Match(sym.Name) {
			matched = append(matched, sym)
		}
	}

	return matched
}
