# Proposal: RAG-Enhanced Context Retrieval

**ID**: PROP-CONTEXT-001  
**Title**: RAG-Enhanced Context Retrieval for ACE System  
**Status**: Draft  
**Created**: 2025-01-20  
**Author**: AI Assistant  
**References**: [LangChain Context Engineering](https://github.com/langchain-ai/how_to_fix_your_context)

## Summary

Enhance Spin's existing ACE retrieval system with advanced RAG (Retrieval-Augmented Generation) techniques to improve the quality and relevance of context provided to the LLM during agent execution.

## Problem Statement

### Current State

Spin already implements semantic retrieval through the ACE playbook system:
- Bullets are embedded using Ollama embeddings
- Retrieval uses cosine similarity via HNSW index
- Query building combines user query with trajectory context
- Results are cached with TTL-based eviction

### Identified Gaps

1. **Single-Vector Limitation**: Current retrieval uses a single query embedding, missing multi-faceted queries
2. **No Re-ranking**: Retrieved bullets are returned in similarity order without relevance re-ranking
3. **Static Query Building**: Query composition doesn't adapt based on retrieval quality
4. **Missing Hybrid Search**: No combination of semantic + keyword search

### Context Rot Risks (per LangChain research)

- **Context Confusion**: Irrelevant bullets may pollute the context
- **Context Distraction**: Too many similar bullets may overshadow critical information

## Proposed Solution

### 1. Multi-Query Retrieval

Decompose complex queries into multiple sub-queries for comprehensive retrieval.

```go
// internal/ace/retrieval/multi_query.go

type MultiQueryRetriever struct {
    base      Retriever
    decomposer QueryDecomposer
    merger    ResultMerger
}

type QueryDecomposer interface {
    // Decompose breaks a complex query into focused sub-queries
    Decompose(ctx context.Context, query string) ([]string, error)
}

func (r *MultiQueryRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error) {
    // 1. Decompose query into sub-queries
    subQueries, err := r.decomposer.Decompose(ctx, query)
    if err != nil {
        // Fallback to single query
        return r.base.Retrieve(ctx, query, topK)
    }
    
    // 2. Retrieve for each sub-query in parallel
    results := make([][]*bullet.Bullet, len(subQueries))
    g, gctx := errgroup.WithContext(ctx)
    
    for i, sq := range subQueries {
        i, sq := i, sq
        g.Go(func() error {
            bullets, err := r.base.Retrieve(gctx, sq, topK)
            if err != nil {
                return err
            }
            results[i] = bullets
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    // 3. Merge and deduplicate results
    return r.merger.Merge(results, topK)
}
```

### 2. Hybrid Search (Semantic + Keyword)

Combine embedding-based retrieval with BM25/keyword matching.

```go
// internal/ace/retrieval/hybrid.go

type HybridRetriever struct {
    semantic  Retriever
    keyword   KeywordRetriever
    alpha     float64 // Weighting factor (0=keyword, 1=semantic)
}

type KeywordRetriever interface {
    Search(ctx context.Context, query string, topK int) ([]ScoredBullet, error)
}

func (r *HybridRetriever) RetrieveWithScores(ctx context.Context, query string, topK int) ([]ScoredBullet, error) {
    // Parallel retrieval
    var semanticResults, keywordResults []ScoredBullet
    g, gctx := errgroup.WithContext(ctx)
    
    g.Go(func() error {
        var err error
        semanticResults, err = r.semantic.RetrieveWithScores(gctx, query, topK*2)
        return err
    })
    
    g.Go(func() error {
        var err error
        keywordResults, err = r.keyword.Search(gctx, query, topK*2)
        return err
    })
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    // Reciprocal Rank Fusion
    return r.fuseResults(semanticResults, keywordResults, topK), nil
}

func (r *HybridRetriever) fuseResults(semantic, keyword []ScoredBullet, topK int) []ScoredBullet {
    scores := make(map[string]float64)
    bullets := make(map[string]*bullet.Bullet)
    
    k := 60.0 // RRF constant
    
    for rank, sb := range semantic {
        scores[sb.Bullet.ID] += r.alpha * (1.0 / (k + float64(rank)))
        bullets[sb.Bullet.ID] = sb.Bullet
    }
    
    for rank, sb := range keyword {
        scores[sb.Bullet.ID] += (1 - r.alpha) * (1.0 / (k + float64(rank)))
        bullets[sb.Bullet.ID] = sb.Bullet
    }
    
    // Sort by fused score and return top K
    // ...
}
```

### 3. Cross-Encoder Re-ranking

Apply a more accurate (but slower) cross-encoder model to re-rank top candidates.

```go
// internal/ace/retrieval/reranker.go

type CrossEncoderReranker struct {
    encoder CrossEncoder
    topN    int // Number of candidates to re-rank
}

type CrossEncoder interface {
    // Score returns relevance score for query-document pair
    Score(ctx context.Context, query string, document string) (float64, error)
    
    // BatchScore scores multiple documents efficiently
    BatchScore(ctx context.Context, query string, documents []string) ([]float64, error)
}

func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, candidates []ScoredBullet) ([]ScoredBullet, error) {
    if len(candidates) <= 1 {
        return candidates, nil
    }
    
    // Limit candidates to re-rank
    toRerank := candidates
    if len(toRerank) > r.topN {
        toRerank = toRerank[:r.topN]
    }
    
    // Extract documents
    docs := make([]string, len(toRerank))
    for i, sb := range toRerank {
        docs[i] = sb.Bullet.Content
    }
    
    // Batch score
    scores, err := r.encoder.BatchScore(ctx, query, docs)
    if err != nil {
        return candidates, nil // Fallback to original ranking
    }
    
    // Apply new scores and sort
    for i := range toRerank {
        toRerank[i].Score = scores[i]
    }
    
    sort.Slice(toRerank, func(i, j int) bool {
        return toRerank[i].Score > toRerank[j].Score
    })
    
    return toRerank, nil
}
```

### 4. Adaptive Query Building

Enhance query building based on retrieval feedback and trajectory patterns.

```go
// internal/agent/query_builder.go (enhanced)

type AdaptiveQueryBuilder struct {
    base           QueryBuilder
    feedbackStore  RetrievalFeedbackStore
    patternMatcher PatternMatcher
}

type QueryComponent struct {
    Content  string
    Weight   float64
    Source   string // "user", "error", "tool", "pattern"
}

func (b *AdaptiveQueryBuilder) BuildQuery(tc *trajectory.Context) string {
    components := []QueryComponent{}
    
    // Base user query (highest weight)
    components = append(components, QueryComponent{
        Content: tc.Query,
        Weight:  1.0,
        Source:  "user",
    })
    
    // Error context (if recent errors)
    if errors := tc.RecentErrors(5); len(errors) > 0 {
        errorSummary := b.summarizeErrors(errors)
        components = append(components, QueryComponent{
            Content: errorSummary,
            Weight:  0.8,
            Source:  "error",
        })
    }
    
    // Tool usage patterns
    if patterns := b.patternMatcher.Match(tc.Steps); len(patterns) > 0 {
        for _, p := range patterns {
            components = append(components, QueryComponent{
                Content: p.Description,
                Weight:  0.5,
                Source:  "pattern",
            })
        }
    }
    
    // Adjust weights based on historical retrieval feedback
    components = b.adjustWeights(components, tc.SessionID)
    
    return b.composeQuery(components)
}
```

### 5. Retrieval Quality Tracking

Track and learn from retrieval outcomes.

```go
// internal/ace/retrieval/feedback.go

type RetrievalFeedback struct {
    SessionID    string
    Query        string
    RetrievedIDs []string
    UsedIDs      []string    // Bullets actually referenced in response
    Outcome      Outcome     // Success, Partial, Failure
    Latency      time.Duration
    Timestamp    time.Time
}

type RetrievalFeedbackStore interface {
    Record(ctx context.Context, feedback RetrievalFeedback) error
    GetPatterns(ctx context.Context, query string) ([]QueryPattern, error)
    GetBulletUtility(ctx context.Context, bulletID string) (float64, error)
}

// Used to adjust retrieval parameters
type QueryPattern struct {
    QueryType       string   // "error_resolution", "code_generation", etc.
    EffectiveTopK   int
    EffectiveMinScore float64
    SuccessfulTags  []string
}
```

## Configuration

```yaml
context:
  ace:
    retrieval:
      # Existing settings
      top_k: 10
      min_score: 0.5
      
      # New RAG enhancements
      rag:
        enabled: true
        
        multi_query:
          enabled: true
          max_sub_queries: 3
          
        hybrid_search:
          enabled: true
          alpha: 0.7  # Semantic weight
          
        reranking:
          enabled: true
          model: "cross-encoder/ms-marco-MiniLM-L-6-v2"
          top_n: 20
          
        adaptive_query:
          enabled: true
          error_weight: 0.8
          pattern_weight: 0.5
          
        feedback:
          enabled: true
          retention_days: 30
```

## Implementation Plan

### Phase 1: Foundation (Week 1-2)
1. Implement `KeywordRetriever` using simple inverted index
2. Implement `HybridRetriever` with RRF fusion
3. Add retrieval feedback recording
4. Write comprehensive tests

### Phase 2: Multi-Query (Week 3)
1. Implement `QueryDecomposer` using LLM
2. Implement `MultiQueryRetriever`
3. Add parallel retrieval with timeout
4. Benchmark performance impact

### Phase 3: Re-ranking (Week 4)
1. Integrate cross-encoder model (optional Ollama model)
2. Implement `CrossEncoderReranker`
3. Add fallback when model unavailable
4. Measure accuracy improvements

### Phase 4: Adaptive Learning (Week 5-6)
1. Implement `RetrievalFeedbackStore`
2. Enhance `QueryBuilder` with adaptation
3. Add pattern detection
4. Create feedback loop integration

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Retrieval Precision@5 | ~60% | 80%+ |
| Query Latency (P50) | 50ms | <100ms |
| Query Latency (P99) | 200ms | <500ms |
| Bullet Utilization | Unknown | 70%+ |
| Context Confusion Rate | Unknown | <10% |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Increased latency | High | Parallel retrieval, caching, configurable features |
| LLM dependency for decomposition | Medium | Simple rule-based fallback |
| Cross-encoder model availability | Medium | Optional feature, graceful degradation |
| Storage growth for feedback | Low | TTL-based cleanup, configurable retention |

## Alternatives Considered

1. **Pure keyword search**: Rejected - loses semantic understanding
2. **Single dense retriever**: Current state - insufficient for complex queries
3. **Full LLM-based retrieval**: Rejected - too slow and expensive for real-time use

## References

- [LangChain RAG Implementation](https://github.com/langchain-ai/how_to_fix_your_context/blob/main/notebooks/1_rag.ipynb)
- [Reciprocal Rank Fusion](https://plg.uwaterloo.ca/~gvcormac/cormacksigir09-rrf.pdf)
- [Cross-Encoder Re-ranking](https://www.sbert.net/examples/applications/cross-encoder/README.html)
