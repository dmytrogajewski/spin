# FRD-7.2: Conversation Manager

## Overview

- **Module**: `internal/core`
- **Feature**: Conversation Manager (Feature 7.2)
- **Goal**: Provide a high-level API to create, resume, list, and archive conversations, wiring dependencies (LLM provider, tools, security) and persisting sessions.

## Problem Statement

We need a `Manager` that owns configuration and dependencies and exposes ergonomic methods to manage conversations and their persistence. This decouples UI layers from low-level wiring and ensures consistent lifecycle and policy use.

## Scope

In-scope:
- `Manager` struct with options (functional options pattern)
- Methods: `NewConversation(ctx, workDir)`, `ResumeConversation(ctx, sessionID)`, `ListConversations(ctx, filter)`, `ArchiveConversation(ctx, sessionID)`
- Dependency injection for LLM provider, executor, validator, event emitter
- Session persistence (using session storage already in core)
- Unit tests (>90% coverage)

Out-of-scope:
- Multi-provider routing (Phase 8.1)
- Tools registry integration (Phase 8.2)
- Security sandbox integration (Phase 8.3)

## API

```go
// Manager coordinates conversation lifecycle and state
// Accepts interfaces and returns concrete types
type Manager struct {
    cfg      *Config
    llm      coretesting.LLMProvider
    emitter  *EventEmitter
    storage  session.Storage  // Added for full storage integration
}

type ManagerOption func(*Manager) error

func WithLLM(llm coretesting.LLMProvider) ManagerOption
func WithEmitter(e *EventEmitter) ManagerOption
func WithStorage(s session.Storage) ManagerOption

func NewManager(cfg *Config, opts ...ManagerOption) (*Manager, error)
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error)
func (m *Manager) ResumeConversation(ctx context.Context, sessionID string) (*Conversation, error)
func (m *Manager) ListConversations(ctx context.Context, filter any) ([]*session.Metadata, error)
func (m *Manager) ArchiveConversation(ctx context.Context, sessionID string) error
```

## Requirements

### Functional
- R1: Create conversation with initialized `Conversation` ✅
- R2: Resume conversation by session ID with full history restoration ✅
- R3: List returns session metadata with filtering support via session.Filter ✅
- R4: Archive marks session state as archived and persists ✅

### Non-Functional
- NF1: Use functional options; validate inputs ✅
- NF2: Thread-safe via storage layer ✅
- NF3: Tests coverage for manager (unit + integration) ✅

## Design

- Manager holds base `Config`, LLM provider, shared `EventEmitter`, and `session.Storage`.
- Storage is initialized automatically to `FileStorage(cfg.SessionDir)` if not provided via options.
- For `NewConversation`, create an `Executor`, `Validator`, `Context`, and `Agent`; initialize `History` with default system message; construct `Conversation` via `NewConversation(agent, history, emitter)`, and return it.
- For `ResumeConversation`, load full `session.Session` from storage, rebuild agent with session workdir, restore history from turns (user input + AI response for each turn), and return new conversation.
- For `ListConversations`, delegate to `storage.ListMetadata(filter)` and return metadata array.
- For `ArchiveConversation`, load session, set `State` to `StateArchived`, save back to storage.

## Implementation Notes

- **Import Cycle Fix**: Removed `Config` field from `session.Session` (was unused) to break cycle between `core` and `session` packages.
- **Session Constructor**: Updated `session.NewSession(workDir)` to take single parameter (no config).
- **History Restoration**: Iterates over `sess.Turns` and adds user/assistant messages to history. Tool messages not yet restored.

## Testing

### Unit Tests
- `TestNewManager_AndNewConversation`: Validates manager creation and conversation instantiation
- `TestManager_ListAndArchive`: Tests listing and archiving operations
- `TestManager_ResumeConversation_NotFound`: Verifies error for non-existent session

### Integration Tests (with real FileStorage)
- `TestManager_Integration_SaveResumeWithHistory`: Tests full save/resume cycle with history
- `TestManager_Integration_ListWithMetadata`: Tests listing multiple sessions with metadata
- `TestManager_Integration_Archive`: Tests archive operation and state persistence
- `TestManager_Integration_MultipleConversations`: Tests creating multiple independent conversations

All tests passing ✅

## DoD
- [x] Code with docs
- [x] Full storage integration (list, resume, archive)
- [x] History restoration from turns
- [x] Unit tests passing
- [x] Integration tests with FileStorage
- [x] Import cycle resolved
- [x] Roadmap updated
