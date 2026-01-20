# Package: summarizer

**Location:** `internal/context/summarizer/`

## Overview

The summarizer package provides progressive context summarization capabilities for the Spin agent. It enables intelligent compression of conversation history and content while preserving essential information, allowing longer effective conversations within token limits.

## Key Concepts

### Summarization vs Pruning

| Aspect | Pruning | Summarization |
|--------|---------|---------------|
| Goal | Remove irrelevant content | Compress all content |
| Output | Subset of original | New condensed text |
| Information | Discarded | Preserved (compressed) |
| Use case | Tool outputs, retrieval | Conversation history |

## Components

### Summarizer Interface

The core interface for all summarization implementations:

```go
type Summarizer interface {
    // Summarize compresses content while preserving key information
    Summarize(ctx context.Context, content string, opts Options) (*Result, error)
    
    // SummarizeMessages summarizes a sequence of conversation messages
    SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error)
}
```

### LLMSummarizer

LLM-based implementation using any `llm.Provider`:

```go
// Create with default config
config := summarizer.DefaultLLMSummarizerConfig()
s := summarizer.NewLLMSummarizer(provider, tokenizer, config)

// Summarize content
result, err := s.Summarize(ctx, content, summarizer.Options{
    MaxTokens:   500,
    TargetRatio: 0.3,
    Style:       summarizer.StyleNarrative,
})
```

### Cache

LRU cache with TTL for efficient summary retrieval:

```go
config := summarizer.DefaultCacheConfig() // MaxSize: 100, TTL: 1 hour
cache := summarizer.NewCache(config)

// Store
cache.Set(content, summary)

// Retrieve
if cached, ok := cache.Get(content); ok {
    // Use cached summary
}
```

### CachingSummarizer

Decorator that adds transparent caching to any Summarizer:

```go
inner := summarizer.NewLLMSummarizer(provider, tok, config)
cache := summarizer.NewCache(summarizer.DefaultCacheConfig())
cached := summarizer.NewCachingSummarizer(inner, cache)

// Uses cache automatically
result, _ := cached.Summarize(ctx, content, opts)
```

## Summary Styles

| Style | Description |
|-------|-------------|
| `StyleBrief` | Minimal, key points only |
| `StyleDetailed` | More context preserved |
| `StyleBullet` | Bullet point format |
| `StyleNarrative` | Flowing narrative (default) |

## Configuration

### LLMSummarizerConfig

```go
type LLMSummarizerConfig struct {
    Model              string        // LLM model (default: "gpt-4o-mini")
    Timeout            time.Duration // Request timeout (default: 10s)
    DefaultMaxTokens   int           // Default target tokens (default: 500)
    DefaultTargetRatio float64       // Default compression ratio (default: 0.3)
    DefaultStyle       SummaryStyle  // Default style (default: StyleNarrative)
}
```

### CacheConfig

```go
type CacheConfig struct {
    MaxSize int           // Max cached entries (default: 100)
    TTL     time.Duration // Time-to-live (default: 1 hour)
}
```

## Usage Examples

### Basic Content Summarization

```go
provider := llm.NewOpenAIProvider(...)
config := summarizer.DefaultLLMSummarizerConfig()
s := summarizer.NewLLMSummarizer(provider, nil, config)

result, err := s.Summarize(ctx, longContent, summarizer.Options{
    MaxTokens:   300,
    TargetRatio: 0.2,
    Style:       summarizer.StyleBullet,
})

fmt.Printf("Original: %d tokens\n", result.OriginalTokens)
fmt.Printf("Summary: %d tokens (%.0f%% compression)\n", 
    result.SummaryTokens, result.CompressionRatio*100)
fmt.Println(result.Summary)
```

### Conversation Summarization

```go
messages := []message.Message{
    {Role: message.RoleUser, Content: "How do I authenticate?"},
    {Role: message.RoleAssistant, Content: "Use JWT tokens..."},
    {Role: message.RoleUser, Content: "Can you show an example?"},
    // ... more messages
}

result, err := s.SummarizeMessages(ctx, messages, summarizer.Options{
    MaxTokens: 500,
    Style:     summarizer.StyleNarrative,
})

// result.Summary contains a single message summarizing the conversation
// result.OriginalCount = number of messages summarized
// result.SummarizedRange = [start, end] indices
```

### With Caching

```go
inner := summarizer.NewLLMSummarizer(provider, tok, config)
cache := summarizer.NewCache(summarizer.CacheConfig{
    MaxSize: 200,
    TTL:     2 * time.Hour,
})
s := summarizer.NewCachingSummarizer(inner, cache)

// First call - hits LLM
result1, _ := s.Summarize(ctx, content, opts)

// Second call - returns cached result (no LLM call)
result2, _ := s.Summarize(ctx, content, opts)
```

## Thread Safety

All implementations are safe for concurrent use:
- `Cache` uses `sync.RWMutex` for thread-safe operations
- `LLMSummarizer` is stateless and thread-safe
- `CachingSummarizer` inherits thread-safety from its components

## Related Packages

- `internal/llm` - LLM provider interface
- `internal/tokenizer` - Token counting
- `internal/message` - Message types
- `internal/history` - Conversation history management

## Future Work (Phase 2+)

- **IncrementalSummarizer**: Progressive summarization as conversation grows
- **WindowManager**: Context window management with strategies
- **SummarizingToolWrapper**: Tool output summarization
- **SummarizingCompressor**: Enhanced history compression integration
