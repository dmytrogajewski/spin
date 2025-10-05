package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilePicker(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)

	assert.NotNil(t, fp.list)
	assert.NotNil(t, fp.matcher)
	assert.Equal(t, tmpDir, fp.baseDir)
	assert.Equal(t, 60, fp.width)
	assert.Equal(t, 15, fp.height)
	assert.False(t, fp.active)
	assert.Len(t, fp.files, 1)
}

func TestFilePicker_SetActive(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFilePicker(tmpDir, 60, 15)

	fp.SetActive(true)

	assert.True(t, fp.active)

	fp.SetActive(false)

	assert.False(t, fp.active)
}

func TestFilePicker_SetQuery_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetQuery("")

	// Empty query shows first N files
	assert.LessOrEqual(t, len(fp.filtered), 20)
}

func TestFilePicker_SetQuery_Match(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetQuery("test")

	// Should filter to only "test.go"
	assert.Contains(t, fp.filtered, "test.go")
	assert.NotContains(t, fp.filtered, "main.go")
}

func TestFilePicker_SetQuery_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetQuery("xyz")

	// No matches
	assert.Len(t, fp.filtered, 0)
}

func TestFilePicker_SetQuery_FuzzyMatch(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file_test.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetQuery("ft")

	// Fuzzy match should find "file_test.go" (f...t)
	assert.Contains(t, fp.filtered, "file_test.go")
}

func TestFilePicker_SetQuery_UpdatesList(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetActive(true)
	fp.SetQuery("test")

	// List should be updated
	items := fp.list.Items()
	assert.Len(t, items, 1)
}

func TestFilePicker_GetSelected_NoSelection(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFilePicker(tmpDir, 60, 15)

	selected := fp.GetSelected()

	assert.Equal(t, "", selected)
}

func TestFilePicker_GetSelected_WithSelection(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetActive(true)
	fp.SetQuery("test")

	selected := fp.GetSelected()

	// First item should be selected by default
	assert.Equal(t, "test.go", selected)
}

func TestFilePicker_Update_Inactive(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFilePicker(tmpDir, 60, 15)

	newFp, cmd := fp.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Inactive picker should not respond
	assert.False(t, newFp.active)
	assert.Nil(t, cmd)
}

func TestFilePicker_Update_Active(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetActive(true)

	// Should update list
	newFp, _ := fp.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Check it processed the update
	_ = newFp // Update should work without panic
}

func TestFilePicker_View_Inactive(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFilePicker(tmpDir, 60, 15)

	view := fp.View()

	assert.Equal(t, "", view)
}

func TestFilePicker_View_Active(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetActive(true)
	fp.SetQuery("test")

	view := fp.View()

	// Should render something
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Select File")
}

func TestFilePicker_SetSize(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFilePicker(tmpDir, 60, 15)

	fp.SetSize(80, 20)

	assert.Equal(t, 80, fp.width)
	assert.Equal(t, 20, fp.height)
}

func TestFilePicker_SetOnSelect(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFilePicker(tmpDir, 60, 15)

	called := false
	fp.SetOnSelect(func(path string) {
		called = true
	})

	fp.onSelect("test.go")
	assert.True(t, called)
}

func TestFilePicker_SetOnCancel(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFilePicker(tmpDir, 60, 15)

	called := false
	fp.SetOnCancel(func() {
		called = true
	})

	fp.onCancel()
	assert.True(t, called)
}

func TestFilePicker_Integration_FilterAndSelect(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "app.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644)

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetActive(true)

	// Filter to "app"
	fp.SetQuery("app")

	// Should show only app.go
	assert.Len(t, fp.filtered, 1)
	assert.Equal(t, "app.go", fp.GetSelected())
}

func TestFilePicker_Integration_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetActive(true)

	// No files
	assert.Len(t, fp.files, 0)
	assert.Len(t, fp.filtered, 0)
	assert.Equal(t, "", fp.GetSelected())
}

func TestFilePicker_MaxResults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 30 files
	for i := 0; i < 30; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file_%03d.go", i))
		require.NoError(t, os.WriteFile(filename, []byte(""), 0644))
	}

	fp := NewFilePicker(tmpDir, 60, 15)
	fp.SetQuery("")

	// Should limit to 20 results
	assert.LessOrEqual(t, len(fp.filtered), 20)
}
