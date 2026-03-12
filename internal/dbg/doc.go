// Package dbg provides debugging and development utilities for Spin.
//
// This package implements tools for:
//   - Sandbox testing (macOS sandbox-exec, Linux Landlock)
//   - Core event stream debugging
//   - LLM request/response logging
//   - Performance profiling helpers
//
// # Sandbox Testing
//
// Test sandbox behavior with different modes:
//
//	executor := debug.NewSandboxExecutor("read-only", "/workspace")
//	result, err := executor.Execute(ctx, "ls", "-la")
//
// # Event Debugging
//
// Capture and log all core events:
//
//	logger := debug.NewEventLogger("json", []string{"tool", "stream"})
//	err := logger.Run(ctx, "fix tests")
//
// # LLM Logging
//
// Intercept and log LLM requests/responses:
//
//	interceptor := debug.NewLLMInterceptor(provider, "text")
//	resp, err := interceptor.Complete(ctx, req)
//
// # Performance Profiling
//
// Enable CPU/memory profiling:
//
//	profiler := debug.NewProfiler("cpu", "spin-profile")
//	profiler.Start()
//	defer profiler.Stop()
//
// All debug utilities are designed for development and troubleshooting.
// They should not be used in production environments.
package dbg
