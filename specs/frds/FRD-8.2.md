# FRD-8.2: Tool Registry Integration

**Feature:** Tool Registry Integration
**Package:** `internal/core` + `internal/tools`
**Status:** Draft
**Version:** 1.0
**Last Updated:** 2025-10-04

---

## 1. Overview

### 1.1 Purpose
Integrate the core module with a dedicated `internal/tools` registry for centralized tool management, schema retrieval, and execution coordination.

### 1.2 Scope
- Define the `internal/tools` package with Tool interface and Registry
- Implement tool schema retrieval for LLM function calling
- Coordinate tool execution through the registry
- Implement tool parameter validation
- Support custom tool registration
- Filter tools by task mode restrictions

### 1.3 Dependencies
- Feature 6.2: Tool Call Processing (completed)
- `internal/llm` package (for tool schema format compatibility)

---

## 2. Functional Requirements

### 2.1 Tool Interface

**FR-8.2.1:** Tool interface SHALL define the contract for all tools:
```go
type Tool interface {
    Name() string
    Description() string
    Schema() ToolSchema
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}
```

**FR-8.2.2:** ToolSchema SHALL be compatible with OpenAI function calling format:
```go
type ToolSchema struct {
    Type       string                 `json:"type"`        // "function"
    Function   FunctionSchema         `json:"function"`
}

type FunctionSchema struct {
    Name        string                `json:"name"`
    Description string                `json:"description"`
    Parameters  ParameterSchema       `json:"parameters"`
}

type ParameterSchema struct {
    Type       string                          `json:"type"`        // "object"
    Properties map[string]PropertyDefinition   `json:"properties"`
    Required   []string                        `json:"required,omitempty"`
}

type PropertyDefinition struct {
    Type        string   `json:"type"`
    Description string   `json:"description"`
    Enum        []string `json:"enum,omitempty"`
}
```

**FR-8.2.3:** ToolResult SHALL include:
- Success: boolean indicating success/failure
- Output: string output for LLM
- Error: optional error message

### 2.2 Tool Registry

**FR-8.2.4:** Registry SHALL manage tool lifecycle:
```go
type Registry struct {
    // internal fields
}

func NewRegistry() *Registry
func (r *Registry) Register(tool Tool) error
func (r *Registry) Get(name string) (Tool, error)
func (r *Registry) List() []Tool
func (r *Registry) ListSchemas() []ToolSchema
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (ToolResult, error)
```

**FR-8.2.5:** Registry SHALL prevent duplicate tool registration (error on duplicate name)

**FR-8.2.6:** Registry SHALL be thread-safe for concurrent access

### 2.3 Tool Execution

**FR-8.2.7:** Registry.Execute() SHALL:
1. Look up tool by name
2. Validate parameters against schema
3. Execute tool with context
4. Return structured result

**FR-8.2.8:** Parameter validation SHALL check:
- All required parameters present
- Parameter types match schema
- Enum values valid (if applicable)

**FR-8.2.9:** Execution errors SHALL be wrapped with operation context

### 2.4 Core Integration

**FR-8.2.10:** Agent SHALL use Registry.ListSchemas() to provide tools to LLM

**FR-8.2.11:** Agent.ProcessToolCall() SHALL use Registry.Execute() for tool invocation

**FR-8.2.12:** Task modes SHALL filter tools using Registry.List() with allowed tool names

### 2.5 Built-in Tools

**FR-8.2.13:** Registry SHALL include these built-in tools:
- `read_file`: Read file contents
- `write_file`: Write/create file
- `list_directory`: List directory contents
- `search_files`: Search for files by pattern
- `search_code`: Search code by regex
- `execute_command`: Execute shell command
- `get_context`: Get environment context

**FR-8.2.14:** Each built-in tool SHALL have comprehensive schema with descriptions

---

## 3. Non-Functional Requirements

### 3.1 Performance
- **NFR-8.2.1:** Tool lookup SHALL be O(1) using map-based storage
- **NFR-8.2.2:** Schema listing SHALL be cached to avoid repeated allocations

### 3.2 Reliability
- **NFR-8.2.3:** Registry SHALL gracefully handle tool execution failures
- **NFR-8.2.4:** Parameter validation SHALL prevent invalid tool calls

### 3.3 Security
- **NFR-8.2.5:** Tool execution SHALL respect context cancellation
- **NFR-8.2.6:** Command execution tool SHALL use existing Executor for safety

### 3.4 Code Quality
- **NFR-8.2.7:** Test coverage ≥90% for Registry
- **NFR-8.2.8:** Cyclomatic complexity ≤10 per function
- **NFR-8.2.9:** All exported symbols SHALL have godoc comments

---

## 4. Data Structures

### 4.1 Core Types

```go
// Tool defines the interface for all tools
type Tool interface {
    Name() string
    Description() string
    Schema() ToolSchema
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}

// ToolResult represents the result of tool execution
type ToolResult struct {
    Success bool   `json:"success"`
    Output  string `json:"output"`
    Error   string `json:"error,omitempty"`
}

// ToolSchema defines the OpenAI-compatible tool schema
type ToolSchema struct {
    Type     string         `json:"type"`
    Function FunctionSchema `json:"function"`
}

// FunctionSchema defines function metadata
type FunctionSchema struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  ParameterSchema `json:"parameters"`
}

// ParameterSchema defines function parameters
type ParameterSchema struct {
    Type       string                        `json:"type"`
    Properties map[string]PropertyDefinition `json:"properties"`
    Required   []string                      `json:"required,omitempty"`
}

// PropertyDefinition defines a parameter property
type PropertyDefinition struct {
    Type        string   `json:"type"`
    Description string   `json:"description"`
    Enum        []string `json:"enum,omitempty"`
}

// Registry manages tool registration and execution
type Registry struct {
    mu    sync.RWMutex
    tools map[string]Tool
}
```

### 4.2 Built-in Tool Implementations

Each built-in tool SHALL implement the Tool interface with proper schema definitions.

---

## 5. API Specification

### 5.1 Registry API

```go
// NewRegistry creates a new tool registry with built-in tools
func NewRegistry() *Registry

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, error)

// List returns all registered tools
func (r *Registry) List() []Tool

// ListSchemas returns schemas for all tools
func (r *Registry) ListSchemas() []ToolSchema

// Execute runs a tool by name with parameters
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (ToolResult, error)
```

### 5.2 Error Handling

```go
var (
    ErrToolNotFound      = errors.New("tool not found")
    ErrDuplicateTool     = errors.New("tool already registered")
    ErrInvalidParameters = errors.New("invalid tool parameters")
)
```

---

## 6. Integration Points

### 6.1 Agent Integration

**Current Implementation:**
```go
// Agent.ProcessToolCall() currently has inline tool execution
func (a *Agent) ProcessToolCall(ctx context.Context, toolCall llm.ToolCall) (llm.ToolResult, error) {
    // Inline implementation for read_file, write_file, etc.
}
```

**After Integration:**
```go
func (a *Agent) ProcessToolCall(ctx context.Context, toolCall llm.ToolCall) (llm.ToolResult, error) {
    result, err := a.toolRegistry.Execute(ctx, toolCall.Function.Name, params)
    // Convert tools.ToolResult to llm.ToolResult
}
```

### 6.2 Manager Integration

```go
// WithToolRegistry option for dependency injection
func WithToolRegistry(registry *tools.Registry) ManagerOption {
    return func(m *Manager) {
        m.toolRegistry = registry
    }
}
```

### 6.3 Task Mode Integration

Task modes SHALL filter tools by name:
```go
func (r *Regular) AllowedTools() []string {
    return []string{"read_file", "write_file", "list_directory", ...}
}

// In Agent initialization:
allowedTools := task.AllowedTools()
schemas := filterSchemasByName(registry.ListSchemas(), allowedTools)
```

---

## 7. Testing Requirements

### 7.1 Unit Tests

**UT-8.2.1:** Test tool registration (success and duplicate)
**UT-8.2.2:** Test tool retrieval (found and not found)
**UT-8.2.3:** Test tool listing and schema generation
**UT-8.2.4:** Test parameter validation (valid, missing required, wrong type)
**UT-8.2.5:** Test tool execution (success, failure, context cancellation)
**UT-8.2.6:** Test concurrent registry access (race detector clean)

### 7.2 Integration Tests

**IT-8.2.1:** Test Agent with Registry (tool call end-to-end)
**IT-8.2.2:** Test task mode filtering (only allowed tools available)
**IT-8.2.3:** Test custom tool registration and execution
**IT-8.2.4:** Test all built-in tools execute correctly

### 7.3 Coverage Target

- Registry core: ≥95%
- Built-in tools: ≥90%
- Integration: ≥85%

---

## 8. Implementation Plan

### 8.1 Phase 1: Registry Core (2 hours)
1. Create `internal/tools` package
2. Define Tool interface and related types
3. Implement Registry struct with map storage
4. Implement Register/Get/List/ListSchemas
5. Write registry unit tests

### 8.2 Phase 2: Built-in Tools (2 hours)
1. Implement ReadFileTool
2. Implement WriteFileTool
3. Implement ListDirectoryTool
4. Implement SearchFilesTool
5. Implement SearchCodeTool
6. Implement ExecuteCommandTool (delegate to core.Executor)
7. Implement GetContextTool
8. Write tool unit tests

### 8.3 Phase 3: Execution & Validation (1 hour)
1. Implement Registry.Execute()
2. Implement parameter validation
3. Add error handling
4. Write execution tests

### 8.4 Phase 4: Core Integration (1 hour)
1. Update Agent to use Registry
2. Add WithToolRegistry option to Manager
3. Update task mode filtering logic
4. Update all existing tests
5. Write integration tests

**Total Estimated Effort:** 6 hours

---

## 9. Acceptance Criteria

- [ ] `internal/tools` package created with all types
- [ ] Registry implements all required methods
- [ ] All 7 built-in tools implemented with proper schemas
- [ ] Parameter validation working for all schema types
- [ ] Agent uses Registry for tool execution
- [ ] Task modes filter tools correctly
- [ ] All unit tests passing (≥90% coverage)
- [ ] All integration tests passing
- [ ] Race detector clean
- [ ] All linters passing
- [ ] Godoc complete for all exported symbols
- [ ] Agent tests updated to use Registry
- [ ] Manager supports WithToolRegistry option

---

## 10. Future Enhancements

- Tool versioning support
- Tool dependency management
- Tool execution metrics/telemetry
- Tool sandboxing per tool (finer-grained than Executor)
- Dynamic tool loading from plugins
- Tool execution history/audit log

---

## 11. References

- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- Feature 6.2: Tool Call Processing
- `internal/llm` package (CompletionRequest.Tools)
- `internal/core` package (Executor, Context)

---

**Author:** AI Agent
**Reviewers:** Development Team
**Approval:** Pending Implementation
