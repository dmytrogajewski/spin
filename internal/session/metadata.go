package session

// Metadata contains session metadata.
type Metadata struct {
	Title       string   // User-friendly session title.
	Description string   // Session description.
	Tags        []string // User-defined tags.
	TotalTurns  int      // Total turn count.
	TokensUsed  int      // Total tokens consumed.
	LastError   string   // Last error message (if any).
}
