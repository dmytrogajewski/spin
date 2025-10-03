// Package core provides the core business logic and orchestration for the Spin AI coding agent.
//
// # Overview
//
// The core package implements all the essential functionality needed for an autonomous
// coding agent, including:
//   - Conversation management and lifecycle
//   - Agent orchestration and decision-making
//   - Task execution with safety controls
//   - State management (sessions, turns, history)
//   - Event streaming for real-time UI updates
//   - Command validation and execution
//   - Environment context gathering
//
// # Architecture
//
// The package is organized into several layers:
//
// Public API Layer:
//   - Manager: High-level conversation manager (entry point)
//   - Conversation: Active conversation instance
//   - Agent: Core agent orchestration
//
// State Management Layer:
//   - session/: Persistent session state
//   - turn/: Turn state machine
//   - History: Conversation history with token-aware truncation
//
// Task Execution Layer:
//   - Executor: Safe command execution
//   - Planner: Task decomposition
//   - Validator: Command safety classification
//
// Supporting Infrastructure:
//   - event.go: Event types and emission
//   - context.go: Environment context gathering
//   - config.go: Configuration management
//   - error.go: Error types and handling
//
// # Usage
//
// Basic usage pattern:
//
//	// Create a manager
//	mgr, err := core.NewManager(cfg,
//		core.WithLLMProvider(provider),
//		core.WithToolRegistry(tools),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Start a conversation
//	conv, err := mgr.NewConversation(ctx, workDir)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Execute a turn
//	err = conv.RunTurn(ctx, "Implement user authentication")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Stream events
//	for event := range conv.Stream() {
//		fmt.Printf("%s: %v\n", event.Type, event.Data)
//	}
//
// # Design Principles
//
// The core package follows these design principles:
//   - Clean Architecture: Dependencies point inward, core is independent
//   - SOLID Principles: Especially interface segregation and dependency inversion
//   - Go Idioms: Accept interfaces, return structs
//   - Concurrency: Safe concurrent access with proper synchronization
//   - Error Handling: Errors are wrapped with context using fmt.Errorf with %w
//   - Context Propagation: context.Context used throughout for cancellation
//
// # Dependencies
//
// External dependencies:
//   - golang.org/x/sync/errgroup: Concurrent error handling
//   - gopkg.in/yaml.v3: Configuration file parsing
//
// Internal dependencies:
//   - internal/llm: LLM provider interface
//   - internal/tools: Tool implementations
//   - internal/security: Sandbox and policy enforcement
//   - internal/mcp: Model Context Protocol client
//
// # Testing
//
// The package provides test utilities in the testing/ subdirectory:
//   - MockProvider: Mock LLM provider for testing
//   - MockToolsRegistry: Mock tools registry
//   - Test helpers for common test scenarios
//
// Run tests with:
//
//	go test ./internal/core/...
//	go test -race ./internal/core/...  # with race detector
//	go test -cover ./internal/core/... # with coverage
package core
