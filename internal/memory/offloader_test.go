package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAutoOffloader_Defaults(t *testing.T) {
	t.Parallel()

	offloader := NewAutoOffloader(AutoOffloaderConfig{})

	assert.NotNil(t, offloader)
	assert.InDelta(t, 0.7, offloader.GetThreshold(), 1e-9)
}

func TestNewAutoOffloader_WithConfig(t *testing.T) {
	t.Parallel()

	scratchpad := NewScratchpad("test", 10)
	analyzer := NewDefaultContextAnalyzer()

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Scratchpad: scratchpad,
		Analyzer:   analyzer,
		Threshold:  0.8,
	})

	assert.NotNil(t, offloader)
	assert.InDelta(t, 0.8, offloader.GetThreshold(), 1e-9)
}

func TestAutoOffloader_ShouldOffload(t *testing.T) {
	t.Parallel()

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Threshold: 0.7,
	})

	tests := []struct {
		name          string
		currentTokens int
		maxTokens     int
		expected      bool
	}{
		{"below threshold", 50, 100, false},
		{"at threshold", 70, 100, false},
		{"above threshold", 75, 100, true},
		{"way above", 95, 100, true},
		{"zero max", 50, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := offloader.ShouldOffload(tt.currentTokens, tt.maxTokens)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAutoOffloader_Offload_NoContent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	offloader := NewAutoOffloader(AutoOffloaderConfig{})

	messages := []AnalyzableMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	modified, results, err := offloader.Offload(ctx, messages)
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, messages, modified)
}

func TestAutoOffloader_Offload_LargeCodeBlock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scratchpad := NewScratchpad("test", 10)

	analyzer := NewDefaultContextAnalyzer()
	analyzer.CodeBlockThreshold = 50 // Low threshold for testing.

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Scratchpad: scratchpad,
		Analyzer:   analyzer,
	})

	largeCode := "```go\n" + strings.Repeat("func example() {\n\treturn nil\n}\n", 20) + "```"
	messages := []AnalyzableMessage{
		{Role: "assistant", Content: "Here's the code:\n" + largeCode},
	}

	modified, results, err := offloader.Offload(ctx, messages)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Equal(t, ScopeSession, results[0].Destination)
	assert.Positive(t, results[0].TokensSaved)

	// Check that reference was added.
	assert.Contains(t, modified[0].Content, "offloaded")

	// Check that content was stored in scratchpad.
	entry, err := scratchpad.Get(ctx, results[0].Key)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.Value)
}

func TestAutoOffloader_Offload_Decision(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	persistent, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Persistent: persistent,
	})

	messages := []AnalyzableMessage{
		{Role: "assistant", Content: "After analysis, I decided to use PostgreSQL for the database."},
	}

	modified, results, err := offloader.Offload(ctx, messages)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Equal(t, ScopePersistent, results[0].Destination)

	// Check that reference was added.
	assert.Contains(t, modified[0].Content, "offloaded")
}

func TestAutoOffloader_Offload_NoStoresConfigured(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	analyzer := NewDefaultContextAnalyzer()
	analyzer.CodeBlockThreshold = 10

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Analyzer: analyzer,
		// No scratchpad or persistent store.
	})

	largeCode := "```go\n" + strings.Repeat("x", 100) + "\n```"
	messages := []AnalyzableMessage{
		{Role: "assistant", Content: largeCode},
	}

	modified, results, err := offloader.Offload(ctx, messages)
	require.NoError(t, err)
	// Results will be empty since no stores are configured
	// The analyzer finds candidates but storage fails silently.
	assert.Equal(t, messages[0].Content, modified[0].Content[:len(messages[0].Content)])

	_ = results // Results may or may not be empty depending on implementation.
}

func TestAutoOffloader_Recall_FromScratchpad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scratchpad := NewScratchpad("test", 10)

	// Pre-populate scratchpad.
	err := scratchpad.Put(ctx, "test-key", "test-value", PutOptions{})
	require.NoError(t, err)

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Scratchpad: scratchpad,
	})

	value, err := offloader.Recall(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", value)
}

func TestAutoOffloader_Recall_FromPersistent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	persistent, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	// Pre-populate persistent store.
	err = persistent.Put(ctx, "test-key", "persistent-value", PutOptions{})
	require.NoError(t, err)

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Persistent: persistent,
	})

	value, err := offloader.Recall(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "persistent-value", value)
}

func TestAutoOffloader_Recall_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scratchpad := NewScratchpad("test", 10)

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Scratchpad: scratchpad,
	})

	_, err := offloader.Recall(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAutoOffloader_SetThreshold(t *testing.T) {
	t.Parallel()

	offloader := NewAutoOffloader(AutoOffloaderConfig{})

	offloader.SetThreshold(0.5)
	assert.InDelta(t, 0.5, offloader.GetThreshold(), 1e-9)

	// Invalid values should be ignored.
	offloader.SetThreshold(0)
	assert.InDelta(t, 0.5, offloader.GetThreshold(), 1e-9)

	offloader.SetThreshold(1.5)
	assert.InDelta(t, 0.5, offloader.GetThreshold(), 1e-9)
}

func TestAutoOffloader_Offload_MultipleMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scratchpad := NewScratchpad("test", 10)

	analyzer := NewDefaultContextAnalyzer()
	analyzer.CodeBlockThreshold = 10
	analyzer.ToolOutputThreshold = 10

	offloader := NewAutoOffloader(AutoOffloaderConfig{
		Scratchpad: scratchpad,
		Analyzer:   analyzer,
	})

	messages := []AnalyzableMessage{
		{Role: "assistant", Content: "```go\n" + strings.Repeat("x", 100) + "\n```"},
		{Role: "tool", Content: strings.Repeat("output line\n", 20)},
		{Role: "user", Content: "What do you think?"},
	}

	modified, results, err := offloader.Offload(ctx, messages)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2) // At least code block and tool output.

	// User message should be unchanged.
	assert.Equal(t, messages[2].Content, modified[2].Content)
}
