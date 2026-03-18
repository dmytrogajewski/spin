# Journey R-T3: Replace Reflection with fmt.Stringer in GetContextTool

**Roadmap Item**: R-T3
**Spec**: [SPEC.md](../refactoring/tools-cleanup/SPEC.md) Section F-4
**Status**: Done

## Context

`GetContextTool` stored its dependency as `any` and called `String()` via
`reflect.ValueOf` and `MethodByName("String")`. This was done to avoid an
import cycle with the `agent` package.

The standard library `fmt.Stringer` interface provides compile-time safety
for the same purpose with zero reflection overhead.

## User Journey

### Persona
Developer adding a new environment property to `agent.Environment`.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Develop | Pass non-Stringer type | Runtime error at test time | Compile-time error |
| Debug | Trace reflection crash | Stack trace inside reflect pkg | N/A — won't compile |
| Maintain | Read get_context.go | 40 lines of reflection ceremony | 5 lines of direct call |

### Friction Points (Resolved)
1. **Runtime fragility**: eliminated — compile-time check.
2. **Reflection ceremony**: eliminated — direct `t.stringer.String()` call.
3. **Code clarity**: `any` field replaced with typed `fmt.Stringer`.

## Implementation

### Files Modified
| File | Change |
|------|--------|
| `internal/tools/get_context.go` | Field type `any` → `fmt.Stringer`; removed `reflect` import; simplified Execute to direct call |
| `internal/tools/registry.go` | `NewDefaultRegistry` env parameter: `any` → `fmt.Stringer`; fixed blank line formatting |
| `internal/tools/get_context_test.go` | Removed `TestGetContextTool_InvalidType` (no longer possible at compile time) |
| `internal/tools/registry_test.go` | Added `registryTestEnv` with `String()` method; updated 5 test functions; modernized `WaitGroup.Go` calls |

## Tests

- All `get_context_test.go` tests pass (nil context, success, output format, schema)
- All `registry_test.go` tests pass (default registry, tools configured, nil env, equivalent, all registered, unique names)
- `go vet ./internal/tools/...` clean
- `go build ./...` compiles cleanly (all callers including `conversation/tools.go`)
