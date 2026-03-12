package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/memory"
)

func TestMemoryTool_Name(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)
	assert.Equal(t, "memory", tool.Name())
}

func TestMemoryTool_Description(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)
	assert.Contains(t, tool.Description(), "persistent")
}

func TestMemoryTool_Schema(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)
	schema := tool.Schema()

	assert.Equal(t, "function", schema.Type)
	assert.Equal(t, "memory", schema.Function.Name)

	// Check required parameters.
	props := schema.Function.Parameters.Properties
	assert.Contains(t, props, "operation")
	assert.Contains(t, props, "key")

	// Check that operation is required.
	assert.Contains(t, schema.Function.Parameters.Required, "operation")
}

func TestMemoryTool_Put(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)

	params, err := FromMap(map[string]any{
		"operation": "put",
		"key":       "test-key",
		"value":     "test-value",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "test-key")

	// Verify entry was stored.
	entry, err := store.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", entry.Value)

	// Verify file was created.
	filePath := filepath.Join(tmpDir, "default", "test-key.json")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

func TestMemoryTool_Get(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)

	// First store an entry.
	err = store.Put(ctx, "test-key", "test-value", memory.PutOptions{})
	require.NoError(t, err)

	params, err := FromMap(map[string]any{
		"operation": "get",
		"key":       "test-key",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "test-value")
}

func TestMemoryTool_Get_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)

	params, err := FromMap(map[string]any{
		"operation": "get",
		"key":       "nonexistent",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not found")
}

func TestMemoryTool_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)

	// First store an entry.
	err = store.Put(ctx, "test-key", "test-value", memory.PutOptions{})
	require.NoError(t, err)

	params, err := FromMap(map[string]any{
		"operation": "delete",
		"key":       "test-key",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "deleted")

	// Verify entry was deleted.
	_, err = store.Get(ctx, "test-key")
	assert.ErrorIs(t, err, memory.ErrNotFound)
}

func TestMemoryTool_List(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)

	// Store some entries.
	require.NoError(t, store.Put(ctx, "key1", "value1", memory.PutOptions{}))
	require.NoError(t, store.Put(ctx, "key2", "value2", memory.PutOptions{}))
	require.NoError(t, store.Put(ctx, "other", "value3", memory.PutOptions{}))

	params, err := FromMap(map[string]any{
		"operation": "list",
		"pattern":   "key*",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "key1")
	assert.Contains(t, result.Output, "key2")
	assert.NotContains(t, result.Output, "other")
}

func TestMemoryTool_Search(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)

	// Store some entries.
	require.NoError(t, store.Put(ctx, "api-response", "contains user data", memory.PutOptions{}))
	require.NoError(t, store.Put(ctx, "config", "database settings", memory.PutOptions{}))

	params, err := FromMap(map[string]any{
		"operation": "search",
		"query":     "user",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "api-response")
}

func TestMemoryTool_BadOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "invalid operation",
			params:    map[string]any{"operation": "invalid"},
			wantError: "unknown operation",
		},
		{
			name:      "missing operation",
			params:    map[string]any{"key": "test"},
			wantError: "operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			tmpDir := t.TempDir()
			store, err := memory.NewPersistentStore(tmpDir)
			require.NoError(t, err)

			tool := NewMemoryTool(store)

			params, err := FromMap(tt.params)
			require.NoError(t, err)

			result, err := tool.Execute(ctx, params)
			require.NoError(t, err)
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, tt.wantError)
		})
	}
}

func TestMemoryTool_Put_WithNamespace(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	tool := NewMemoryTool(store)

	params, err := FromMap(map[string]any{
		"operation": "put",
		"key":       "test-key",
		"value":     "test-value",
		"namespace": "custom",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify file was created in custom namespace.
	filePath := filepath.Join(tmpDir, "custom", "test-key.json")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

func TestMemoryTool_NilStore(t *testing.T) {
	t.Parallel()
	tool := NewMemoryTool(nil)
	assert.Nil(t, tool)
}
