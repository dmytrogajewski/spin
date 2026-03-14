package storage

// Journey: specs/journeys/JOURNEY-extract-atomic-write.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicWriteFile_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []byte(`{"key": "value"}`)

	err := AtomicWriteFile(path, data, DefaultFilePerm)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestAtomicWriteFile_Permissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "perms.json")

	err := AtomicWriteFile(path, []byte("test"), 0o644)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestAtomicWriteFile_NoTempFileOnSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "clean.json")

	err := AtomicWriteFile(path, []byte("data"), 0o600)
	require.NoError(t, err)

	// Only the final file should exist.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "clean.json", entries[0].Name())
}

func TestAtomicWriteFile_ParentDirNotExist(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nonexistent", "sub", "file.json")

	err := AtomicWriteFile(path, []byte("data"), 0o600)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create temp file")
}

func TestAtomicWriteFile_OverwriteExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.json")

	// Write original.
	err := AtomicWriteFile(path, []byte("original"), 0o600)
	require.NoError(t, err)

	// Overwrite.
	err = AtomicWriteFile(path, []byte("updated"), 0o600)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("updated"), got)
}

func TestAtomicWriteFile_EmptyData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	err := AtomicWriteFile(path, []byte{}, 0o600)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAtomicWriteFile_NilData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nil.json")

	err := AtomicWriteFile(path, nil, 0o600)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, got)
}
