package session_test

// Journey: specs/journeys/JOURNEY-R6.2.md.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/session"
)

const (
	testSessionID1   = "sess-001"
	testSessionID2   = "sess-002"
	testSessionID3   = "sess-003"
	testSessionTitle = "Test Session"
	testWorkDirA     = "/project/a"
	testWorkDirB     = "/project/b"
	testMsgCount     = 42
	testIndexFile    = "sessions-index.json"
)

func indexPath(dir string) string {
	return filepath.Join(dir, testIndexFile)
}

func testEntry(sessionID, title, workDir string, msgCount int, lastMod time.Time) session.IndexEntry {
	return session.IndexEntry{
		ID:           sessionID,
		Title:        title,
		MessageCount: msgCount,
		LastModified: lastMod,
		WorkDir:      workDir,
	}
}

func TestSessionIndex_UpdateAndList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	entry := testEntry(testSessionID1, testSessionTitle, testWorkDirA, testMsgCount, now)
	require.NoError(t, idx.Update(entry))

	entries := idx.List("")
	require.Len(t, entries, 1)
	require.Equal(t, testSessionID1, entries[0].ID)
	require.Equal(t, testSessionTitle, entries[0].Title)
	require.Equal(t, testMsgCount, entries[0].MessageCount)
}

func TestSessionIndex_UpdateUpserts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)
	updatedTitle := "Updated Title"

	entry := testEntry(testSessionID1, testSessionTitle, testWorkDirA, testMsgCount, now)
	require.NoError(t, idx.Update(entry))

	const updatedCount = 99

	entry.Title = updatedTitle
	entry.MessageCount = updatedCount
	require.NoError(t, idx.Update(entry))

	entries := idx.List("")
	require.Len(t, entries, 1)
	require.Equal(t, updatedTitle, entries[0].Title)
	require.Equal(t, updatedCount, entries[0].MessageCount)
}

func TestSessionIndex_Remove(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	require.NoError(t, idx.Update(testEntry(testSessionID1, testSessionTitle, testWorkDirA, testMsgCount, now)))
	require.NoError(t, idx.Update(testEntry(testSessionID2, testSessionTitle, testWorkDirA, testMsgCount, now)))

	require.NoError(t, idx.Remove(testSessionID1))

	entries := idx.List("")
	require.Len(t, entries, 1)
	require.Equal(t, testSessionID2, entries[0].ID)
}

func TestSessionIndex_RemoveNonExistent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	// Should not error.
	require.NoError(t, idx.Remove("nonexistent"))
}

func TestSessionIndex_FilterByWorkDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	require.NoError(t, idx.Update(testEntry(testSessionID1, "A1", testWorkDirA, testMsgCount, now)))
	require.NoError(t, idx.Update(testEntry(testSessionID2, "B1", testWorkDirB, testMsgCount, now)))
	require.NoError(t, idx.Update(testEntry(testSessionID3, "A2", testWorkDirA, testMsgCount, now)))

	entriesA := idx.List(testWorkDirA)
	require.Len(t, entriesA, 2)

	entriesB := idx.List(testWorkDirB)
	require.Len(t, entriesB, 1)
	require.Equal(t, testSessionID2, entriesB[0].ID)
}

func TestSessionIndex_ListSortedByLastModifiedDesc(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	oneHour := time.Hour

	require.NoError(t, idx.Update(testEntry(testSessionID1, "oldest", testWorkDirA, 1, base)))
	require.NoError(t, idx.Update(testEntry(testSessionID2, "newest", testWorkDirA, 2, base.Add(2*oneHour))))
	require.NoError(t, idx.Update(testEntry(testSessionID3, "middle", testWorkDirA, 3, base.Add(oneHour))))

	entries := idx.List("")
	require.Len(t, entries, 3)
	require.Equal(t, testSessionID2, entries[0].ID, "newest should be first")
	require.Equal(t, testSessionID3, entries[1].ID, "middle should be second")
	require.Equal(t, testSessionID1, entries[2].ID, "oldest should be last")
}

func TestSessionIndex_PersistAndReload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	now := time.Now().Truncate(time.Second)

	// Create and populate index.
	idx1, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)
	require.NoError(t, idx1.Update(testEntry(testSessionID1, testSessionTitle, testWorkDirA, testMsgCount, now)))

	// Reload from disk.
	idx2, reloadErr := session.NewSessionIndex(path, nil)
	require.NoError(t, reloadErr)

	entries := idx2.List("")
	require.Len(t, entries, 1)
	require.Equal(t, testSessionID1, entries[0].ID)
	require.Equal(t, testSessionTitle, entries[0].Title)
}

func TestSessionIndex_AutoRebuildOnCorruption(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	// Write corrupted index file.
	require.NoError(t, os.WriteFile(path, []byte("{corrupted json!!"), 0o600))

	var rebuilt bool

	scanner := &mockMetadataScanner{
		entries: []session.IndexEntry{
			testEntry(testSessionID1, "rebuilt", testWorkDirA, 5, time.Now().Truncate(time.Second)),
		},
	}

	idx, err := session.NewSessionIndex(path, scanner, session.WithRebuildCallback(func() {
		rebuilt = true
	}))
	require.NoError(t, err)
	require.True(t, rebuilt, "rebuild callback should have been called")

	entries := idx.List("")
	require.Len(t, entries, 1)
	require.Equal(t, "rebuilt", entries[0].Title)
}

func TestSessionIndex_AutoRebuildOnMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	var rebuilt bool

	scanner := &mockMetadataScanner{
		entries: []session.IndexEntry{
			testEntry(testSessionID1, "scanned", testWorkDirA, 3, time.Now().Truncate(time.Second)),
		},
	}

	idx, err := session.NewSessionIndex(path, scanner, session.WithRebuildCallback(func() {
		rebuilt = true
	}))
	require.NoError(t, err)
	require.True(t, rebuilt, "rebuild should trigger on missing file")

	entries := idx.List("")
	require.Len(t, entries, 1)
	require.Equal(t, "scanned", entries[0].Title)
}

func TestSessionIndex_RebuildFromMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	now := time.Now().Truncate(time.Second)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	// Add stale entry.
	require.NoError(t, idx.Update(testEntry("stale", "stale", testWorkDirA, 1, now)))

	scanner := &mockMetadataScanner{
		entries: []session.IndexEntry{
			testEntry(testSessionID1, "fresh-1", testWorkDirA, 10, now),
			testEntry(testSessionID2, "fresh-2", testWorkDirB, 20, now),
		},
	}

	require.NoError(t, idx.Rebuild(scanner))

	// Stale entry should be gone, fresh entries should be present.
	entries := idx.List("")
	require.Len(t, entries, 2)

	ids := make(map[string]bool)
	for _, entry := range entries {
		ids[entry.ID] = true
	}

	require.True(t, ids[testSessionID1])
	require.True(t, ids[testSessionID2])
	require.False(t, ids["stale"])
}

func TestSessionIndex_AtomicWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)
	require.NoError(t, idx.Update(testEntry(testSessionID1, testSessionTitle, testWorkDirA, testMsgCount, now)))

	// Verify the file is valid JSON.
	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)

	var parsed session.IndexData
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Entries, 1)
}

func TestSessionIndex_EmptyList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	entries := idx.List("")
	require.Empty(t, entries)
}

func TestSessionIndex_Count(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	require.Equal(t, 0, idx.Count())

	now := time.Now().Truncate(time.Second)
	require.NoError(t, idx.Update(testEntry(testSessionID1, testSessionTitle, testWorkDirA, testMsgCount, now)))
	require.Equal(t, 1, idx.Count())
}

func TestSessionIndex_RebuildNilScanner(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	idx, err := session.NewSessionIndex(path, nil)
	require.NoError(t, err)

	// Rebuild with nil scanner should return error.
	require.Error(t, idx.Rebuild(nil))
}

// mockMetadataScanner implements session.MetadataScanner for testing.
type mockMetadataScanner struct {
	entries []session.IndexEntry
	err     error
}

func (m *mockMetadataScanner) ScanSessions() ([]session.IndexEntry, error) {
	return m.entries, m.err
}
