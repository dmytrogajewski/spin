// Package suggest provides TUI prompt completion for / commands, skills, and @ files.
// Journey: specs/journeys/JOURNEY-027-prompt-slash-at-paste.md.
package suggest

// Kind is the suggestion row type.
type Kind int

const (
	// KindNone is an inactive token.
	KindNone Kind = iota
	// KindSlash is a /command or /skill token.
	KindSlash
	// KindFile is an @path token.
	KindFile
)

const (
	// MaxSuggestions is the on-screen list cap.
	MaxSuggestions = 12
	// MaxAttachBytes skips files larger than this when attaching.
	MaxAttachBytes = 64 << 10
)

// Item is one completion row.
type Item struct {
	Kind   Kind
	Insert string
	Label  string
	Detail string
}

// Token is the word at the cursor that can be completed.
type Token struct {
	Kind  Kind
	Query string
	Start int
	End   int
}
