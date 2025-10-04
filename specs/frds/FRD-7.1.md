# FRD-7.1: Conversation Implementation

## Overview

- **Module**: `internal/core`
- **Feature**: Conversation Implementation (Feature 7.1)
- **Goal**: Implement `Conversation` to coordinate turns, stream events, and manage lifecycle.

## Problem Statement

The core agent loop and supporting components exist, but an active conversation wrapper is missing. We need a thread-safe `Conversation` that:
- Executes a single user turn end-to-end via `Agent`
- Streams events to subscribers using `EventEmitter`
- Prevents overlapping/concurrent turns
- Supports graceful shutdown and state inspection

## Scope

In-scope:
- `Conversation` struct with session, history, agent, and event stream
- Methods: `RunTurn(ctx, userInput)`, `Stream() <-chan Event`, `Stop(ctx) error`, `State() State`
- Concurrency control to avoid overlapping `RunTurn`
- Event fan-out from `EventEmitter`
- Unit tests (>90% coverage)

Out-of-scope:
- Multi-conversation Manager (Feature 7.2)
- Provider/tool registry integration beyond existing mocks

## Definitions

- Turn: single interaction cycle processed by `Agent.Execute`
- State: simple conversation state string: `idle`, `running`, `stopped`

## Requirements

### Functional
- R1: `RunTurn` starts a turn if none is running; returns error if already running
- R2: `RunTurn` builds `AgentRequest` from history and context, calls `Agent.Execute`
- R3: Events from `Agent` must be emitted to the conversation stream
- R4: `Stream` returns a read-only channel for events, persists for conversation lifetime
- R5: `Stop` gracefully stops: closes event stream and marks state `stopped`
- R6: `State` returns current conversation state, thread-safe

### Non-Functional
- NF1: Thread-safe with mutexes
- NF2: No goroutine leaks; event channel closed on `Stop`
- NF3: Tests: unit tests >90%; race detector clean

## API

```go
// Conversation represents an active agent conversation
// Thread-safe for Stream/State; RunTurn serializes by mutex
type Conversation struct {
    session     *session.Session
    agent       *Agent
    history     *History
    emitter     *EventEmitter
    events      chan Event
    state       string // idle|running|stopped
    mu          sync.RWMutex
}

func NewConversation(sess *session.Session, agent *Agent, history *History, emitter *EventEmitter) *Conversation
func (c *Conversation) RunTurn(ctx context.Context, userInput string) error
func (c *Conversation) Stream() <-chan Event
func (c *Conversation) Stop(ctx context.Context) error
func (c *Conversation) State() string
```

## Design

- A dedicated `events` channel is created per conversation. The `Conversation` subscribes to the shared `EventEmitter` and forwards events into `events` using non-blocking send. On `Stop`, cancel subscription and close `events`.
- `RunTurn` acquires a lock to ensure no concurrent turn. It appends a user message to `history`, constructs an `AgentRequest`, calls `Agent.Execute`, and then appends the assistant response. Errors cause state transition to `idle` and surface to caller.
- Event forwarding runs while a turn is running; after completion, the stream remains open for further turns until `Stop`.

## Edge Cases
- Double `RunTurn` while running → error
- `Stop` while running → cancel context; return once agent returns; close `events`
- Subscribers after `Stop` read from closed channel without blocking

## Validation & Testing

- Unit tests for:
  - RunTurn success emits events and updates history
  - RunTurn error surfaces and state resets
  - Concurrency: second RunTurn during running returns error
  - Stream channel delivers EventTurnStart/Complete
  - Stop closes stream and moves to stopped state

- Run with:
```
go test -race ./internal/core/...
```

## DoD
- Code implemented with docs
- Tests >90% coverage for `conversation.go`
- Linters pass
- ROADMAP 7.1 updated to completed
