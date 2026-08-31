package nav

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
)

const (
	lsCommand         = "ls"
	whyEmptyDirectory = "empty directory"
)

// Paths lists one directory as an R10 tree plus a single path pointer record.
func (idx *Index) Paths(dir string) (Result, error) {
	if dir == "" {
		return Result{}, ErrPathRequired
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{}, fmt.Errorf("read directory: %w", err)
	}

	var raw strings.Builder

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}

		fmt.Fprintln(&raw, "./"+name)
	}

	applied := compact.Default().Apply(lsCommand, []byte(raw.String()), nil, 0)
	listing := string(applied.Stdout)

	title := filepath.Base(dir)
	if title == "." || title == "/" {
		title = dir
	}

	return Result{
		Records: []Record{{
			Kind:  KindPath,
			ID:    dir,
			Title: title,
			Why:   oneLine(firstLine(listing), whyEmptyDirectory),
			Open:  dir,
		}},
		Listing: listing,
	}, nil
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")

	return line
}
