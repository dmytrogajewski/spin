# Proposal: Dynamic Tool Loadout

**ID**: PROP-CONTEXT-002  
**Title**: Dynamic Tool Selection Based on Query Context  
**Status**: Draft  
**Created**: 2025-01-20  
**Author**: AI Assistant  
**References**: [LangChain Context Engineering](https://github.com/langchain-ai/how_to_fix_your_context)

## Summary

Implement dynamic tool selection that provides the LLM with only the most relevant tools for each query, reducing context confusion and improving tool selection accuracy.

## Problem Statement

### Current State

Spin currently exposes all registered tools to the LLM in every request:
- Built-in tools: `read_file`, `write_file`, `shell_command`, `apply_patch`, `file_search`, `git_operation`, `get_context`
- MCP tools: Dynamically loaded from external servers
- All tool definitions are included in every LLM prompt

### Identified Issues

1. **Context Bloat**: Tool definitions consume significant tokens (~500-1000 tokens per tool)
2. **Tool Confusion**: Similar tools (e.g., `write_file` vs `apply_patch`) may confuse the model
3. **Irrelevant Options**: Git tools are irrelevant for non-git operations
4. **MCP Scaling**: As MCP tools grow, context overhead increases linearly

### Context Rot Risks

- **Context Confusion**: Too many tool options degrade selection quality
- **Context Distraction**: Verbose tool descriptions overshadow the actual task

## Proposed Solution

### 1. Tool Categorization and Tagging

Organize tools into semantic categories with searchable tags.

```go
// internal/tools/category.go

type ToolCategory string

const (
    CategoryFileRead    ToolCategory = "file_read"
    CategoryFileWrite   ToolCategory = "file_write"
    CategorySearch      ToolCategory = "search"
    CategoryGit         ToolCategory = "git"
    CategoryShell       ToolCategory = "shell"
    CategoryContext     ToolCategory = "context"
    CategoryExternal    ToolCategory = "external"
)

type ToolMetadata struct {
    Name        string
    Description string
    Category    ToolCategory
    Tags        []string           // Semantic tags for search
    Embedding   []float32          // Pre-computed embedding
    UsageHints  []string           // When to use this tool
    Conflicts   []string           // Tools that shouldn't be used together
    Requires    []string           // Tools that should be available if this is
}

// Example metadata
var ReadFileMetadata = ToolMetadata{
    Name:        "read_file",
    Description: "Read contents of a file",
    Category:    CategoryFileRead,
    Tags:        []string{"file", "read", "content", "view", "inspect", "open"},
    UsageHints:  []string{"viewing file contents", "reading configuration", "inspecting code"},
    Conflicts:   []string{},
    Requires:    []string{},
}
```

### 2. Tool Selector Interface

Define the interface for dynamic tool selection.

```go
// internal/tools/selector.go

type ToolSelector interface {
    // SelectTools returns the most relevant tools for a given query/context
    SelectTools(ctx context.Context, query string, maxTools int) ([]Tool, error)
    
    // SelectToolsWithContext uses trajectory context for better selection
    SelectToolsWithContext(ctx context.Context, tc *trajectory.Context, maxTools int) ([]Tool, error)
}

type SelectionResult struct {
    Tools       []Tool
    Scores      map[string]float64  // Tool name -> relevance score
    Categories  []ToolCategory      // Selected categories
    Reasoning   string              // Why these tools were selected
}
```

### 3. Semantic Tool Selector

Implement selection based on embedding similarity.

```go
// internal/tools/semantic_selector.go

type SemanticToolSelector struct {
    registry    *Registry
    embedder    embedder.Embedder
    metadata    map[string]ToolMetadata
    index       *hnsw.Index  // Pre-built index of tool embeddings
}

func (s *SemanticToolSelector) SelectTools(ctx context.Context, query string, maxTools int) ([]Tool, error) {
    // 1. Embed the query
    queryEmb, err := s.embedder.Embed(ctx, query)
    if err != nil {
        return s.fallbackSelection(maxTools)
    }
    
    // 2. Search for similar tools
    neighbors, err := s.index.SearchKNN(queryEmb, maxTools*2)
    if err != nil {
        return s.fallbackSelection(maxTools)
    }
    
    // 3. Apply category constraints
    selected := s.applyCategoryConstraints(neighbors, maxTools)
    
    // 4. Add required dependencies
    selected = s.addRequiredTools(selected)
    
    // 5. Convert to Tool instances
    return s.resolveTools(selected)
}

func (s *SemanticToolSelector) applyCategoryConstraints(candidates []string, maxTools int) []string {
    // Ensure diversity: at least one tool per relevant category
    categorySelected := make(map[ToolCategory]bool)
    result := []string{}
    
    for _, toolName := range candidates {
        if len(result) >= maxTools {
            break
        }
        
        meta := s.metadata[toolName]
        
        // Prioritize first tool from each category
        if !categorySelected[meta.Category] {
            result = append(result, toolName)
            categorySelected[meta.Category] = true
            continue
        }
        
        // Add additional tools if space permits
        result = append(result, toolName)
    }
    
    return result
}
```

### 4. Context-Aware Tool Selection

Enhance selection using trajectory context.

```go
// internal/tools/context_selector.go

type ContextAwareSelector struct {
    base        ToolSelector
    analyzer    TrajectoryAnalyzer
}

func (s *ContextAwareSelector) SelectToolsWithContext(
    ctx context.Context, 
    tc *trajectory.Context, 
    maxTools int,
) ([]Tool, error) {
    // 1. Analyze trajectory for tool usage patterns
    analysis := s.analyzer.Analyze(tc)
    
    // 2. Build enhanced query
    query := s.buildEnhancedQuery(tc.Query, analysis)
    
    // 3. Get base selection
    tools, err := s.base.SelectTools(ctx, query, maxTools)
    if err != nil {
        return nil, err
    }
    
    // 4. Boost recently successful tools
    tools = s.boostSuccessfulTools(tools, analysis.SuccessfulTools)
    
    // 5. Include tools used in recent steps (continuity)
    tools = s.ensureContinuity(tools, analysis.RecentTools, maxTools)
    
    return tools, nil
}

type TrajectoryAnalysis struct {
    SuccessfulTools []string          // Tools that led to good outcomes
    FailedTools     []string          // Tools that caused errors
    RecentTools     []string          // Tools used in last N steps
    Patterns        []UsagePattern    // Detected tool usage patterns
    DominantCategory ToolCategory     // Most used category
}

type UsagePattern struct {
    Sequence    []string  // Tool sequence (e.g., read_file -> apply_patch)
    Frequency   int
    SuccessRate float64
}
```

### 5. Core Tool Set (Always Included)

Define a minimal set of tools always available.

```go
// internal/tools/core_set.go

// CoreTools are always included regardless of selection
var CoreTools = []string{
    "read_file",      // Fundamental for understanding context
    "shell_command",  // General-purpose execution
}

// CategoryCoreTools are included when their category is relevant
var CategoryCoreTools = map[ToolCategory][]string{
    CategoryFileWrite: {"write_file", "apply_patch"},
    CategoryGit:       {"git_operation"},
    CategorySearch:    {"file_search"},
    CategoryContext:   {"get_context"},
}
```

### 6. Integration with Agent Loop

Modify the agent to use dynamic tool selection.

```go
// internal/agent/agent.go (modified)

func (a *Agent) buildToolDefinitions(ctx context.Context, tc *trajectory.Context) ([]llm.Tool, error) {
    if !a.config.DynamicToolSelection.Enabled {
        // Fallback to all tools
        return a.registry.AllToolDefinitions(), nil
    }
    
    // Select relevant tools
    maxTools := a.config.DynamicToolSelection.MaxTools
    tools, err := a.toolSelector.SelectToolsWithContext(ctx, tc, maxTools)
    if err != nil {
        a.logger.Warn("tool selection failed, using all tools", "error", err)
        return a.registry.AllToolDefinitions(), nil
    }
    
    // Convert to LLM tool definitions
    definitions := make([]llm.Tool, len(tools))
    for i, tool := range tools {
        definitions[i] = tool.Definition()
    }
    
    // Log selection for observability
    a.emitToolSelectionEvent(tools)
    
    return definitions, nil
}
```

### 7. Tool Selection Events

Add observability for tool selection.

```go
// internal/events/tool_selection.go

type ToolSelectionEvent struct {
    Timestamp       time.Time
    Query           string
    SelectedTools   []string
    Scores          map[string]float64
    TotalAvailable  int
    SelectionMethod string  // "semantic", "context", "fallback"
    Latency         time.Duration
}
```

## Configuration

```yaml
tools:
  dynamic_selection:
    enabled: true
    max_tools: 10           # Maximum tools to include
    min_score: 0.3          # Minimum relevance score
    
    core_tools:             # Always included
      - read_file
      - shell_command
    
    category_budgets:       # Max tools per category
      file_read: 2
      file_write: 3
      search: 2
      git: 2
      shell: 1
      external: 5
    
    continuity:
      enabled: true
      lookback_steps: 3     # Consider tools from last N steps
      boost_factor: 1.2     # Boost for recently used tools
    
    caching:
      enabled: true
      ttl: 5m               # Cache selection for similar queries
```

## Implementation Plan

### Phase 1: Foundation (Week 1)
1. Define `ToolMetadata` structure
2. Add metadata to all existing tools
3. Implement `ToolSelector` interface
4. Create simple keyword-based selector as baseline

### Phase 2: Semantic Selection (Week 2)
1. Pre-compute tool embeddings
2. Build HNSW index for tools
3. Implement `SemanticToolSelector`
4. Add caching layer

### Phase 3: Context-Aware Selection (Week 3)
1. Implement `TrajectoryAnalyzer`
2. Build `ContextAwareSelector`
3. Add continuity logic
4. Integrate with agent loop

### Phase 4: Optimization (Week 4)
1. Add selection events/metrics
2. Implement A/B testing framework
3. Tune thresholds based on data
4. Document best practices

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Tool Definition Tokens | ~3000 | <1500 |
| Tool Selection Accuracy | N/A | 90%+ |
| Tool-Related Errors | Unknown | -30% |
| Selection Latency | N/A | <50ms |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Missing critical tool | High | Core tool set always included |
| Selection latency | Medium | Caching, pre-computed embeddings |
| Poor semantic matching | Medium | Fallback to all tools |
| MCP tool metadata | Low | Default metadata generation |

## Examples

### Example 1: Git-focused Query

**Query**: "Create a new branch and commit these changes"

**Selected Tools**:
1. `git_operation` (score: 0.95) - Direct match
2. `shell_command` (score: 0.70) - Core tool
3. `read_file` (score: 0.65) - Core tool

**Excluded**: `file_search`, `apply_patch`, `write_file` (irrelevant)

### Example 2: Code Modification Query

**Query**: "Fix the type error in the User model"

**Selected Tools**:
1. `read_file` (score: 0.90) - Need to see code
2. `file_search` (score: 0.85) - Find the file
3. `apply_patch` (score: 0.80) - Make changes
4. `shell_command` (score: 0.65) - Run type checker

**Excluded**: `git_operation` (not requested)

### Example 3: Trajectory Continuity

**Previous Steps**: Used `read_file`, `file_search`
**Query**: "Now fix it"

**Selected Tools** (boosted for continuity):
1. `apply_patch` (score: 0.90) - Logical next step
2. `read_file` (score: 0.85, boosted) - Recently used
3. `write_file` (score: 0.80) - Alternative fix method
4. `shell_command` (score: 0.70) - Verify fix

## References

- [LangChain Tool Loadout](https://github.com/langchain-ai/how_to_fix_your_context/blob/main/notebooks/2_tool_loadout.ipynb)
- [Dynamic Tool Selection in LLM Agents](https://arxiv.org/abs/2305.14318)
