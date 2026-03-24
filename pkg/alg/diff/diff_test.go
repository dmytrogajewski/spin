package diff

// Journey: specs/journeys/JOURNEY-R-REF-19.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	got := Generate("test.go", "old line", "new line")

	require.Contains(t, got, "--- test.go")
	require.Contains(t, got, "+++ test.go")
	require.Contains(t, got, "-old line")
	require.Contains(t, got, "+new line")
}

func TestGenerate_MultiLine(t *testing.T) {
	t.Parallel()

	got := Generate("f.go", "a\nb", "c\nd")

	require.Contains(t, got, "-a\n")
	require.Contains(t, got, "-b\n")
	require.Contains(t, got, "+c\n")
	require.Contains(t, got, "+d\n")
}

func TestParse_Simple(t *testing.T) {
	t.Parallel()

	input := "--- test.go\n+++ test.go\n@@ -1 +1 @@\n-old\n+new"

	filename, hunks, err := Parse(input)
	require.NoError(t, err)
	require.Equal(t, "test.go", filename)
	require.Len(t, hunks, 1)
	require.Len(t, hunks[0].Changes, 2)
	require.Equal(t, LineDelete, hunks[0].Changes[0].Type)
	require.Equal(t, "old", hunks[0].Changes[0].Text)
	require.Equal(t, LineInsert, hunks[0].Changes[1].Type)
	require.Equal(t, "new", hunks[0].Changes[1].Text)
}

func TestParse_GitPrefix(t *testing.T) {
	t.Parallel()

	input := "--- a/src/main.go\n+++ b/src/main.go\n@@ -1 +1 @@\n-old\n+new"

	filename, _, err := Parse(input)
	require.NoError(t, err)
	require.Equal(t, "src/main.go", filename)
}

func TestParse_ContextLines(t *testing.T) {
	t.Parallel()

	input := "--- f.go\n+++ f.go\n@@ -1 +1 @@\n context\n-old\n+new\n context2"

	_, hunks, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, hunks, 1)
	require.Len(t, hunks[0].Changes, 4)
	require.Equal(t, LineContext, hunks[0].Changes[0].Type)
	require.Equal(t, "context", hunks[0].Changes[0].Text)
}

func TestParse_TooShort(t *testing.T) {
	t.Parallel()

	_, _, err := Parse("one\ntwo\n")
	require.Error(t, err)
}

func TestParse_BadHeader(t *testing.T) {
	t.Parallel()

	_, _, err := Parse("not a diff\nline two\nline three\n")
	require.Error(t, err)
}

func TestParse_MultipleHunks(t *testing.T) {
	t.Parallel()

	input := "--- f.go\n+++ f.go\n@@ -1 +1 @@\n-a\n@@ -5 +5 @@\n+b"

	_, hunks, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, hunks, 2)
	require.Len(t, hunks[0].Changes, 1)
	require.Len(t, hunks[1].Changes, 1)
}

func TestParse_StarHeader(t *testing.T) {
	t.Parallel()

	input := "*** file.txt\n+++ file.txt\n@@ -1 +1 @@\n-old\n+new"

	filename, _, err := Parse(input)
	require.NoError(t, err)
	require.Equal(t, "file.txt", filename)
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	original := Generate("main.go", "hello", "world")

	filename, hunks, err := Parse(original)
	require.NoError(t, err)
	require.Equal(t, "main.go", filename)
	require.NotEmpty(t, hunks)
}
