package undo_test

// Journey: specs/journeys/JOURNEY-R5.1.md.

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/undo"
)

func TestOperationLog_RecordAndUndo_Create(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "new.txt")

	// Create the file first (simulating what WriteFileTool would do after recording).
	require.NoError(t, os.WriteFile(filePath, []byte("created"), 0o600))

	log := undo.NewOperationLog()
	log.Record(undo.Operation{
		Type: undo.OpCreate,
		Path: filePath,
	})

	require.Equal(t, 1, log.Len())

	op, err := log.Undo()
	require.NoError(t, err)
	require.Equal(t, undo.OpCreate, op.Type)
	require.Equal(t, filePath, op.Path)

	// File should be deleted after undo.
	_, statErr := os.Stat(filePath)
	require.True(t, os.IsNotExist(statErr))
	require.Equal(t, 0, log.Len())
}

func TestOperationLog_RecordAndUndo_Modify(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	originalContent := []byte("original content")

	require.NoError(t, os.WriteFile(filePath, originalContent, 0o600))

	log := undo.NewOperationLog()
	log.Record(undo.Operation{
		Type:          undo.OpModify,
		Path:          filePath,
		BeforeContent: originalContent,
	})

	// Simulate the tool modifying the file.
	require.NoError(t, os.WriteFile(filePath, []byte("modified content"), 0o600))

	op, err := log.Undo()
	require.NoError(t, err)
	require.Equal(t, undo.OpModify, op.Type)

	// File should be restored to original content.
	content, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	require.Equal(t, originalContent, content)
}

func TestOperationLog_EmptyLog_ReturnsError(t *testing.T) {
	t.Parallel()

	log := undo.NewOperationLog()

	_, err := log.Undo()
	require.ErrorIs(t, err, undo.ErrEmptyLog)
}

func TestOperationLog_FIFOEviction(t *testing.T) {
	t.Parallel()

	log := undo.NewOperationLog()

	// Fill to capacity + 1.
	for idx := range undo.MaxEntries + 1 {
		log.Record(undo.Operation{
			Type: undo.OpModify,
			Path: filepath.Join(t.TempDir(), fmt.Sprintf("file%c", rune('a'+idx%26))),
		})
	}

	require.Equal(t, undo.MaxEntries, log.Len())
}

func TestOperationLog_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	log := undo.NewOperationLog()

	const goroutineCount = 20

	var wg sync.WaitGroup

	wg.Add(goroutineCount)

	errs := make([]error, goroutineCount)

	for idx := range goroutineCount {
		go func() {
			defer wg.Done()

			filePath := filepath.Join(tmpDir, "concurrent"+string(rune('a'+idx%26))+".txt")

			if writeErr := os.WriteFile(filePath, []byte("test"), 0o600); writeErr != nil {
				errs[idx] = writeErr

				return
			}

			log.Record(undo.Operation{
				Type:          undo.OpModify,
				Path:          filePath,
				BeforeContent: []byte("test"),
			})
		}()
	}

	wg.Wait()

	for idx, writeErr := range errs {
		require.NoError(t, writeErr, "goroutine %d failed to write file", idx)
	}

	require.Equal(t, goroutineCount, log.Len())
}

func TestOperationLog_MultipleUndos(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fileOne := filepath.Join(tmpDir, "one.txt")
	fileTwo := filepath.Join(tmpDir, "two.txt")
	contentOne := []byte("one")
	contentTwo := []byte("two")

	require.NoError(t, os.WriteFile(fileOne, contentOne, 0o600))
	require.NoError(t, os.WriteFile(fileTwo, contentTwo, 0o600))

	log := undo.NewOperationLog()

	log.Record(undo.Operation{Type: undo.OpModify, Path: fileOne, BeforeContent: contentOne})
	require.NoError(t, os.WriteFile(fileOne, []byte("one-modified"), 0o600))

	log.Record(undo.Operation{Type: undo.OpModify, Path: fileTwo, BeforeContent: contentTwo})
	require.NoError(t, os.WriteFile(fileTwo, []byte("two-modified"), 0o600))

	// Undo second operation first (LIFO).
	op, err := log.Undo()
	require.NoError(t, err)
	require.Equal(t, fileTwo, op.Path)

	content, readErr := os.ReadFile(fileTwo)
	require.NoError(t, readErr)
	require.Equal(t, contentTwo, content)

	// Undo first operation.
	op, err = log.Undo()
	require.NoError(t, err)
	require.Equal(t, fileOne, op.Path)

	content, readErr = os.ReadFile(fileOne)
	require.NoError(t, readErr)
	require.Equal(t, contentOne, content)

	require.Equal(t, 0, log.Len())
}

func TestOperationLog_Len(t *testing.T) {
	t.Parallel()

	log := undo.NewOperationLog()
	require.Equal(t, 0, log.Len())

	log.Record(undo.Operation{Type: undo.OpCreate, Path: "/tmp/a"})
	require.Equal(t, 1, log.Len())

	log.Record(undo.Operation{Type: undo.OpCreate, Path: "/tmp/b"})
	require.Equal(t, 2, log.Len())
}

func TestOperationLog_RecordFileChange_NewFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nonexistent.txt")

	log := undo.NewOperationLog()

	err := log.RecordFileChange(filePath)
	require.NoError(t, err)
	require.Equal(t, 1, log.Len())
}

func TestOperationLog_RecordFileChange_ExistingFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "existing.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("existing"), 0o600))

	log := undo.NewOperationLog()

	err := log.RecordFileChange(filePath)
	require.NoError(t, err)
	require.Equal(t, 1, log.Len())
}

func TestOperationType_String(t *testing.T) {
	t.Parallel()

	require.Equal(t, "create", undo.OpCreate.String())
	require.Equal(t, "modify", undo.OpModify.String())
	require.Equal(t, "delete", undo.OpDelete.String())
}

func TestOperationLog_UndoDelete_RestoresFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "deleted.txt")
	content := []byte("restore me")

	log := undo.NewOperationLog()
	log.Record(undo.Operation{
		Type:          undo.OpDelete,
		Path:          filePath,
		BeforeContent: content,
	})

	op, err := log.Undo()
	require.NoError(t, err)
	require.Equal(t, undo.OpDelete, op.Type)

	restored, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	require.Equal(t, content, restored)
}
