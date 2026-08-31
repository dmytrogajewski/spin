package nav

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
)

// Records lists catalog rows for kind. Path listings use Paths, not this method.
func (idx *Index) Records(kind Kind) ([]Record, error) {
	if idx == nil {
		return nil, nil
	}

	switch kind {
	case KindSkill:
		return idx.skillRecords(), nil
	case KindPlugin:
		return idx.pluginRecords(), nil
	case KindSession:
		return idx.sessionRecords(), nil
	case KindPeer:
		return idx.peerRecords(), nil
	case KindSymbol:
		return idx.symbolRecords(""), nil
	case KindPath:
		return nil, fmt.Errorf("%w", ErrPathRequired)
	default:
		return nil, unknownKind(kind)
	}
}

// Lookup returns records for a query. kind=path fills Result.Listing via R10.
func (idx *Index) Lookup(query Query) (Result, error) {
	if idx == nil {
		return Result{}, nil
	}

	if query.Kind == KindPath {
		return idx.Paths(query.Path)
	}

	records, err := idx.Records(query.Kind)
	if err != nil {
		return Result{}, err
	}

	if query.Kind == KindSymbol && query.Name != "" {
		records = idx.symbolRecords(query.Name)
	}

	return Result{Records: filterID(records, query.ID)}, nil
}

func (idx *Index) skillRecords() []Record {
	records := make([]Record, 0, len(idx.sources.Skills))
	for _, entry := range idx.sources.Skills {
		records = append(records, Record{
			Kind:  KindSkill,
			ID:    entry.Name,
			Title: entry.Name,
			Why:   oneLine(entry.Description, whySkillFallback),
			Open:  entry.Location,
		})
	}

	return records
}

func (idx *Index) pluginRecords() []Record {
	records := make([]Record, 0, len(idx.sources.Plugins))
	for _, plugin := range idx.sources.Plugins {
		records = append(records, Record{
			Kind:  KindPlugin,
			ID:    plugin.Name,
			Title: plugin.Name,
			Why:   oneLine(plugin.Description, whyPluginFallback),
			Open:  plugin.Root,
		})
	}

	return records
}

func (idx *Index) sessionRecords() []Record {
	if idx.sources.Sessions == nil {
		return nil
	}

	entries := idx.sources.Sessions.List("")
	records := make([]Record, 0, len(entries))

	for _, entry := range entries {
		open := entry.WorkDir
		if open == "" {
			open = entry.ID
		}

		title := entry.Title
		if title == "" {
			title = entry.ID
		}

		records = append(records, Record{
			Kind:  KindSession,
			ID:    entry.ID,
			Title: title,
			Why:   oneLine(entry.Title, whySessionFallback),
			Open:  open,
		})
	}

	return records
}

func (idx *Index) peerRecords() []Record {
	if len(idx.sources.Peers) > 0 {
		records := make([]Record, 0, len(idx.sources.Peers))
		for _, peer := range idx.sources.Peers {
			records = append(records, Record{
				Kind:  KindPeer,
				ID:    peer.ID,
				Title: oneLine(peer.Title, peer.ID),
				Why:   oneLine(peer.Why, whyPeerFallback),
				Open:  peer.Open,
			})
		}

		return records
	}

	return specsToPeers(subagent.Builtins())
}

func (idx *Index) symbolRecords(name string) []Record {
	if idx.sources.Symbols == nil {
		return nil
	}

	hits := idx.sources.Symbols.Find(name)
	records := make([]Record, 0, len(hits))

	for _, hit := range hits {
		records = append(records, Record{
			Kind:  KindSymbol,
			ID:    hit.Name,
			Title: hit.Name,
			Why:   oneLine(hit.Why, whySymbolFallback),
			Open:  hit.Open,
		})
	}

	return records
}
