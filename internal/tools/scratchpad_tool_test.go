package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/memory"
)

func TestScratchpadTool_Name(t *testing.T) {
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)
	assert.Equal(t, "scratchpad", tool.Name())
}

func TestScratchpadTool_Description(t *testing.T) {
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)
	assert.Contains(t, tool.Description(), "session-scoped")
}

func TestScratchpadTool_Schema(t *testing.T) {
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)
	schema := tool.Schema()

	assert.Equal(t, "function", schema.Type)
	assert.Equal(t, "scratchpad", schema.Function.Name)

	// Check required parameters.
	props := schema.Function.Parameters.Properties
	assert.Contains(t, props, "operation")
	assert.Contains(t, props, "key")

	// Check that operation is required.
	assert.Contains(t, schema.Function.Parameters.Required, "operation")
}

func TestScratchpadTool_Put(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

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
	entry, err := pad.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", entry.Value)
}

func TestScratchpadTool_Get(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// First store an entry.
	err := pad.Put(ctx, "test-key", "test-value", memory.PutOptions{})
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

func TestScratchpadTool_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

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

func TestScratchpadTool_Delete(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// First store an entry.
	err := pad.Put(ctx, "test-key", "test-value", memory.PutOptions{})
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
	_, err = pad.Get(ctx, "test-key")
	assert.ErrorIs(t, err, memory.ErrNotFound)
}

func TestScratchpadTool_List(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// Store some entries.
	require.NoError(t, pad.Put(ctx, "key1", "value1", memory.PutOptions{}))
	require.NoError(t, pad.Put(ctx, "key2", "value2", memory.PutOptions{}))
	require.NoError(t, pad.Put(ctx, "other", "value3", memory.PutOptions{}))

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

func TestScratchpadTool_List_All(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// Store some entries.
	require.NoError(t, pad.Put(ctx, "key1", "value1", memory.PutOptions{}))
	require.NoError(t, pad.Put(ctx, "key2", "value2", memory.PutOptions{}))

	params, err := FromMap(map[string]any{
		"operation": "list",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "key1")
	assert.Contains(t, result.Output, "key2")
}

func TestScratchpadTool_Search(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// Store some entries.
	require.NoError(t, pad.Put(ctx, "api-response", "contains user data", memory.PutOptions{}))
	require.NoError(t, pad.Put(ctx, "config", "database settings", memory.PutOptions{}))

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

func TestScratchpadTool_Pin(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// First store an entry.
	require.NoError(t, pad.Put(ctx, "important", "critical data", memory.PutOptions{}))

	params, err := FromMap(map[string]any{
		"operation": "pin",
		"key":       "important",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "pinned")
}

func TestScratchpadTool_Unpin(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// First store and pin an entry.
	require.NoError(t, pad.Put(ctx, "important", "critical data", memory.PutOptions{}))
	require.NoError(t, pad.Pin("important"))

	params, err := FromMap(map[string]any{
		"operation": "unpin",
		"key":       "important",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "unpinned")
}

func TestScratchpadTool_Clear(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	// Store some entries.
	require.NoError(t, pad.Put(ctx, "key1", "value1", memory.PutOptions{}))
	require.NoError(t, pad.Put(ctx, "key2", "value2", memory.PutOptions{}))

	params, err := FromMap(map[string]any{
		"operation": "clear",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "cleared")

	// Verify entries were deleted.
	keys, _ := pad.List(ctx, "*")
	assert.Empty(t, keys)
}

func TestScratchpadTool_InvalidOperation(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	params, err := FromMap(map[string]any{
		"operation": "invalid",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unknown operation")
}

func TestScratchpadTool_MissingOperation(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	params, err := FromMap(map[string]any{
		"key": "test",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "operation")
}

func TestScratchpadTool_MissingKey_Put(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	params, err := FromMap(map[string]any{
		"operation": "put",
		"value":     "test-value",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "key")
}

func TestScratchpadTool_MissingValue_Put(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	params, err := FromMap(map[string]any{
		"operation": "put",
		"key":       "test-key",
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "value")
}

func TestScratchpadTool_Put_WithNamespace(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

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

	// Verify namespace was set.
	entry, err := pad.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "custom", entry.Namespace)
}

func TestScratchpadTool_Put_WithTags(t *testing.T) {
	ctx := context.Background()
	pad := memory.NewScratchpad("test-session", 10)
	tool := NewScratchpadTool(pad)

	params, err := FromMap(map[string]any{
		"operation": "put",
		"key":       "test-key",
		"value":     "test-value",
		"tags":      []string{"important", "api"},
	})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify tags were set.
	entry, err := pad.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Contains(t, entry.Tags, "important")
	assert.Contains(t, entry.Tags, "api")
}

func TestScratchpadTool_NilScratchpad(t *testing.T) {
	tool := NewScratchpadTool(nil)
	assert.Nil(t, tool)
}
