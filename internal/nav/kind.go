// Package nav is the structured navigation index (kind records, not raw trees).
package nav

// Kind is one index record family.
type Kind string

const (
	// KindSkill is a skill catalog row.
	KindSkill Kind = "skill"
	// KindPlugin is a plugin catalog row.
	KindPlugin Kind = "plugin"
	// KindSession is a resume-index session.
	KindSession Kind = "session"
	// KindPeer is an A2A peer (card or local spec).
	KindPeer Kind = "peer"
	// KindPath is a filesystem listing pointer.
	KindPath Kind = "path"
	// KindSymbol is a symbol pointer (path, not body).
	KindSymbol Kind = "symbol"

	// ValidKinds is the pipe-separated kind list for errors and schemas.
	ValidKinds = "skill|plugin|session|peer|path|symbol"
)

// Record is one index row. Open is a pointer, never a file body.
type Record struct {
	Kind  Kind   `json:"kind"`
	ID    string `json:"id"`
	Title string `json:"title"`
	Why   string `json:"why"`
	Open  string `json:"open"`
}
