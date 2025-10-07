# Package: internal/tools

**Path:** `internal/tools`  
**Purpose:** Tool registry and built-in tool implementations

---

## Overview

The `tools` package provides a registry for LLM-callable tools (functions) and implements built-in tools for common operations like file manipulation, shell commands, and code execution.

## Key Features

- **Tool Registry**: Central registration system with strict schema validation
- **Built-in Tools**: Command execution, file access, and search utilities
- **Custom Tools**: Functional interface that matches OpenAI function calling
- **Argument Helpers**: Shared `ToolCallArguments` container for tool interop
- **Async Execution**: Non-blocking invocation with structured results

## Package Structure

```
internal/tools/
├── types.go            # Tool interface, results, and schema types
├── registry.go         # Tool registry implementation with validation
├── builtin.go          # Built-in tools (bash, read_file, write_file, etc.)
├── builtin_test.go     # Built-in tool tests
└── registry_test.go    # Registry tests
```

## Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Schema() ToolSchema
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}
```

### ToolResult

```go
type ToolResult struct {
    Success bool   `json:"success"`
    Output  string `json:"output"`
    Error   string `json:"error,omitempty"`
}
```

### ToolSchema

```go
type ToolSchema struct {
    Type     string         `json:"type"`
    Function FunctionSchema `json:"function"`
}

type FunctionSchema struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  ParameterSchema `json:"parameters"`
}

type ParameterSchema struct {
    Type       string                       `json:"type"`
    Properties map[string]PropertyDefinition `json:"properties"`
    Required   []string                     `json:"required,omitempty"`
}
```

## Built-in Tools

### bash
Execute shell commands.

```json
{
  "name": "bash",
  "description": "Execute a bash command",
  "input_schema": {
    "type": "object",
    "properties": {
      "command": {"type": "string", "description": "Command to execute"}
    },
    "required": ["command"]
  }
}
```

### read_file
Read file contents.

### write_file
Write content to a file.

### edit_file
Apply edits to a file.

### grep
Search for patterns in files.

## Usage

### Register Built-in Tools

```go
import "github.com/dmytrogajewski/spin/internal/tools"

// Create registry
registry := tools.NewRegistry()

// Register built-in tools
registry.RegisterBuiltin()

// List tools
for _, tool := range registry.List() {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}
```

### Custom Tool

```go
type MyTool struct{}

func (t *MyTool) Name() string        { return "my_tool" }
func (t *MyTool) Description() string { return "Does something useful" }

func (t *MyTool) Schema() tools.ToolSchema {
    return tools.ToolSchema{
        Type: "function",
        Function: tools.FunctionSchema{
            Name:        "my_tool",
            Description: "Does something useful",
            Parameters: tools.ParameterSchema{
                Type: "object",
                Properties: map[string]tools.PropertyDefinition{
                    "input": {
                        Type:        "string",
                        Description: "Input parameter",
                    },
                },
                Required: []string{"input"},
            },
        },
    }
}

func (t *MyTool) Execute(ctx context.Context, params map[string]interface{}) (tools.ToolResult, error) {
    inputStr, _ := params["input"].(string)
    return tools.ToolResult{Success: true, Output: "Processed: " + inputStr}, nil
}

// Register
registry.Register(&MyTool{})
```

### Execute Tool

```go
result, err := registry.Execute(ctx, "bash", map[string]any{
    "command": "git status",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)
```

## Tool Schema

```go
type PropertyDefinition struct {
    Type        string   `json:"type"`
    Description string   `json:"description"`
    Enum        []string `json:"enum,omitempty"`
}

// Strict validation rejects unknown parameters and mismatched types.
```

## Shared Parameter Helpers

Use `types.ToolCallArguments` (from `internal/types`) to safely extract typed parameters and to convert between raw maps and structured arguments when integrating with the TUI and core event system.
```

---

**Last Updated:** 2025-10-05  
**Status:** ✅ Production Ready
