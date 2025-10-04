package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Comprehensive performance benchmark suite for Feature 8.6

// =============================================================================
// History Truncation Benchmarks
// =============================================================================

// BenchmarkHistoryTruncate_Small benchmarks truncation with 10 messages
func BenchmarkHistoryTruncate_Small(b *testing.B) {
	benchmarkHistoryTruncate(b, 10)
}

// BenchmarkHistoryTruncate_Medium benchmarks truncation with 100 messages
func BenchmarkHistoryTruncate_Medium(b *testing.B) {
	benchmarkHistoryTruncate(b, 100)
}

// BenchmarkHistoryTruncate_Large benchmarks truncation with 1000 messages
func BenchmarkHistoryTruncate_Large(b *testing.B) {
	benchmarkHistoryTruncate(b, 1000)
}

// BenchmarkHistoryTruncate_XLarge benchmarks truncation with 10000 messages
func BenchmarkHistoryTruncate_XLarge(b *testing.B) {
	benchmarkHistoryTruncate(b, 10000)
}

func benchmarkHistoryTruncate(b *testing.B, msgCount int) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System prompt")

	// Add messages
	for i := 0; i < msgCount; i++ {
		_ = h.AddMessage(Message{
			Role:    "user",
			Content: strings.Repeat("word ", 50),
		})
	}

	budget := 8192

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Make a copy to benchmark on fresh data
		hCopy := h.Clone()
		err := hCopy.Truncate(budget)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHistoryTruncate_NoTruncNeeded benchmarks when no truncation needed
func BenchmarkHistoryTruncate_NoTruncNeeded(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System")
	_ = h.AddUserMessage("Hello")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.Truncate(8192)
	}
}

// BenchmarkHistoryTruncate_SystemOnly benchmarks truncation with only system message
func BenchmarkHistoryTruncate_SystemOnly(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage(strings.Repeat("word ", 100))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hCopy := h.Clone()
		_ = hCopy.Truncate(500)
	}
}

// =============================================================================
// Token Counting Benchmarks
// =============================================================================

// BenchmarkTokenCounting_Simple benchmarks simple tokenizer
func BenchmarkTokenCounting_Simple(b *testing.B) {
	tokenizer := &SimpleTokenizer{}
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 10)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tokenizer.Count(text)
	}
}

// BenchmarkTokenCounting_Long benchmarks tokenizing long text
func BenchmarkTokenCounting_Long(b *testing.B) {
	tokenizer := &SimpleTokenizer{}
	text := strings.Repeat("word ", 10000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tokenizer.Count(text)
	}
}

// =============================================================================
// Context Gathering Benchmarks
// =============================================================================

// BenchmarkContextGather_Sequential benchmarks sequential context gathering (baseline)
func BenchmarkContextGather_Sequential(b *testing.B) {
	workDir := b.TempDir()

	// Create some test files
	createTestProject(b, workDir)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GatherEnvironment(workDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextGather_Concurrent benchmarks parallel context gathering
func BenchmarkContextGather_Concurrent(b *testing.B) {
	workDir := b.TempDir()

	// Create some test files
	createTestProject(b, workDir)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GatherEnvironmentConcurrent(workDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// createTestProject creates a test project structure for benchmarking
func createTestProject(tb testing.TB, dir string) {
	tb.Helper()

	files := map[string]string{
		"main.go":         "package main\n\nfunc main() {}\n",
		"go.mod":          "module test\n\ngo 1.24\n",
		"README.md":       "# Test Project\n",
		"src/app.go":      "package src\n",
		"src/util.go":     "package src\n",
		"test/app_test.go": "package test\n",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			tb.Fatal(err)
		}
	}
}

// =============================================================================
// Command Cache Benchmarks
// =============================================================================

// BenchmarkCommandCache_Hit benchmarks cache hit performance
func BenchmarkCommandCache_Hit(b *testing.B) {
	cache := NewCommandCache(5*time.Second, 10*1024*1024)

	cmd := &Command{
		Program: "git",
		Args:    []string{"status"},
		WorkDir: "/tmp",
	}

	result := &Result{
		Command:   cmd,
		Stdout:    "On branch main\nnothing to commit",
		ExitCode:  0,
		Duration:  10 * time.Millisecond,
		StartedAt: time.Now(),
	}

	key := cache.Key(cmd)
	cache.Set(key, result)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, ok := cache.Get(key)
		if !ok {
			b.Fatal("expected cache hit")
		}
	}
}

// BenchmarkCommandCache_Miss benchmarks cache miss overhead
func BenchmarkCommandCache_Miss(b *testing.B) {
	cache := NewCommandCache(5*time.Second, 10*1024*1024)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := cache.Key(&Command{
			Program: "test",
			Args:    []string{string(rune(i))},
		})
		_, _ = cache.Get(key)
	}
}

// BenchmarkCommandCache_Set benchmarks cache set performance
func BenchmarkCommandCache_Set(b *testing.B) {
	cache := NewCommandCache(5*time.Second, 100*1024*1024)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cmd := &Command{
			Program: "test",
			Args:    []string{string(rune(i % 1000))},
		}
		result := &Result{
			Stdout:   "output",
			ExitCode: 0,
		}
		key := cache.Key(cmd)
		cache.Set(key, result)
	}
}

// BenchmarkCommandCache_Eviction benchmarks cache eviction
func BenchmarkCommandCache_Eviction(b *testing.B) {
	// Small cache to force evictions
	cache := NewCommandCache(5*time.Second, 1024) // 1KB cache

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cmd := &Command{
			Program: "test",
			Args:    []string{string(rune(i))},
		}
		result := &Result{
			Stdout:   strings.Repeat("x", 100), // 100 bytes
			ExitCode: 0,
		}
		key := cache.Key(cmd)
		cache.Set(key, result) // Will trigger evictions
	}
}

// BenchmarkCommandCache_KeyGeneration benchmarks cache key generation
func BenchmarkCommandCache_KeyGeneration(b *testing.B) {
	cache := NewCommandCache(5*time.Second, 10*1024*1024)

	cmd := &Command{
		Program: "git",
		Args:    []string{"log", "--oneline", "-n", "10"},
		WorkDir: "/path/to/project",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cache.Key(cmd)
	}
}

// =============================================================================
// Event Emission Benchmarks
// =============================================================================

// BenchmarkEventEmission benchmarks event emission throughput
func BenchmarkEventEmission(b *testing.B) {
	emitter := NewEventEmitter(100)

	// Start consumer goroutine
	id, events, err := emitter.Subscribe()
	if err != nil {
		b.Fatal(err)
	}
	defer emitter.Unsubscribe(id)

	go func() {
		for range events {
			// Consume events
		}
	}()

	event := Event{
		Type: EventContentDelta,
		Data: map[string]interface{}{
			"content": "Hello",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		emitter.Emit(event)
	}
}

// BenchmarkEventEmission_MultipleSubscribers benchmarks with multiple subscribers
func BenchmarkEventEmission_MultipleSubscribers(b *testing.B) {
	emitter := NewEventEmitter(100)

	// Start 10 consumer goroutines
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		id, events, err := emitter.Subscribe()
		if err != nil {
			b.Fatal(err)
		}
		ids[i] = id
		go func() {
			for range events {
				// Consume events
			}
		}()
	}

	defer func() {
		for _, id := range ids {
			emitter.Unsubscribe(id)
		}
	}()

	event := Event{
		Type: EventContentDelta,
		Data: map[string]interface{}{
			"content": "Hello",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		emitter.Emit(event)
	}
}

// =============================================================================
// Channel Throughput Benchmarks
// =============================================================================

// BenchmarkChannelThroughput_Unbuffered benchmarks unbuffered channel
func BenchmarkChannelThroughput_Unbuffered(b *testing.B) {
	benchmarkChannelThroughput(b, 0)
}

// BenchmarkChannelThroughput_Buffer10 benchmarks channel with buffer 10
func BenchmarkChannelThroughput_Buffer10(b *testing.B) {
	benchmarkChannelThroughput(b, 10)
}

// BenchmarkChannelThroughput_Buffer100 benchmarks channel with buffer 100
func BenchmarkChannelThroughput_Buffer100(b *testing.B) {
	benchmarkChannelThroughput(b, 100)
}

// BenchmarkChannelThroughput_Buffer1000 benchmarks channel with buffer 1000
func BenchmarkChannelThroughput_Buffer1000(b *testing.B) {
	benchmarkChannelThroughput(b, 1000)
}

func benchmarkChannelThroughput(b *testing.B, bufSize int) {
	ch := make(chan int, bufSize)
	done := make(chan struct{})

	// Consumer
	go func() {
		for range ch {
			// Consume
		}
		done <- struct{}{}
	}()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ch <- i
	}

	close(ch)
	<-done
}

// =============================================================================
// Validator Benchmarks
// =============================================================================

// BenchmarkValidator_ClassifySafe benchmarks classifying safe commands
func BenchmarkValidator_ClassifySafe(b *testing.B) {
	validator := NewValidator()

	cmd := &Command{
		Program: "ls",
		Args:    []string{"-la"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Classify(cmd)
	}
}

// BenchmarkValidator_ClassifyDangerous benchmarks classifying dangerous commands
func BenchmarkValidator_ClassifyDangerous(b *testing.B) {
	validator := NewValidator()

	cmd := &Command{
		Program: "rm",
		Args:    []string{"-rf", "/"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Classify(cmd)
	}
}

// =============================================================================
// History Operations Benchmarks
// =============================================================================

// BenchmarkHistory_AddMessage_Concurrent benchmarks adding messages concurrently
func BenchmarkHistory_AddMessage_Concurrent(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System")

	msg := Message{
		Role:    "user",
		Content: "Test message",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.AddMessage(msg)
	}
}

// BenchmarkHistory_Messages benchmarks retrieving all messages
func BenchmarkHistory_Messages(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System")

	for i := 0; i < 100; i++ {
		_ = h.AddUserMessage("Message")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.Messages()
	}
}

// BenchmarkHistory_TokenCount benchmarks token counting
func BenchmarkHistory_TokenCount(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System")

	for i := 0; i < 100; i++ {
		_ = h.AddUserMessage(strings.Repeat("word ", 50))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.TokenCount()
	}
}

// BenchmarkHistory_Clone benchmarks history cloning
func BenchmarkHistory_Clone(b *testing.B) {
	h := NewHistoryWithDefaults()
	_ = h.AddSystemMessage("System")

	for i := 0; i < 100; i++ {
		_ = h.AddUserMessage("Message")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.Clone()
	}
}

// =============================================================================
// String Builder Benchmarks
// =============================================================================

// BenchmarkStringConcatenation_Plus benchmarks string concatenation with +
func BenchmarkStringConcatenation_Plus(b *testing.B) {
	parts := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := ""
		for _, part := range parts {
			result = result + part
		}
		_ = result
	}
}

// BenchmarkStringConcatenation_Builder benchmarks strings.Builder
func BenchmarkStringConcatenation_Builder(b *testing.B) {
	parts := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		for _, part := range parts {
			sb.WriteString(part)
		}
		_ = sb.String()
	}
}

// BenchmarkStringConcatenation_BuilderPrealloc benchmarks strings.Builder with Grow
func BenchmarkStringConcatenation_BuilderPrealloc(b *testing.B) {
	parts := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.Grow(10) // Pre-allocate
		for _, part := range parts {
			sb.WriteString(part)
		}
		_ = sb.String()
	}
}
