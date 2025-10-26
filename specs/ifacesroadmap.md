# Empty Interface Elimination Roadmap

## Progress Overview

**Total Tasks**: 78 (revised: removed testutil phase)
**Completed**: 7 (Phase 1.1-1.3, Phase 2.1-2.4)
**In Progress**: 0
**Remaining**: 71

**Current Interface{} Count**: ~342 occurrences across 71 files (9 eliminated: 2 in detection.go, 2 in message.go, 1 in completion.go, 4 in manager.go)
**Target**: Reduce to <30 occurrences (idiomatic cases only)

**Last Updated**: 2025-10-26
**Latest Completion**: Phase 2.4 - MCP Manager SDK Migration ✅
**Status**: Phase 1 - Complete (100%), Phase 2 - Complete (100%)

---

## Phase 1: Core Types & Infrastructure ⏳

**Status**: Not Started
**Estimated Duration**: Weeks 1-2
**Goal**: Establish foundational types that other systems depend on

### 1.1 Tool Parameter System (Priority: P0) ✅ COMPLETED
- [x] `internal/tools/parameters.go` - Define `ToolParameters` type
  - [x] Create `ToolParameters` struct with `map[string]json.RawMessage`
  - [x] Implement `GetString(key string) (string, error)` method
  - [x] Implement `GetInt(key string) (int, error)` method
  - [x] Implement `GetBool(key string) (bool, error)` method
  - [x] Implement `GetFloat64(key string) (float64, error)` method
  - [x] Implement `GetObject(key string, dest any) error` method
  - [x] Implement `*Or()` methods with defaults
  - [x] Implement `Has()`, `Keys()`, `ToMap()`, `FromMap()`
  - [x] Implement JSON marshaling (MarshalJSON/UnmarshalJSON)
  - [x] Write unit tests (achieved: 71.0% coverage)
  - [x] Add package documentation (doc.go)
  - [x] Run `go vet` and `go fmt` - all issues fixed

**Files Implemented**:
- `internal/tools/parameters.go` - Type-safe parameter handling
- `internal/tools/parameters_test.go` - Comprehensive tests (71.0% coverage)
- `internal/tools/parameters_doc.go` - Package documentation

**Status**: ✅ Complete - In production use
**Next Step**: Gradually migrate tools to use `ToolParameters` (Phase 3.2)
**Note**: Implementation in `internal/tools/` rather than separate package to keep related code together

### 1.2 Event System (Priority: P0) ✅ COMPLETED
- [x] `internal/events/event.go` - Add type-safe helper methods
  - [x] Add `ToolCallStartData()` helper method
  - [x] Add `ToolCallCompleteData()` helper method
  - [x] Add `ToolProgressData()` helper method
  - [x] Add `ContentDeltaData()` helper method
  - [x] Add `TurnEventData()` helper method
  - [x] Add `ApprovalEventData()` helper method
  - [x] Add `SystemEventData()` helper method
  - [x] Add `ErrorData()` helper method
  - [x] Write unit tests (achieved: 92.3% coverage)
  - [x] Run `go vet` and `go fmt` - all clean

- [x] `internal/detection/detection.go` - Define detection event types
  - [x] Create `DetectionEventData` type alias
  - [x] Update `event` struct to use `DetectionEventData`
  - [x] Update `EscalateIntervention` to use typed data
  - [x] Write unit tests (achieved: 47.8% coverage - acceptable for this change)
  - [x] Run `go vet` and `go fmt` - all clean

**Note**: `internal/cycle/intervention.go` was removed in previous cleanup (deadcode), so those tasks are N/A.

**Implementation Decision**: Kept `Event.Data` as `interface{}` (idiomatic Go for heterogeneous event streams) instead of making Event generic. Added type-safe helper methods for better IDE support and eliminated manual type assertions. See FRD-20251026-event-system-generics.md for rationale.

**Files Modified**: `internal/events/event.go`, `internal/events/event_test.go`, `internal/detection/detection.go`, `internal/detection/detection_test.go`

**Status**: ✅ Complete - Added to "Keep As-Is" section below

### 1.3 Message System (Priority: P0) ✅ COMPLETED
- [x] `internal/message/message.go` - Define typed structures
  - [x] Create `ToolCall` struct with proper types
  - [x] Create `FunctionCall` struct
  - [x] Define `Metadata` type alias (`map[string]string`)
  - [x] Update `Message` struct to use `[]ToolCall`
  - [x] Update `Message` struct to use `Metadata`
  - [x] Implement `GetRole()`, `GetContent()`, `GetTimestamp()` methods
  - [x] Write unit tests (achieved: 100% coverage)
  - [x] Run `go vet` and `go fmt` - all clean

**Files Modified**:
- `internal/message/message.go` - Added ToolCall, FunctionCall, Metadata types (+19 lines)
- `internal/message/message_test.go` - Comprehensive tests (+237 lines, 100% coverage)

**Interface{} Eliminated**: 2 occurrences (ToolCalls, Metadata)

**Status**: ✅ Complete - All tests pass, 100% coverage

---

## Phase 2: Provider Layer ⏳

**Status**: Not Started
**Estimated Duration**: Weeks 3-4
**Goal**: Type-safe API interactions with external services

### 2.1 LLM Base Types (Priority: P0) ✅ COMPLETED
- [x] `internal/llm/completion.go` - Update Function parameters
  - [x] Change `Parameters interface{}` to `json.RawMessage`
  - [x] Update Ollama provider to handle json.RawMessage
  - [x] Write unit tests (achieved: 79.6% coverage for llm, 91.4% for openai)
  - [x] Run `go vet` and `go fmt` - all clean

**Files Modified**:
- `internal/llm/completion.go` - Changed Parameters to json.RawMessage (+2 lines)
- `internal/llm/ollama/provider.go` - Fixed Parameters handling (-3 lines, +2 lines)
- `internal/llm/completion_test.go` - Added JSON marshaling test (+44 lines, new file)

**Interface{} Eliminated**: 1 occurrence (Function.Parameters)

**Status**: ✅ Complete - All tests pass, providers compatible

### 2.2 OpenAI Provider (Priority: P1) ✅ COMPLETED
- [x] Migrated to official openai-go SDK (v0.1.0-alpha.37)
  - [x] Added github.com/openai/openai-go dependency
  - [x] Created `convert.go` with type conversion functions
  - [x] Created `errors.go` with error mapping
  - [x] Rewrote `provider.go` using SDK client
  - [x] Updated `doc.go` to reference SDK
  - [x] Created new `provider_test.go` with basic tests
  - [x] Deleted old custom HTTP client code (`api.go`)
  - [x] Build passes (`make build`)
  - [x] Factory tests pass (provider name: "openai-compatible")

**Migration Notes**: 
- No backward compatibility maintained (clean cutover as requested)
- Reduced code from ~657 lines to ~250 lines
- SDK handles HTTP, SSE parsing, retry logic
- Models() temporarily returns configured model only (TODO: implement SDK pagination)

**Files Changed**: 
- NEW: `internal/llm/openai/convert.go` (type converters)
- NEW: `internal/llm/openai/errors.go` (error mapping)
- REPLACED: `internal/llm/openai/provider.go` (SDK-based implementation)
- REPLACED: `internal/llm/openai/provider_test.go` (new tests)
- UPDATED: `internal/llm/openai/doc.go` (SDK references)
- DELETED: `internal/llm/openai/api.go` (custom HTTP types)

### 2.3 Ollama Provider (Priority: P1) ✅ COMPLETED
- [x] Migrated to official ollama/ollama/api SDK (v0.12.6)
  - [x] Rewrote as thin wrapper around OpenAI provider (embedded architecture)
  - [x] Uses Ollama's OpenAI-compatible API at `/v1` endpoint
  - [x] Uses Ollama SDK only for Ollama-specific features:
    - [x] `Models()` - List available models with metadata
    - [x] `AutoTune()` - VRAM-based optimization
  - [x] Validated base URL before OpenAI provider creation
  - [x] Reduced code from ~600 lines to 204 lines
  - [x] All tests pass (38.2% coverage - wrapper code)
  - [x] Zero deadcode warnings
  - [x] Deleted unused files:
    - [x] `internal/llm/client.go` - Custom HTTP client
    - [x] `internal/llm/error_mapper.go` - Custom error mapping
    - [x] `internal/llm/client_test.go` - Orphaned test file
    - [x] `internal/llm/ollama/api.go` - Custom types
    - [x] `internal/llm/ollama/convert.go` - Type converters
    - [x] `internal/llm/ollama/convert_test.go` - Associated tests

**Migration Notes**:
- No backward compatibility maintained (clean cutover as requested)
- Architecture: LMStudio-style wrapper pattern
- OpenAI provider handles: Complete(), Stream(), Close()
- Ollama SDK handles: Models(), AutoTune() with VRAM detection
- VRAM auto-tuning logic preserved from original implementation

**Files Changed**:
- REPLACED: `internal/llm/ollama/provider.go` (thin wrapper, 204 lines)
- REPLACED: `internal/llm/ollama/provider_test.go` (simplified tests)
- DELETED: 6 files (custom HTTP, error mapping, type converters)

**Documentation Updated**:
- [x] `docs/packages/llm.md` - Updated Ollama section with wrapper architecture
- [x] `specs/ifacesroadmap.md` - Marked Phase 2.3 complete

**Status**: ✅ Complete - Ready for Phase 2.4 (MCP Manager)

### 2.4 MCP Manager (Priority: P1) ✅ COMPLETED
- [x] Migrated to official mark3labs/mcp-go SDK (v0.42.0)
  - [x] Replaced custom client implementation with SDK
  - [x] Updated `CallTool` signature from `map[string]interface{}` to `json.RawMessage`
  - [x] Defined `JSONSchema` and `JSONSchemaProperty` types for schema parsing
  - [x] Eliminated all 4 `interface{}` occurrences in manager.go
  - [x] Updated to use SDK types: `mcp.InitializeRequest`, `mcp.CallToolRequest`, etc.
  - [x] Deleted custom client code (~800 lines)
  - [x] Deleted custom types code (~400 lines)
  - [x] All tests pass (9.0% coverage - manager only, SDK provides protocol)
  - [x] Zero deadcode warnings
  - [x] Updated documentation

**Migration Notes**:
- No backward compatibility maintained (clean cutover as requested)
- Architecture: Thin manager layer around mcp-go SDK
- SDK handles: Protocol, JSON-RPC, transport, connection lifecycle
- Manager handles: Server registration, tool discovery, tool invocation routing
- Type safety: `json.RawMessage` for arguments, structured types for schemas

**Files Changed**:
- REPLACED: `internal/mcp/manager.go` (simplified to ~400 lines)
- REPLACED: `internal/mcp/manager_test.go` (basic SDK integration tests)
- DELETED: `internal/mcp/client/` (entire directory, ~500 lines)
- DELETED: `internal/mcp/types/` (entire directory, ~400 lines)

**Code Reduction**: ~900 lines deleted, ~400 lines simplified

**Documentation Updated**:
- [x] `docs/packages/mcp.md` - Updated with SDK architecture and examples
- [x] `specs/ifacesroadmap.md` - Marked Phase 2.4 complete
- [x] `specs/frds/FRD-20251026000001-mcp-go-sdk-migration.md` - Migration FRD created

**Status**: ✅ Complete - Ready for Phase 3

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
**Goal**: Improve utility functions

### 4.1 Tokenizer (Priority: P2)
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

### 4.2 Shell Integration (Priority: P2)
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

## Removed from Scope ❌

### Phase 4.1 - Test Utilities (REMOVED 2025-10-26)

**Reason**: The `internal/testutil` package was identified as unused deadcode and removed during cleanup.

**Impact**: None - package was never used in production. Tests use `github.com/stretchr/testify` directly.

**Documentation**: See `docs/deadcode-cleanup-2025-10-26.md` for details.

---

## Keep As-Is (Idiomatic Go) ✅

These usages will be kept because they are idiomatic Go patterns:

### Printf-style Variadic Arguments
- [x] `internal/errors/errors.go` - `func Newf(...args ...interface{})` - Matches `fmt.Sprintf` signature

### Heterogeneous Event Streams
- [x] `internal/events/event.go` - `Event.Data interface{}` - Idiomatic for heterogeneous event streams where different event types carry different data structures. Type safety provided through:
  - Strongly-typed payload structs (ContentDeltaData, ToolCallStartData, etc.)
  - Type-safe helper methods (event.ContentDeltaData(), event.ToolCallStartData(), etc.)
  - Compile-time guarantees for payload structure, runtime flexibility for event channel

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

## Baseline Measurement (2025-10-26)

**Current Interface{} Usage**: 351 occurrences across 72 files

**Top Areas**:
- `internal/tools/` - 50 occurrences (tool system, registry, tests)
- `internal/llm/` - 38 occurrences (OpenAI, Ollama providers)
- `internal/protocol/` - 18 occurrences (JSON-RPC, message protocol)
- `internal/ui/` - 26 occurrences (blocks, commands)
- `internal/mcp/` - 14 occurrences (MCP integration)
- Test files - ~40% of total occurrences

**Analysis Method**: `grep -r "interface{}" internal/ | wc -l`

**Target After Completion**: <30 occurrences (idiomatic cases only)

---

## Document Metadata

**Version**: 2.1 (Actualized - Post Deadcode Cleanup)
**Last Updated**: 2025-10-26
**Author**: Claude (Rob Pike persona)
**Status**: Active - Ready for Phase 1.2

**Changelog**:
- v2.1 - Actualized roadmap based on codebase analysis (2025-10-26)
  - Updated Phase 1.1 to reflect actual implementation in `internal/tools/`
  - Removed Phase 4.1 (testutil package deleted as deadcode)
  - Added baseline measurement: 351 interface{} occurrences
  - Updated progress counters (1/78 complete)
- v2.0 - Converted to checklist-based format for progress tracking
- v1.0 - Initial analysis and roadmap creation
