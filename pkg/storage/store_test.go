package storage

// Journey: specs/journeys/JOURNEY-CTX-1.2.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestData is a simple struct for testing.
type TestData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestFileStore_SaveLoad(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	data := TestData{ID: "test-1", Name: "Test", Value: 42}

	// Save.
	err = store.Save(t.Context(), "test-1", data)
	require.NoError(t, err)

	// Load.
	loaded, err := store.Load(t.Context(), "test-1")
	require.NoError(t, err)
	assert.Equal(t, data, loaded)
}

func TestFileStore_Delete(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	data := TestData{ID: "test-1", Name: "Test", Value: 42}
	_ = store.Save(t.Context(), "test-1", data)

	// Delete.
	err = store.Delete(t.Context(), "test-1")
	require.NoError(t, err)

	// Should not exist.
	exists, _ := store.Exists(t.Context(), "test-1")
	assert.False(t, exists)
}

func TestFileStore_Exists(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	// Should not exist initially.
	exists, err := store.Exists(t.Context(), "test-1")
	require.NoError(t, err)
	assert.False(t, exists)

	// Save and check.
	_ = store.Save(t.Context(), "test-1", TestData{ID: "test-1"})

	exists, err = store.Exists(t.Context(), "test-1")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestFileStore_List(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	// Save multiple items.
	for i := range 5 {
		_ = store.Save(t.Context(), string(rune('a'+i)), TestData{ID: string(rune('a' + i))})
	}

	keys, err := store.List(t.Context())
	require.NoError(t, err)
	assert.Len(t, keys, 5)
}

func TestFileStore_CustomSuffix(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".history.json",
	})
	require.NoError(t, err)

	data := TestData{ID: "test-1", Name: "Test", Value: 42}
	_ = store.Save(t.Context(), "test-1", data)

	// Load should work.
	loaded, err := store.Load(t.Context(), "test-1")
	require.NoError(t, err)
	assert.Equal(t, data.ID, loaded.ID)
}

func TestFileStore_EmptyKey(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	ctx := t.Context()

	// All operations should fail with empty key.
	err = store.Save(ctx, "", TestData{})
	require.Error(t, err)

	_, err = store.Load(ctx, "")
	require.Error(t, err)

	err = store.Delete(ctx, "")
	require.Error(t, err)

	_, err = store.Exists(ctx, "")
	require.Error(t, err)
}

func TestFileStore_NotFound(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	_, err = store.Load(t.Context(), "nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestFileStore_CanceledContext_Save(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	ctx := canceledContext()

	err = store.Save(ctx, "test-1", TestData{ID: "test-1"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestFileStore_CanceledContext_Load(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	// Save with valid context first.
	_ = store.Save(t.Context(), "test-1", TestData{ID: "test-1"})

	ctx := canceledContext()

	_, err = store.Load(ctx, "test-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFileStore_CanceledContext_Delete(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	ctx := canceledContext()

	err = store.Delete(ctx, "test-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFileStore_CanceledContext_Exists(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	ctx := canceledContext()

	_, err = store.Exists(ctx, "test-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFileStore_CanceledContext_List(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	require.NoError(t, err)

	ctx := canceledContext()

	_, err = store.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
}
