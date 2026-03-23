// Package filesearch provides file search with indexing and fuzzy matching.
// The core matching algorithm lives in [github.com/dmytrogajewski/spin/pkg/alg/pathx].
package filesearch

import "github.com/dmytrogajewski/spin/pkg/alg/pathx"

// Match is an alias for [pathx.Match].
type Match = pathx.Match

// Matcher is an alias for [pathx.Matcher].
type Matcher = pathx.Matcher

// NewMatcher creates a new fuzzy file path matcher.
func NewMatcher(caseSensitive bool) *Matcher {
	return pathx.NewMatcher(caseSensitive)
}
