package suggest

// Journey: specs/journeys/JOURNEY-027-prompt-slash-at-paste.md.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
)

func TestTokenAt_SlashAndFile(t *testing.T) {
	t.Parallel()

	slash := TokenAt("/sk", 3)
	require.Equal(t, KindSlash, slash.Kind)
	require.Equal(t, "sk", slash.Query)

	file := TokenAt("see @cmd/s", 10)
	require.Equal(t, KindFile, file.Kind)
	require.Equal(t, "cmd/s", file.Query)

	plain := TokenAt("hello", 5)
	require.Equal(t, KindNone, plain.Kind)
}

func TestFilter_PrefixFirst(t *testing.T) {
	t.Parallel()

	items := []Item{
		{Insert: "/mode", Label: "/mode"},
		{Insert: "/skills", Label: "/skills"},
	}
	got := Filter(items, "sk", MaxSuggestions)
	require.NotEmpty(t, got)
	require.Equal(t, "/skills", got[0].Insert)
}

func TestApply_ReplacesToken(t *testing.T) {
	t.Parallel()

	tok := TokenAt("/sk", 3)
	next, cur := Apply("/sk", tok, Item{Insert: "/skills"})
	require.Equal(t, "/skills", next)
	require.Equal(t, len([]rune("/skills")), cur)
}

func TestExpand_SkillKeepsRemainderCase(t *testing.T) {
	t.Parallel()

	dir := writeSkill(t, t.TempDir(), "review-pr", "Review a PR.", "# Review\n")
	catalog := skills.Catalog{{Name: "review-pr", Location: dir, Source: skills.SourceProject}}

	got := Expand("/review-pr Auth.go", t.TempDir(), catalog)
	require.False(t, got.IsCommand)
	require.Contains(t, got.Prompt, "# Review")
	require.Contains(t, got.Prompt, "Auth.go")
	require.Contains(t, got.Prompt, `name="review-pr"`)
}

func TestExpand_CommandWinsOverSkill(t *testing.T) {
	t.Parallel()

	dir := writeSkill(t, t.TempDir(), "help", "Skill help.", "# Skill body\n")
	catalog := skills.Catalog{{Name: "help", Location: dir, Source: skills.SourceProject}}

	got := Expand("/help", t.TempDir(), catalog)
	require.True(t, got.IsCommand)
	require.Equal(t, "/help", got.Command)
	require.NotContains(t, got.Prompt, "# Skill body")
}

func TestExpand_UnknownSlashStaysCommand(t *testing.T) {
	t.Parallel()

	got := Expand("/nope", t.TempDir(), nil)
	require.True(t, got.IsCommand)
	require.Equal(t, "/nope", got.Command)
}

func TestExpand_AttachFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o600))

	got := Expand("look at @a.go", root, nil)
	require.False(t, got.IsCommand)
	require.Contains(t, got.Prompt, "<attached path=\"a.go\">")
	require.Contains(t, got.Prompt, "package a")
	require.Contains(t, got.Prompt, "look at @a.go")
}

func TestExpand_AttachEscape(t *testing.T) {
	t.Parallel()

	got := Expand("x @../secret", t.TempDir(), nil)
	require.NotContains(t, got.Prompt, "<attached")
}

func TestClassifyPaste_PathsAndText(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "p1.go"), []byte("one"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "p2.go"), []byte("two"), 0o600))

	paths := ClassifyPaste([]byte("p1.go\np2.go\n"), root)
	require.Equal(t, "@p1.go @p2.go", paths.Text)

	text := ClassifyPaste([]byte("hello"), root)
	require.Equal(t, "hello", text.Text)
}

func TestClassifyPaste_PNG(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	raw := append([]byte{0x89, 0x50, 0x4e, 0x47}, []byte("not-a-real-png")...)
	got := ClassifyPaste(raw, root)
	require.True(t, strings.HasPrefix(got.Text, "@.spin/paste/"))
	require.True(t, strings.HasSuffix(got.Text, ".png"))

	rel := strings.TrimPrefix(got.Text, "@")
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	require.NoError(t, err)
}

func TestSource_SlashItemsIncludeCommands(t *testing.T) {
	t.Parallel()

	src := NewSource(t.TempDir(), nil)
	items := src.Items("/", 1)
	names := make([]string, 0, len(items))

	for _, item := range items {
		names = append(names, item.Insert)
	}

	require.Contains(t, strings.Join(names, " "), "/help")
}

func writeSkill(t *testing.T, parent, name, desc, body string) string {
	t.Helper()

	dir := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(dir, 0o750))

	content := "---\nname: " + name + "\ndescription: " + desc + "\n---\n\n" + body
	require.NoError(t, os.WriteFile(filepath.Join(dir, skills.FileName), []byte(content), 0o600))

	return dir
}
