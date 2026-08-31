package suggest

import (
	"context"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/commands"
	"github.com/dmytrogajewski/spin/internal/filesearch"
	"github.com/dmytrogajewski/spin/internal/plugins"
	"github.com/dmytrogajewski/spin/internal/skills"
)

// Source supplies slash and file suggestion rows for a workdir.
type Source struct {
	workDir string
	catalog skills.Catalog

	mu    sync.Mutex
	files []string
}

// NewSource builds a source for workDir. Catalog may be nil (rediscovered).
func NewSource(workDir string, catalog skills.Catalog) *Source {
	return &Source{workDir: workDir, catalog: catalog}
}

// Load scans the workdir for @ file suggestions.
func (s *Source) Load(ctx context.Context) {
	if s == nil || s.workDir == "" {
		return
	}

	files, err := filesearch.NewScanner(s.workDir).ScanWithContext(ctx)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.files = files
	s.mu.Unlock()
}

// Items returns filtered suggestions for the token at cursor.
func (s *Source) Items(text string, cursor int) []Item {
	if s == nil {
		return nil
	}

	tok := TokenAt(text, cursor)
	switch tok.Kind {
	case KindSlash:
		return Filter(s.slashItems(), tok.Query, MaxSuggestions)
	case KindFile:
		return Filter(s.fileItems(), tok.Query, MaxSuggestions)
	default:
		return nil
	}
}

func (s *Source) slashItems() []Item {
	items := commandItems()
	catalog := s.skillCatalog()

	for _, entry := range catalog {
		items = append(items, Item{
			Kind:   KindSlash,
			Insert: "/" + entry.Name,
			Label:  "/" + entry.Name,
			Detail: entry.Description,
		})
	}

	return items
}

func (s *Source) skillCatalog() skills.Catalog {
	if s.catalog != nil {
		return s.catalog
	}

	return plugins.DiscoverCatalog(s.workDir, nil)
}

func commandItems() []Item {
	cmds := commands.ListCommands()
	items := make([]Item, 0, len(cmds))

	for _, cmd := range cmds {
		items = append(items, Item{
			Kind:   KindSlash,
			Insert: cmd.Name(),
			Label:  cmd.Name(),
			Detail: cmd.Description(),
		})
	}

	return items
}

func (s *Source) fileItems() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Item, 0, len(s.files))

	for _, rel := range s.files {
		slash := strings.ReplaceAll(rel, "\\", "/")
		items = append(items, Item{
			Kind:   KindFile,
			Insert: "@" + slash,
			Label:  slash,
			Detail: "project file",
		})
	}

	return items
}
