package patchapply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApplier_UpdateFile_UnifiedDiffContextLines verifies that hunk context
// matching works with lines extracted from a unified diff format.
func TestApplier_UpdateFile_UnifiedDiffContextLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")

	original := "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"

	require.NoError(t, os.WriteFile(filePath, []byte(original), 0o600))

	patch := &Patch{
		Operations: []FileOperation{
			&UpdateFile{
				FilePath: "main.go",
				Hunks: []Hunk{
					{
						Changes: []LineChange{
							{Type: LineContext, Text: "package main"},
							{Type: LineContext, Text: ""},
							{Type: LineContext, Text: "func main() {"},
							{Type: LineDelete, Text: "\tfmt.Println(\"hello\")"},
							{Type: LineInsert, Text: "\tfmt.Println(\"greetings\")"},
							{Type: LineContext, Text: "}"},
						},
					},
				},
			},
		},
	}

	applier, err := NewApplier(dir)
	require.NoError(t, err)

	result, applyErr := applier.Apply(t.Context(), patch)
	require.NoError(t, applyErr, "apply should succeed with context lines from unified diff")
	require.Contains(t, result.FilesUpdated, "main.go")

	content, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	require.Contains(t, string(content), "greetings")
	require.NotContains(t, string(content), "\"hello\"")
}
