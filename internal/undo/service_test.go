package undo_test

// Journey: specs/journeys/JOURNEY-R5.2.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/undo"
)

func TestUndoService_UndoLast_DelegatesToLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")

	log := undo.NewOperationLog()

	// Record a create operation.
	log.Record(undo.Operation{
		Type: undo.OpCreate,
		Path: filePath,
	})

	// Create the file so undo can delete it.
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o600))

	svc := undo.NewService(log, nil)
	op, err := svc.UndoLast()

	require.NoError(t, err)
	require.Equal(t, undo.OpCreate, op.Type)
	require.Equal(t, filePath, op.Path)

	// File should be deleted.
	_, statErr := os.Stat(filePath)
	require.True(t, os.IsNotExist(statErr))
}

func TestUndoService_TakeSnapshot_RecordsHash(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	svc := undo.NewService(undo.NewOperationLog(), mgr)

	hash, err := svc.TakeSnapshot()
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.Equal(t, 1, svc.SnapshotCount())
}

func TestUndoService_TakeSnapshot_NilSnapshot_ReturnsError(t *testing.T) {
	t.Parallel()

	svc := undo.NewService(undo.NewOperationLog(), nil)

	_, err := svc.TakeSnapshot()
	require.ErrorIs(t, err, undo.ErrShadowRepoNotInitialized)
}

func TestUndoService_UndoToStep_RestoresSnapshot(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	svc := undo.NewService(undo.NewOperationLog(), mgr)

	// Take snapshot of initial state (step 0).
	_, err := svc.TakeSnapshot()
	require.NoError(t, err)

	// Modify the file.
	writeTestFile(t, dir, testFileName, testFileUpdated)

	// Take snapshot of modified state (step 1).
	_, snapErr := svc.TakeSnapshot()
	require.NoError(t, snapErr)

	// Undo to step 0.
	require.NoError(t, svc.UndoToStep(0))

	// File should be back to original.
	content := readTestFile(t, dir, testFileName)
	require.Equal(t, testFileContent, content)
}

func TestUndoService_UndoToStep_InvalidStep(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	svc := undo.NewService(undo.NewOperationLog(), mgr)

	err := svc.UndoToStep(0)
	require.Error(t, err)
}

func TestUndoService_UndoToStep_NilSnapshot(t *testing.T) {
	t.Parallel()

	svc := undo.NewService(undo.NewOperationLog(), nil)

	err := svc.UndoToStep(0)
	require.ErrorIs(t, err, undo.ErrShadowRepoNotInitialized)
}

func TestUndoService_OperationLog_ReturnsLog(t *testing.T) {
	t.Parallel()

	log := undo.NewOperationLog()
	svc := undo.NewService(log, nil)

	require.Equal(t, log, svc.OperationLog())
}
