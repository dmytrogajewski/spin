# Package: internal/tools

**Path:** `internal/tools`  
**Purpose:** Tool registry and built-in tool implementations

---

## Overview

The `tools` package provides a registry for LLM-callable tools (functions) and implements built-in tools for common operations like file manipulation, shell commands, and code execution.

## Key Features

- **Tool Registry**: Central registration system
- **Built-in Tools**: Common operations pre-implemented
- **Custom Tools**: Easy custom tool creation
- **Schema Validation**: JSON Schema for parameters
- **Type Safety**: Strongly typed tool definitions
- **Async Execution**: Non-blocking tool calls

## Package Structure

```
internal/tools/
├── types.go        # Tool interface and schema types
├── registry.go     # Tool registry implementation
├── builtin.go      # All built-in tool implementations
├── builtin_test.go # Built-in tools tests
└── registry_test.go # Registry tests
```

**Note:** All built-in tools (bash, read_file, write_file, edit_file, grep) are implemented in a single `builtin.go` file for easy discovery and maintenance.

## Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() Schema
    Execute(ctx context.Context, input map[string]any) (string, error)
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
// Define custom tool
type MyTool struct{}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "Does something useful"
}

func (t *MyTool) InputSchema() tools.Schema {
    return tools.Schema{
        Type: "object",
        Properties: map[string]tools.Property{
            "input": {
                Type:        "string",
                Description: "Input parameter",
            },
        },
        Required: []string{"input"},
    }
}

func (t *MyTool) Execute(ctx context.Context, input map[string]any) (string, error) {
    inputStr := input["input"].(string)
    return fmt.Sprintf("Processed: %s", inputStr), nil
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
type Schema struct {
    Type        string              `json:"type"`
    Properties  map[string]Property `json:"properties"`
    Required    []string            `json:"required"`
    Description string              `json:"description,omitempty"`
}

type Property struct {
    Type        string   `json:"type"`
    Description string   `json:"description"`
    Enum        []string `json:"enum,omitempty"`
}
```

---

**Last Updated:** 2025-10-05  
**Status:** ✅ Production Ready
