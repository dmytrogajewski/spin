# FRD-20260120-004: Context Summarization - Phase 1: Core Summarizer

**Created:** 2026-01-20  
**Author:** Architecture  
**Status:** Draft  
**Priority:** P1 (High)  
**Related Proposal:** PROP-CONTEXT-005  

## Problem Statement

Spin currently implements reactive context compression (`internal/history/compress/`) that:
- Triggers at 80% of token limit
- Uses importance-weighted selection (critical/high/medium/low)
- Provides greedy selection algorithm within token budget
- Has **no summarization** - just selection/removal

### Identified Issues

1. **Binary Retention**: Content is either kept fully or removed entirely
2. **Loss of Continuity**: Removed messages break conversation flow
3. **Reactive Not Proactive**: Compression happens at crisis point
4. **No Semantic Compression**: Cannot condense verbose content intelligently

### Key Distinction from Pruning

| Aspect | Pruning | Summarization |
|--------|---------|---------------|
| Goal | Remove irrelevant content | Compress all content |
| Output | Subset of original | New condensed text |
| Information | Discarded | Preserved (compressed) |
| Use case | Tool outputs, retrieval | Conversation history |

## Solution

Implement Phase 1 of Progressive Context Summarization:

1. **Summarizer Interface** - Core abstraction for summarization
2. **LLMSummarizer** - LLM-based summarization implementation
3. **SummaryCache** - Cache layer for efficient retrieval

## Design

### Package Structure

```
internal/context/
└── summarizer/
    ├── summarizer.go       # Interface definitions
    ├── summarizer_test.go  # Interface tests
    ├── llm.go              # LLM-based summarizer
    ├── llm_test.go         # LLM summarizer tests
    ├── cache.go            # Summary cache
    ├── cache_test.go       # Cache tests
    └── doc.go              # Package documentation
```

### 1. Summarizer Interface

```go
// internal/context/summarizer/summarizer.go

package summarizer

import (
    "context"
    
    "github.com/dmytrogajewski/spin/internal/message"
)

// ContentType represents the type of content being summarized.
type ContentType string

const (
    ContentTypeConversation ContentType = "conversation"
    ContentTypeToolOutput   ContentType = "tool_output"
    ContentTypeDocument     ContentType = "document"
)

// SummaryStyle represents the style of summary output.
type SummaryStyle string

const (
    StyleBrief     SummaryStyle = "brief"     // Minimal, key points only
    StyleDetailed  SummaryStyle = "detailed"  // More context preserved
    StyleBullet    SummaryStyle = "bullet"    // Bullet point format
    StyleNarrative SummaryStyle = "narrative" // Flowing narrative
)

// Options configures summarization behavior.
type Options struct {
    // MaxTokens is the target output size in tokens.
    MaxTokens int
    
    // TargetRatio is the target compression ratio (e.g., 0.3 = 30% of original).
    TargetRatio float64
    
    // PreserveList contains items to preserve verbatim in the summary.
    PreserveList []string
    
    // ContentType indicates the type of content being summarized.
    ContentType ContentType
    
    // Style is the desired summary style.
    Style SummaryStyle
}

// Result contains the summarization output and metadata.
type Result struct {
    // Summary is the condensed content.
    Summary string
    
    // OriginalTokens is the token count of the original content.
    OriginalTokens int
    
    // SummaryTokens is the token count of the summary.
    SummaryTokens int
    
    // CompressionRatio is the ratio of summary to original tokens.
    CompressionRatio float64
    
    // PreservedItems contains items that were preserved verbatim.
    PreservedItems []string
    
    // KeyPoints contains extracted key points from the content.
    KeyPoints []string
}

// MessageResult contains the result of summarizing messages.
type MessageResult struct {
    // Summary is the single summary message.
    Summary message.Message
    
    // OriginalCount is the number of messages summarized.
    OriginalCount int
    
    // SummarizedRange contains the indices of summarized messages [start, end].
    SummarizedRange [2]int
    
    // KeyDecisions contains important decisions preserved from the conversation.
    KeyDecisions []string
    
    // KeyActions contains actions taken during the conversation.
    KeyActions []string
}

// Summarizer defines the interface for content summarization.
type Summarizer interface {
    // Summarize compresses content while preserving key information.
    Summarize(ctx context.Context, content string, opts Options) (*Result, error)
    
    // SummarizeMessages summarizes a sequence of conversation messages.
    SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error)
}
```

### 2. LLMSummarizer

```go
// internal/context/summarizer/llm.go

package summarizer

import (
    "context"
    "fmt"
    "strings"
    "time"
    
    "github.com/dmytrogajewski/spin/internal/llm"
    "github.com/dmytrogajewski/spin/internal/message"
    "github.com/dmytrogajewski/spin/internal/tokenizer"
    "github.com/openai/openai-go"
)

// LLMSummarizerConfig configures the LLM summarizer.
type LLMSummarizerConfig struct {
    // Model is the LLM model to use for summarization.
    Model string
    
    // Timeout is the maximum time for a summarization request.
    Timeout time.Duration
    
    // DefaultMaxTokens is the default target token count.
    DefaultMaxTokens int
    
    // DefaultTargetRatio is the default compression ratio.
    DefaultTargetRatio float64
    
    // DefaultStyle is the default summary style.
    DefaultStyle SummaryStyle
}

// DefaultLLMSummarizerConfig returns sensible default configuration.
func DefaultLLMSummarizerConfig() LLMSummarizerConfig {
    return LLMSummarizerConfig{
        Model:              "gpt-4o-mini",
        Timeout:            10 * time.Second,
        DefaultMaxTokens:   500,
        DefaultTargetRatio: 0.3,
        DefaultStyle:       StyleNarrative,
    }
}

// LLMSummarizer implements Summarizer using an LLM provider.
type LLMSummarizer struct {
    provider  llm.Provider
    tokenizer tokenizer.Tokenizer
    config    LLMSummarizerConfig
}

// NewLLMSummarizer creates a new LLM-based summarizer.
func NewLLMSummarizer(provider llm.Provider, tok tokenizer.Tokenizer, config LLMSummarizerConfig) *LLMSummarizer {
    if tok == nil {
        tok = &tokenizer.SimpleTokenizer{}
    }
    return &LLMSummarizer{
        provider:  provider,
        tokenizer: tok,
        config:    config,
    }
}

// Summarize implements Summarizer.Summarize.
func (s *LLMSummarizer) Summarize(ctx context.Context, content string, opts Options) (*Result, error) {
    if content == "" {
        return &Result{
            Summary:          "",
            OriginalTokens:   0,
            SummaryTokens:    0,
            CompressionRatio: 1.0,
        }, nil
    }
    
    // Apply defaults
    opts = s.applyDefaults(opts)
    
    // Count original tokens
    originalTokens := s.tokenizer.Count(content)
    
    // Build prompt
    prompt := s.buildPrompt(content, opts)
    
    // Apply timeout
    ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
    defer cancel()
    
    // Call LLM
    params := openai.ChatCompletionNewParams{
        Model: openai.F(s.config.Model),
        Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
            openai.UserMessage(prompt),
        }),
        MaxTokens: openai.Int(int64(opts.MaxTokens)),
    }
    
    completion, err := s.provider.Complete(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("summarization failed: %w", err)
    }
    
    // Extract summary from response
    summary := ""
    if len(completion.Choices) > 0 {
        summary = completion.Choices[0].Message.Content
    }
    
    summaryTokens := s.tokenizer.Count(summary)
    
    return &Result{
        Summary:          summary,
        OriginalTokens:   originalTokens,
        SummaryTokens:    summaryTokens,
        CompressionRatio: float64(summaryTokens) / float64(originalTokens),
        PreservedItems:   opts.PreserveList,
    }, nil
}

// SummarizeMessages implements Summarizer.SummarizeMessages.
func (s *LLMSummarizer) SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error) {
    if len(messages) == 0 {
        return &MessageResult{
            Summary: message.Message{
                Role:    message.RoleAssistant,
                Content: "",
            },
            OriginalCount:   0,
            SummarizedRange: [2]int{0, 0},
        }, nil
    }
    
    // Apply defaults
    opts = s.applyDefaults(opts)
    
    // Format messages for summarization
    formatted := s.formatMessages(messages)
    
    // Build specialized prompt for messages
    prompt := s.buildMessagePrompt(formatted, opts, len(messages))
    
    // Apply timeout
    ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
    defer cancel()
    
    // Call LLM
    params := openai.ChatCompletionNewParams{
        Model: openai.F(s.config.Model),
        Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
            openai.UserMessage(prompt),
        }),
        MaxTokens: openai.Int(int64(opts.MaxTokens)),
    }
    
    completion, err := s.provider.Complete(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("message summarization failed: %w", err)
    }
    
    // Extract summary from response
    summaryContent := ""
    if len(completion.Choices) > 0 {
        summaryContent = completion.Choices[0].Message.Content
    }
    
    return &MessageResult{
        Summary: message.Message{
            Role:    message.RoleAssistant,
            Content: fmt.Sprintf("[Summary of previous %d messages]\n%s", len(messages), summaryContent),
        },
        OriginalCount:   len(messages),
        SummarizedRange: [2]int{0, len(messages) - 1},
    }, nil
}

func (s *LLMSummarizer) applyDefaults(opts Options) Options {
    if opts.MaxTokens <= 0 {
        opts.MaxTokens = s.config.DefaultMaxTokens
    }
    if opts.TargetRatio <= 0 {
        opts.TargetRatio = s.config.DefaultTargetRatio
    }
    if opts.Style == "" {
        opts.Style = s.config.DefaultStyle
    }
    return opts
}

func (s *LLMSummarizer) buildPrompt(content string, opts Options) string {
    styleGuide := s.getStyleGuide(opts.Style)
    preserveGuide := ""
    if len(opts.PreserveList) > 0 {
        preserveGuide = fmt.Sprintf("\nPreserve these items verbatim: %s", strings.Join(opts.PreserveList, ", "))
    }
    
    return fmt.Sprintf(`Summarize the following content concisely while preserving all essential information.

Target: approximately %d tokens (%d%% of original)
Style: %s
%s%s

Content:
---
%s
---

Summary:`,
        opts.MaxTokens,
        int(opts.TargetRatio*100),
        opts.Style,
        styleGuide,
        preserveGuide,
        content)
}

func (s *LLMSummarizer) buildMessagePrompt(formatted string, opts Options, count int) string {
    return fmt.Sprintf(`Summarize this conversation segment while preserving:
1. All decisions made
2. Actions taken and their outcomes
3. Current state/context
4. Unresolved questions or tasks

Target: approximately %d tokens
Format: Single coherent narrative

Conversation (%d messages):
---
%s
---

Summary (as a single message capturing the above):`,
        opts.MaxTokens,
        count,
        formatted)
}

func (s *LLMSummarizer) getStyleGuide(style SummaryStyle) string {
    switch style {
    case StyleBrief:
        return "\nUse minimal words. Include only critical points."
    case StyleDetailed:
        return "\nPreserve important context and nuance."
    case StyleBullet:
        return "\nFormat as bullet points. One point per key item."
    case StyleNarrative:
        return "\nWrite as flowing prose. Maintain logical flow."
    default:
        return ""
    }
}

func (s *LLMSummarizer) formatMessages(messages []message.Message) string {
    var sb strings.Builder
    for _, msg := range messages {
        sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
    }
    return sb.String()
}
```

### 3. SummaryCache

```go
// internal/context/summarizer/cache.go

package summarizer

import (
    "crypto/sha256"
    "encoding/hex"
    "sync"
    "time"
)

// CacheConfig configures the summary cache.
type CacheConfig struct {
    // MaxSize is the maximum number of cached summaries.
    MaxSize int
    
    // TTL is the time-to-live for cached entries.
    TTL time.Duration
}

// DefaultCacheConfig returns sensible default configuration.
func DefaultCacheConfig() CacheConfig {
    return CacheConfig{
        MaxSize: 100,
        TTL:     time.Hour,
    }
}

// CachedSummary contains a cached summary with metadata.
type CachedSummary struct {
    Key         string
    Summary     *Result
    ContentHash string
    CreatedAt   time.Time
    AccessCount int
}

// Cache provides caching for summaries.
type Cache struct {
    cache   map[string]*CachedSummary
    config  CacheConfig
    mu      sync.RWMutex
}

// NewCache creates a new summary cache.
func NewCache(config CacheConfig) *Cache {
    return &Cache{
        cache:  make(map[string]*CachedSummary),
        config: config,
    }
}

// Get retrieves a cached summary if available and not expired.
func (c *Cache) Get(content string) (*Result, bool) {
    hash := hashContent(content)
    
    c.mu.RLock()
    cached, ok := c.cache[hash]
    c.mu.RUnlock()
    
    if !ok {
        return nil, false
    }
    
    // Check TTL
    if time.Since(cached.CreatedAt) > c.config.TTL {
        c.mu.Lock()
        delete(c.cache, hash)
        c.mu.Unlock()
        return nil, false
    }
    
    // Update access count
    c.mu.Lock()
    cached.AccessCount++
    c.mu.Unlock()
    
    return cached.Summary, true
}

// Set stores a summary in the cache.
func (c *Cache) Set(content string, summary *Result) {
    hash := hashContent(content)
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Evict if at capacity
    if len(c.cache) >= c.config.MaxSize {
        c.evictLRU()
    }
    
    c.cache[hash] = &CachedSummary{
        Key:         hash,
        Summary:     summary,
        ContentHash: hash,
        CreatedAt:   time.Now(),
        AccessCount: 0,
    }
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return len(c.cache)
}

// Clear removes all cached entries.
func (c *Cache) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache = make(map[string]*CachedSummary)
}

// evictLRU removes the least recently used entry.
// Must be called with lock held.
func (c *Cache) evictLRU() {
    if len(c.cache) == 0 {
        return
    }
    
    var lruKey string
    var lruCount int = -1
    var lruTime time.Time
    
    for key, entry := range c.cache {
        if lruCount == -1 || entry.AccessCount < lruCount ||
            (entry.AccessCount == lruCount && entry.CreatedAt.Before(lruTime)) {
            lruKey = key
            lruCount = entry.AccessCount
            lruTime = entry.CreatedAt
        }
    }
    
    if lruKey != "" {
        delete(c.cache, lruKey)
    }
}

func hashContent(content string) string {
    h := sha256.New()
    h.Write([]byte(content))
    return hex.EncodeToString(h.Sum(nil))
}
```

### 4. CachingSummarizer (Decorator)

```go
// internal/context/summarizer/caching.go

package summarizer

import (
    "context"
    
    "github.com/dmytrogajewski/spin/internal/message"
)

// CachingSummarizer wraps a Summarizer with caching.
type CachingSummarizer struct {
    inner Summarizer
    cache *Cache
}

// NewCachingSummarizer creates a caching wrapper around a summarizer.
func NewCachingSummarizer(inner Summarizer, cache *Cache) *CachingSummarizer {
    return &CachingSummarizer{
        inner: inner,
        cache: cache,
    }
}

// Summarize implements Summarizer.Summarize with caching.
func (s *CachingSummarizer) Summarize(ctx context.Context, content string, opts Options) (*Result, error) {
    // Check cache first
    if cached, ok := s.cache.Get(content); ok {
        return cached, nil
    }
    
    // Not cached, call inner summarizer
    result, err := s.inner.Summarize(ctx, content, opts)
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    s.cache.Set(content, result)
    
    return result, nil
}

// SummarizeMessages implements Summarizer.SummarizeMessages.
// Note: Message summarization is not cached as message sequences vary.
func (s *CachingSummarizer) SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error) {
    return s.inner.SummarizeMessages(ctx, messages, opts)
}
```

## Acceptance Criteria

1. [ ] `Summarizer` interface defined with `Summarize` and `SummarizeMessages` methods
2. [ ] `LLMSummarizer` implements `Summarizer` interface
3. [ ] `Cache` provides LRU eviction with TTL
4. [ ] `CachingSummarizer` decorator provides transparent caching
5. [ ] All tests pass with >= 90% coverage
6. [ ] `make lint` passes with zero errors
7. [ ] `uast/herr` analysis shows YELLOW or better
8. [ ] Package documentation complete

## Test Cases

### Unit Tests

#### Summarizer Interface Tests
1. **Options defaults** - Default values applied correctly
2. **Result fields** - All fields populated correctly
3. **MessageResult fields** - All fields populated correctly

#### LLMSummarizer Tests
1. **Empty content** - Returns empty result without LLM call
2. **Basic summarization** - Content summarized successfully
3. **Message summarization** - Messages formatted and summarized
4. **Timeout handling** - Context timeout respected
5. **Style guides** - Each style produces appropriate guide
6. **Preserve list** - Preserved items included in prompt
7. **Default application** - Defaults applied when options empty

#### Cache Tests
1. **Get miss** - Returns false for uncached content
2. **Get hit** - Returns cached summary
3. **TTL expiration** - Expired entries not returned
4. **LRU eviction** - Least used entry evicted at capacity
5. **Access count** - Access count incremented on hit
6. **Clear** - All entries removed
7. **Concurrent access** - Thread-safe operations

#### CachingSummarizer Tests
1. **Cache hit** - Returns cached result without inner call
2. **Cache miss** - Calls inner and caches result
3. **Message passthrough** - Messages not cached

## Configuration

Add to `internal/config/config_v2.go`:

```go
// SummarizationConfigV2 configures context summarization.
type SummarizationConfigV2 struct {
    // Enabled controls whether summarization is active.
    Enabled bool `yaml:"enabled" mapstructure:"enabled"`
    
    // Model is the LLM model to use for summarization.
    Model string `yaml:"model" mapstructure:"model"`
    
    // Timeout is the maximum time for summarization requests.
    Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`
    
    // Cache configures the summary cache.
    Cache SummarizationCacheConfigV2 `yaml:"cache" mapstructure:"cache"`
}

// SummarizationCacheConfigV2 configures the summary cache.
type SummarizationCacheConfigV2 struct {
    // Enabled controls whether caching is active.
    Enabled bool `yaml:"enabled" mapstructure:"enabled"`
    
    // MaxSize is the maximum number of cached summaries.
    MaxSize int `yaml:"max_size" mapstructure:"max_size"`
    
    // TTL is the time-to-live for cached entries.
    TTL time.Duration `yaml:"ttl" mapstructure:"ttl"`
}
```

## Files to Create

- `internal/context/summarizer/doc.go`
- `internal/context/summarizer/summarizer.go`
- `internal/context/summarizer/summarizer_test.go`
- `internal/context/summarizer/llm.go`
- `internal/context/summarizer/llm_test.go`
- `internal/context/summarizer/cache.go`
- `internal/context/summarizer/cache_test.go`
- `internal/context/summarizer/caching.go`
- `internal/context/summarizer/caching_test.go`

## Non-Goals

- Incremental summarization (Phase 2)
- Context window management (Phase 2)
- Tool output summarization (Phase 3)
- Enhanced compression integration (Phase 4)
- Configuration integration (deferred to Phase 2+)

## Risks

- **Medium:** LLM summarization quality varies by model
  - Mitigation: Allow configurable model, provide sensible defaults
  
- **Low:** Cache memory usage with large summaries
  - Mitigation: LRU eviction, configurable max size

## References

- Proposal: PROP-CONTEXT-005 (specs/proposals/context/005-context-summarization.md)
- LangChain Context Summarization: https://github.com/langchain-ai/how_to_fix_your_context
- Existing History: internal/history/history.go
- Existing Tokenizer: internal/tokenizer/tokenizer.go
