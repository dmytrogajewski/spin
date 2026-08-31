package nav

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/skills"
)

var (
	// ErrUnknownKind is returned when Records/Lookup sees a kind outside ValidKinds.
	ErrUnknownKind = errors.New("unknown navigate kind")
	// ErrPathRequired is returned when kind=path is missing a directory.
	ErrPathRequired = errors.New("path is required for kind=path")
)

const (
	whySkillFallback   = "skill catalog entry"
	whyPluginFallback  = "plugin catalog entry"
	whySessionFallback = "resume session"
	whyPeerFallback    = "A2A peer"
	whySymbolFallback  = "symbol"
	peerURLPrefix      = "stdio://"
)

// SessionSource is the resume index list surface (session.Index.List).
type SessionSource interface {
	List(workDir string) []session.IndexEntry
}

// SymbolHit is a pointer to a symbol, never a source body.
type SymbolHit struct {
	Name string
	Open string
	Why  string
}

// SymbolSource finds symbols without returning file bodies.
type SymbolSource interface {
	Find(name string) []SymbolHit
}

// Peer is a local or remote A2A peer pointer.
type Peer struct {
	ID    string
	Title string
	Why   string
	Open  string
}

// PluginRow is a live plugin catalog pointer (root path, not plugin.json body).
type PluginRow struct {
	Name        string
	Description string
	Root        string
}

// Sources are live catalogs the index reads. Missing sources yield empty lists.
type Sources struct {
	Skills   skills.Catalog
	Plugins  []PluginRow
	Sessions SessionSource
	Peers    []Peer
	Symbols  SymbolSource
	WorkDir  string
}

// Query selects a kind and optional filters.
type Query struct {
	Kind Kind
	ID   string
	Path string
	Name string
}

// Result is records plus an optional R10 path listing.
type Result struct {
	Records []Record `json:"records"`
	Listing string   `json:"listing,omitempty"`
}

// Index returns structured navigation records instead of raw trees.
type Index struct {
	sources Sources
}

// New builds an index over injected catalogs.
func New(sources Sources) *Index {
	return &Index{sources: sources}
}

func oneLine(text, fallback string) string {
	trimmed := strings.Join(strings.Fields(text), " ")
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

func filterID(records []Record, id string) []Record {
	if id == "" {
		return records
	}

	out := make([]Record, 0, 1)

	for _, record := range records {
		if record.ID == id {
			out = append(out, record)
		}
	}

	return out
}

func unknownKind(kind Kind) error {
	return fmt.Errorf("%w: %s (valid: %s)", ErrUnknownKind, kind, ValidKinds)
}

func specsToPeers(specs []*subagent.Spec) []Record {
	records := make([]Record, 0, len(specs))

	for _, spec := range specs {
		if spec == nil {
			continue
		}

		records = append(records, Record{
			Kind:  KindPeer,
			ID:    spec.Name,
			Title: spec.Name,
			Why:   oneLine(spec.Description, whyPeerFallback),
			Open:  peerURLPrefix + spec.Name,
		})
	}

	return records
}
