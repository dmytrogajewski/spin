# Empty Interface Elimination Roadmap

## Progress Overview

**Total Tasks**: 79  
**Completed**: 10  
**In Progress**: 0  
**Remaining**: 69  

**Last Updated**: 2025-10-26  
**Latest Completion**: Phase 1.1 - Tool Parameter System ✅

---

## Phase 1: Core Types & Infrastructure ⏳

**Status**: Not Started  
**Estimated Duration**: Weeks 1-2  
**Goal**: Establish foundational types that other systems depend on

### 1.1 Tool Parameter System (Priority: P0) ✅ COMPLETED
- [x] `internal/toolparams/arguments.go` - Define `ToolParameters` type
  - [x] Create `ToolParameters` struct with `map[string]json.RawMessage`
  - [x] Implement `GetString(key string) (string, error)` method
  - [x] Implement `GetInt(key string) (int, error)` method
  - [x] Implement `GetBool(key string) (bool, error)` method
  - [x] Implement `GetFloat64(key string) (float64, error)` method
  - [x] Implement `GetObject(key string, dest any) error` method
  - [x] Implement `*Or()` methods with defaults
  - [x] Implement `Has()`, `Keys()`, `ToMap()`, `FromMap()`
  - [x] Implement JSON marshaling (MarshalJSON/UnmarshalJSON)
  - [x] Write unit tests (achieved: 91.7% coverage ✅)
  - [x] Add package documentation (doc.go)
  - [x] Run `go vet` and `go fmt` - all issues fixed
  - [x] Fix package naming: renamed `types` → `toolparams` (Go best practices)
  - [x] Create lint test to prevent anti-pattern package names (`internal/lint_test.go`)
  - [x] Update `docs/packages/toolparams.md` with full API documentation
  - [x] Create FRD-001.md specification

**Files Created**: 
- `internal/toolparams/arguments.go` (226 lines)
- `internal/toolparams/arguments_test.go` (627 lines, 91.7% coverage)
- `internal/toolparams/doc.go` (90 lines)
- `internal/lint_test.go` (prevents "types", "common", "util" anti-patterns)
- `specs/frds/FRD-001.md` (full specification)

**Files Updated**:
- `docs/packages/toolparams.md` (comprehensive documentation)

**Status**: ✅ Complete - Ready for Phase 1.2
**Note**: Tool interface update deferred to maintain backward compatibility (gradual migration strategy)

### 1.2 Event System (Priority: P0)
- [ ] `internal/events/event.go` - Define generic Event type
  - [ ] Create `Event[T any]` generic struct
  - [ ] Update `GetData()` method to return typed data
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/cycle/intervention.go` - Update intervention events
  - [ ] Convert `eventImpl` to generic `eventImpl[T any]`
  - [ ] Update `GetData()` return type
  - [ ] Update all event creators to use typed events
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/cycle/intervention_test.go`
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/detection/detection.go` - Define detection event types
  - [ ] Create `DetectionEventData` struct
  - [ ] Update `Event` interface to return `DetectionEventData`
  - [ ] Update `event` implementation
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/events/event.go`, `internal/cycle/intervention.go`, `internal/cycle/intervention_test.go`, `internal/detection/detection.go`

### 1.3 Message System (Priority: P0)
- [ ] `internal/message/message.go` - Define typed structures
  - [ ] Create `ToolCall` struct with proper types
  - [ ] Create `FunctionCall` struct
  - [ ] Define `Metadata` type alias (`map[string]string`)
  - [ ] Update `Message` struct to use `[]ToolCall`
  - [ ] Update `Message` struct to use `Metadata`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/message/message.go`

### 1.4 Error Handling (Priority: P1)
- [ ] `internal/errors/errors.go` - Convert As to generics
  - [ ] Update `As[T error](err error, target *T) bool` signature
  - [ ] Update implementation to use generics
  - [ ] Keep `Newf` as-is (idiomatic with `...interface{}`)
  - [ ] Update `tryAssign` to work with generics
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/errors/errors.go`

---

## Phase 2: Provider Layer ⏳

**Status**: Not Started  
**Estimated Duration**: Weeks 3-4  
**Goal**: Type-safe API interactions with external services

### 2.1 LLM Base Types (Priority: P0)
- [ ] `internal/llm/types.go` - Update Function parameters
  - [ ] Change `Parameters interface{}` to `json.RawMessage`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/llm/types.go`

### 2.2 OpenAI Provider (Priority: P1)
- [ ] `internal/llm/openai/types.go` - Define OpenAI types
  - [ ] Create `OpenAIRequest` struct
  - [ ] Create `OpenAIMessage` struct
  - [ ] Create `OpenAITool` struct
  - [ ] Create `OpenAIToolCall` struct
  - [ ] Create `OpenAIFunction` struct
  - [ ] Update `Function.Parameters` to use `json.RawMessage`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/llm/openai/provider.go` - Update provider implementation
  - [ ] Update `buildRequest` to return `OpenAIRequest`
  - [ ] Update `convertMessages` to return `[]OpenAIMessage`
  - [ ] Update `convertMessage` to return `OpenAIMessage`
  - [ ] Update `convertToolCalls` to return `[]OpenAIToolCall`
  - [ ] Update `addTools` to use typed structures
  - [ ] Update `addOptionalParameters` signature
  - [ ] Update `newRequest` to accept typed body
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/llm/openai/provider_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/llm/openai/types.go`, `internal/llm/openai/provider.go`, `internal/llm/openai/provider_test.go`

### 2.3 Ollama Provider (Priority: P1)
- [ ] `internal/llm/ollama/types.go` - Define Ollama types
  - [ ] Create `OllamaOptions` struct
  - [ ] Create `OllamaRequest` struct
  - [ ] Create `OllamaMessage` struct
  - [ ] Create `OllamaTool` struct
  - [ ] Create `OllamaToolCall` struct
  - [ ] Update `GenerateRequest.Options` to use `OllamaOptions`
  - [ ] Update `ToolFunction.Parameters` to use `json.RawMessage`
  - [ ] Update `ToolCall.Arguments` to use `json.RawMessage`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/llm/ollama/provider.go` - Update provider implementation
  - [ ] Update options initialization to use `OllamaOptions`
  - [ ] Update `newRequest` to accept typed body
  - [ ] Update tool parameter extraction logic
  - [ ] Update argument handling in tool calls
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/llm/ollama/provider_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/llm/ollama/types.go`, `internal/llm/ollama/provider.go`, `internal/llm/ollama/provider_test.go`

### 2.4 MCP Manager (Priority: P1)
- [ ] `internal/mcp/manager.go` - Define MCP types
  - [ ] Create `ToolArguments` type alias or `MCPToolParams` struct
  - [ ] Implement `GetString(key string) (string, error)` method
  - [ ] Implement `GetInt(key string) (int, error)` method
  - [ ] Implement `GetBool(key string) (bool, error)` method
  - [ ] Implement `Unmarshal(key string, v any) error` method
  - [ ] Update `CallTool` signature to use new type
  - [ ] Update `MCPToolWrapper.Execute` signature
  - [ ] Update schema parsing logic
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/mcp/manager.go`

---

## Phase 3: Protocol & Orchestration ⏳

**Status**: Not Started  
**Estimated Duration**: Week 5  
**Goal**: Type-safe message handling and orchestration

### 3.1 Protocol Layer (Priority: P1)
- [ ] `internal/protocol/protocol.go` - Define ParsedMessage types
  - [ ] Create `ParsedMessage` interface with `messageType()` method
  - [ ] Create `TurnStartMessage` struct implementing `ParsedMessage`
  - [ ] Create `AssistantDeltaMessage` struct implementing `ParsedMessage`
  - [ ] Create `ToolCallProposedMessage` struct implementing `ParsedMessage`
  - [ ] Create `ToolCallExecutingMessage` struct implementing `ParsedMessage`
  - [ ] Create `ToolCallResultMessage` struct implementing `ParsedMessage`
  - [ ] Create `TurnCompleteMessage` struct implementing `ParsedMessage`
  - [ ] Create `StatusUpdateMessage` struct implementing `ParsedMessage`
  - [ ] Update `ParseMessage` to return `ParsedMessage`
  - [ ] Update `messageParser` type definition
  - [ ] Update all parser functions to return `ParsedMessage`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/protocol/protocol.go`

### 3.2 JSON-RPC Layer (Priority: P1)
- [ ] `internal/protocol/jsonrpc/jsonrpc.go` - Update config types
  - [ ] Option A: Create `Config` struct OR Option B: Use `json.RawMessage`
  - [ ] Update `InitializeRequest.Config` field
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/protocol/jsonrpc/jsonrpc_test.go`
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/protocol/jsonrpc/server.go` - Update handler interface
  - [ ] Update `RequestHandler` return type to `json.RawMessage`
  - [ ] Update all implementations
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/protocol/jsonrpc/server_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/protocol/jsonrpc/jsonrpc.go`, `internal/protocol/jsonrpc/jsonrpc_test.go`, `internal/protocol/jsonrpc/server.go`, `internal/protocol/jsonrpc/server_test.go`

### 3.3 Orchestration (Priority: P1)
- [ ] `internal/orchestration/tool_executor.go` - Update argument parsing
  - [ ] Create `ToolParameters` type alias
  - [ ] Update `parseToolArguments` return type
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/orchestration/tool_executor_test.go` mock tools
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/orchestration/turn.go` - Define TurnMetadata
  - [ ] Create `TurnMetadata` type alias
  - [ ] Update `Turn.Metadata` field type
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/orchestration/tool_executor.go`, `internal/orchestration/tool_executor_test.go`, `internal/orchestration/turn.go`

---

## Phase 4: Utilities & Infrastructure ⏳

**Status**: Not Started  
**Estimated Duration**: Week 6  
**Goal**: Improve testing and utility functions

### 4.1 Test Utilities (Priority: P2)
- [ ] `internal/testutil/helpers.go` - Convert to generics
  - [ ] Update `AssertEqual[T comparable](t *testing.T, want, got T, msgAndArgs ...interface{})`
  - [ ] Update `AssertNotNil[T any](t *testing.T, value T, msgAndArgs ...interface{})`
  - [ ] Keep `RequireNoError` as-is (idiomatic)
  - [ ] Keep `RequireError` as-is (idiomatic)
  - [ ] Keep `AssertContains` as-is (string-specific)
  - [ ] Update `TableTest[F, R any]` struct
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/testutil/doc.go` - Update documentation
  - [ ] Update examples to show generic usage
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/testutil/helpers.go`, `internal/testutil/doc.go`

### 4.2 Tokenizer (Priority: P2)
- [ ] `internal/tokenizer/tokenizer.go` - Define TokenizableMessage
  - [ ] Create `TokenizableMessage` interface
  - [ ] Add `GetRole() string` method
  - [ ] Add `GetContent() string` method
  - [ ] Add `GetToolCalls() []ToolCall` method
  - [ ] Update `Tokenizer` interface to use `[]TokenizableMessage`
  - [ ] Update `SimpleTokenizer.CountMessages` signature
  - [ ] Update implementation logic
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/tokenizer/tokenizer.go`

### 4.3 Shell Integration (Priority: P2)
- [ ] `internal/shell/integration.go` - Define ShellContextInfo
  - [ ] Create `ShellContextInfo` struct
  - [ ] Update `GetContextInfo()` return type
  - [ ] Update implementation
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/shell/operation_tool.go` - Define ShellOperationParams
  - [ ] Create `ShellOperationParams` struct
  - [ ] Update `Execute` signature to use `ShellOperationParams`
  - [ ] Update implementation
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/shell/operation_tool_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/shell/integration.go`, `internal/shell/operation_tool.go`, `internal/shell/operation_tool_test.go`

---

## Phase 5: UI & Configuration ⏳

**Status**: Not Started  
**Estimated Duration**: Week 7  
**Goal**: Clean up remaining UI and config usage

### 5.1 UI Blocks (Priority: P3)
- [ ] `internal/ui/blocks/model.go` - Define Metadata type
  - [ ] Create `Metadata` type alias (`map[string]string`)
  - [ ] Update `Block.Meta` field type
  - [ ] Update `GetMeta(key string) (string, bool)` signature
  - [ ] Update `SetMeta(key string, value string)` signature
  - [ ] Update metadata initialization
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/ui/blocks/model_test.go`
  - [ ] Update `internal/ui/blocks/renderer_bench_test.go`
  - [ ] Update `internal/ui/blocks/renderer_tool_test.go`
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/ui/blocks/metadata.go` - Update metadata extraction
  - [ ] Update all metadata extraction functions
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/ui/adapters/puretty.go` - Update metadata init
  - [ ] Update metadata initialization
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/ui/blocks/model.go`, `internal/ui/blocks/model_test.go`, `internal/ui/blocks/renderer_bench_test.go`, `internal/ui/blocks/renderer_tool_test.go`, `internal/ui/blocks/metadata.go`, `internal/ui/adapters/puretty.go`

### 5.2 UI Commands (Priority: P3)
- [ ] `internal/ui/overlay/command.go` - Define CommandArgs
  - [ ] Create `CommandArgs` struct
  - [ ] Implement `Get(index int) (string, error)` method
  - [ ] Implement `GetInt(index int) (int, error)` method
  - [ ] Update `CommandHandler` type definition
  - [ ] Update all command implementations
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/ui/overlay/command_test.go`
  - [ ] Update `internal/ui/overlay/palette_renderer_test.go`
  - [ ] Update `internal/ui/overlay/palette_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/ui/overlay/command.go`, `internal/ui/overlay/command_test.go`, `internal/ui/overlay/palette_renderer_test.go`, `internal/ui/overlay/palette_test.go`

### 5.3 Configuration (Priority: P2)
- [ ] `cmd/spin/config.go` - Convert to type-safe
  - [ ] Create `ConfigData` type (or use generic approach)
  - [ ] Replace inline `interface{ Write([]byte) (int, error) }` with `io.Writer`
  - [ ] Update `redactSensitiveValues` signature
  - [ ] Update `printJSON[T any](out io.Writer, data T) error`
  - [ ] Update `printYAML[T any](out io.Writer, data T) error`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `cmd/spin/config_test.go`
  - [ ] Run `make lint` and fix all issues

- [ ] `cmd/spin/mcp.go` - Convert outputJSON to generic
  - [ ] Update `outputJSON[T any](data T) error`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `cmd/spin/config.go`, `cmd/spin/config_test.go`, `cmd/spin/mcp.go`

### 5.4 Manager Configuration (Priority: P1)
- [ ] `internal/manager/config.go` - Update ProviderConfig
  - [ ] Option A: Use `json.RawMessage` OR Option B: Define specific types
  - [ ] Update `ProviderConfig` field
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/manager/config.go`

---

## Phase 6: Additional Components ⏳

**Status**: Not Started  
**Estimated Duration**: Week 8  
**Goal**: Clean up remaining components

### 6.1 Security (Priority: P2)
- [ ] `internal/security/approval.go` - Define ApprovalContext
  - [ ] Create `ApprovalContext` struct
  - [ ] Update `ApprovalRequest.Context` field
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/security/approval.go`

### 6.2 Git Integration (Priority: P2)
- [ ] `internal/git/integration.go` - Define GitOperationParams
  - [ ] Create `GitOperationParams` struct
  - [ ] Update git operation functions
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/git/integration.go`

### 6.3 History (Priority: P2)
- [ ] `internal/history/history.go` - Update type assertions
  - [ ] Review and update tool call extraction logic
  - [ ] Replace type assertions with typed structures
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/history/history.go`

### 6.4 Debug Events (Priority: P3)
- [ ] `internal/debug/events.go` - Update event structures
  - [ ] Review and update event data structures
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/debug/events_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/debug/events.go`, `internal/debug/events_test.go`

### 6.5 LLM Builder (Priority: P2)
- [ ] `internal/llm/builder/builder.go` - Update configuration handling
  - [ ] Review builder configuration structures
  - [ ] Update to use typed configs
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/llm/builder/builder_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/llm/builder/builder.go`, `internal/llm/builder/builder_test.go`

### 6.6 Tools System Extensions (Priority: P1)
- [ ] `internal/tools/builtin.go` - Update to use ToolParameters
  - [ ] Update all builtin tools to use `ToolParameters`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/tools/builtin_test.go`
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/tools/git_operation_tool.go` - Update to use ToolParameters
  - [ ] Update git tool to use `ToolParameters`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/tools/parser.go` - Update argument parsing
  - [ ] Update parser to work with `ToolParameters`
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/tools/parser_test.go`
  - [ ] Run `make lint` and fix all issues

- [ ] `internal/tools/registry.go` - Update registry
  - [ ] Update registry to work with typed tools
  - [ ] Write unit tests (target: 90%+ coverage)
  - [ ] Update `internal/tools/registry_test.go`
  - [ ] Run `make lint` and fix all issues

**Files Affected**: `internal/tools/builtin.go`, `internal/tools/builtin_test.go`, `internal/tools/git_operation_tool.go`, `internal/tools/parser.go`, `internal/tools/parser_test.go`, `internal/tools/registry.go`, `internal/tools/registry_test.go`

---

## Phase 7: Testing & Documentation ⏳

**Status**: Not Started  
**Estimated Duration**: Week 9  
**Goal**: Ensure all changes are tested and documented

### 7.1 Integration Testing
- [ ] Run full test suite across all packages
- [ ] Verify all tests pass with new types
- [ ] Check test coverage (target: 90%+ overall)
- [ ] Run benchmarks to check for performance regressions
- [ ] Test with real OpenAI API calls
- [ ] Test with real Ollama API calls
- [ ] Test MCP integrations
- [ ] Test shell operations

### 7.2 Static Analysis
- [ ] Run `make lint` on entire codebase
- [ ] Fix all linting errors
- [ ] Run `go vet ./...`
- [ ] Run `staticcheck ./...`
- [ ] Check for dead code
- [ ] Verify no `interface{}` remains (except idiomatic cases)

### 7.3 Documentation
- [ ] Update `docs/` with type changes
- [ ] Create migration guide in `docs/migration/interface-elimination.md`
- [ ] Update API documentation
- [ ] Update code examples
- [ ] Update AGENTS.md if needed
- [ ] Update README.md if needed
- [ ] Add inline code comments for new types
- [ ] Document breaking changes

### 7.4 Final Verification
- [ ] Build entire project successfully
- [ ] Run all tests one final time
- [ ] Create comprehensive test report
- [ ] Review all changed files
- [ ] Ensure backward compatibility where needed
- [ ] Prepare release notes

---

## Keep As-Is (Idiomatic Go) ✅

These usages will be kept because they are idiomatic Go patterns:

### Printf-style Variadic Arguments
- [x] `internal/errors/errors.go:84` - `func Newf(...args ...interface{})`
- [x] `internal/testutil/helpers.go:189` - `RequireNoError(msgAndArgs ...interface{})`
- [x] `internal/testutil/helpers.go:201` - `RequireError(msgAndArgs ...interface{})`
- [x] `internal/testutil/helpers.go:237` - `AssertContains(msgAndArgs ...interface{})`

---

## Appendix: Implementation Guidelines

### When to Use Each Approach

#### ✅ Use `json.RawMessage`:
- JSON schema parameters that vary by provider
- Configuration values that need delayed parsing
- API request/response bodies passed through unchanged

#### ✅ Use Generics:
- Container types (caches, wrappers)
- Generic utility functions (assertions, conversions)
- Event systems with typed data
- Table-driven tests

#### ✅ Use Defined Structs:
- API request/response types
- Configuration structures
- Message formats
- Tool parameters with known fields

#### ✅ Use Type Aliases with Helper Methods:
- Metadata fields
- Parameter maps that need type-safe accessors
- Flexible but structured data

#### ✅ Keep `interface{}`:
- Printf-style variadic arguments (`...interface{}`)
- Stdlib-compatible signatures
- True polymorphic usage where any type is valid

---

## Success Metrics

### Code Quality
- [ ] 0 `interface{}` occurrences (except idiomatic cases)
- [ ] 90%+ test coverage for all new types
- [ ] No increase in cyclomatic complexity
- [ ] All static analysis passes (vet, staticcheck, lint)
- [ ] No dead code

### Performance
- [ ] No regression in benchmark tests
- [ ] Memory usage unchanged or improved
- [ ] API response times unchanged
- [ ] No allocation increases in hot paths

### Developer Experience
- [ ] Type completion works in IDEs
- [ ] Compile-time errors for type mismatches
- [ ] Clear error messages for validation failures
- [ ] Documentation complete and clear
- [ ] Migration guide available

---

## Risk Mitigation

### High Risk Areas
1. ⚠️ **Tool Execution System** - Heavy usage, affects many components
2. ⚠️ **Event System** - Core to application flow
3. ⚠️ **Protocol Layer** - Message parsing critical
4. ⚠️ **Provider APIs** - External API compatibility

### Mitigation Actions
- [ ] Create feature flags for gradual rollout
- [ ] Maintain parallel implementations during migration
- [ ] Set up comprehensive integration tests
- [ ] Monitor metrics after each phase
- [ ] Have rollback plan for each phase
- [ ] Document all breaking changes
- [ ] Create backward compatibility adapters where needed

---

## Document Metadata

**Version**: 2.0 (Checklist Format)  
**Last Updated**: 2025-10-26  
**Author**: Claude (Automated Analysis)  
**Status**: Active - Ready for Implementation  

**Changelog**:
- v2.0 - Converted to checklist-based format for progress tracking
- v1.0 - Initial analysis and roadmap creation
