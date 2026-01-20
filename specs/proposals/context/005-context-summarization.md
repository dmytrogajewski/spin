# Proposal: Progressive Context Summarization

**ID**: PROP-CONTEXT-005  
**Title**: Progressive Context Summarization for Long Conversations  
**Status**: Draft  
**Created**: 2025-01-20  
**Author**: AI Assistant  
**References**: [LangChain Context Engineering](https://github.com/langchain-ai/how_to_fix_your_context)

## Summary

Implement progressive context summarization that condenses older conversation history and verbose content while preserving essential information, enabling longer effective conversations within token limits.

## Problem Statement

### Current State

Spin implements reactive context compression (`internal/history/compress/`):
- Triggers at 80% of token limit
- Uses importance-weighted selection (critical/high/medium/low)
- Greedy selection algorithm within token budget
- No summarization - just selection/removal

### Identified Issues

1. **Binary Retention**: Content is either kept fully or removed entirely
2. **Loss of Continuity**: Removed messages break conversation flow
3. **Reactive Not Proactive**: Compression happens at crisis point
4. **No Semantic Compression**: Can't condense verbose content intelligently

### Context Rot Risks

- **Context Distraction**: Old, verbose content dominates
- **Context Confusion**: Disjointed history after compression
- **Lost Decisions**: Important reasoning may be lost in compression

### Key Distinction from Pruning

| Aspect | Pruning | Summarization |
|--------|---------|---------------|
| Goal | Remove irrelevant content | Compress all content |
| Output | Subset of original | New condensed text |
| Information | Discarded | Preserved (compressed) |
| Use case | Tool outputs, retrieval | Conversation history |

## Proposed Solution

### 1. Summarizer Interface

Define the core summarization interface.

```go
// internal/context/summarizer/summarizer.go

type Summarizer interface {
    // Summarize compresses content while preserving key information
    Summarize(ctx context.Context, content string, opts SummarizeOptions) (*SummaryResult, error)
    
    // SummarizeMessages summarizes a sequence of conversation messages
    SummarizeMessages(ctx context.Context, messages []Message, opts SummarizeOptions) (*MessageSummary, error)
}

type SummarizeOptions struct {
    MaxTokens       int              // Target output size
    TargetRatio     float64          // Target compression ratio (e.g., 0.3 = 30%)
    PreserveList    []string         // Items to preserve verbatim
    ContentType     ContentType      // Type of content
    Style           SummaryStyle     // Summary style
}

type SummaryStyle string

const (
    StyleBrief      SummaryStyle = "brief"      // Minimal, key points only
    StyleDetailed   SummaryStyle = "detailed"   // More context preserved
    StyleBullet     SummaryStyle = "bullet"     // Bullet point format
    StyleNarrative  SummaryStyle = "narrative"  // Flowing narrative
)

type SummaryResult struct {
    Summary         string
    OriginalTokens  int
    SummaryTokens   int
    CompressionRatio float64
    PreservedItems  []string      // Items kept verbatim
    KeyPoints       []string      // Extracted key points
}

type MessageSummary struct {
    Summary         Message       // Single summary message
    OriginalCount   int
    SummarizedRange [2]int        // Message indices summarized
    KeyDecisions    []string      // Important decisions preserved
    KeyActions      []string      // Actions taken
}
```

### 2. LLM-Based Summarizer

Use LLM for intelligent summarization.

```go
// internal/context/summarizer/llm_summarizer.go

type LLMSummarizer struct {
    client    llm.Client
    model     string
    timeout   time.Duration
}

func (s *LLMSummarizer) Summarize(ctx context.Context, content string, opts SummarizeOptions) (*SummaryResult, error) {
    prompt := s.buildPrompt(content, opts)
    
    ctx, cancel := context.WithTimeout(ctx, s.timeout)
    defer cancel()
    
    response, err := s.client.Complete(ctx, llm.Request{
        Model:    s.model,
        Messages: []llm.Message{{Role: "user", Content: prompt}},
        MaxTokens: opts.MaxTokens,
    })
    if err != nil {
        return nil, err
    }
    
    return &SummaryResult{
        Summary:          response.Content,
        OriginalTokens:   countTokens(content),
        SummaryTokens:    countTokens(response.Content),
        CompressionRatio: float64(countTokens(response.Content)) / float64(countTokens(content)),
    }, nil
}

func (s *LLMSummarizer) buildPrompt(content string, opts SummarizeOptions) string {
    styleGuide := s.getStyleGuide(opts.Style)
    preserveGuide := ""
    if len(opts.PreserveList) > 0 {
        preserveGuide = fmt.Sprintf("\nPreserve these items verbatim: %s", strings.Join(opts.PreserveList, ", "))
    }
    
    return fmt.Sprintf(`Summarize the following content concisely while preserving all essential information.

Target: %d tokens (approximately %d%% of original)
Style: %s
%s
%s

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

func (s *LLMSummarizer) SummarizeMessages(ctx context.Context, messages []Message, opts SummarizeOptions) (*MessageSummary, error) {
    // Format messages for summarization
    formatted := s.formatMessages(messages)
    
    prompt := fmt.Sprintf(`Summarize this conversation segment while preserving:
1. All decisions made
2. Actions taken and their outcomes
3. Current state/context
4. Unresolved questions or tasks

Target: %d tokens
Format: Single coherent narrative

Conversation:
---
%s
---

Summary (as a single message capturing the above):`, opts.MaxTokens, formatted)
    
    response, err := s.client.Complete(ctx, llm.Request{
        Model:    s.model,
        Messages: []llm.Message{{Role: "user", Content: prompt}},
    })
    if err != nil {
        return nil, err
    }
    
    // Extract key decisions and actions from summary
    keyPoints := s.extractKeyPoints(response.Content)
    
    return &MessageSummary{
        Summary: Message{
            Role:    "assistant",
            Content: fmt.Sprintf("[Summary of previous %d messages]\n%s", len(messages), response.Content),
        },
        OriginalCount:   len(messages),
        SummarizedRange: [2]int{0, len(messages) - 1},
        KeyDecisions:    keyPoints.Decisions,
        KeyActions:      keyPoints.Actions,
    }, nil
}
```

### 3. Incremental Summarizer

Summarize progressively as conversation grows.

```go
// internal/context/summarizer/incremental.go

type IncrementalSummarizer struct {
    summarizer    Summarizer
    summaryChain  []ChainedSummary
    config        IncrementalConfig
}

type IncrementalConfig struct {
    ChunkSize       int     // Messages per chunk to summarize
    OverlapMessages int     // Messages to overlap between chunks
    MaxChainLength  int     // Max number of chained summaries
    MinTokensToSummarize int // Minimum tokens before summarizing
}

type ChainedSummary struct {
    Summary       string
    OriginalRange [2]int      // Original message indices
    Timestamp     time.Time
    TokenCount    int
}

func (s *IncrementalSummarizer) ProcessMessages(ctx context.Context, messages []Message) ([]Message, error) {
    // Check if summarization needed
    totalTokens := s.countTotalTokens(messages)
    if totalTokens < s.config.MinTokensToSummarize {
        return messages, nil
    }
    
    // Identify messages to summarize (older messages)
    summarizeCount := s.calculateSummarizeCount(messages)
    if summarizeCount < s.config.ChunkSize {
        return messages, nil
    }
    
    toSummarize := messages[:summarizeCount]
    toKeep := messages[summarizeCount-s.config.OverlapMessages:]
    
    // Summarize older messages
    summary, err := s.summarizer.SummarizeMessages(ctx, toSummarize, SummarizeOptions{
        MaxTokens:   s.config.ChunkSize * 50, // Rough estimate
        TargetRatio: 0.3,
        Style:       StyleNarrative,
    })
    if err != nil {
        return messages, nil // Keep original on error
    }
    
    // Chain summaries if needed
    s.addToChain(summary, summarizeCount)
    
    // Return summary + recent messages
    result := []Message{summary.Summary}
    result = append(result, toKeep...)
    
    return result, nil
}

func (s *IncrementalSummarizer) addToChain(summary *MessageSummary, count int) {
    newChain := ChainedSummary{
        Summary:       summary.Summary.Content,
        OriginalRange: [2]int{0, count},
        Timestamp:     time.Now(),
        TokenCount:    countTokens(summary.Summary.Content),
    }
    
    s.summaryChain = append(s.summaryChain, newChain)
    
    // Consolidate chain if too long
    if len(s.summaryChain) > s.config.MaxChainLength {
        s.consolidateChain()
    }
}

func (s *IncrementalSummarizer) consolidateChain() {
    // Summarize summaries (meta-summarization)
    chainContent := strings.Builder{}
    for _, cs := range s.summaryChain {
        chainContent.WriteString(cs.Summary)
        chainContent.WriteString("\n---\n")
    }
    
    // Create meta-summary
    // ... (similar to regular summarization)
}
```

### 4. Context Window Manager

Manage context window with summarization.

```go
// internal/context/window_manager.go

type WindowManager struct {
    maxTokens           int
    summarizer          Summarizer
    incrementalSummarizer *IncrementalSummarizer
    strategy            WindowStrategy
}

type WindowStrategy string

const (
    StrategySliding     WindowStrategy = "sliding"     // Summarize oldest, keep recent
    StrategyHierarchical WindowStrategy = "hierarchical" // Multi-level summaries
    StrategyImportance  WindowStrategy = "importance"  // Summarize low-importance first
)

func (m *WindowManager) ManageWindow(ctx context.Context, messages []Message) ([]Message, error) {
    currentTokens := m.countTokens(messages)
    
    // Under limit - no action needed
    if currentTokens < int(float64(m.maxTokens)*0.7) {
        return messages, nil
    }
    
    switch m.strategy {
    case StrategySliding:
        return m.slidingWindowSummarize(ctx, messages)
    case StrategyHierarchical:
        return m.hierarchicalSummarize(ctx, messages)
    case StrategyImportance:
        return m.importanceSummarize(ctx, messages)
    default:
        return m.slidingWindowSummarize(ctx, messages)
    }
}

func (m *WindowManager) slidingWindowSummarize(ctx context.Context, messages []Message) ([]Message, error) {
    // Keep most recent N messages
    recentCount := m.calculateRecentCount(messages)
    
    if recentCount >= len(messages) {
        return messages, nil
    }
    
    // Summarize older messages
    oldMessages := messages[:len(messages)-recentCount]
    recentMessages := messages[len(messages)-recentCount:]
    
    summary, err := m.summarizer.SummarizeMessages(ctx, oldMessages, SummarizeOptions{
        TargetRatio: 0.2,
        Style:       StyleNarrative,
    })
    if err != nil {
        // Fallback: just truncate
        return recentMessages, nil
    }
    
    result := []Message{summary.Summary}
    result = append(result, recentMessages...)
    return result, nil
}

func (m *WindowManager) hierarchicalSummarize(ctx context.Context, messages []Message) ([]Message, error) {
    // Level 1: Individual message summaries
    // Level 2: Chunk summaries (5-10 messages)
    // Level 3: Section summaries (multiple chunks)
    
    levels := m.buildHierarchy(messages)
    
    // Select appropriate level based on token budget
    targetTokens := int(float64(m.maxTokens) * 0.5)
    
    for level := len(levels) - 1; level >= 0; level-- {
        if m.countTokens(levels[level]) <= targetTokens {
            return levels[level], nil
        }
    }
    
    // Even highest summary too large - summarize again
    return m.summarizeLevel(ctx, levels[len(levels)-1])
}
```

### 5. Tool Output Summarizer

Summarize verbose tool outputs before adding to history.

```go
// internal/tools/summarizing_wrapper.go

type SummarizingToolWrapper struct {
    tool       Tool
    summarizer Summarizer
    config     ToolSummaryConfig
}

type ToolSummaryConfig struct {
    Enabled         bool
    ThresholdTokens int              // Summarize if over this
    TargetTokens    int              // Target after summarization
    KeepOriginal    bool             // Store original separately
    ToolConfigs     map[string]ToolSummaryRule
}

type ToolSummaryRule struct {
    Enabled       bool
    TargetRatio   float64
    PreservePatterns []string       // Regex patterns to preserve
    Style         SummaryStyle
}

func (w *SummarizingToolWrapper) Execute(ctx context.Context, params interface{}) (*ToolResult, error) {
    result, err := w.tool.Execute(ctx, params)
    if err != nil {
        return result, err
    }
    
    // Check if summarization needed
    tokens := countTokens(result.Output)
    if tokens < w.config.ThresholdTokens {
        return result, nil
    }
    
    // Get tool-specific config
    rule := w.getRule(w.tool.Name())
    if !rule.Enabled {
        return result, nil
    }
    
    // Preserve patterns (errors, specific content)
    preserved := w.extractPreserved(result.Output, rule.PreservePatterns)
    
    // Summarize
    summary, err := w.summarizer.Summarize(ctx, result.Output, SummarizeOptions{
        MaxTokens:    w.config.TargetTokens,
        TargetRatio:  rule.TargetRatio,
        PreserveList: preserved,
        Style:        rule.Style,
    })
    if err != nil {
        return result, nil // Keep original on error
    }
    
    // Store original if configured
    if w.config.KeepOriginal {
        result.Metadata["original_output"] = result.Output
    }
    
    result.Output = summary.Summary
    result.Metadata["summarized"] = true
    result.Metadata["original_tokens"] = summary.OriginalTokens
    result.Metadata["summary_tokens"] = summary.SummaryTokens
    
    return result, nil
}
```

### 6. Conversation Summary Cache

Cache summaries for efficient retrieval.

```go
// internal/context/summarizer/cache.go

type SummaryCache struct {
    cache     map[string]*CachedSummary
    maxSize   int
    ttl       time.Duration
    mu        sync.RWMutex
}

type CachedSummary struct {
    Key          string
    Summary      *SummaryResult
    ContentHash  string
    CreatedAt    time.Time
    AccessCount  int
}

func (c *SummaryCache) Get(content string) (*SummaryResult, bool) {
    hash := hashContent(content)
    
    c.mu.RLock()
    cached, ok := c.cache[hash]
    c.mu.RUnlock()
    
    if !ok || time.Since(cached.CreatedAt) > c.ttl {
        return nil, false
    }
    
    cached.AccessCount++
    return cached.Summary, true
}

func (c *SummaryCache) Set(content string, summary *SummaryResult) {
    hash := hashContent(content)
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Evict if at capacity
    if len(c.cache) >= c.maxSize {
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
```

### 7. Enhanced History Compression

Upgrade existing compression with summarization.

```go
// internal/history/compress/summarizing_compressor.go

type SummarizingCompressor struct {
    base       Compressor
    summarizer Summarizer
    config     SummarizingCompressorConfig
}

type SummarizingCompressorConfig struct {
    UseSummarization     bool
    SummaryThreshold     float64  // Token ratio to trigger summarization
    FallbackToSelection  bool     // Fall back to selection if summarization fails
}

func (c *SummarizingCompressor) Compress(ctx context.Context, messages []Message, budget int) ([]Message, *CompressStats, error) {
    totalTokens := countTotalTokens(messages)
    
    // Under budget - no compression
    if totalTokens <= budget {
        return messages, &CompressStats{}, nil
    }
    
    requiredReduction := float64(totalTokens-budget) / float64(totalTokens)
    
    // Light compression: use selection
    if requiredReduction < c.config.SummaryThreshold {
        return c.base.Compress(ctx, messages, budget)
    }
    
    // Heavy compression: use summarization
    return c.summarizeCompress(ctx, messages, budget)
}

func (c *SummarizingCompressor) summarizeCompress(ctx context.Context, messages []Message, budget int) ([]Message, *CompressStats, error) {
    // Classify messages by importance
    classified := c.classifyMessages(messages)
    
    // Always keep critical messages
    result := classified.Critical
    usedTokens := countTotalTokens(result)
    
    // Summarize high/medium importance together
    toSummarize := append(classified.High, classified.Medium...)
    if len(toSummarize) > 0 {
        targetTokens := budget - usedTokens - 500 // Reserve for low
        
        summary, err := c.summarizer.SummarizeMessages(ctx, toSummarize, SummarizeOptions{
            MaxTokens:   targetTokens,
            TargetRatio: 0.3,
            Style:       StyleNarrative,
        })
        
        if err != nil && c.config.FallbackToSelection {
            return c.base.Compress(ctx, messages, budget)
        }
        
        if err == nil {
            result = append(result, summary.Summary)
            usedTokens += countTokens(summary.Summary.Content)
        }
    }
    
    // Add low priority if space remains
    for _, msg := range classified.Low {
        msgTokens := countTokens(msg.Content)
        if usedTokens+msgTokens <= budget {
            result = append(result, msg)
            usedTokens += msgTokens
        }
    }
    
    // Sort by original order
    sortByOriginalOrder(result, messages)
    
    return result, &CompressStats{
        OriginalTokens:  countTotalTokens(messages),
        CompressedTokens: usedTokens,
        MessagesRemoved: len(messages) - len(result),
        Summarized:      true,
    }, nil
}
```

## Configuration

```yaml
context:
  summarization:
    enabled: true
    
    conversation:
      strategy: "sliding"           # sliding, hierarchical, importance
      chunk_size: 10                # Messages per summary chunk
      overlap_messages: 2           # Overlap between chunks
      target_ratio: 0.3             # Target 30% of original
      
    tool_outputs:
      enabled: true
      threshold_tokens: 1000        # Summarize if over
      target_tokens: 300
      preserve_errors: true
      
      per_tool:
        shell_command:
          enabled: true
          target_ratio: 0.2
          preserve_patterns:
            - "error|Error|ERROR"
            - "fail|Fail|FAIL"
            
        read_file:
          enabled: false            # Usually want full file
          
    compression:
      use_summarization: true
      summary_threshold: 0.4        # Use summarization for >40% reduction
      fallback_to_selection: true
      
    cache:
      enabled: true
      max_size: 100
      ttl: 1h
      
    llm:
      model: "gpt-4o-mini"          # Fast model for summarization
      timeout: 10s
```

## Implementation Plan

### Phase 1: Core Summarizer (Week 1)
1. Implement `Summarizer` interface
2. Build `LLMSummarizer`
3. Add summary caching
4. Unit tests

### Phase 2: Conversation Summarization (Week 2)
1. Implement `IncrementalSummarizer`
2. Build `WindowManager`
3. Integrate with conversation history
4. Add strategy implementations

### Phase 3: Tool Output Summarization (Week 3)
1. Build `SummarizingToolWrapper`
2. Add tool-specific rules
3. Integrate with tool execution
4. Preserve error patterns

### Phase 4: Enhanced Compression (Week 4)
1. Upgrade `SummarizingCompressor`
2. Integrate with existing compression
3. Add fallback mechanisms
4. Performance optimization

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Effective Context Length | ~50K tokens | ~150K equivalent |
| Compression Ratio | 0% | 50-70% |
| Information Retention | 100% (no summary) | 95%+ |
| Summary Quality Score | N/A | 4.0/5.0 (human eval) |
| Summarization Latency | N/A | <5s per summary |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Information loss | High | Quality validation, preserve critical items |
| Latency increase | Medium | Caching, async summarization |
| LLM cost | Medium | Use smallest effective model, cache |
| Summary hallucination | Medium | Extractive bias, validation |

## Examples

### Example 1: Conversation Summarization

**Original** (20 messages, 8000 tokens):
```
User: Help me fix the authentication bug
Assistant: I'll look at the auth code...
[reads file]
Assistant: Found the issue in validateToken...
[applies patch]
User: Still failing
Assistant: Let me check the tests...
[runs tests]
Assistant: Ah, there's also an issue in refreshToken...
[more debugging]
```

**Summary** (1 message, 500 tokens):
```
[Summary of previous 20 messages]
Investigated authentication bug in auth.go. Found and fixed 
issue in validateToken() - was not checking token expiry correctly.
Applied patch to add expiry validation. Subsequent testing revealed
additional issue in refreshToken() - currently investigating.
Key decisions: Using JWT validation library instead of manual parsing.
Current state: First fix applied, second issue in progress.
```

### Example 2: Tool Output Summarization

**Original shell_command output** (3000 tokens):
```
Running tests...
=== RUN TestAuth
--- PASS: TestAuth (0.01s)
... (100 more passing tests)
=== RUN TestRefreshToken
    auth_test.go:234: token refresh failed: invalid signature
--- FAIL: TestRefreshToken (0.02s)
```

**Summary** (200 tokens):
```
Test Results Summary:
- Total: 102 tests
- Passed: 101
- Failed: 1

FAILURE:
TestRefreshToken (auth_test.go:234)
Error: token refresh failed: invalid signature
```

## References

- [LangChain Context Summarization](https://github.com/langchain-ai/how_to_fix_your_context/blob/main/notebooks/5_context_summarization.ipynb)
- [Conversation Summarization](https://arxiv.org/abs/2109.10460)
- [Hierarchical Summarization](https://arxiv.org/abs/2301.13298)
