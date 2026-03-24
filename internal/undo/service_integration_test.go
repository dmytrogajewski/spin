package undo_test

// Journey: specs/journeys/JOURNEY-2.1.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/undo"
)

func TestUndoService_FullRollbackFlow(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init(context.Background()))

	svc := undo.NewService(undo.NewOperationLog(), mgr)

	// Step 0: snapshot the initial state (hello.txt = "hello world").
	hash0, err := svc.TakeSnapshot()
	require.NoError(t, err)
	require.NotEmpty(t, hash0)

	// Modify the file.
	writeTestFile(t, dir, testFileName, "modified content")

	// Step 1: snapshot the modified state.
	hash1, err := svc.TakeSnapshot()
	require.NoError(t, err)
	require.NotEqual(t, hash0, hash1)
	require.Equal(t, 2, svc.SnapshotCount())

	// Undo to step 0 — should restore original content.
	require.NoError(t, svc.UndoToStep(0))

	content := readTestFile(t, dir, testFileName)
	require.Equal(t, testFileContent, content)
}

func TestUndoService_MultipleSnapshots_RollbackToMiddle(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init(context.Background()))

	svc := undo.NewService(undo.NewOperationLog(), mgr)

	// Step 0: initial state (hello.txt = "hello world").
	_, err := svc.TakeSnapshot()
	require.NoError(t, err)

	// Modify file to state-1.
	const state1Content = "state one"
	writeTestFile(t, dir, testFileName, state1Content)

	// Step 1: snapshot state-1.
	_, err = svc.TakeSnapshot()
	require.NoError(t, err)

	// Modify file to state-2.
	writeTestFile(t, dir, testFileName, "state two")

	// Step 2: snapshot state-2.
	_, err = svc.TakeSnapshot()
	require.NoError(t, err)
	require.Equal(t, 3, svc.SnapshotCount())

	// Rollback to step 1 — should restore state-1 content.
	require.NoError(t, svc.UndoToStep(1))

	content := readTestFile(t, dir, testFileName)
	require.Equal(t, state1Content, content)
}

func TestUndoService_OperationLog_IntegratedWithSnapshot(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init(context.Background()))

	opLog := undo.NewOperationLog()
	svc := undo.NewService(opLog, mgr)

	// Step 0: snapshot initial state (hello.txt = "hello world").
	_, err := svc.TakeSnapshot()
	require.NoError(t, err)

	// Create a new file and record the operation in OpLog.
	newFilePath := filepath.Join(dir, "extra.txt")
	require.NoError(t, opLog.RecordFileChange(newFilePath)) // records OpCreate (file doesn't exist yet).
	writeTestFile(t, dir, "extra.txt", "extra data")

	// Record a modify operation on the original file in OpLog.
	require.NoError(t, opLog.RecordFileChange(filepath.Join(dir, testFileName))) // records OpModify with before-content.
	writeTestFile(t, dir, testFileName, "overwritten")

	require.Equal(t, 2, opLog.Len())

	// Step 1: snapshot the modified state.
	_, err = svc.TakeSnapshot()
	require.NoError(t, err)

	// Undo individual operations via OpLog (fine-grained undo).

	// Undo the modify on hello.txt — restores original "hello world" content.
	op, undoErr := svc.UndoLast()
	require.NoError(t, undoErr)
	require.Equal(t, undo.OpModify, op.Type)

	content := readTestFile(t, dir, testFileName)
	require.Equal(t, testFileContent, content)

	// Undo the create of extra.txt — deletes the file.
	op, undoErr = svc.UndoLast()
	require.NoError(t, undoErr)
	require.Equal(t, undo.OpCreate, op.Type)

	_, statErr := os.Stat(newFilePath)
	require.True(t, os.IsNotExist(statErr))

	// OpLog should now be empty.
	require.Equal(t, 0, opLog.Len())

	// --- Now use snapshot rollback to step 0 (initial state) ---
	// After OpLog undos, the working tree is already back to initial state.
	// Take a fresh snapshot to update the shadow index to reflect the current
	// working tree, then undo to step 0 to verify snapshot rollback works.
	_, err = svc.TakeSnapshot() // step 2: captures current (post-OpLog-undo) state.
	require.NoError(t, err)

	// Mutate the file again so we have something to undo with snapshots.
	writeTestFile(t, dir, testFileName, "post-undo mutation")

	// Take another snapshot so shadow index is current.
	_, err = svc.TakeSnapshot() // step 3.
	require.NoError(t, err)

	// Rollback to step 0 via snapshot — should restore original content.
	require.NoError(t, svc.UndoToStep(0))

	content = readTestFile(t, dir, testFileName)
	require.Equal(t, testFileContent, content)

	// extra.txt should not exist in step 0 (it was created after step 0).
	_, statErr = os.Stat(newFilePath)
	require.True(t, os.IsNotExist(statErr))
}
