# Core Module Implementation Roadmap

**Project:** Spin AI Agent - Core Module  
**Package:** `internal/core`  
**Target:** Go 1.24  
**Architecture:** Clean Architecture, SOLID principles  

---

## Overview

This roadmap breaks down the implementation of the `internal/core` package into manageable features with clear dependencies, Definition of Ready (DoR), and Definition of Done (DoD) criteria. The implementation follows a bottom-up approach, building foundational components before higher-level orchestration.

---

## Phase 0: Foundation & Setup

### Feature 0.1: Project Structure & Dependencies ✅

**Description:** Initialize the core module structure, setup dependency management, and establish coding standards.

**DoR:**
- [x] Go 1.24 installed and configured
- [x] Project root structure exists
- [x] Git repository initialized

**DoD:**
- [x] `internal/core/` directory structure created
- [x] `go.mod` with Go 1.24 and required dependencies
- [x] `.golangci.yml` configured with linters
- [x] `Makefile` with build, test, lint targets
- [x] `README.md` in `internal/core/` with package overview
- [x] All standard library imports verified
- [x] External dependencies documented (`golang.org/x/sync/errgroup`)

**Tasks:**
1. Create directory structure matching spec
2. Initialize go.mod with Go 1.24
3. Add external dependencies
4. Configure golangci-lint
5. Create Makefile
6. Write package-level documentation

**Priority:** P0 (Blocker)  
**Estimated Effort:** 4 hours

---

### Feature 0.2: Core Types & Errors ✅

**Description:** Define fundamental types, error definitions, and common interfaces used across the core module.

**DoR:**
- [x] Feature 0.1 completed
- [x] Error handling patterns documented

**DoD:**
- [x] `error.go` implemented with all core error types
- [x] Error wrapping with `Error` struct implemented
- [x] Error matching with `Is()` and `Unwrap()` implemented
- [x] Error codes defined as constants
- [x] Unit tests for error handling (92.2% coverage)
- [x] Error documentation with usage examples
- [x] Common type definitions (State, Filter, etc.)

**Tasks:**
1. Implement core error types (ErrInvalidInput, ErrSessionNotFound, etc.)
2. Implement Error struct with Op, Err, Code fields
3. Implement error wrapping and unwrapping
4. Write error handling tests
5. Define common types and constants
6. Document error handling patterns

**Priority:** P0 (Blocker)  
**Estimated Effort:** 6 hours

---

### Feature 0.3: Configuration System

**Description:** Implement the configuration system with file loading, environment variables, CLI flag merging, and validation.

**DoR:**
- [ ] Feature 0.2 completed
- [ ] Configuration schema defined
- [ ] Default values determined

**DoD:**
- [ ] `config.go` implemented with Config struct
- [ ] `Load()` function for YAML config files
- [ ] `Validate()` method with comprehensive checks
- [ ] `Merge()` method for configuration precedence
- [ ] Environment variable support (SPIN_*)
- [ ] Configuration defaults defined
- [ ] Unit tests for all config scenarios (>90% coverage)
- [ ] Example config file (`config.example.yaml`)
- [ ] Configuration documentation

**Tasks:**
1. Define Config struct with all fields
2. Implement YAML loading (use `gopkg.in/yaml.v3`)
3. Implement validation logic
4. Implement merge logic (CLI > Env > File > Defaults)
5. Add environment variable parsing
6. Write comprehensive config tests
7. Create example configuration file
8. Document configuration options

**Priority:** P0 (Blocker)  
**Estimated Effort:** 8 hours

---

## Phase 1: State Management

### Feature 1.1: Session Management

**Description:** Implement session state management including session creation, persistence, and retrieval.

**DoR:**
- [ ] Feature 0.3 completed
- [ ] Storage format decided (JSON/Protobuf)
- [ ] Session directory structure defined

**DoD:**
- [ ] `session/session.go` implemented
- [ ] `session/storage.go` with persistence layer
- [ ] `session/metadata.go` with session metadata
- [ ] Session CRUD operations (Create, Read, Update, Delete)
- [ ] File-based storage in `~/.spin/sessions/`
- [ ] Session ID generation (UUIDs)
- [ ] Concurrent access safety (mutex protection)
- [ ] Unit tests for session operations (>90% coverage)
- [ ] Integration tests for persistence
- [ ] Session migration support (version field)

**Tasks:**
1. Implement Session struct with all fields
2. Implement NewSession() constructor
3. Implement AddTurn() method
4. Implement Save() persistence method
5. Implement Load() retrieval method
6. Implement storage backend (file-based)
7. Add session metadata handling
8. Write unit tests for session logic
9. Write integration tests for storage
10. Document session lifecycle

**Priority:** P0 (Blocker)  
**Estimated Effort:** 12 hours

---

### Feature 1.2: Turn State Machine

**Description:** Implement turn state management with state transitions and turn execution tracking.

**DoR:**
- [ ] Feature 1.1 completed
- [ ] Turn state machine diagram defined
- [ ] State transition rules documented

**DoD:**
- [ ] `turn/turn.go` implemented with Turn struct
- [ ] `turn/state.go` with TurnState enum and transitions
- [ ] `turn/result.go` with turn execution results
- [ ] All TurnState constants defined
- [ ] State transition validation
- [ ] Turn ID generation
- [ ] Token usage tracking
- [ ] Timestamp tracking (start/complete)
- [ ] Unit tests for state machine (>90% coverage)
- [ ] State transition tests (all valid/invalid paths)
- [ ] Turn serialization/deserialization

**Tasks:**
1. Define Turn struct with all fields
2. Define TurnState enum (Pending, Running, WaitingApproval, Completed, Failed, Cancelled)
3. Implement state transition logic
4. Implement Execute() method stub
5. Implement result tracking
6. Add token usage tracking
7. Write state machine tests
8. Write turn lifecycle tests
9. Document turn states and transitions

**Priority:** P0 (Blocker)  
**Estimated Effort:** 10 hours

---

### Feature 1.3: History Management

**Description:** Implement conversation history with token-aware truncation and message management.

**DoR:**
- [ ] Feature 1.2 completed
- [ ] Tokenizer interface defined (from llm package)
- [ ] Truncation strategy documented

**DoD:**
- [ ] `history.go` implemented with History struct
- [ ] AddMessage() method with thread-safety
- [ ] Messages() retrieval method
- [ ] Truncate() with token budget awareness
- [ ] Export() for saving history to file
- [ ] Message struct with all fields (Role, Content, ToolCalls, etc.)
- [ ] Token counting integration
- [ ] Smart truncation (preserve system message)
- [ ] Unit tests for history operations (>90% coverage)
- [ ] Truncation algorithm tests
- [ ] Concurrent access tests

**Tasks:**
1. Implement History struct with message slice
2. Implement Message struct
3. Add thread-safe AddMessage() with mutex
4. Implement Messages() accessor
5. Implement token-aware Truncate()
6. Add Export() for history persistence
7. Write history management tests
8. Write truncation strategy tests
9. Test concurrent access scenarios
10. Document history management

**Priority:** P1 (Critical)  
**Estimated Effort:** 10 hours

---

## Phase 2: Safety & Execution

### Feature 2.1: Command Validator

**Description:** Implement command safety classification with pattern matching against security policies.

**DoR:**
- [ ] Feature 0.3 completed
- [ ] Security patterns documented
- [ ] Command classification rules defined
- [ ] `internal/security` package available

**DoD:**
- [ ] `validator.go` implemented with Validator struct
- [ ] CommandClass enum (Safe, Interactive, Dangerous, Forbidden)
- [ ] Classify() method with pattern matching
- [ ] IsSafe() and IsDangerous() helper methods
- [ ] Pattern-based command classification
- [ ] Security policy integration
- [ ] Unit tests for all command classes (>95% coverage)
- [ ] Test cases for safe commands (ls, cat, git status, etc.)
- [ ] Test cases for dangerous commands (rm -rf, sudo, etc.)
- [ ] Test cases for forbidden commands (fork bomb, etc.)
- [ ] Classification documentation

**Tasks:**
1. Define CommandClass enum
2. Implement Validator struct
3. Implement Classify() with pattern matching
4. Define safe command patterns
5. Define interactive command patterns
6. Define dangerous command patterns
7. Define forbidden command patterns
8. Implement IsSafe() helper
9. Implement IsDangerous() helper
10. Write comprehensive classification tests
11. Document command classification rules

**Priority:** P0 (Blocker - Security Critical)  
**Estimated Effort:** 12 hours

---

### Feature 2.2: Command Executor

**Description:** Implement safe command execution with sandboxing, timeouts, and result capture.

**DoR:**
- [ ] Feature 2.1 completed
- [ ] `internal/security` sandbox available
- [ ] Execution environment defined
- [ ] Timeout policies defined

**DoD:**
- [ ] `executor.go` implemented with Executor struct
- [ ] Execute() method with full execution flow
- [ ] ExecuteStreaming() for long-running commands
- [ ] Validate() pre-execution checks
- [ ] Sandbox integration
- [ ] Timeout enforcement (context-based)
- [ ] Output capture (stdout/stderr)
- [ ] Exit code handling
- [ ] Environment variable management
- [ ] Working directory management
- [ ] Unit tests for executor (>90% coverage)
- [ ] Integration tests with real commands
- [ ] Timeout tests
- [ ] Sandbox integration tests

**Tasks:**
1. Implement Executor struct
2. Implement Execute() with validation flow
3. Add policy checking integration
4. Add sandbox wrapping
5. Implement command execution (os/exec)
6. Add timeout handling with context
7. Implement output capture
8. Implement ExecuteStreaming() for streaming output
9. Add environment management
10. Write executor unit tests
11. Write integration tests
12. Document execution flow

**Priority:** P0 (Blocker - Security Critical)  
**Estimated Effort:** 14 hours

---

### Feature 2.3: Task Planner

**Description:** Implement task decomposition and planning for complex multi-step operations.

**DoR:**
- [ ] Feature 0.3 completed
- [ ] LLM provider interface available
- [ ] Planning prompt templates defined

**DoD:**
- [ ] `planner.go` implemented with Planner struct
- [ ] Plan() method for task decomposition
- [ ] Plan struct with steps and dependencies
- [ ] Step struct with ID, description, dependencies
- [ ] Dependency graph construction
- [ ] Step status tracking
- [ ] Estimated duration calculation
- [ ] LLM integration for planning
- [ ] Unit tests for planner (>85% coverage)
- [ ] Integration tests with mock LLM
- [ ] Dependency resolution tests
- [ ] Planning documentation

**Tasks:**
1. Implement Planner struct
2. Define Plan struct
3. Define Step struct with dependencies
4. Implement Plan() method
5. Add LLM integration for decomposition
6. Implement dependency graph building
7. Add step status tracking
8. Add duration estimation
9. Write planner tests
10. Write dependency resolution tests
11. Document planning process

**Priority:** P2 (Nice to Have)  
**Estimated Effort:** 10 hours

---

## Phase 3: Context & Environment

### Feature 3.1: Environment Context Gathering

**Description:** Implement comprehensive environment context collection including OS, Git, project structure, and environment variables.

**DoR:**
- [ ] Feature 0.2 completed
- [ ] Context gathering requirements documented
- [ ] Privacy/security considerations defined

**DoD:**
- [ ] `context.go` implemented with Context struct
- [ ] Gather() function for context collection
- [ ] OSInfo gathering (OS, arch, kernel, shell)
- [ ] GitInfo gathering (branch, changes, remotes)
- [ ] FileInfo scanning (project structure)
- [ ] Project type detection (Go, Python, Node.js, etc.)
- [ ] Language detection from files
- [ ] Environment variable filtering (exclude sensitive vars)
- [ ] Git repository detection and info extraction
- [ ] Unit tests for context gathering (>85% coverage)
- [ ] Integration tests with real project directories
- [ ] Tests for non-git directories
- [ ] Sensitive data filtering tests
- [ ] Context serialization for LLM prompts

**Tasks:**
1. Implement Context struct with all info types
2. Implement OSInfo struct and gathering
3. Implement GitInfo struct and gathering
4. Implement FileInfo struct and scanning
5. Implement Gather() orchestration
6. Add project type detection logic
7. Add language detection from extensions
8. Implement environment variable filtering
9. Add Git repository detection
10. Write context gathering tests
11. Write integration tests
12. Document context structure

**Priority:** P1 (Critical)  
**Estimated Effort:** 14 hours

---

## Phase 4: Event System

### Feature 4.1: Event Infrastructure

**Description:** Implement event streaming system for real-time UI updates and monitoring.

**DoR:**
- [ ] Feature 0.2 completed
- [ ] Event types documented
- [ ] Event flow diagrams created

**DoD:**
- [ ] `event.go` implemented with Event struct
- [ ] EventType enum with all event types
- [ ] EventEmitter struct with pub/sub pattern
- [ ] Emit() method for event publication
- [ ] Subscribe() method for event subscription
- [ ] Thread-safe event distribution
- [ ] Event buffering strategy
- [ ] Event timestamp tracking
- [ ] Event data type safety
- [ ] Unit tests for event system (>90% coverage)
- [ ] Concurrent subscription tests
- [ ] Event ordering tests
- [ ] Subscriber cleanup tests

**Tasks:**
1. Define Event struct
2. Define EventType enum (ContentDelta, ToolCallStart, etc.)
3. Implement EventEmitter struct
4. Implement Emit() with fan-out to subscribers
5. Implement Subscribe() returning event channel
6. Add thread-safety with RWMutex
7. Add event buffering
8. Implement unsubscribe mechanism
9. Write event system tests
10. Write concurrency tests
11. Document event system usage

**Priority:** P1 (Critical)  
**Estimated Effort:** 10 hours

---

### Feature 4.2: Stream Management

**Description:** Implement stream handling, buffering, and event type management for LLM streaming.

**DoR:**
- [ ] Feature 4.1 completed
- [ ] Streaming requirements defined
- [ ] Buffer sizing determined

**DoD:**
- [ ] `stream/stream.go` implemented
- [ ] `stream/buffer.go` with smart buffering
- [ ] `stream/types.go` with stream event types
- [ ] Stream event handling
- [ ] Backpressure management
- [ ] Buffer overflow handling
- [ ] Stream completion detection
- [ ] Error propagation in streams
- [ ] Unit tests for stream handling (>90% coverage)
- [ ] Backpressure tests
- [ ] Buffer tests
- [ ] Stream lifecycle tests

**Tasks:**
1. Implement stream event types
2. Implement stream handler
3. Add buffer management
4. Implement backpressure handling
5. Add stream completion detection
6. Add error propagation
7. Write stream tests
8. Write buffer tests
9. Document streaming architecture

**Priority:** P1 (Critical)  
**Estimated Effort:** 8 hours

---

## Phase 5: Task Execution Modes

### Feature 5.1: Task Interface & Registry

**Description:** Define the Task interface and implement task registry for different execution modes.

**DoR:**
- [ ] Feature 0.3 completed
- [ ] Task modes documented
- [ ] Tool restrictions per mode defined

**DoD:**
- [ ] `task/task.go` with Task interface
- [ ] Task interface methods defined (Name, SystemPrompt, AllowedTools, MaxTokens, Validate)
- [ ] Task registry for mode management
- [ ] Task validation framework
- [ ] Task switching mechanism
- [ ] Unit tests for task interface (>85% coverage)
- [ ] Registry tests
- [ ] Task interface documentation

**Tasks:**
1. Define Task interface
2. Implement task registry
3. Add task registration mechanism
4. Implement task lookup
5. Add task validation
6. Write interface tests
7. Document task system

**Priority:** P1 (Critical)  
**Estimated Effort:** 6 hours

---

### Feature 5.2: Regular Task Mode

**Description:** Implement standard interactive coding mode with full tool access.

**DoR:**
- [ ] Feature 5.1 completed
- [ ] Regular mode requirements defined
- [ ] System prompts written

**DoD:**
- [ ] `task/regular.go` implemented
- [ ] Regular struct implementing Task interface
- [ ] Full tool set enabled (read_file, write_file, shell, search, git)
- [ ] Regular mode system prompt
- [ ] Standard token budget
- [ ] Configuration options
- [ ] Unit tests for regular mode (>90% coverage)
- [ ] Integration tests with executor
- [ ] Mode documentation

**Tasks:**
1. Implement Regular struct
2. Implement Name() returning "regular"
3. Implement SystemPrompt() with regular prompt
4. Implement AllowedTools() with full tool list
5. Implement MaxTokens() with standard budget
6. Implement Validate()
7. Write regular mode tests
8. Document regular mode usage

**Priority:** P1 (Critical)  
**Estimated Effort:** 4 hours

---

### Feature 5.3: Review Task Mode

**Description:** Implement read-only code review mode with restricted tool access.

**DoR:**
- [ ] Feature 5.1 completed
- [ ] Review mode requirements defined
- [ ] Review system prompts written

**DoD:**
- [ ] `task/review.go` implemented
- [ ] Review struct implementing Task interface
- [ ] Read-only tools (read_file, search)
- [ ] Review mode system prompt
- [ ] File list configuration
- [ ] Validation for read-only operations
- [ ] Unit tests for review mode (>90% coverage)
- [ ] Tool restriction tests
- [ ] Mode documentation

**Tasks:**
1. Implement Review struct
2. Implement Name() returning "review"
3. Implement SystemPrompt() with review prompt
4. Implement AllowedTools() with read-only tools
5. Add file list management
6. Implement validation
7. Write review mode tests
8. Test tool restrictions
9. Document review mode

**Priority:** P2 (Nice to Have)  
**Estimated Effort:** 4 hours

---

### Feature 5.4: Compact Task Mode

**Description:** Implement minimal context mode for constrained environments or quick tasks.

**DoR:**
- [ ] Feature 5.1 completed
- [ ] Compact mode requirements defined
- [ ] Token budget constraints determined

**DoD:**
- [ ] `task/compact.go` implemented
- [ ] Compact struct implementing Task interface
- [ ] Reduced token budget (4096 tokens)
- [ ] Minimal context system prompt
- [ ] Limited tool set for efficiency
- [ ] Fast response optimization
- [ ] Unit tests for compact mode (>90% coverage)
- [ ] Token budget tests
- [ ] Mode documentation

**Tasks:**
1. Implement Compact struct
2. Implement Name() returning "compact"
3. Implement SystemPrompt() with minimal prompt
4. Implement MaxTokens() returning 4096
5. Implement limited tool set
6. Write compact mode tests
7. Test token constraints
8. Document compact mode

**Priority:** P2 (Nice to Have)  
**Estimated Effort:** 4 hours

---

## Phase 6: Agent Core

### Feature 6.1: Agent Orchestration

**Description:** Implement the core agent decision loop with LLM interaction and tool execution coordination.

**DoR:**
- [ ] Features 2.1, 2.2, 3.1, 4.1 completed
- [ ] `internal/llm` Provider interface available
- [ ] `internal/tools` Registry available
- [ ] Agent loop algorithm defined

**DoD:**
- [ ] `agent.go` implemented with Agent struct
- [ ] Execute() method with agent loop
- [ ] ProcessToolCall() for tool invocations
- [ ] ShouldApprove() for approval decisions
- [ ] buildPrompt() for context construction
- [ ] LLM streaming integration
- [ ] Tool call accumulation and execution
- [ ] Multi-turn loop with max turns limit
- [ ] Timeout enforcement
- [ ] Finish reason detection
- [ ] Unit tests for agent loop (>85% coverage)
- [ ] Integration tests with mock LLM
- [ ] Tool execution tests
- [ ] Approval logic tests
- [ ] Error handling tests

**Tasks:**
1. Implement Agent struct with dependencies
2. Implement Execute() with main agent loop
3. Implement prompt building logic
4. Add LLM streaming integration
5. Implement tool call processing
6. Add tool result handling
7. Implement ShouldApprove() logic
8. Add timeout and max turns enforcement
9. Implement finish reason detection
10. Write agent tests with mocks
11. Write integration tests
12. Document agent loop flow

**Priority:** P0 (Blocker)  
**Estimated Effort:** 20 hours

---

### Feature 6.2: Tool Call Processing

**Description:** Implement robust tool call handling with validation, execution, and result formatting.

**DoR:**
- [ ] Feature 6.1 in progress
- [ ] Tool call format defined
- [ ] Tool result format defined

**DoD:**
- [ ] Tool call parsing and validation
- [ ] Tool parameter extraction
- [ ] Tool execution coordination
- [ ] Result formatting for LLM
- [ ] Error handling for tool failures
- [ ] Streaming tool output support
- [ ] Tool timeout handling
- [ ] Concurrent tool execution (if safe)
- [ ] Unit tests for tool processing (>90% coverage)
- [ ] Error scenario tests
- [ ] Timeout tests

**Tasks:**
1. Implement tool call parsing
2. Add parameter validation
3. Implement tool lookup in registry
4. Add tool execution coordination
5. Implement result formatting
6. Add error handling and recovery
7. Implement streaming support
8. Add timeout handling
9. Write tool call tests
10. Document tool call flow

**Priority:** P0 (Blocker)  
**Estimated Effort:** 12 hours

---

## Phase 7: Conversation Management

### Feature 7.1: Conversation Implementation

**Description:** Implement active conversation with turn execution and event streaming.

**DoR:**
- [ ] Features 1.1, 1.2, 1.3, 4.1, 6.1 completed
- [ ] Conversation state machine defined

**DoD:**
- [ ] `conversation.go` implemented with Conversation struct
- [ ] RunTurn() method for turn execution
- [ ] Stream() method returning event channel
- [ ] Stop() method for graceful shutdown
- [ ] State() method for state access
- [ ] Conversation lifecycle management
- [ ] Event emission during turns
- [ ] Concurrent turn safety (prevent overlapping turns)
- [ ] Graceful cancellation
- [ ] Unit tests for conversation (>90% coverage)
- [ ] Turn execution tests
- [ ] Cancellation tests
- [ ] Event streaming tests

**Tasks:**
1. Implement Conversation struct
2. Implement RunTurn() with turn coordination
3. Add event stream setup
4. Implement Stream() accessor
5. Implement Stop() with cleanup
6. Add State() accessor
7. Implement turn safety (mutex)
8. Add cancellation handling
9. Write conversation tests
10. Write lifecycle tests
11. Document conversation usage

**Priority:** P0 (Blocker)  
**Estimated Effort:** 16 hours

---

### Feature 7.2: Conversation Manager

**Description:** Implement high-level conversation manager for lifecycle, persistence, and multi-conversation management.

**DoR:**
- [ ] Features 1.1, 7.1 completed
- [ ] Storage backend available
- [ ] Manager responsibilities defined

**DoD:**
- [ ] `manager.go` implemented with Manager struct
- [ ] NewManager() constructor with options
- [ ] NewConversation() for starting conversations
- [ ] ResumeConversation() for resuming from storage
- [ ] ListConversations() with filtering
- [ ] ArchiveConversation() for cleanup
- [ ] Dependency injection (LLM, tools, security)
- [ ] Provider management
- [ ] Functional options pattern
- [ ] Unit tests for manager (>90% coverage)
- [ ] Integration tests with storage
- [ ] Multi-conversation tests
- [ ] Resume tests

**Tasks:**
1. Implement Manager struct
2. Implement NewManager() with options
3. Implement functional options (WithProvider, etc.)
4. Implement NewConversation()
5. Implement ResumeConversation()
6. Implement ListConversations() with filtering
7. Implement ArchiveConversation()
8. Add dependency coordination
9. Write manager tests
10. Write storage integration tests
11. Document manager API

**Priority:** P0 (Blocker)  
**Estimated Effort:** 14 hours

---

## Phase 8: Integration & Polish

### Feature 8.1: LLM Provider Integration

**Description:** Integrate with `internal/llm` provider interface for multi-provider support.

**DoR:**
- [ ] Feature 6.1 completed
- [ ] `internal/llm` package available
- [ ] Provider interface defined

**DoD:**
- [ ] Provider interface integration
- [ ] Multi-provider support in Manager
- [ ] Provider switching mechanism
- [ ] Streaming integration verified
- [ ] Tool schema passing
- [ ] Token counting integration
- [ ] Integration tests with mock provider
- [ ] Integration tests with real providers (if available)
- [ ] Error handling for provider failures
- [ ] Provider documentation

**Tasks:**
1. Integrate Provider interface
2. Implement provider switching
3. Add streaming integration
4. Add tool schema passing
5. Integrate token counting
6. Write provider integration tests
7. Test with multiple providers
8. Document provider usage

**Priority:** P1 (Critical)  
**Estimated Effort:** 8 hours

---

### Feature 8.2: Tool Registry Integration

**Description:** Integrate with `internal/tools` registry for tool management.

**DoR:**
- [ ] Feature 6.2 completed
- [ ] `internal/tools` package available
- [ ] Tool interface defined

**DoD:**
- [ ] Tool registry integration
- [ ] Tool schema retrieval
- [ ] Tool execution via registry
- [ ] Tool parameter validation
- [ ] Custom tool registration support
- [ ] Tool filtering by task mode
- [ ] Integration tests with tool registry
- [ ] Tool execution tests
- [ ] Tool documentation

**Tasks:**
1. Integrate tools.Registry
2. Implement tool schema retrieval
3. Add tool execution coordination
4. Implement parameter validation
5. Add tool filtering logic
6. Write tool integration tests
7. Document tool usage

**Priority:** P1 (Critical)  
**Estimated Effort:** 6 hours

---

### Feature 8.3: Security Integration

**Description:** Integrate with `internal/security` for sandbox and policy enforcement.

**DoR:**
- [ ] Feature 2.2 completed
- [ ] `internal/security` package available
- [ ] Security policy format defined

**DoD:**
- [ ] Sandbox integration in Executor
- [ ] Policy loading and validation
- [ ] Policy enforcement in validator
- [ ] Sandbox mode configuration
- [ ] Security violation handling
- [ ] Audit logging for security events
- [ ] Integration tests with security module
- [ ] Security policy tests
- [ ] Security documentation

**Tasks:**
1. Integrate security.Sandbox
2. Integrate security.Policy
3. Add policy enforcement
4. Implement sandbox wrapping
5. Add security audit logging
6. Write security integration tests
7. Document security features

**Priority:** P0 (Blocker - Security Critical)  
**Estimated Effort:** 8 hours

---

### Feature 8.4: MCP Integration

**Description:** Integrate Model Context Protocol (MCP) for extended context capabilities.

**DoR:**
- [ ] Feature 6.1 completed
- [ ] `internal/mcp` package available
- [ ] MCP configuration defined

**DoD:**
- [ ] MCP client integration
- [ ] MCP server configuration
- [ ] MCP resource access
- [ ] MCP tool invocation
- [ ] MCP context extension
- [ ] Configuration-based MCP enablement
- [ ] Integration tests with MCP
- [ ] MCP documentation

**Tasks:**
1. Integrate mcp.Client
2. Add MCP configuration
3. Implement MCP resource access
4. Add MCP tool support
5. Write MCP integration tests
6. Document MCP usage

**Priority:** P2 (Nice to Have)  
**Estimated Effort:** 10 hours

---

### Feature 8.5: Comprehensive Testing Suite

**Description:** Implement comprehensive test coverage including unit, integration, and end-to-end tests.

**DoR:**
- [ ] All Phase 0-7 features completed
- [ ] Test utilities implemented

**DoD:**
- [ ] Unit test coverage >90% for all packages
- [ ] Integration tests for all major flows
- [ ] End-to-end tests for complete conversations
- [ ] Benchmark tests for performance-critical paths
- [ ] Race condition tests (`go test -race`)
- [ ] Table-driven tests for complex logic
- [ ] Mock implementations (MockProvider, MockTools)
- [ ] Test helpers and utilities
- [ ] Testdata fixtures
- [ ] CI/CD pipeline integration
- [ ] Test documentation

**Tasks:**
1. Review and fill coverage gaps
2. Implement integration test suite
3. Create end-to-end test scenarios
4. Add benchmark tests
5. Run race detector tests
6. Create mock implementations
7. Build test helper library
8. Organize testdata fixtures
9. Configure CI/CD testing
10. Document testing strategy

**Priority:** P0 (Blocker)  
**Estimated Effort:** 24 hours

---

### Feature 8.6: Performance Optimization

**Description:** Optimize performance for streaming, concurrency, context management, and caching.

**DoR:**
- [ ] Feature 8.5 completed
- [ ] Performance benchmarks established
- [ ] Performance targets defined

**DoD:**
- [ ] Streaming optimized (minimal buffering)
- [ ] Concurrent operations with errgroup
- [ ] Context truncation optimization
- [ ] Command result caching implemented
- [ ] Channel buffer sizing optimized
- [ ] Memory profiling completed
- [ ] CPU profiling completed
- [ ] Benchmark tests showing improvements
- [ ] Performance documentation

**Tasks:**
1. Profile memory usage
2. Profile CPU usage
3. Optimize streaming paths
4. Implement concurrent context gathering
5. Add context truncation optimization
6. Implement command caching
7. Optimize channel buffers
8. Run benchmark comparisons
9. Document performance characteristics

**Priority:** P2 (Nice to Have)  
**Estimated Effort:** 16 hours

---

### Feature 8.7: Observability & Debugging

**Description:** Implement comprehensive logging, tracing, and debugging capabilities.

**DoR:**
- [ ] All core features completed
- [ ] Logging strategy defined

**DoD:**
- [ ] Structured logging with `log/slog`
- [ ] Debug mode with verbose logging
- [ ] OpenTelemetry tracing integration (optional)
- [ ] Span creation for major operations
- [ ] Context propagation for tracing
- [ ] Environment variable configuration (SPIN_DEBUG, SPIN_TRACE)
- [ ] Log level configuration
- [ ] Error logging with context
- [ ] Performance tracing
- [ ] Debugging documentation

**Tasks:**
1. Add structured logging throughout
2. Implement debug mode
3. Integrate OpenTelemetry (optional)
4. Add tracing spans
5. Configure log levels
6. Add error logging
7. Write debugging guide
8. Document observability features

**Priority:** P1 (Critical)  
**Estimated Effort:** 10 hours

---

### Feature 8.8: Documentation & Examples

**Description:** Create comprehensive documentation, examples, and godoc.

**DoR:**
- [ ] All features completed
- [ ] Documentation standards defined

**DoD:**
- [ ] Godoc comments for all exported symbols
- [ ] Package-level documentation
- [ ] Architecture documentation (this document)
- [ ] API reference generated
- [ ] Usage examples in `examples/` directory
- [ ] Runnable example programs
- [ ] Tutorial documentation
- [ ] Troubleshooting guide
- [ ] Contributing guidelines
- [ ] README for core package

**Tasks:**
1. Write godoc for all exported symbols
2. Add package documentation
3. Create usage examples
4. Write tutorial documentation
5. Create troubleshooting guide
6. Write API reference
7. Generate documentation site
8. Review and polish documentation

**Priority:** P1 (Critical)  
**Estimated Effort:** 16 hours

---

## Dependency Graph

```
Phase 0 (Foundation)
  ├─ 0.1: Project Structure
  ├─ 0.2: Core Types & Errors
  └─ 0.3: Configuration System

Phase 1 (State Management)
  ├─ 1.1: Session Management [depends on 0.3]
  ├─ 1.2: Turn State Machine [depends on 1.1]
  └─ 1.3: History Management [depends on 1.2]

Phase 2 (Safety & Execution)
  ├─ 2.1: Command Validator [depends on 0.3]
  ├─ 2.2: Command Executor [depends on 2.1]
  └─ 2.3: Task Planner [depends on 0.3]

Phase 3 (Context & Environment)
  └─ 3.1: Environment Context [depends on 0.2]

Phase 4 (Event System)
  ├─ 4.1: Event Infrastructure [depends on 0.2]
  └─ 4.2: Stream Management [depends on 4.1]

Phase 5 (Task Modes)
  ├─ 5.1: Task Interface [depends on 0.3]
  ├─ 5.2: Regular Mode [depends on 5.1]
  ├─ 5.3: Review Mode [depends on 5.1]
  └─ 5.4: Compact Mode [depends on 5.1]

Phase 6 (Agent Core)
  ├─ 6.1: Agent Orchestration [depends on 2.1, 2.2, 3.1, 4.1]
  └─ 6.2: Tool Call Processing [depends on 6.1]

Phase 7 (Conversation Management)
  ├─ 7.1: Conversation Implementation [depends on 1.1, 1.2, 1.3, 4.1, 6.1]
  └─ 7.2: Conversation Manager [depends on 1.1, 7.1]

Phase 8 (Integration & Polish)
  ├─ 8.1: LLM Provider Integration [depends on 6.1]
  ├─ 8.2: Tool Registry Integration [depends on 6.2]
  ├─ 8.3: Security Integration [depends on 2.2]
  ├─ 8.4: MCP Integration [depends on 6.1]
  ├─ 8.5: Comprehensive Testing [depends on all above]
  ├─ 8.6: Performance Optimization [depends on 8.5]
  ├─ 8.7: Observability & Debugging [depends on all core features]
  └─ 8.8: Documentation & Examples [depends on all features]
```

---

## Implementation Timeline

**Estimated Total Effort:** 242 hours (~6 weeks at 40 hours/week)

### Week 1-2: Foundation & State
- Phase 0: Foundation & Setup (18 hours)
- Phase 1: State Management (32 hours)

### Week 3: Safety & Context
- Phase 2: Safety & Execution (36 hours)
- Phase 3: Context & Environment (14 hours)

### Week 4: Events & Tasks
- Phase 4: Event System (18 hours)
- Phase 5: Task Execution Modes (18 hours)

### Week 5: Core Agent
- Phase 6: Agent Core (32 hours)

### Week 6: Conversation & Integration
- Phase 7: Conversation Management (30 hours)
- Phase 8: Integration & Polish (Begin - 44 hours remaining)

### Week 7+: Polish & Documentation
- Phase 8: Integration & Polish (Complete remaining ~44 hours)

---

## Risk Assessment

### High Risk
- **Agent Loop Complexity:** The agent decision loop is complex with multiple state transitions. Mitigation: Incremental implementation with extensive testing.
- **Concurrency Issues:** Event streaming and turn execution have concurrency challenges. Mitigation: Use proven patterns (channels, mutexes), extensive race detector testing.
- **Security Vulnerabilities:** Command execution has inherent security risks. Mitigation: Defense in depth with validator, sandbox, and policy enforcement.

### Medium Risk
- **LLM Integration:** Dependencies on external LLM providers. Mitigation: Mock providers for testing, graceful error handling.
- **Performance:** Streaming and context management may have performance bottlenecks. Mitigation: Early benchmarking, profiling, optimization phase.
- **Token Management:** Context window limits require careful truncation. Mitigation: Smart truncation algorithm, testing with various history sizes.

### Low Risk
- **Configuration Management:** Well-understood problem. Mitigation: Standard YAML parsing with validation.
- **Session Persistence:** File-based storage is straightforward. Mitigation: Atomic writes, validation on load.

---

## Success Criteria

### Functional Requirements
- [ ] All features implemented according to DoD
- [ ] >90% test coverage for critical paths
- [ ] >85% test coverage overall
- [ ] All integration points working
- [ ] Security features validated
- [ ] Performance targets met

### Non-Functional Requirements
- [ ] Code follows Go 1.24 idioms
- [ ] Clean Architecture principles applied
- [ ] SOLID principles followed
- [ ] DRY principle maintained
- [ ] KISS principle for complexity management
- [ ] Effective Go guidelines followed
- [ ] All linters passing
- [ ] No race conditions detected
- [ ] Comprehensive documentation
- [ ] Runnable examples provided

### Quality Gates
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] Race detector clean
- [ ] Linters passing (golangci-lint)
- [ ] Code review completed
- [ ] Documentation reviewed
- [ ] Security review completed
- [ ] Performance benchmarks acceptable

---

## Notes

### Go 1.24 Considerations
- Use latest standard library features
- Leverage improved type inference
- Use `log/slog` for structured logging
- Follow latest error handling patterns
- Use `context.Context` throughout
- Leverage generics where appropriate

### Clean Architecture
- Core business logic independent of frameworks
- Dependency inversion (depend on interfaces)
- Clear separation of concerns
- Testable in isolation
- UI/infrastructure depends on core, not vice versa

### SOLID Principles
- **S**ingle Responsibility: Each component has one reason to change
- **O**pen/Closed: Open for extension, closed for modification
- **L**iskov Substitution: Interfaces usable without knowing implementation
- **I**nterface Segregation: Small, focused interfaces
- **D**ependency Inversion: Depend on abstractions, not concretions

---

## References

- [Spin Architecture Overview](./spec.md)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

---

**Last Updated:** October 3, 2025  
**Status:** Ready for Implementation  
**Reviewers:** Development Team

