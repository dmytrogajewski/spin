package core

import (
	"strings"
	"testing"
)

// Simple benchmark tests for performance-critical paths

// BenchmarkHistory_Truncate_Small benchmarks history truncation with small history
func BenchmarkHistory_Truncate_Small(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System prompt")

	for i := 0; i < 10; i++ {
		_ = h.AddMessage(Message{
			Role:    "user",
			Content: strings.Repeat("word ", 50),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to benchmark on fresh data
		hCopy := *h
		_ = hCopy.Truncate(8192)
	}
}

// BenchmarkHistory_AddMessage benchmarks adding messages
func BenchmarkHistory_AddMessage(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System")

	msg := Message{
		Role:    "user",
		Content: "Test message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.AddMessage(msg)
	}
}

// BenchmarkContext_Gather benchmarks environment context gathering
func BenchmarkContext_Gather(b *testing.B) {
	workDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GatherEnvironment(workDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkState_String benchmarks state string conversion
func BenchmarkState_String(b *testing.B) {
	state := StateRunning

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.String()
	}
}

// BenchmarkError_Error benchmarks error string formatting
func BenchmarkError_Error(b *testing.B) {
	err := &Error{
		Op:   "TestOperation",
		Code: ErrCodeInternal,
		Err:  ErrInvalidInput,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// BenchmarkValidator_Classify benchmarks command classification
func BenchmarkValidator_Classify(b *testing.B) {
	validator := NewValidator()

	cmd := &Command{
		Program: "git",
		Args:    []string{"commit", "-m", "test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Classify(cmd)
	}
}

// BenchmarkExecutor_Validate benchmarks command validation
func BenchmarkExecutor_Validate(b *testing.B) {
	workDir := b.TempDir()
	executor, err := NewExecutor(workDir)
	if err != nil {
		b.Fatalf("NewExecutor() error: %v", err)
	}

	cmd := &Command{
		Program: "echo",
		Args:    []string{"test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := executor.Validate(cmd)
		if err != nil {
			b.Fatal(err)
		}
	}
}
