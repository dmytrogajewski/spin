# UI Modules Implementation Roadmap

## Overview

This roadmap covers the implementation of all three UI modules for Spin:
- **spin-tui**: Interactive terminal user interface (Bubble Tea)
- **spin-exec**: Non-interactive/headless execution mode
- **spin-cli**: Main CLI multitool and entry point

All implementations follow TDD principles, SOLID design, and the quality standards defined in [AGENTS.md](../../AGENTS.md).

---

## Phase 1: Foundation & CLI Framework

### 1.1 Main CLI Entry Point (`cmd/spin/`) ✅

**DoR:**
- [x] Project structure follows Go Standard Layout
- [x] Dependencies specified in go.mod
- [x] Architecture overview reviewed

**Implementation:**
- [x] Create `cmd/spin/main.go` with Cobra framework
- [x] Implement global flags (--model, --provider, --sandbox, --cd, --config)
- [x] Add command registration structure
- [x] Implement help system and version command
- [x] Add binary name detection for special modes (spin-apply-patch, spin-sandbox)
- [x] Create shell completion command (bash, zsh, fish, powershell)

**DoD:**
- [x] All tests passing (≥85% coverage) - **86.1% coverage achieved**
- [x] Linter clean (golangci-lint) - **Passing (duplication warnings are expected for stubs)**
- [x] Complexity ≤15 - **All functions < 15**
- [x] Godoc on all exports - **Complete**
- [x] Can execute: `spin --help`, `spin --version` - **Working**
- [x] Shell completions working - **bash, zsh, fish, powershell complete**

**FRD:** [FRD-UI-1.1](../frds/FRD-UI-1.1.md)
**Status:** ✅ **COMPLETE**

---

### 1.2 Configuration Management

WARNING: We use spf13/viper, you should review configuration management and base this implementation on what we already have

**DoR:**
- [ ] Config spec from core-module reviewed
- [ ] Configuration file format defined (TOML)
- [ ] Environment variable naming convention established

**Implementation:**
- [ ] Create `internal/config/` package
- [ ] Implement Config struct with TOML tags
- [ ] Add Load() function with precedence (CLI > --config > file > env > defaults)
- [ ] Implement Validate() method
- [ ] Add Merge() for config overrides
- [ ] Support for `~/.config/spin/config.toml`
- [ ] MCP server configuration support

**DoD:**
- [ ] Tests for all loading methods (≥90% coverage)
- [ ] Tests for precedence rules
- [ ] Error handling for invalid configs
- [ ] Example config.toml provided
- [ ] Documentation complete

---

### 1.3 Logging Infrastructure

**DoR:**
- [ ] Go 1.24+ confirmed (for log/slog)
- [ ] Logging levels defined (debug, info, warn, error)

**Implementation:**
- [ ] Setup log/slog with TextHandler
- [ ] Implement SPIN_LOG_LEVEL environment variable support
- [ ] Create context-aware logger helpers
- [ ] Add structured logging for events
- [ ] Implement log rotation (optional)

**DoD:**
- [ ] Tests for log level parsing
- [ ] Tests for structured logging
- [ ] No logs in stdout during normal operation (TUI/exec)
- [ ] All errors logged to stderr
- [ ] Performance: <1ms per log call

---

## Phase 2: Non-Interactive Mode (spin-exec)

**Implementation Strategy:**
- **Phases 2.1-2.3**: Build infrastructure (args, output, approval) with placeholder logic
- **Phase 2.4**: Connect everything to the core module (⚡ actual integration happens here)

This approach allows us to:
1. Build and test the exec infrastructure independently
2. Ensure clean separation of concerns
3. Integrate with a stable core interface in Phase 2.4

### 2.1 Exec Command Structure ✅

**DoR:**
- [x] Core module integration interface defined
- [x] Exit code specification reviewed
- [x] Output format requirements clear

**Implementation:**
- [x] Create `cmd/spin-exec/main.go`
- [x] Implement argument parsing (prompt from args or stdin)
- [x] Add exec-specific flags (--auto-approve, --timeout, --format)
- [x] Implement run() function with context
- [x] Add timeout support via context.WithTimeout
- [x] Setup graceful shutdown (SIGINT/SIGTERM handling)

**DoD:**
- [x] Tests for argument parsing (76.7% coverage overall)
- [x] Tests for timeout handling
- [x] Tests for signal handling
- [x] Can execute: `spin exec "test prompt"`
- [x] Can accept stdin: `echo "prompt" | spin exec`
- [x] All tests passing
- [x] Complexity ≤15 (all functions pass)
- [x] Linter clean (golangci-lint)
- [x] Binary builds successfully

**FRD:** [FRD-UI-2.1](../frds/FRD-UI-2.1.md)
**Status:** ✅ **COMPLETE**

**Note:** This phase implements the infrastructure (args, signals, errors) only. **Core module integration happens in Phase 2.4**. Current implementation uses placeholder logic.

---

### 2.2 Output Formatting ✅

**Note:** Infrastructure only. **Core integration in Phase 2.4**.

**DoR:**
- [x] Output format spec reviewed (text, json)
- [x] Streaming requirements understood

**Implementation:**
- [x] Create `internal/exec/format/` package with types
- [x] Implement text output formatter (human-readable)
- [x] Implement JSON output formatter (structured)
- [x] Add streaming output support (stdout)
- [x] Implement summary generation (tokens used, files modified, etc.)
- [x] Create formatter factory `NewFormatter()`

**DoD:**
- [x] Tests for both output formats (≥90% coverage) - **98.8% achieved**
- [x] Streaming works correctly (tested)
- [x] JSON output is valid and parseable (tested with json.Unmarshal)
- [x] NO_COLOR support implemented
- [x] Exit codes structure defined
- [x] All tests passing
- [x] Linter clean
- [x] Complexity ≤15 for all production code
- [x] Godoc on all exports

**FRD:** [FRD-UI-2.2](../frds/FRD-UI-2.2.md)
**Status:** ✅ **COMPLETE**

**Coverage:** 98.8% (format package), 100.0% (exec package)

**Note:** This phase implements formatters with placeholder data. **Core module integration happens in Phase 2.4**.

---

### 2.3 Non-Interactive Tool Approval ✅

**Status:** ✅ **COMPLETE** - Integrated with existing core.Validator (no duplication needed)

**Note:** Phase 2.3 was simplified after discovering `internal/core/validator.go` already implements comprehensive command classification. Instead of duplicating, integrated directly with core in Phase 2.4.

**Accomplished:**
- ✅ Reviewed existing `core.Validator` with safety classification (Safe, Interactive, Dangerous, Forbidden, Unverified)
- ✅ Understood approval flow: core emits `EventCommandApproval` when validation fails
- ✅ Implemented `--auto-approve` flag in spin-exec args
- ✅ Integrated approval handling in runner.go using core's existing patterns

**Key Decision:** No separate approval package needed - core already has all classification logic.

---

### 2.4 Exec Integration with Core ✅

**Status:** ✅ **COMPLETE**

**DoR:**
- [x] Core module interface stable ✅
- [x] Event protocol understood ✅
- [x] Channel-based communication pattern defined ✅
- [x] Phases 2.1, 2.2, 2.3 complete ✅

**Implementation:**
- [x] Create `cmd/spin-exec/runner.go` ✅
- [x] Implement runTask() function ✅
- [x] Setup core event channel handling ✅
- [x] Add streaming delta processing (EventContentDelta) ✅
- [x] Implement completion detection (EventTurnComplete) ✅
- [x] Error propagation from core (EventError) ✅
- [x] Integrated with core.Validator for command approval ✅
- [x] Added audit logging for approval decisions ✅

**DoD:**
- [x] Integration with real core ✅
- [x] Event streaming works ✅
- [x] Error handling works ✅
- [x] --auto-approve flag functional ✅
- [x] Linter clean ✅
- [x] Builds successfully ✅

**Implementation Notes:**
- Uses `core.DefaultConfig()` with mock LLM provider
- `--auto-approve` sets `AllowedCommands = ["*"]` to bypass validation
- Without `--auto-approve`, dangerous commands trigger `EventCommandApproval` and exec denies them
- Audit logging via `log/slog` to stderr in JSON format
- Streams content to stdout as `EventContentDelta` events arrive

---

## ⚠️ IMPORTANT: Core Integration Gaps Identified

**Comprehensive core analysis completed**

### Critical Blockers Found

Phase 3 (TUI) and full spin-exec functionality are **BLOCKED** by missing core functionality and integration:

1. **❌ Approval Response Mechanism** - Core emits `EventCommandApproval` but has NO WAY to respond
2. **❌ Pause/Resume Turn** - Can stop conversation but can't pause mid-execution
3. **❌ Event Backpressure** - Slow consumers drop events (fire-and-forget)
4. **⚠️ Provider Factory Integration** - cmd modules use mock providers, real LLM integration not connected

**Action Required:** Complete Phase 0 (Fix Core Gaps) below before continuing with Phase 3.

---

## Phase 0: Fix Core Gaps (CRITICAL - Must Complete First)

**Status:** 🔴 **BLOCKING** all TUI work

This phase fixes critical gaps in `internal/core` and integrates LLM provider factory into cmd modules for real-world usage.

### 0.1 Approval Response Mechanism ✅ COMPLETE

**Status:** ✅ **COMPLETE** (2025-10-05)

**Implemented:**
- ✅ Public `WithApprovalHandler` option for Agent
- ✅ `ApprovalRequest` type with UUID, command details, reason, workdir, timestamp
- ✅ `ApprovalResponse` type with request ID, approved boolean, reason, modified command, timestamp
- ✅ `requestApproval()` updated with full approval flow (timeout, validation, modification)
- ✅ Event types: `EventCommandApproval`, `EventCommandApproved`, `EventCommandDenied`
- ✅ Command modification support with re-validation
- ✅ Timeout handling (default: 60s, configurable via `ApprovalTimeout`)
- ✅ Context cancellation support
- ✅ Request ID validation
- ✅ Auto-deny when no handler set

**Testing:**
- ✅ 9 comprehensive test cases in `approval_simplified_test.go`
- ✅ All tests passing with race detector
- ✅ Covers: approval/denial, timeout, cancellation, ID mismatch, command modification, validation errors

**Quality:**
- ✅ Linter clean (`make lint` passes)
- ✅ Complexity: 12 (within ≤15 threshold)
- ✅ Godoc complete on all exports
- ✅ No dead code

**DoD:**
- [x] Public approval handler API available
- [x] Request/response types defined and documented
- [x] Approval flow emits both request and result events
- [x] Tests with mock handler (9 test cases)
- [x] Documentation with usage examples
- [x] FRD created: `specs/frds/FRD-CORE-0.1.md`

**Files:**
- [internal/core/agent.go](../../internal/core/agent.go) - Types, `WithApprovalHandler`, `requestApproval`
- [internal/core/event.go](../../internal/core/event.go) - Event types
- [internal/core/approval_simplified_test.go](../../internal/core/approval_simplified_test.go) - Tests
- [specs/frds/FRD-CORE-0.1.md](../frds/FRD-CORE-0.1.md) - FRD documentation

---

### 0.2 Pause/Resume Turn Execution ✨ CRITICAL

**Current Problem:** Can only `Stop()` conversation, no way to pause mid-execution and resume.

**What to Build:**
- `Pause()` method on Conversation to pause running turn
- `Resume()` method on Conversation to continue paused turn
- Internal control channel for pause/resume/cancel signals
- `StatePaused` state added to conversation state machine
- Update `RunTurn()` to check control signals during execution loop
- State transition validation (only pause when running, only resume when paused)

**Why:** Interactive UI needs ability to pause turn for user review (e.g., during approval dialogs) and resume after user input, without cancelling the entire conversation.

**DoD:**
- [ ] Pause/Resume API available on Conversation
- [ ] Control signals integrated into turn execution loop
- [ ] State machine properly handles transitions
- [ ] Tests for pause/resume scenarios (≥90% coverage)
- [ ] Documentation updated
- [ ] FRD created: `specs/frds/FRD-CORE-0.2.md`

---

### 0.3 Event Streaming Control ✨ IMPORTANT

**Current Problem:** EventEmitter uses unbuffered channels with fire-and-forget ([event.go](../../internal/core/event.go)). Slow consumers (like UI rendering) drop events silently.

**What to Build:**
- Configurable backpressure strategies for EventEmitter
  - `BackpressureDrop` - Current behavior, drop if channel full
  - `BackpressureBlock` - Block emitter until consumer ready
  - `BackpressureBuffer` - Dynamic buffer growth up to limit
- `EventEmitterConfig` with buffer size, backpressure mode, buffer limit
- Constructor `NewEventEmitterWithConfig()` to create configured emitter
- Update `Emit()` to apply backpressure strategy based on config

**Why:** UI components process events slower than core generates them (rendering takes time). Without backpressure control, critical events like approval requests or error messages can be silently dropped, breaking UI state.

**DoD:**
- [ ] All three backpressure modes implemented
- [ ] Configurable buffer size and limits
- [ ] Tests for each mode (≥90% coverage)
- [ ] Performance benchmarks showing overhead
- [ ] Documentation with mode selection guidance
- [ ] FRD created: `specs/frds/FRD-CORE-0.3.md`

---

### 0.4 Provider Factory Integration in CMD ✨ IMPORTANT

**Current Problem:** `cmd/spin-exec` and future cmd modules need to create LLM providers but currently use hardcoded mock providers. The `internal/llm/factory` package provides provider creation with auth integration but is not connected to cmd modules.

**What to Build:**
- Integrate `internal/llm/factory` into cmd modules for provider instantiation
- Add provider type configuration support (--provider flag: openai, ollama, lmstudio, etc.)
- Add model selection support (--model flag)
- Integrate with `internal/auth` for secure credential management via keystore
- Support multiple authentication methods:
  - KeyName (recommended) - credentials from keystore
  - APIKey (deprecated) - direct API key
  - Environment variables (OPENAI_API_KEY, etc.)
- Add provider-specific configuration (base URL, timeout, options)

**Why:** Users need to connect to real LLM providers (OpenAI, Ollama, LMStudio) instead of mock providers. The factory pattern with auth integration is already implemented in `internal/llm/factory` but not exposed to cmd modules. This unblocks real-world usage of spin-exec and future spin-tui.

**Target Modules:**
- `cmd/spin-exec` - Update runner.go to use factory instead of mock provider
- `cmd/spin` - Add global provider/model flags
- Future: `cmd/spin-tui` - Use same factory pattern

**DoD:**
- [ ] Provider factory integrated in cmd/spin-exec
- [ ] --provider flag works (openai, ollama, lmstudio, openai-compatible)
- [ ] --model flag works with provider validation
- [ ] Auth manager integrated for keystore credentials
- [ ] Environment variable fallback works
- [ ] Tests with multiple provider types (≥90% coverage)
- [ ] Documentation for provider configuration
- [ ] Example configurations for each provider type
- [ ] FRD created: `specs/frds/FRD-CORE-0.4.md`

**Related Files:**
- [internal/llm/factory/factory.go](../../internal/llm/factory/factory.go) - Provider factory implementation
- [internal/auth/manager.go](../../internal/auth/manager.go) - Credential management
- [cmd/spin-exec/runner.go](../../cmd/spin-exec/runner.go) - Current mock usage

---

## Phase 3: Interactive TUI (spin-tui) - BLOCKED ⚠️

**Status:** ⚠️ **BLOCKED** - Waiting for Phase 0 complete

**Blockers:**
- ❌ Need Phase 0.1: Approval response mechanism (critical for approval dialogs)
- ❌ Need Phase 0.2: Pause/Resume capability (important for interactive flow)
- ❌ Need Phase 0.3: Event backpressure control (critical for UI updates)
- ⚠️ Phase 0.4: Provider factory integration (important for real LLM usage)

**Implementation Strategy:**
- **Phases 3.1-3.10**: Build TUI infrastructure (can start while Phase 0 is being fixed)
- **Phase 3.11**: Connect to core module (⚡ requires Phase 0 complete)
- **Phase 3.12**: Error handling integration

### 3.1 Bubble Tea Application Setup

**DoR:**
- [ ] Bubble Tea framework studied (The Elm Architecture)
- [ ] Dependencies added (bubbletea, lipgloss, bubbles)
- [ ] TUI state machine designed

**Implementation:**
- [ ] Create `cmd/spin-tui/main.go`
- [ ] Implement Model struct (Bubble Tea model)
- [ ] Create AppState enum (Idle, WaitingResponse, ToolApproval, etc.)
- [ ] Implement Init() function
- [ ] Implement Update() function (message routing)
- [ ] Implement View() function (render pipeline)
- [ ] Add window resize handling

**DoD:**
- [ ] Tests for state transitions (≥85% coverage)
- [ ] Tests for message routing
- [ ] Basic TUI launches successfully
- [ ] Window resize works correctly
- [ ] Render latency <16ms (60 FPS)

---

### 3.2 Chat Interface Components

**DoR:**
- [ ] UI component spec reviewed
- [ ] Rendering pipeline understood
- [ ] Markdown/code highlighting requirements clear

**Implementation:**
- [ ] Create `cmd/spin-tui/ui/chat.go`
- [ ] Implement transcript rendering (viewport)
- [ ] Add streaming delta display
- [ ] Integrate glamour (markdown rendering)
- [ ] Integrate chroma (syntax highlighting)
- [ ] Add ANSI color preservation
- [ ] Implement reasoning block display

**DoD:**
- [ ] Tests for rendering (≥80% coverage)
- [ ] Snapshot tests for layout
- [ ] Markdown renders correctly
- [ ] Code blocks highlighted
- [ ] Streaming is smooth (no flickering)
- [ ] Memory usage <30MB

---

### 3.3 Input Widget & Multi-line Support

**DoR:**
- [ ] Input requirements reviewed (multi-line, paste support)
- [ ] Keyboard shortcuts defined
- [ ] @ file picker trigger understood

**Implementation:**
- [ ] Create `cmd/spin-tui/ui/input.go`
- [ ] Implement textinput.Model integration
- [ ] Add multi-line input support
- [ ] Implement paste handling (large text blocks)
- [ ] Add @ trigger detection
- [ ] Implement input history (up/down arrows)

**DoD:**
- [ ] Tests for input handling (≥90% coverage)
- [ ] Multi-line works correctly
- [ ] Paste doesn't freeze UI
- [ ] @ triggers file picker
- [ ] Input latency <5ms

---

### 3.4 File Picker Widget (@-trigger)

**DoR:**
- [ ] File search requirements reviewed
- [ ] Fuzzy search algorithm selected
- [ ] UI/UX flow designed

**Implementation:**
- [ ] Create `cmd/spin-tui/ui/filepicker.go`
- [ ] Create `internal/filesearch/` package
- [ ] Implement fuzzy search algorithm
- [ ] Add keyboard navigation (↑↓ arrows)
- [ ] Implement Tab/Enter selection
- [ ] Add instant path insertion
- [ ] Filter gitignored files

**DoD:**
- [ ] Tests for fuzzy search (≥90% coverage)
- [ ] Tests for keyboard navigation
- [ ] Search is fast (<50ms for 10k files)
- [ ] UI updates in real-time
- [ ] File picker can be dismissed (Esc)

---

### 3.5 Tool Approval UI

**DoR:**
- [ ] Tool approval UX designed
- [ ] Modal overlay pattern selected
- [ ] Approve/Deny/Modify flow defined

**Implementation:**
- [ ] Create `cmd/spin-tui/handlers/approval.go`
- [ ] Implement approval modal overlay
- [ ] Add [A]pprove / [D]eny / [M]odify keyboard handling
- [ ] Implement command modification editor
- [ ] Add confirmation display
- [ ] Send approval/denial to core

**DoD:**
- [ ] Tests for approval flow (≥90% coverage)
- [ ] Modal displays correctly
- [ ] All three actions work (A/D/M)
- [ ] Modification editor functional
- [ ] Core receives correct approval events

---

### 3.6 Status Bar

**DoR:**
- [ ] Status bar information list defined
- [ ] Layout specification reviewed
- [ ] Dynamic updates understood

**Implementation:**
- [ ] Create `cmd/spin-tui/ui/statusbar.go`
- [ ] Display current model
- [ ] Display sandbox mode (🔒/📝 icons)
- [ ] Display working directory
- [ ] Display connection status
- [ ] Display token usage (turn/session)
- [ ] Add color coding for status

**DoD:**
- [ ] Tests for status rendering (≥85% coverage)
- [ ] Status bar updates in real-time
- [ ] Layout works at different widths
- [ ] Icons display correctly
- [ ] Token count accurate

---

### 3.7 Transcript View & History

**DoR:**
- [ ] Transcript storage format defined
- [ ] Scroll behavior specified
- [ ] Search requirements understood

**Implementation:**
- [ ] Create `cmd/spin-tui/ui/transcript.go`
- [ ] Implement viewport for scrolling
- [ ] Add PgUp/PgDn navigation
- [ ] Implement search (/ key)
- [ ] Add export functionality (Ctrl+E)
- [ ] Optimize rendering (lazy loading)

**DoD:**
- [ ] Tests for transcript management (≥85% coverage)
- [ ] Scrolling is smooth
- [ ] Search works correctly
- [ ] Export produces valid output
- [ ] Large transcripts don't slow UI

---

### 3.8 Backtrack Mode (Esc-Esc)

**DoR:**
- [ ] Backtrack UX flow reviewed
- [ ] State management designed
- [ ] Conversation forking understood

**Implementation:**
- [ ] Create `cmd/spin-tui/handlers/backtrack.go`
- [ ] Implement Esc-Esc detection (when input empty)
- [ ] Add backtrack state management
- [ ] Highlight selected message
- [ ] Implement Esc to step backward
- [ ] Load message on Enter
- [ ] Handle conversation forking

**DoD:**
- [ ] Tests for backtrack flow (≥90% coverage)
- [ ] State transitions correct
- [ ] Message selection works
- [ ] Loading into input works
- [ ] Conversation forks properly

---

### 3.9 Keyboard Shortcuts & Events

**DoR:**
- [ ] All shortcuts documented
- [ ] Key binding conflicts checked
- [ ] Event handling pattern defined

**Implementation:**
- [ ] Create `cmd/spin-tui/handlers/keyboard.go`
- [ ] Implement all keyboard shortcuts (Enter, Ctrl+C, Ctrl+D, etc.)
- [ ] Add command cancellation (Ctrl+C)
- [ ] Implement screen clear (Ctrl+L)
- [ ] Add graceful exit (Ctrl+D)
- [ ] Handle special key combinations

**DoD:**
- [ ] Tests for all shortcuts (≥90% coverage)
- [ ] No key binding conflicts
- [ ] Cancellation works immediately
- [ ] Exit is graceful (cleanup done)
- [ ] Help text shows all shortcuts

---

### 3.10 TUI Styling & Themes

**DoR:**
- [ ] Color palette defined
- [ ] Lipgloss patterns studied
- [ ] NO_COLOR support specified

**Implementation:**
- [ ] Create `cmd/spin-tui/renderer/styles.go`
- [ ] Define color styles (User, Assistant, System, Error)
- [ ] Implement NO_COLOR support
- [ ] Add terminal color scheme detection
- [ ] Create layout utilities (lipgloss)
- [ ] Add responsive styling

**DoD:**
- [ ] Tests for style application (≥80% coverage)
- [ ] Colors display correctly
- [ ] NO_COLOR works
- [ ] Fallback to plain text works
- [ ] Styles are consistent

---

### 3.11 TUI Integration with Core

**DoR:**
- [ ] Core event protocol understood
- [ ] Channel-based communication pattern reviewed
- [ ] Async Bubble Tea commands studied

**Implementation:**
- [ ] Create `cmd/spin-tui/handlers/events.go`
- [ ] Implement waitForCoreEvent() Bubble Tea command
- [ ] Handle EventAssistantDelta (streaming)
- [ ] Handle EventToolCallProposed (approval)
- [ ] Handle EventTurnComplete (transition to idle)
- [ ] Handle errors from core
- [ ] Add reconnection logic

**DoD:**
- [ ] Tests with mock core events (≥90% coverage)
- [ ] Streaming works smoothly
- [ ] Tool approval triggers correctly
- [ ] State transitions correct
- [ ] Error handling works
- [ ] Reconnection succeeds

---

### 3.12 TUI Error Handling & Display

**DoR:**
- [ ] Error display strategy defined
- [ ] Error severity levels understood
- [ ] Recovery mechanisms specified

**Implementation:**
- [ ] Create `internal/tui/errors.go`
- [ ] Implement inline error display (in transcript)
- [ ] Add status bar error messages (transient)
- [ ] Create modal overlays (critical errors)
- [ ] Implement error recovery (auto-reconnect)
- [ ] Add dismissible error notifications

**DoD:**
- [ ] Tests for error scenarios (≥90% coverage)
- [ ] Errors display correctly
- [ ] Recovery works (network failures)
- [ ] Critical errors show modal
- [ ] User can dismiss errors
- [ ] No error causes crash

---

## Phase 4: Advanced Features & Polish

### 4.1 MCP Management Commands

**DoR:**
- [ ] MCP configuration format defined
- [ ] CRUD operations specified
- [ ] Integration with config system understood

**Implementation:**
- [ ] Create `cmd/spin/mcp.go`
- [ ] Implement `spin mcp add <name> <command> [args...]`
- [ ] Implement `spin mcp list`
- [ ] Implement `spin mcp get <name>`
- [ ] Implement `spin mcp remove <name>`
- [ ] Update config.toml with MCP servers

**DoD:**
- [ ] Tests for all MCP commands (≥90% coverage)
- [ ] Add/remove updates config correctly
- [ ] List displays all servers
- [ ] Get shows server details
- [ ] Integration with config system works

---

### 4.2 Config Management Commands

**DoR:**
- [ ] Config operations defined
- [ ] Validation requirements clear
- [ ] Interactive editing specified

**Implementation:**
- [ ] Create `cmd/spin/config.go`
- [ ] Implement `spin config show`
- [ ] Implement `spin config validate`
- [ ] Implement `spin config edit` (opens $EDITOR)
- [ ] Implement `spin config path` (show config file location)
- [ ] Add config migration support

**DoD:**
- [ ] Tests for config commands (≥90% coverage)
- [ ] Show displays current config
- [ ] Validate catches errors
- [ ] Edit opens correct editor
- [ ] Path returns correct location

---

### 4.3 Debug Commands

**DoR:**
- [ ] Debug features list reviewed
- [ ] Sandbox testing requirements understood
- [ ] Development utilities specified

**Implementation:**
- [ ] Create `cmd/spin/debug.go`
- [ ] Implement `spin debug sandbox <command>` (macOS)
- [ ] Implement `spin debug landlock <command>` (Linux)
- [ ] Add core event debugging
- [ ] Add LLM request/response logging
- [ ] Implement performance profiling helpers

**DoD:**
- [ ] Tests for debug commands (≥85% coverage)
- [ ] Sandbox testing works (macOS/Linux)
- [ ] Event debugging shows all events
- [ ] Request logging captures correctly
- [ ] Performance tools useful

---

### 4.4 Version & Update System

**DoR:**
- [ ] Versioning scheme defined (semver)
- [ ] Build info injection understood
- [ ] Update check mechanism designed

**Implementation:**
- [ ] Create `internal/version/version.go`
- [ ] Inject version at build time (-ldflags)
- [ ] Implement `spin version` command
- [ ] Add version display (--version flag)
- [ ] Show build info (commit, date, Go version)
- [ ] Add update check (optional)

**DoD:**
- [ ] Tests for version display (≥90% coverage)
- [ ] Version shows correctly
- [ ] Build info accurate
- [ ] Update check works (if enabled)
- [ ] Can be disabled via flag

---

### 4.5 Testing & Examples

**DoR:**
- [ ] Integration test strategy defined
- [ ] Example scenarios identified
- [ ] Test data prepared

**Implementation:**
- [ ] Create integration tests (`cmd/spin_test.go`)
- [ ] Add end-to-end TUI tests (with mock)
- [ ] Create exec mode integration tests
- [ ] Add example scripts (`examples/`)
- [ ] Create tutorial documentation
- [ ] Add CI/CD integration examples

**DoD:**
- [ ] Integration tests passing (≥80% coverage)
- [ ] All examples work
- [ ] Tutorial is complete
- [ ] CI/CD examples tested
- [ ] Documentation updated

---

### 4.6 Performance Optimization

**DoR:**
- [ ] Performance targets defined (from spec)
- [ ] Profiling tools ready
- [ ] Bottlenecks identified

**Implementation:**
- [ ] Run benchmarks (`go test -bench=.`)
- [ ] Profile CPU usage (pprof)
- [ ] Profile memory allocations
- [ ] Optimize hot paths (rendering, event processing)
- [ ] Add lazy loading where needed
- [ ] Implement viewport optimizations

**DoD:**
- [ ] TUI startup <80ms
- [ ] Exec startup <40ms
- [ ] TUI render latency <16ms
- [ ] Input latency <5ms
- [ ] Memory usage targets met
- [ ] Benchmark results documented

---

### 4.7 Documentation & Polish

**DoR:**
- [ ] Documentation structure planned
- [ ] User guide outline created
- [ ] Examples collected

**Implementation:**
- [ ] Write user guide (`docs/user-guide.md`)
- [ ] Create developer documentation (`docs/dev-guide.md`)
- [ ] Add architecture diagrams
- [ ] Create troubleshooting guide
- [ ] Write FAQ
- [ ] Add godoc for all packages

**DoD:**
- [ ] User guide complete
- [ ] Developer guide complete
- [ ] All diagrams created
- [ ] Troubleshooting covers common issues
- [ ] FAQ answers key questions
- [ ] godoc.org looks good

---

## Quality Gates

All features must pass these gates before being marked complete:

### Code Quality
- [ ] Tests passing (≥85% coverage overall, ≥90% for critical paths)
- [ ] Race detector clean (`go test -race`)
- [ ] Linter passing (`make lint`)
- [ ] Complexity ≤15 (cyclomatic complexity via gocyclo)
- [ ] No security issues (`gosec`)

### Documentation
- [ ] Godoc comments on all exports
- [ ] README updated
- [ ] Examples provided
- [ ] Architecture diagrams current

### Performance
- [ ] TUI startup <100ms
- [ ] TUI render <16ms
- [ ] Input latency <5ms
- [ ] Memory usage <30MB (TUI), <20MB (exec)
- [ ] Binary size <15MB (statically linked)

### User Experience
- [ ] Error messages actionable
- [ ] Help text complete
- [ ] Keyboard shortcuts documented
- [ ] Exit codes correct
- [ ] Logging appropriate

---

## Dependencies

### External Libraries

**TUI:**
- `github.com/charmbracelet/bubbletea` (TUI framework)
- `github.com/charmbracelet/lipgloss` (styling)
- `github.com/charmbracelet/bubbles` (UI components)
- `github.com/charmbracelet/glamour` (markdown)
- `github.com/alecthomas/chroma` (syntax highlighting)

**CLI:**
- `github.com/spf13/cobra` (CLI framework)

**Core:**
- `golang.org/x/sync/errgroup` (concurrency)
- `golang.org/x/term` (terminal utilities)

### Internal Dependencies
- `internal/core` (core business logic)
- `internal/llm` (LLM provider integration)
- `internal/auth` (authentication)
- `internal/protocol` (event protocol)

---

## Progress Tracking

### Phase 1: Foundation & CLI Framework
- [x] 1.1 Main CLI Entry Point ✅
- [ ] 1.2 Configuration Management
- [ ] 1.3 Logging Infrastructure

### Phase 2: Non-Interactive Mode ✅ **COMPLETE**
- [x] 2.1 Exec Command Structure ✅
- [x] 2.2 Output Formatting ✅
- [x] 2.3 Non-Interactive Tool Approval ✅ (integrated with core.Validator)
- [x] 2.4 Exec Integration with Core ✅

### Phase 3: Interactive TUI
- [ ] 3.1 Bubble Tea Application Setup
- [ ] 3.2 Chat Interface Components
- [ ] 3.3 Input Widget & Multi-line Support
- [ ] 3.4 File Picker Widget
- [ ] 3.5 Tool Approval UI
- [ ] 3.6 Status Bar
- [ ] 3.7 Transcript View & History
- [ ] 3.8 Backtrack Mode
- [ ] 3.9 Keyboard Shortcuts & Events
- [ ] 3.10 TUI Styling & Themes
- [ ] 3.11 TUI Integration with Core
- [ ] 3.12 TUI Error Handling

### Phase 4: Advanced Features
- [ ] 4.1 MCP Management Commands
- [ ] 4.2 Config Management Commands
- [ ] 4.3 Debug Commands
- [ ] 4.4 Version & Update System
- [ ] 4.5 Testing & Examples
- [ ] 4.6 Performance Optimization
- [ ] 4.7 Documentation & Polish

---

## Notes

- Follow TDD: write tests first, then implementation
- All functions must pass complexity checks (≤15)
- Use `internal/` for private packages
- Keep main packages (`cmd/`) thin (business logic in `internal/`)
- Regular integration with core module
- Continuous testing with race detector
- Performance profiling throughout development

---

## Success Criteria

The UI modules implementation is complete when:

1. **Functionality:**
   - All three modes work (CLI, TUI, exec)
   - All features from spec implemented
   - Integration with core module complete

2. **Quality:**
   - All tests passing (≥85% coverage)
   - All quality gates passed
   - No known critical bugs

3. **Performance:**
   - All performance targets met
   - Benchmarks documented
   - Resource usage within limits

4. **Documentation:**
   - User guide complete
   - Developer guide complete
   - All godoc comments present

5. **Polish:**
   - Error messages clear
   - UX smooth and intuitive
   - Help system comprehensive

---

## Core Integration Timeline

```
Phase 1: Foundation & CLI Framework
├── 1.1 Main CLI Entry Point ✅ (infrastructure only)
├── 1.2 Configuration Management (infrastructure only)
└── 1.3 Logging Infrastructure (infrastructure only)

Phase 2: Non-Interactive Mode (spin-exec)
├── 2.1 Exec Command Structure ✅ (infrastructure: args, signals, errors)
├── 2.2 Output Formatting (infrastructure: formatters)
├── 2.3 Non-Interactive Tool Approval (infrastructure: approval logic)
└── 2.4 Exec Integration with Core ⚡ CORE INTEGRATION HAPPENS HERE
    └── Connects 2.1-2.3 to internal/core module

Phase 3: Interactive TUI (spin-tui)
├── 3.1-3.10 TUI Infrastructure (UI components, styling, keyboard handling)
├── 3.11 TUI Integration with Core ⚡ CORE INTEGRATION HAPPENS HERE
└── 3.12 TUI Error Handling

Phase 4: Advanced Features & Polish
└── (Various enhancements)
```

**Key Points:**
- Infrastructure phases (2.1-2.3, 3.1-3.10) use **placeholder logic**
- Core integration happens in dedicated phases (2.4, 3.11)
- This allows parallel development and cleaner interfaces
- Each mode (exec/TUI) integrates with core independently

