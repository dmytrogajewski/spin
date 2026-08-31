package filesearch

// Journey: specs/journeys/JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls.md.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrep_FixtureHits(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "pkg"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "pkg", "a.go"), []byte("short match\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "pkg", "b.go"), []byte("other file match\n"), 0o600))

	got, err := Grep(t.Context(), root, "match")
	require.NoError(t, err)

	if !strings.Contains(got, "pkg/a.go:1:short match") {
		t.Fatalf("Grep = %q, want pkg/a.go hit", got)
	}

	if !strings.Contains(got, "pkg/b.go:1:other file match") {
		t.Fatalf("Grep = %q, want pkg/b.go hit", got)
	}
}
