package core

import (
	"strings"
	"testing"
	"time"
)

func TestHistory_CompressionTrigger(t *testing.T) {
	// Test that compression triggers at 80% threshold
	maxTokens := 1000
	threshold := 0.8

	history := NewHistory(maxTokens, &SimpleTokenizer{})

	// Add messages until 80% capacity
	targetTokens := int(float64(maxTokens) * threshold)
	tokensAdded := 0

	for tokensAdded < targetTokens {
		msg := Message{
			Role:    RoleAssistant,
			Content: "This is a test message to fill up context",
			Tokens:  50,
		}
		if err := history.AddMessage(msg); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
		tokensAdded += 50
	}

	// Verify compression hasn't triggered yet
	if history.Count() == 0 {
		t.Errorf("expected messages before compression")
	}

	// Add one more message to exceed 80%
	msg := Message{
		Role:    RoleAssistant,
		Content: "Trigger compression",
		Tokens:  100,
	}
	if err := history.AddMessage(msg); err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	// Verify compression occurred (message count reduced)
	// This is a rough check - exact behavior depends on implementation
	finalTokens := history.TokenCount()
	if finalTokens > maxTokens {
		t.Errorf("expected tokens under max after compression, got %d (max: %d)", finalTokens, maxTokens)
	}
}

func TestHistory_CriticalRetention(t *testing.T) {
	maxTokens := 500
	history := NewHistory(maxTokens, &SimpleTokenizer{})

	// Add critical messages (user, tool, error)
	criticalMessages := []Message{
		{ID: "user1", Role: RoleUser, Content: "Important user question", Tokens: 50},
		{ID: "tool1", Role: RoleTool, Content: "Tool result data", Tokens: 50},
		{ID: "user2", Role: RoleUser, Content: "Another user question", Tokens: 50},
	}

	for _, msg := range criticalMessages {
		if err := history.AddMessage(msg); err != nil {
			t.Fatalf("failed to add critical message: %v", err)
		}
	}

	// Fill up with low-importance messages to trigger compression
	for i := 0; i < 20; i++ {
		msg := Message{
			Role:    RoleAssistant,
			Content: generateLongVerboseContent(),
			Tokens:  100,
		}
		if err := history.AddMessage(msg); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	// Verify all critical messages are still present
	messages := history.Messages()
	for _, critical := range criticalMessages {
		found := false
		for _, msg := range messages {
			if msg.ID == critical.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("critical message %s was removed during compression", critical.ID)
		}
	}
}

func TestHistory_200TurnsNoOverflow(t *testing.T) {
	// Simulate 200-turn conversation
	maxTokens := 16384 // Regular mode
	history := NewHistory(maxTokens, &SimpleTokenizer{})

	for turn := 0; turn < 200; turn++ {
		// User message
		userMsg := Message{
			Role:    RoleUser,
			Content: "Turn " + string(rune(turn)) + " user question",
			Tokens:  20,
		}
		if err := history.AddMessage(userMsg); err != nil {
			t.Fatalf("turn %d: failed to add user message: %v", turn, err)
		}

		// Assistant response
		assistantMsg := Message{
			Role:    RoleAssistant,
			Content: "Turn " + string(rune(turn)) + " assistant response with some detail",
			Tokens:  50,
		}
		if err := history.AddMessage(assistantMsg); err != nil {
			t.Fatalf("turn %d: failed to add assistant message: %v", turn, err)
		}

		// Occasional tool call
		if turn%10 == 0 {
			toolMsg := Message{
				Role:    RoleTool,
				Content: "Tool execution result for turn " + string(rune(turn)),
				Tokens:  80,
			}
			if err := history.AddMessage(toolMsg); err != nil {
				t.Fatalf("turn %d: failed to add tool message: %v", turn, err)
			}
		}
	}

	// Verify we never exceeded max tokens
	finalTokens := history.TokenCount()
	if finalTokens > maxTokens {
		t.Errorf("exceeded max tokens after 200 turns: %d > %d", finalTokens, maxTokens)
	}

	// Verify we have messages (not all removed)
	if history.Count() == 0 {
		t.Errorf("all messages removed after 200 turns")
	}
}

func TestHistory_CompressionDisabled(t *testing.T) {
	maxTokens := 1000

	// Create config with compression disabled
	config := &HistoryConfig{
		CompressionEnabled:   false,
		CompressionThreshold: 0.8,
		PreserveCritical:     true,
		MinRetention:         0.3,
	}

	history := NewHistoryWithConfig(maxTokens, &SimpleTokenizer{}, config)

	// Fill beyond capacity
	for i := 0; i < 50; i++ {
		msg := Message{
			Role:    RoleAssistant,
			Content: "Message " + string(rune(i)),
			Tokens:  50,
		}
		if err := history.AddMessage(msg); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	// With compression disabled, should exceed max tokens
	finalTokens := history.TokenCount()
	if finalTokens <= maxTokens {
		t.Errorf("expected to exceed max tokens with compression disabled, got %d (max: %d)", finalTokens, maxTokens)
	}
}

func TestHistory_CompressionEvent(t *testing.T) {
	maxTokens := 1000
	history := NewHistory(maxTokens, &SimpleTokenizer{})

	// Create event emitter and subscribe
	emitter := NewEventEmitter(100)
	history.SetEventEmitter(emitter)

	subID, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer emitter.Unsubscribe(subID)

	// Fill history to trigger compression
	for i := 0; i < 50; i++ {
		msg := Message{
			Role:    RoleAssistant,
			Content: "Test message " + string(rune(i)),
			Tokens:  50,
		}
		if err := history.AddMessage(msg); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	// Check if compression event was emitted
	eventReceived := false
	timeout := time.After(100 * time.Millisecond)

drainLoop:
	for {
		select {
		case event := <-events:
			if event.Type == EventInfo {
				if data, ok := event.Data.(SystemEventData); ok {
					if strings.Contains(data.Message, "compressed") {
						eventReceived = true
						// Verify details contain stats
						if !strings.Contains(data.Details, "messages:") {
							t.Errorf("expected compression stats in details, got: %s", data.Details)
						}
						break drainLoop
					}
				}
			}
		case <-timeout:
			break drainLoop
		}
	}

	if !eventReceived {
		t.Errorf("compression event not emitted")
	}
}

func TestHistory_MultipleCompressions(t *testing.T) {
	maxTokens := 500
	history := NewHistory(maxTokens, &SimpleTokenizer{})

	// Trigger multiple compression cycles
	for cycle := 0; cycle < 5; cycle++ {
		// Add enough messages to trigger compression
		for i := 0; i < 30; i++ {
			msg := Message{
				Role:    RoleAssistant,
				Content: "Cycle " + string(rune(cycle)) + " message " + string(rune(i)),
				Tokens:  50,
			}
			if err := history.AddMessage(msg); err != nil {
				t.Fatalf("cycle %d: failed to add message: %v", cycle, err)
			}
		}
	}

	// Should still be under max tokens after multiple cycles
	finalTokens := history.TokenCount()
	if finalTokens > maxTokens {
		t.Errorf("exceeded max tokens after multiple compressions: %d > %d", finalTokens, maxTokens)
	}
}

func TestHistory_CodeReviewMode(t *testing.T) {
	// Review mode: 12K tokens
	maxTokens := 12288
	history := NewHistory(maxTokens, &SimpleTokenizer{})

	// Simulate code review session with many file reads
	// Note: With 50 files * 620 tokens each = 31K total, compression will be aggressive
	for fileNum := 0; fileNum < 50; fileNum++ {
		// User request to review file
		userMsg := Message{
			Role:    RoleUser,
			Content: "Please review file" + string(rune(fileNum)) + ".go",
			Tokens:  20,
		}
		if err := history.AddMessage(userMsg); err != nil {
			t.Fatalf("failed to add user message: %v", err)
		}

		// Tool result with file contents (large)
		toolMsg := Message{
			Role:    RoleTool,
			Content: "File contents: " + strings.Repeat("code line\n", 100),
			Tokens:  500,
		}
		if err := history.AddMessage(toolMsg); err != nil {
			t.Fatalf("failed to add tool message: %v", err)
		}

		// Assistant analysis
		assistantMsg := Message{
			Role:    RoleAssistant,
			Content: "Analysis of file" + string(rune(fileNum)),
			Tokens:  100,
		}
		if err := history.AddMessage(assistantMsg); err != nil {
			t.Fatalf("failed to add assistant message: %v", err)
		}
	}

	// Verify some user requests preserved (not all - critical messages too large)
	messages := history.Messages()
	userCount := 0
	for _, msg := range messages {
		if msg.Role == RoleUser {
			userCount++
		}
	}

	if userCount == 0 {
		t.Errorf("expected at least some user messages preserved, got 0")
	}

	// Verify compression kept us reasonably close to budget
	// With PreserveCritical, we may exceed budget if critical messages alone are too large
	// But should not be wildly over (allow 2x overage for critical preservation)
	finalTokens := history.TokenCount()
	if finalTokens > maxTokens*2 {
		t.Errorf("exceeded 2x max tokens in review mode: %d > %d", finalTokens, maxTokens*2)
	}
}

func TestHistory_PlanningMode(t *testing.T) {
	// Planning mode: 4K tokens
	maxTokens := 4096
	history := NewHistory(maxTokens, &SimpleTokenizer{})

	// User creates detailed plan
	userMsg := Message{
		Role:    RoleUser,
		Content: "Create a plan for implementing user authentication with the following requirements: " + strings.Repeat("requirement ", 50),
		Tokens:  300,
	}
	if err := history.AddMessage(userMsg); err != nil {
		t.Fatalf("failed to add user message: %v", err)
	}

	// Many turns refining the plan
	for turn := 0; turn < 50; turn++ {
		assistantMsg := Message{
			Role:    RoleAssistant,
			Content: "Plan iteration " + string(rune(turn)) + ": " + strings.Repeat("step ", 20),
			Tokens:  100,
		}
		if err := history.AddMessage(assistantMsg); err != nil {
			t.Fatalf("failed to add assistant message: %v", err)
		}
	}

	// User's original requirements should be preserved
	messages := history.Messages()
	foundOriginal := false
	for _, msg := range messages {
		if msg.Role == RoleUser && strings.Contains(msg.Content, "authentication") {
			foundOriginal = true
			break
		}
	}

	if !foundOriginal {
		t.Errorf("original user requirements not preserved")
	}

	// Under token limit
	finalTokens := history.TokenCount()
	if finalTokens > maxTokens {
		t.Errorf("exceeded max tokens in planning mode: %d > %d", finalTokens, maxTokens)
	}
}

// Benchmark compression overhead

func BenchmarkHistory_AddWithCompression_100Messages(b *testing.B) {
	maxTokens := 8000
	tokenizer := &SimpleTokenizer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		history := NewHistory(maxTokens, tokenizer)

		// Add 100 messages
		for j := 0; j < 100; j++ {
			msg := Message{
				Role:    RoleAssistant,
				Content: "Test message number " + string(rune(j)),
				Tokens:  50,
			}
			_ = history.AddMessage(msg)
		}
	}
}

func BenchmarkHistory_AddWithCompression_1000Messages(b *testing.B) {
	maxTokens := 16000
	tokenizer := &SimpleTokenizer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		history := NewHistory(maxTokens, tokenizer)

		// Add 1000 messages (will trigger multiple compressions)
		for j := 0; j < 1000; j++ {
			msg := Message{
				Role:    RoleAssistant,
				Content: "Test message number " + string(rune(j)),
				Tokens:  50,
			}
			_ = history.AddMessage(msg)
		}
	}
}

// Helper function
func generateLongVerboseContent() string {
	var content strings.Builder
	for i := 0; i < 50; i++ {
		content.WriteString("This is verbose reasoning content that can be compressed. ")
	}
	return content.String()
}

// Race condition test
func TestHistory_ConcurrentAddWithCompression(t *testing.T) {
	maxTokens := 5000
	history := NewHistory(maxTokens, &SimpleTokenizer{})

	// Multiple goroutines adding messages concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				msg := Message{
					Role:    RoleAssistant,
					Content: "Goroutine " + string(rune(id)) + " message " + string(rune(j)),
					Tokens:  50,
				}
				if err := history.AddMessage(msg); err != nil {
					t.Errorf("concurrent add failed: %v", err)
					return
				}
				time.Sleep(time.Microsecond) // Small delay to interleave operations
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify history is consistent
	if history.Count() == 0 {
		t.Errorf("no messages after concurrent adds")
	}

	finalTokens := history.TokenCount()
	if finalTokens > maxTokens*2 {
		t.Errorf("tokens significantly exceeded max after concurrent adds: %d > %d", finalTokens, maxTokens*2)
	}
}
