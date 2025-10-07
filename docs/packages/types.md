# Package: internal/types

**Path:** `internal/types`

The `types` package provides shared helper types that are consumed across the core, tools, and UI layers. It currently focuses on type-safe tool argument handling that bridges JSON payloads from LLM providers, UI transports, and internal Go structs.

## Key Concepts

- **ToolCallArguments**: A thin wrapper over `map[string]json.RawMessage` that preserves original JSON while offering typed accessors.
- **Argument Conversion**: Helpers for converting between raw `map[string]any` values and `ToolCallArguments` without losing fidelity.
- **Error Contracts**: Shared `ErrParameterNotFound` sentinel error to simplify caller-side handling.

## API Surface

```go
var ErrParameterNotFound = errors.New("parameter not found")

type ToolCallArguments map[string]json.RawMessage

func (t ToolCallArguments) Get(key string, dest any) error
func (t ToolCallArguments) GetString(key string) (string, error)
func (t ToolCallArguments) GetInt(key string) (int, error)
func (t ToolCallArguments) GetBool(key string) (bool, error)
func (t ToolCallArguments) ToMap() map[string]any

func FromMap(m map[string]any) (ToolCallArguments, error)
```

### Typed Accessors

`Get`, `GetString`, `GetInt`, and `GetBool` abstract JSON decoding and return `ErrParameterNotFound` when a key is missing. Callers can supply their own struct to `Get` for complex parameter shapes.

### Map Conversion

- `ToMap` eagerly decodes each argument into `any`, returning only keys that successfully unmarshal.
- `FromMap` mirrors `ToMap` by marshalling each value back to `json.RawMessage`, preserving data for transport over JSON-RPC or MCP channels.

## Usage

```go
args, _ := types.FromMap(map[string]any{"command": "ls"})
cmd, err := args.GetString("command")

if errors.Is(err, types.ErrParameterNotFound) {
    // handle missing parameter
}
```

The package is used by:

- `internal/tools/types_generic.go` for tool adapters that work with strongly typed structs
- `internal/core/event_generic.go` to embed tool arguments inside typed event payloads
- TUI modules that render approval and tool execution dialogs

## Testing

Unit tests should cover round-tripping conversions and edge cases such as unknown keys or invalid JSON. (Tests are pending as of this update and should be added alongside future enhancements.)
