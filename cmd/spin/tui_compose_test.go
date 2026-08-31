package main

// Journey: specs/journeys/JOURNEY-027-prompt-slash-at-paste.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
)

func TestComposeTUILine_HelpIsCommand(t *testing.T) {
	t.Parallel()

	got := composeTUILine("/help", t.TempDir())
	require.True(t, got.IsCommand)
	require.Equal(t, "/help", got.Command)
}

func TestComposeTUILine_AttachesFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "n.go"), []byte("package n\n"), 0o600))

	got := composeTUILine("see @n.go", root)
	require.False(t, got.IsCommand)
	require.Contains(t, got.Prompt, "package n")
}

func TestComposeTUILine_SkillFromWorkdir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, ".agents", "skills", "review-pr")
	require.NoError(t, os.MkdirAll(dir, 0o750))

	body := "---\nname: review-pr\ndescription: Review a pull request for tests.\n---\n\n# Review body\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, skills.FileName), []byte(body), 0o600))

	got := composeTUILine("/review-pr Auth.go", root)
	require.False(t, got.IsCommand)
	require.Contains(t, got.Prompt, "# Review body")
	require.Contains(t, got.Prompt, "Auth.go")
}
