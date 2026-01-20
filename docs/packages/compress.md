# compress Package

Package `compress` provides context compression for conversation history.

## Overview

The compression system uses importance-weighted message selection to preserve critical information while staying within token budgets. It optionally integrates with the summarizer package to create semantic summaries of removed content.

## Location

`internal/history/compress/`

## Components

### Compressor Interface

The core interface for compression strategies:

```go
type Compressor interface {
    Compress(ctx context.Context, messages []message.Message, targetTokens int, tok tokenizer.Tokenizer) ([]message.Message, error)
    Name() string
}
```

### MessageClassifier

Assigns importance levels to messages based on role and content:

```go
type Classifier struct{}

func NewClassifier() *Classifier
func (c *Classifier) Classify(msg message.Message) Importance
```

**Importance Levels:**
- `ImportanceCritical` (3): User messages, tool results, errors - always preserved
- `ImportanceHigh` (2): Code blocks, decisions
- `ImportanceMedium` (1): Regular assistant responses
- `ImportanceLow` (0): Verbose reasoning, "thinking" content - compressed first

### HybridCompressor

The main compressor implementation using greedy selection with optional LLM summarization:

```go
type HybridCompressor struct{}

func NewHybridCompressor(classifier *Classifier, config CompressorConfig) *HybridCompressor
func (c *HybridCompressor) WithSummarizer(s summarizer.Summarizer) *HybridCompressor
func (c *HybridCompressor) Compress(ctx context.Context, messages []message.Message, targetTokens int, tok tokenizer.Tokenizer) ([]message.Message, error)
func (c *HybridCompressor) CompressWithStats(ctx context.Context, messages []message.Message, targetTokens int, tok tokenizer.Tokenizer) ([]message.Message, *Stats, error)
```

## Configuration

```go
type CompressorConfig struct {
    PreserveCritical bool    // Always keep critical messages
    MinRetention     float64 // Minimum message retention (0.0-1.0)
}

// Default configuration
cfg := compress.DefaultCompressorConfig()
// PreserveCritical: true
// MinRetention: 0.3
```

## Usage

### Basic Usage

```go
import (
    "github.com/dmytrogajewski/spin/internal/history/compress"
    "github.com/dmytrogajewski/spin/internal/tokenizer"
)

// Create classifier and compressor
classifier := compress.NewClassifier()
compressor := compress.NewHybridCompressor(classifier, compress.DefaultCompressorConfig())

// Compress messages to fit within token budget
compressed, err := compressor.Compress(ctx, messages, 8000, &tokenizer.SimpleTokenizer{})
```

### With Summarization

```go
// Create with summarizer for semantic compression
compressor := compress.NewHybridCompressor(classifier, config).
    WithSummarizer(llmSummarizer)

// Removed messages will be summarized instead of discarded
compressed, err := compressor.Compress(ctx, messages, 8000, tok)
```

### With Statistics

```go
// Get compression statistics
compressed, stats, err := compressor.CompressWithStats(ctx, messages, 8000, tok)

fmt.Printf("Compression ratio: %.1f%%\n", stats.CompressionRatio()*100)
fmt.Printf("Messages: %d -> %d\n", stats.OriginalCount, stats.CompressedCount)
```

### Integration with History

```go
import (
    "github.com/dmytrogajewski/spin/internal/history"
    "github.com/dmytrogajewski/spin/internal/history/compress"
)

// Create history with compression
h := history.NewHistory(16000, &tokenizer.SimpleTokenizer{})

// Set up compressor
classifier := compress.NewClassifier()
compressor := compress.NewHybridCompressor(classifier, compress.DefaultCompressorConfig())
h.SetCompressor(compressor)

// Configure automatic compression
h.SetCompressionConfig(history.CompressionConfig{
    Enabled:     true,
    Threshold:   0.8,  // Compress at 80% capacity
    TargetRatio: 0.7,  // Compress to 70% of max
})

// Messages are automatically compressed when threshold is exceeded
h.AddUserMessage("Hello")
h.AddMessage(assistantResponse)
// ... automatic compression triggers when needed
```

## Classification Rules

The classifier uses deterministic rules:

1. **Critical (100% retention):**
   - System messages
   - User messages
   - Tool role messages
   - Messages with tool calls
   - Messages containing error indicators

2. **High:**
   - Messages with code blocks (```)
   - Messages with diff markers (@@)
   - Messages with indented code

3. **Medium:**
   - Regular assistant responses

4. **Low:**
   - Long assistant messages without code (verbose "thinking")

## Algorithm

The HybridCompressor uses a greedy selection algorithm:

1. Classify all messages by importance
2. Sort by importance (stable sort preserves chronological order)
3. Greedily select messages within token budget
4. Optionally summarize removed messages using LLM
5. Enforce minimum retention ratio
6. Restore chronological order

## Statistics

The `Stats` struct provides compression metrics:

```go
type Stats struct {
    OriginalCount    int    // Messages before compression
    CompressedCount  int    // Messages after compression
    OriginalTokens   int    // Tokens before compression
    CompressedTokens int    // Tokens after compression
    Summarized       bool   // Whether LLM summarization was used
    Strategy         string // Compression strategy name
}

// Helper methods
stats.CompressionRatio()  // Token reduction ratio (0.0-1.0)
stats.MessageReduction()  // Message reduction ratio (0.0-1.0)
```

## Thread Safety

All compression operations are stateless and thread-safe. When integrated with History, the History mutex protects concurrent access.

## Future Work

- Sliding window compression strategy
- Semantic compression using embeddings
- Configurable classification rules
- Compression event streaming
