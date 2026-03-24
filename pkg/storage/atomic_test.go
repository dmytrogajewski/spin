package storage

// Journey: specs/journeys/JOURNEY-CTX-1.1.md.

import (
	"context"
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

	err := AtomicWriteFile(context.Background(), path, data, DefaultFilePerm)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestAtomicWriteFile_Permissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "perms.json")

	err := AtomicWriteFile(context.Background(), path, []byte("test"), 0o644)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestAtomicWriteFile_NoTempFileOnSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "clean.json")

	err := AtomicWriteFile(context.Background(), path, []byte("data"), 0o600)
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

	err := AtomicWriteFile(context.Background(), path, []byte("data"), 0o600)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create temp file")
}

func TestAtomicWriteFile_OverwriteExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.json")

	// Write original.
	err := AtomicWriteFile(context.Background(), path, []byte("original"), 0o600)
	require.NoError(t, err)

	// Overwrite.
	err = AtomicWriteFile(context.Background(), path, []byte("updated"), 0o600)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("updated"), got)
}

func TestAtomicWriteFile_EmptyData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	err := AtomicWriteFile(context.Background(), path, []byte{}, 0o600)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAtomicWriteFile_NilData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nil.json")

	err := AtomicWriteFile(context.Background(), path, nil, 0o600)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAtomicWriteFile_CanceledContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "canceled.json")
	ctx := canceledContext()

	err := AtomicWriteFile(ctx, path, []byte("should not be written"), 0o600)
	require.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "atomic write")

	// Target file must not exist.
	_, statErr := os.Stat(path)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestAtomicWriteFile_CanceledContextNoTempFileLeak(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "leak-check.json")
	ctx := canceledContext()

	_ = AtomicWriteFile(ctx, path, []byte("data"), 0o600)

	// No temp files should remain in the directory.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "temp files leaked after canceled context")
}

func TestAtomicWriteFile_ValidContextStillWorks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "valid-ctx.json")
	data := []byte(`{"ctx": "valid"}`)

	err := AtomicWriteFile(t.Context(), path, data, DefaultFilePerm)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

// canceledContext returns a context that is already canceled.
func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}
