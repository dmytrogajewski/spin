package tools

// Journey: specs/journeys/JOURNEY-R4.1.md.

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sleepBeyondTolerance is the duration to sleep to ensure a file modification
// is detectable beyond the modTimeTolerance window.
const sleepBeyondTolerance = modTimeTolerance * 2

func TestFileTracker_RecordAndAssertFresh(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	tracker := NewFileTracker()

	err := tracker.RecordRead(filePath)
	require.NoError(t, err)

	err = tracker.AssertFresh(filePath)
	require.NoError(t, err)
}

func TestFileTracker_FailsAfterModification(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	tracker := NewFileTracker()

	err := tracker.RecordRead(filePath)
	require.NoError(t, err)

	// Wait beyond tolerance window, then modify.
	time.Sleep(sleepBeyondTolerance)

	require.NoError(t, os.WriteFile(filePath, []byte("modified"), 0o600))

	err = tracker.AssertFresh(filePath)
	require.ErrorIs(t, err, ErrFileModifiedSinceRead)
}

// TestFileTracker_AllowsWriteWithoutPriorRead verifies that files never read
// through the tracker can be written freely — the agent may know about them
// through other means (shell commands, cargo new, etc.).
func TestFileTracker_AllowsWriteWithoutPriorRead(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	tracker := NewFileTracker()

	err := tracker.AssertFresh(filePath)
	require.NoError(t, err, "files not previously read should be writable")
}

func TestFileTracker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	tracker := NewFileTracker()

	const goroutineCount = 20

	var wg sync.WaitGroup

	wg.Add(goroutineCount)

	for range goroutineCount {
		go func() {
			defer wg.Done()

			_ = tracker.RecordRead(filePath)
			_ = tracker.AssertFresh(filePath)
		}()
	}

	wg.Wait()
}

func TestFileTracker_ToleranceWindow(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	tracker := NewFileTracker()

	err := tracker.RecordRead(filePath)
	require.NoError(t, err)

	// File not modified — assert fresh should pass even if mod time is within tolerance.
	err = tracker.AssertFresh(filePath)
	require.NoError(t, err)
}

func TestFileTracker_RecordRead_NonExistentFile(t *testing.T) {
	t.Parallel()

	tracker := NewFileTracker()

	err := tracker.RecordRead("/nonexistent/path/file.txt")
	require.Error(t, err)
}

func TestFileTracker_AssertFresh_DeletedFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	tracker := NewFileTracker()

	err := tracker.RecordRead(filePath)
	require.NoError(t, err)

	require.NoError(t, os.Remove(filePath))

	err = tracker.AssertFresh(filePath)
	require.Error(t, err)
}

func TestFileTracker_MultipleFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fileOne := filepath.Join(tmpDir, "one.txt")
	fileTwo := filepath.Join(tmpDir, "two.txt")

	require.NoError(t, os.WriteFile(fileOne, []byte("one"), 0o600))
	require.NoError(t, os.WriteFile(fileTwo, []byte("two"), 0o600))

	tracker := NewFileTracker()

	require.NoError(t, tracker.RecordRead(fileOne))
	require.NoError(t, tracker.RecordRead(fileTwo))

	require.NoError(t, tracker.AssertFresh(fileOne))
	require.NoError(t, tracker.AssertFresh(fileTwo))
}
