# FRD-20251012113756: Phase 5.1 - Core Integration

**Feature:** Core Integration of Tools & Utility Modules
**Created:** 2025-10-12
**Status:** Planning
**Priority:** P0 (Production Blocker)
**Estimated Effort:** 2-3 days

---

## Overview

Integrate the newly implemented tool and utility modules (`patchapply`, `filesearch`, `git`) into Spin's core tool registry, making them available to the AI agent through the LLM function calling interface.

## Background

Phase 1-4 delivered:
- ✅ Foundation packages: `pkg/pathutil`, `pkg/strutil`, `pkg/ansi`
- ✅ Patch application: `internal/patchapply` (Parser, Matcher, Applier)
- ✅ Enhanced file search: `internal/filesearch` (Scanner, Matcher, IgnoreHandler, Searcher)
- ✅ Git operations: `internal/git` (Repository operations, patch application)

These packages are production-ready with excellent test coverage (84-97%) but are not yet integrated into the core tool registry. This feature makes them accessible to the AI agent.

## Goals

1. **Primary Goal**: Integrate new packages into `internal/tools` registry
2. **Secondary Goal**: Maintain backward compatibility with existing tools
3. **Tertiary Goal**: Provide comprehensive error handling and user feedback

## Non-Goals

- E2E testing with real AI interactions (Phase 5.3)
- Additional documentation (Phase 5.2)
- CLI tools for filesearch (Feature 3.3, P2)
- Performance optimization beyond existing implementations

## Requirements

### Functional Requirements

#### FR-1: Apply Patch Tool

**Description**: Tool that allows the AI agent to apply structured patches to files.

**Tool Schema**:
```json
{
  "name": "apply_patch",
  "description": "Apply a structured patch to modify files in the workspace",
  "parameters": {
    "type": "object",
    "properties": {
      "patch_text": {
        "type": "string",
        "description": "The patch text in Spin's patch format"
      },
      "workspace_root": {
        "type": "string",
        "description": "The workspace root directory (optional, defaults to current workspace)"
      },
      "dry_run": {
        "type": "boolean",
        "description": "If true, validate without applying changes"
      },
      "force": {
        "type": "boolean",
        "description": "If true, allow overwriting existing files on Add operations"
      }
    },
    "required": ["patch_text"]
  }
}
```

**Success Criteria**:
- Parses patch text using `patchapply.Parser`
- Validates paths using `pkg/pathutil`
- Applies patches using `patchapply.Applier`
- Returns structured result with files modified/created/deleted
- Handles errors gracefully with clear messages

#### FR-2: File Search Tool

**Description**: Tool that allows the AI agent to search for files using fuzzy matching with intelligent ranking.

**Tool Schema**:
```json
{
  "name": "file_search",
  "description": "Search for files in the workspace using fuzzy matching with .gitignore support",
  "parameters": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "The search query (fuzzy matched against file paths)"
      },
      "workspace_root": {
        "type": "string",
        "description": "The workspace root directory (optional, defaults to current workspace)"
      },
      "limit": {
        "type": "integer",
        "description": "Maximum number of results to return (default: 10)"
      }
    },
    "required": ["query"]
  }
}
```

**Success Criteria**:
- Uses `filesearch.Searcher` for async indexing and search
- Respects .gitignore and .spinignore patterns
- Returns ranked results using 7-tier scoring algorithm
- Handles large workspaces efficiently (<100ms for 10k files)
- Clear error messages for filesystem issues

#### FR-3: Git Context Tool

**Description**: Tool that provides Git repository context to the AI agent.

**Tool Schema**:
```json
{
  "name": "git_context",
  "description": "Get Git repository context including branch, status, and modifications",
  "parameters": {
    "type": "object",
    "properties": {
      "workspace_root": {
        "type": "string",
        "description": "The workspace root directory (optional, defaults to current workspace)"
      },
      "include_diff": {
        "type": "boolean",
        "description": "If true, include diff summary (default: false)"
      }
    },
    "required": []
  }
}
```

**Success Criteria**:
- Uses `git.Discover` to find repository
- Returns branch name, tracking info, modified/staged/untracked files
- Optionally includes diff summary
- Handles non-repository directories gracefully
- Fast response (<200ms for typical repositories)

### Non-Functional Requirements

#### NFR-1: Performance

- Apply patch: <1s for typical patches (<1000 lines)
- File search: <100ms for 10k files after indexing
- Git context: <200ms for typical repositories
- Minimal memory overhead (<10MB per tool instance)

#### NFR-2: Reliability

- All tools must handle errors gracefully without panics
- Clear, actionable error messages for all failure modes
- Context cancellation support for long-running operations
- Thread-safe concurrent tool execution

#### NFR-3: Test Coverage

- Tool implementations: ≥90% coverage
- Integration with registry: 100% coverage
- Error paths: 100% coverage
- Edge cases: comprehensive coverage

#### NFR-4: Code Quality

- `make lint` passes with zero errors
- Cyclomatic complexity ≤15 per function
- Comprehensive godoc comments
- Follows existing tool patterns in `builtin.go`

## Design

### Architecture

```
internal/tools/
├── types.go              (unchanged)
├── registry.go           (unchanged)
├── builtin.go            (ADD new tools here)
│   ├── ApplyPatchTool    ← NEW
│   ├── FileSearchTool    ← NEW
│   └── GitContextTool    ← NEW
└── builtin_test.go       (ADD tests here)
```

### Data Flow

```
AI Agent → Tool Call → Registry → ApplyPatchTool
                                     ↓
                                patchapply.Parser
                                     ↓
                                patchapply.Applier
                                     ↓
                                Filesystem
                                     ↓
                                ToolResult → AI Agent
```

### Tool Implementations

#### ApplyPatchTool

```go
type ApplyPatchTool struct {
    workspaceRoot string
}

func NewApplyPatchTool(workspaceRoot string) *ApplyPatchTool
func (t *ApplyPatchTool) Name() string // "apply_patch"
func (t *ApplyPatchTool) Description() string
func (t *ApplyPatchTool) Schema() ToolSchema
func (t *ApplyPatchTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
```

**Execute Logic**:
1. Extract parameters: `patch_text`, `workspace_root`, `dry_run`, `force`
2. Parse patch: `parser := patchapply.NewParser(patchText); patch, err := parser.Parse()`
3. Create applier: `applier, err := patchapply.NewApplier(workspaceRoot)`
4. Configure: `applier.SetDryRun(dryRun); applier.SetForceOverwrite(force)`
5. Apply: `result, err := applier.Apply(patch)`
6. Format result: Return structured output with files affected

#### FileSearchTool

```go
type FileSearchTool struct {
    workspaceRoot string
    searcher      *filesearch.Searcher
    mu            sync.RWMutex
}

func NewFileSearchTool(workspaceRoot string) *FileSearchTool
func (t *FileSearchTool) Name() string // "file_search"
func (t *FileSearchTool) Description() string
func (t *FileSearchTool) Schema() ToolSchema
func (t *FileSearchTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
```

**Execute Logic**:
1. Extract parameters: `query`, `workspace_root`, `limit`
2. Lazy-init searcher: `t.getOrCreateSearcher(workspaceRoot)`
3. Index if needed: `searcher.IndexAsync(ctx)`
4. Search: `matches := searcher.Search(query, limit)`
5. Format results: Return ranked file paths with scores

#### GitContextTool

```go
type GitContextTool struct {
    workspaceRoot string
}

func NewGitContextTool(workspaceRoot string) *GitContextTool
func (t *GitContextTool) Name() string // "git_context"
func (t *GitContextTool) Description() string
func (t *GitContextTool) Schema() ToolSchema
func (t *GitContextTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
```

**Execute Logic**:
1. Extract parameters: `workspace_root`, `include_diff`
2. Discover repository: `repo, err := git.Discover(ctx, workspaceRoot)`
3. Get status: `status, err := repo.Status(ctx)`
4. Optionally get diff: if `include_diff`, compute diff summary
5. Format output: Structured text with branch, files, tracking info

## Error Handling

### Error Categories

| Error Type | Handling | Example |
|------------|----------|---------|
| Parse Error | Return ToolResult with Error field | "line 5: invalid path '/etc/passwd'" |
| Path Error | Return ToolResult with Error field | "path outside workspace: ../../etc" |
| Filesystem Error | Return ToolResult with Error field | "file not found: src/main.go" |
| Git Error | Return ToolResult with Error field | "not a git repository" |
| Cancellation | Return context.Canceled | User cancels long operation |

### Error Messages

All error messages must be:
- **Clear**: Explain what went wrong
- **Actionable**: Suggest how to fix it
- **Specific**: Include relevant details (line numbers, paths, etc.)

**Examples**:
```
✅ GOOD: "Parse error at line 12: expected '*** End Patch' but found '*** EndPatch'"
❌ BAD:  "parse error"

✅ GOOD: "File not found: 'src/handler.go'. Did you mean 'src/handlers.go'?"
❌ BAD:  "file error"

✅ GOOD: "Not a Git repository. Initialize with 'git init' first."
❌ BAD:  "git error"
```

## Testing Strategy

### Unit Tests

Each tool must have comprehensive unit tests covering:

1. **Happy Path**:
   - Valid inputs return expected results
   - All optional parameters work correctly
   - Default values are applied

2. **Error Cases**:
   - Missing required parameters
   - Invalid parameter types
   - Malformed input (patches, queries, etc.)
   - Filesystem errors (permissions, not found, etc.)

3. **Edge Cases**:
   - Empty inputs
   - Very large inputs (10k line patches, 100k files)
   - Special characters in paths
   - Unicode content

4. **Context Cancellation**:
   - Long-running operations respect context cancellation
   - Partial results handled correctly

### Integration Tests

Test integration with Registry:

1. **Registration**:
   - Tools register without errors
   - Schemas are valid
   - No naming conflicts

2. **Execution**:
   - Registry.Execute() calls tool correctly
   - Parameters validated before execution
   - Results returned correctly

3. **Concurrent Access**:
   - Multiple tools can execute concurrently
   - No race conditions (run with `-race`)

### Test Coverage Targets

- ApplyPatchTool: ≥90%
- FileSearchTool: ≥90%
- GitContextTool: ≥90%
- Integration tests: 100%

## Acceptance Criteria

### Definition of Done

- [ ] All three tools implemented in `internal/tools/builtin.go`
- [ ] Comprehensive tests in `internal/tools/builtin_test.go`
- [ ] All tests pass: `go test ./internal/tools/...`
- [ ] Race detector clean: `go test -race ./internal/tools/...`
- [ ] Test coverage ≥90%: `go test -cover ./internal/tools/...`
- [ ] `make lint` passes with zero errors
- [ ] Complexity ≤15: `gocyclo -over 15 ./internal/tools/`
- [ ] Code analysis clean: `uast parse | herr analyze` (at least YELLOW)
- [ ] Godoc complete for all exported functions
- [ ] Integration with existing tools verified (no regressions)

### Acceptance Tests

#### AT-1: Apply Patch Tool

```go
// Given a valid patch
patch := `*** Begin Patch
*** Add File: test.txt
+Hello, World!
*** End Patch`

// When the tool is executed
result, err := tool.Execute(ctx, map[string]interface{}{
    "patch_text": patch,
    "workspace_root": tmpDir,
})

// Then the tool succeeds
assert.NoError(t, err)
assert.True(t, result.Success)
assert.Contains(t, result.Output, "test.txt")

// And the file is created
assert.FileExists(t, filepath.Join(tmpDir, "test.txt"))
```

#### AT-2: File Search Tool

```go
// Given a workspace with files
// When searching for "test"
result, err := tool.Execute(ctx, map[string]interface{}{
    "query": "test",
    "limit": 5,
})

// Then the tool returns ranked results
assert.NoError(t, err)
assert.True(t, result.Success)
assert.Contains(t, result.Output, "test_")
```

#### AT-3: Git Context Tool

```go
// Given a Git repository
// When getting context
result, err := tool.Execute(ctx, map[string]interface{}{
    "workspace_root": repoDir,
})

// Then the tool returns branch and status
assert.NoError(t, err)
assert.True(t, result.Success)
assert.Contains(t, result.Output, "Branch:")
assert.Contains(t, result.Output, "main")
```

## Implementation Plan

### Phase 1: ApplyPatchTool (Day 1)

1. ✅ Write FRD
2. Implement `ApplyPatchTool` struct and methods
3. Write comprehensive tests
4. Run lint and analysis
5. Iterate until all quality gates pass

### Phase 2: FileSearchTool (Day 1-2)

1. Implement `FileSearchTool` struct and methods
2. Add lazy initialization and caching logic
3. Write comprehensive tests
4. Run lint and analysis
5. Iterate until all quality gates pass

### Phase 3: GitContextTool (Day 2)

1. Implement `GitContextTool` struct and methods
2. Write comprehensive tests
3. Run lint and analysis
4. Iterate until all quality gates pass

### Phase 4: Integration and Polish (Day 2-3)

1. Test all tools together with Registry
2. Verify no regressions in existing tools
3. Run full test suite with race detector
4. Final lint and analysis pass
5. Update roadmap
6. Update package documentation (brief)

## Dependencies

### Internal Dependencies

- `internal/patchapply` - Parser, Matcher, Applier
- `internal/filesearch` - Scanner, Matcher, Searcher, IgnoreHandler
- `internal/git` - Discover, Repository, Status
- `pkg/pathutil` - Path validation
- `pkg/strutil` - String utilities
- `internal/tools` - Registry, types

### External Dependencies

None (all dependencies already integrated in Phase 1-4)

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Circular import with core | High | Low | Use interface{} pattern like ExecuteCommandTool |
| Workspace root not available | Medium | Medium | Provide sensible defaults, allow override |
| Large file search overhead | Medium | Low | Lazy initialization, async indexing, caching |
| Git operations fail in non-repo | Low | High | Graceful error handling, clear messages |
| Tool registration conflicts | Low | Low | Unique tool names, comprehensive tests |

## Open Questions

1. **Q**: Should file search tool maintain a persistent index across tool calls?
   **A**: Yes, use lazy initialization with caching per workspace root.

2. **Q**: Should git context tool cache repository status?
   **A**: No, status changes frequently. Always fetch fresh.

3. **Q**: How to handle workspace root when not provided?
   **A**: Use current working directory as default. Tools should accept optional override.

4. **Q**: Should tools be registered automatically or manually?
   **A**: Manual registration in core package initialization (existing pattern).

## References

- [Tools & Utility Modules Spec](../tools-modules/tools-modules.md)
- [Tools-Modules ROADMAP](../tools-modules/ROADMAP.md)
- [patchapply Documentation](../../docs/packages/patchapply.md)
- [filesearch Documentation](../../docs/packages/filesearch.md)
- [git Documentation](../../docs/packages/git.md)
- [tools Package Documentation](../../docs/packages/tools.md)
- [AGENTS.md Workflow](../../AGENTS.md)

---

**Last Updated:** 2025-10-12
**Author:** Spin Development Team
**Status:** Ready for Implementation
