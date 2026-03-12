package conversation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/memory"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestMemoryService_NewMemoryService(t *testing.T) {
	scratchpad := memory.NewScratchpad("test-session", 10)
	tmpDir := t.TempDir()
	persistent, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	service := NewMemoryService(scratchpad, persistent)
	assert.NotNil(t, service)
	assert.Equal(t, scratchpad, service.Scratchpad())
	assert.Equal(t, persistent, service.Persistent())
}

func TestMemoryService_NilStores(t *testing.T) {
	service := NewMemoryService(nil, nil)
	assert.NotNil(t, service)
	assert.Nil(t, service.Scratchpad())
	assert.Nil(t, service.Persistent())
}

func TestBuilder_registerMemoryTools_NilService(t *testing.T) {
	builder := &Builder{}
	registry := tools.NewRegistry()

	err := builder.registerMemoryTools(registry)
	assert.NoError(t, err)

	// No memory tools should be registered.
	_, err = registry.Get("scratchpad")
	assert.Error(t, err)
	_, err = registry.Get("memory")
	assert.Error(t, err)
}

func TestBuilder_registerMemoryTools_WithScratchpad(t *testing.T) {
	scratchpad := memory.NewScratchpad("test-session", 10)
	service := NewMemoryService(scratchpad, nil)
	builder := &Builder{memoryService: service}
	registry := tools.NewRegistry()

	err := builder.registerMemoryTools(registry)
	assert.NoError(t, err)

	// Scratchpad tool should be registered.
	tool, err := registry.Get("scratchpad")
	assert.NoError(t, err)
	assert.Equal(t, "scratchpad", tool.Name())

	// Memory tool should not be registered.
	_, err = registry.Get("memory")
	assert.Error(t, err)
}

func TestBuilder_registerMemoryTools_WithPersistent(t *testing.T) {
	tmpDir := t.TempDir()
	persistent, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	service := NewMemoryService(nil, persistent)
	builder := &Builder{memoryService: service}
	registry := tools.NewRegistry()

	err = builder.registerMemoryTools(registry)
	assert.NoError(t, err)

	// Scratchpad tool should not be registered.
	_, err = registry.Get("scratchpad")
	assert.Error(t, err)

	// Memory tool should be registered.
	tool, err := registry.Get("memory")
	assert.NoError(t, err)
	assert.Equal(t, "memory", tool.Name())
}

func TestBuilder_registerMemoryTools_WithBoth(t *testing.T) {
	scratchpad := memory.NewScratchpad("test-session", 10)
	tmpDir := t.TempDir()
	persistent, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	service := NewMemoryService(scratchpad, persistent)
	builder := &Builder{memoryService: service}
	registry := tools.NewRegistry()

	err = builder.registerMemoryTools(registry)
	assert.NoError(t, err)

	// Both tools should be registered.
	tool1, err := registry.Get("scratchpad")
	assert.NoError(t, err)
	assert.Equal(t, "scratchpad", tool1.Name())

	tool2, err := registry.Get("memory")
	assert.NoError(t, err)
	assert.Equal(t, "memory", tool2.Name())
}

func TestBuilder_registerMemoryTools_ToolsWork(t *testing.T) {
	ctx := context.Background()
	scratchpad := memory.NewScratchpad("test-session", 10)
	service := NewMemoryService(scratchpad, nil)
	builder := &Builder{memoryService: service}
	registry := tools.NewRegistry()

	err := builder.registerMemoryTools(registry)
	require.NoError(t, err)

	// Test executing the scratchpad tool.
	params, err := tools.FromMap(map[string]any{
		"operation": "put",
		"key":       "test-key",
		"value":     "test-value",
	})
	require.NoError(t, err)

	result, err := registry.Execute(ctx, "scratchpad", params)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify entry was stored.
	entry, err := scratchpad.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", entry.Value)
}

func TestMemoryService_NewAutoOffloader(t *testing.T) {
	scratchpad := memory.NewScratchpad("test-session", 10)
	service := NewMemoryService(scratchpad, nil)

	offloader := service.NewAutoOffloader(0.7)
	assert.NotNil(t, offloader)
	assert.Equal(t, 0.7, offloader.GetThreshold())
}

func TestMemoryService_NewAutoOffloader_NilStores(t *testing.T) {
	service := NewMemoryService(nil, nil)
	offloader := service.NewAutoOffloader(0.7)
	assert.Nil(t, offloader)
}

func TestMemoryService_NewSessionHandoff(t *testing.T) {
	tmpDir := t.TempDir()
	persistent, err := memory.NewPersistentStore(tmpDir)
	require.NoError(t, err)

	service := NewMemoryService(nil, persistent)

	// With nil summarizer (uses default).
	handoff := service.NewSessionHandoff(nil)
	assert.NotNil(t, handoff)
}

func TestMemoryService_NewSessionHandoff_NoPersistent(t *testing.T) {
	scratchpad := memory.NewScratchpad("test-session", 10)
	service := NewMemoryService(scratchpad, nil)

	handoff := service.NewSessionHandoff(nil)
	assert.Nil(t, handoff)
}
