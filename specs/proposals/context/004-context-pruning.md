# Proposal: Intelligent Context Pruning

**ID**: PROP-CONTEXT-004  
**Title**: Intelligent Context Pruning for Relevance Filtering  
**Status**: Draft  
**Created**: 2025-01-20  
**Author**: AI Assistant  
**References**: [LangChain Context Engineering](https://github.com/langchain-ai/how_to_fix_your_context)

## Summary

Implement intelligent context pruning that removes irrelevant content from retrieved context and tool outputs before passing to the main LLM, reducing token usage while maintaining answer quality.

## Problem Statement

### Current State

Spin's current context handling has these characteristics:
- Tool outputs are passed to LLM in full
- File contents are included completely when read
- Command outputs include all stdout/stderr
- ACE bullet retrieval returns complete bullet content
- Compression happens reactively at 80% token threshold

### Identified Issues

1. **Verbose Tool Outputs**: `shell_command` may return thousands of lines
2. **Full File Reads**: Reading a 1000-line file when only 10 lines are relevant
3. **Irrelevant Bullet Content**: Retrieved bullets may contain tangential information
4. **Build/Test Noise**: Test output includes passing tests, not just failures
5. **Wasted Tokens**: Paying for tokens that don't improve response quality

### Context Rot Risks

- **Context Confusion**: Irrelevant content pressures LLM to use all information
- **Context Distraction**: Verbose outputs overshadow critical information
- **Token Exhaustion**: Reaching limits before completing tasks

### Real Example (from LangChain research)

A RAG query about "agent memory" retrieved 25K tokens but only 11K were relevant. Pruning achieved 56% token reduction with identical answer quality.

## Proposed Solution

### 1. Content Pruner Interface

Define the core interface for content pruning.

```go
// internal/context/pruner/pruner.go

type Pruner interface {
    // Prune removes irrelevant content based on query context
    Prune(ctx context.Context, content string, query string, opts PruneOptions) (*PruneResult, error)
}

type PruneOptions struct {
    MaxTokens       int             // Maximum output tokens
    MinRetention    float64         // Minimum content to keep (0.0-1.0)
    ContentType     ContentType     // Type of content being pruned
    PreserveStructure bool          // Keep code structure intact
}

type PruneResult struct {
    Content         string
    OriginalTokens  int
    PrunedTokens    int
    ReductionRatio  float64
    Sections        []PrunedSection // What was kept/removed
}

type PrunedSection struct {
    Content   string
    Kept      bool
    Reason    string
    Relevance float64
}

type ContentType string

const (
    ContentTypeCode       ContentType = "code"
    ContentTypeLog        ContentType = "log"
    ContentTypeDocument   ContentType = "document"
    ContentTypeBullet     ContentType = "bullet"
    ContentTypeCommandOut ContentType = "command_output"
)
```

### 2. LLM-Based Pruner

Use a small, fast LLM to extract relevant content.

```go
// internal/context/pruner/llm_pruner.go

type LLMPruner struct {
    client     llm.Client
    model      string  // Use fast model like gpt-4o-mini or haiku
    maxTokens  int
    timeout    time.Duration
}

func (p *LLMPruner) Prune(ctx context.Context, content string, query string, opts PruneOptions) (*PruneResult, error) {
    // Build pruning prompt
    prompt := p.buildPrompt(content, query, opts)
    
    // Call small/fast LLM
    ctx, cancel := context.WithTimeout(ctx, p.timeout)
    defer cancel()
    
    response, err := p.client.Complete(ctx, llm.Request{
        Model:    p.model,
        Messages: []llm.Message{{Role: "user", Content: prompt}},
        MaxTokens: opts.MaxTokens,
    })
    if err != nil {
        // Fallback to rule-based pruning
        return p.fallbackPrune(content, opts)
    }
    
    return p.parseResponse(response, content)
}

func (p *LLMPruner) buildPrompt(content string, query string, opts PruneOptions) string {
    return fmt.Sprintf(`Extract only the parts of the following content that are directly relevant to answering this query.

Query: %s

Content Type: %s

Instructions:
- Remove sections that don't help answer the query
- Keep exact quotes when relevant
- Preserve code structure if content is code
- Keep error messages and stack traces if debugging
- Target maximum %d tokens in output
- Maintain coherence - don't leave dangling references

Content:
---
%s
---

Extracted relevant content:`, query, opts.ContentType, opts.MaxTokens, content)
}
```

### 3. Rule-Based Pruner

Fast, deterministic pruning for common patterns.

```go
// internal/context/pruner/rule_pruner.go

type RuleBasedPruner struct {
    rules []PruningRule
}

type PruningRule interface {
    Applies(contentType ContentType) bool
    Prune(content string, query string) string
}

// Log pruning: keep errors, warnings, and query-relevant lines
type LogPruningRule struct{}

func (r *LogPruningRule) Applies(ct ContentType) bool {
    return ct == ContentTypeLog || ct == ContentTypeCommandOut
}

func (r *LogPruningRule) Prune(content string, query string) string {
    lines := strings.Split(content, "\n")
    relevant := []string{}
    
    queryTerms := extractTerms(query)
    
    for i, line := range lines {
        lower := strings.ToLower(line)
        
        // Always keep errors and warnings
        if containsAny(lower, []string{"error", "fail", "panic", "fatal", "warn"}) {
            // Include context lines
            relevant = append(relevant, getContextLines(lines, i, 2)...)
            continue
        }
        
        // Keep lines matching query terms
        for _, term := range queryTerms {
            if strings.Contains(lower, strings.ToLower(term)) {
                relevant = append(relevant, getContextLines(lines, i, 1)...)
                break
            }
        }
    }
    
    return strings.Join(deduplicate(relevant), "\n")
}

// Test output pruning: focus on failures
type TestOutputPruningRule struct{}

func (r *TestOutputPruningRule) Prune(content string, query string) string {
    lines := strings.Split(content, "\n")
    relevant := []string{}
    
    inFailure := false
    failureIndent := 0
    
    for _, line := range lines {
        // Detect failure start
        if containsAny(line, []string{"FAIL", "FAILED", "Error:", "panic:"}) {
            inFailure = true
            failureIndent = countLeadingSpaces(line)
            relevant = append(relevant, line)
            continue
        }
        
        // Continue capturing failure details
        if inFailure {
            if countLeadingSpaces(line) > failureIndent || line == "" {
                relevant = append(relevant, line)
            } else {
                inFailure = false
            }
        }
        
        // Keep summary lines
        if containsAny(line, []string{"PASS", "FAIL", "ok ", "---"}) {
            relevant = append(relevant, line)
        }
    }
    
    return strings.Join(relevant, "\n")
}

// Code pruning: keep relevant functions/methods
type CodePruningRule struct {
    parser CodeParser
}

func (r *CodePruningRule) Prune(content string, query string) string {
    // Parse code into AST
    ast, err := r.parser.Parse(content)
    if err != nil {
        return content // Can't parse, return as-is
    }
    
    queryTerms := extractTerms(query)
    relevant := []string{}
    
    // Walk AST and keep relevant declarations
    for _, decl := range ast.Declarations {
        if r.isRelevant(decl, queryTerms) {
            relevant = append(relevant, decl.Source)
        }
    }
    
    return strings.Join(relevant, "\n\n")
}
```

### 4. Hybrid Pruner

Combine rule-based (fast) and LLM-based (accurate) pruning.

```go
// internal/context/pruner/hybrid.go

type HybridPruner struct {
    ruleBased  *RuleBasedPruner
    llmBased   *LLMPruner
    threshold  int  // Token threshold for LLM pruning
}

func (p *HybridPruner) Prune(ctx context.Context, content string, query string, opts PruneOptions) (*PruneResult, error) {
    // Count tokens
    originalTokens := countTokens(content)
    
    // Small content: no pruning needed
    if originalTokens < p.threshold/2 {
        return &PruneResult{
            Content:        content,
            OriginalTokens: originalTokens,
            PrunedTokens:   originalTokens,
            ReductionRatio: 0,
        }, nil
    }
    
    // First pass: rule-based pruning (fast)
    ruleResult := p.ruleBased.Prune(content, query, opts)
    
    // If sufficient reduction, return
    if ruleResult.ReductionRatio > 0.3 {
        return ruleResult, nil
    }
    
    // Second pass: LLM pruning for better accuracy
    return p.llmBased.Prune(ctx, ruleResult.Content, query, opts)
}
```

### 5. Tool Output Pruner

Integrate pruning into tool execution.

```go
// internal/tools/pruning_wrapper.go

type PruningToolWrapper struct {
    tool    Tool
    pruner  Pruner
    config  PruningConfig
}

type PruningConfig struct {
    Enabled       bool
    MaxOutputTokens int
    ContentTypeMap  map[string]ContentType
}

func (w *PruningToolWrapper) Execute(ctx context.Context, params interface{}) (*ToolResult, error) {
    // Execute underlying tool
    result, err := w.tool.Execute(ctx, params)
    if err != nil {
        return result, err
    }
    
    // Check if pruning is needed
    tokens := countTokens(result.Output)
    if tokens < w.config.MaxOutputTokens {
        return result, nil
    }
    
    // Get current query from context
    query := getQueryFromContext(ctx)
    
    // Determine content type
    contentType := w.getContentType(w.tool.Name())
    
    // Prune output
    pruned, err := w.pruner.Prune(ctx, result.Output, query, PruneOptions{
        MaxTokens:   w.config.MaxOutputTokens,
        ContentType: contentType,
    })
    if err != nil {
        // Return original on error
        return result, nil
    }
    
    // Return pruned result with metadata
    result.Output = pruned.Content
    result.Metadata["original_tokens"] = pruned.OriginalTokens
    result.Metadata["pruned_tokens"] = pruned.PrunedTokens
    result.Metadata["reduction_ratio"] = pruned.ReductionRatio
    
    return result, nil
}
```

### 6. Retrieved Context Pruner

Prune ACE bullet content before injection.

```go
// internal/ace/retrieval/pruning_retriever.go

type PruningRetriever struct {
    base   Retriever
    pruner Pruner
    config RetrievalPruningConfig
}

type RetrievalPruningConfig struct {
    Enabled         bool
    MaxTokensPerBullet int
    MaxTotalTokens  int
}

func (r *PruningRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error) {
    // Get bullets from base retriever
    bullets, err := r.base.Retrieve(ctx, query, topK)
    if err != nil {
        return nil, err
    }
    
    // Calculate total tokens
    totalTokens := 0
    for _, b := range bullets {
        totalTokens += countTokens(b.Content)
    }
    
    // No pruning needed
    if totalTokens < r.config.MaxTotalTokens {
        return bullets, nil
    }
    
    // Prune each bullet
    prunedBullets := make([]*bullet.Bullet, 0, len(bullets))
    remainingTokens := r.config.MaxTotalTokens
    
    for _, b := range bullets {
        bulletTokens := countTokens(b.Content)
        
        // Prune if over budget
        if bulletTokens > r.config.MaxTokensPerBullet || bulletTokens > remainingTokens {
            maxTokens := min(r.config.MaxTokensPerBullet, remainingTokens)
            
            pruned, err := r.pruner.Prune(ctx, b.Content, query, PruneOptions{
                MaxTokens:   maxTokens,
                ContentType: ContentTypeBullet,
            })
            if err == nil {
                b = b.Clone()
                b.Content = pruned.Content
                bulletTokens = pruned.PrunedTokens
            }
        }
        
        if remainingTokens >= bulletTokens {
            prunedBullets = append(prunedBullets, b)
            remainingTokens -= bulletTokens
        }
    }
    
    return prunedBullets, nil
}
```

### 7. Streaming Pruner for Long Outputs

Handle very long outputs by pruning incrementally.

```go
// internal/context/pruner/streaming.go

type StreamingPruner struct {
    pruner      Pruner
    chunkSize   int
    maxChunks   int
}

func (p *StreamingPruner) PruneStream(ctx context.Context, reader io.Reader, query string, opts PruneOptions) (*PruneResult, error) {
    chunks := []string{}
    scanner := bufio.NewScanner(reader)
    scanner.Buffer(make([]byte, p.chunkSize), p.chunkSize)
    
    currentChunk := strings.Builder{}
    chunkLines := 0
    
    for scanner.Scan() {
        line := scanner.Text()
        currentChunk.WriteString(line)
        currentChunk.WriteString("\n")
        chunkLines++
        
        // Process chunk when full
        if chunkLines >= 100 {
            pruned, err := p.pruner.Prune(ctx, currentChunk.String(), query, opts)
            if err == nil {
                chunks = append(chunks, pruned.Content)
            }
            currentChunk.Reset()
            chunkLines = 0
            
            // Limit total chunks
            if len(chunks) >= p.maxChunks {
                break
            }
        }
    }
    
    // Process remaining
    if currentChunk.Len() > 0 {
        pruned, _ := p.pruner.Prune(ctx, currentChunk.String(), query, opts)
        chunks = append(chunks, pruned.Content)
    }
    
    // Final merge and prune
    merged := strings.Join(chunks, "\n---\n")
    return p.pruner.Prune(ctx, merged, query, opts)
}
```

## Configuration

```yaml
context:
  pruning:
    enabled: true
    
    tool_outputs:
      enabled: true
      max_tokens: 2000
      
      rules:
        shell_command:
          content_type: command_output
          max_tokens: 1500
          preserve_errors: true
          
        read_file:
          content_type: code
          max_tokens: 3000
          preserve_structure: true
          
        file_search:
          content_type: log
          max_tokens: 1000
          
    retrieval:
      enabled: true
      max_tokens_per_bullet: 500
      max_total_tokens: 3000
      
    llm_pruner:
      enabled: true
      model: "gpt-4o-mini"  # or "claude-3-haiku"
      timeout: 5s
      fallback_to_rules: true
```

## Implementation Plan

### Phase 1: Rule-Based Foundation (Week 1)
1. Implement `Pruner` interface
2. Build `RuleBasedPruner` with common rules
3. Add log/test output pruning rules
4. Integrate with tool wrapper

### Phase 2: LLM Pruning (Week 2)
1. Implement `LLMPruner`
2. Build `HybridPruner`
3. Add fallback mechanisms
4. Benchmark accuracy vs speed

### Phase 3: Integration (Week 3)
1. Integrate with tool execution
2. Add retrieval pruning
3. Implement streaming pruner
4. Add pruning events/metrics

### Phase 4: Optimization (Week 4)
1. Tune thresholds based on data
2. Add content-type specific rules
3. Implement caching for similar queries
4. Document best practices

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Average Tool Output Tokens | Variable | <2000 |
| Token Reduction Ratio | 0% | 40-60% |
| Answer Quality (with pruning) | Baseline | ≥95% of baseline |
| Pruning Latency (rule-based) | N/A | <10ms |
| Pruning Latency (LLM) | N/A | <2s |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Over-pruning (lost info) | High | Min retention ratio, quality validation |
| Latency increase | Medium | Rule-based first, LLM only when needed |
| LLM cost | Medium | Use smallest effective model |
| Edge cases | Medium | Comprehensive test suite, fallbacks |

## Examples

### Example 1: Test Output Pruning

**Original** (5000 tokens):
```
=== RUN   TestUserCreation
--- PASS: TestUserCreation (0.01s)
=== RUN   TestUserUpdate
--- PASS: TestUserUpdate (0.01s)
... (100 more passing tests)
=== RUN   TestUserDelete
    user_test.go:150: Expected nil error, got: permission denied
--- FAIL: TestUserDelete (0.02s)
FAIL
```

**Pruned** (200 tokens):
```
=== RUN   TestUserDelete
    user_test.go:150: Expected nil error, got: permission denied
--- FAIL: TestUserDelete (0.02s)
FAIL
```

### Example 2: File Content Pruning

**Query**: "Fix the authentication bug"

**Original file** (3000 tokens): Full `auth.go` file

**Pruned** (800 tokens): Only `Authenticate()`, `ValidateToken()`, and error handling code

### Example 3: Bullet Pruning

**Original bullet** (600 tokens):
```
When handling Go errors, there are several patterns to consider.
First, always wrap errors with context using fmt.Errorf with %w.
Second, define sentinel errors for expected conditions.
Third, use errors.Is and errors.As for checking.
The history of error handling in Go started with...
[extensive background]
```

**Pruned for query "wrap errors"** (150 tokens):
```
When handling Go errors: always wrap errors with context using 
fmt.Errorf with %w verb. Example: fmt.Errorf("failed to process: %w", err)
```

## References

- [LangChain Context Pruning](https://github.com/langchain-ai/how_to_fix_your_context/blob/main/notebooks/4_context_pruning.ipynb)
- [Lost in the Middle](https://arxiv.org/abs/2307.03172) - Why LLMs struggle with long contexts
- [Selective Context](https://arxiv.org/abs/2310.06201) - Relevance filtering research
