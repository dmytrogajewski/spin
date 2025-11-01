package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/curator"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
	"github.com/dmytrogajewski/spin/internal/ace/retrieval"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapter_StartSession(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Start session
	sessionID, err := adapter.StartSession(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, sessionID)

	// Should be able to get session
	session, err := adapter.GetSession(sessionID)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, sessionID, session.ID)
	assert.Equal(t, 0, session.SignalCount)
	assert.Equal(t, 0, session.UpdateCount)
}

func TestAdapter_GetSession_NotFound(t *testing.T) {
	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Get non-existent session
	session, err := adapter.GetSession("non-existent")

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session not found")
}

func TestAdapter_EndSession(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// End session
	err = adapter.EndSession(ctx, sessionID)
	require.NoError(t, err)

	// Session should no longer exist
	session, err := adapter.GetSession(sessionID)
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestAdapter_AdaptOnline_SkipSuccess(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// Create success signal
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Context:    "Test passed",
		Outcome:    OutcomeSuccess,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
	}

	// Adapt online
	result, err := adapter.AdaptOnline(ctx, signal)
	require.NoError(t, err)

	// Should skip success signals
	assert.Equal(t, ActionSkip, result.Action)
	assert.Equal(t, 0, result.BulletsAdded)
	assert.Equal(t, 0, pb.Stats().TotalBullets)
}

func TestAdapter_AdaptOnline_FullReflect(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Configure mock LLM to return insights JSON
	insightsJSON := `[{
		"content": "Always check for nil pointers before dereferencing",
		"evidence": ["Test failed with nil pointer panic"],
		"confidence": 0.9,
		"category": "error_mode"
	}]`

	llmProvider := llm.NewMockProvider("test-llm", llm.WithResponse(insightsJSON))
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// Initial playbook should be empty
	assert.Equal(t, 0, pb.Stats().TotalBullets)

	// Create test failure signal
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Context:    "TestFoo failed with nil pointer panic",
		Outcome:    OutcomeFailure,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
	}

	// Adapt online - should trigger reflection
	result, err := adapter.AdaptOnline(ctx, signal)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should use Reflect action
	assert.Equal(t, ActionReflect, result.Action)

	// Should have added bullets to playbook
	assert.Equal(t, 1, result.BulletsAdded)
	assert.Equal(t, 1, pb.Stats().TotalBullets)

	// Session should have update recorded
	session, err := adapter.GetSession(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 1, session.UpdateCount)

	// Verify bullet content
	bullets := pb.List(nil)
	require.Len(t, bullets, 1)
	assert.Contains(t, bullets[0].Content, "nil pointer")
	assert.Equal(t, 9, bullets[0].HelpfulCount) // 0.9 * 10 = 9
}

func TestAdapter_AdaptOnline_QuickAddWithGenerator(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Mock LLM for generator
	bulletResponse := "1. Always run 'go mod tidy' before building\n2. Check for syntax errors in go.mod\n3. Ensure all dependencies are available"
	llmProvider := llm.NewMockProvider("test-llm", llm.WithResponse(bulletResponse))

	// Create retriever and generator
	retr := retrieval.NewSemanticRetriever(pb, embedder)
	gen, err := generator.NewGenerator(generator.Config{
		LLM:       llmProvider,
		Playbook:  pb,
		Retriever: retr,
	})
	require.NoError(t, err)

	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter with generator
	adapter := NewAdapterWithConfig(Config{
		Playbook:     pb,
		Reflector:    refl,
		Curator:      cur,
		Generator:    gen,
		MemoryConfig: DefaultMemoryConfig(),
	})

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// Create build failure signal
	signal := ExecutionSignal{
		SignalType: SignalTypeBuild,
		Context:    "build failed: go.mod syntax error",
		Outcome:    OutcomeFailure,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
	}

	// Adapt online - should trigger quick add
	result, err := adapter.AdaptOnline(ctx, signal)
	require.NoError(t, err)

	// Should use QuickAdd action
	assert.Equal(t, ActionQuickAdd, result.Action)

	// Should have added bullets
	assert.Greater(t, result.BulletsAdded, 0)
	assert.Equal(t, result.BulletsAdded, pb.Stats().TotalBullets)
}

func TestAdapter_AdaptOnline_MemoryRefinement(t *testing.T) {
	ctx := context.Background()

	// Create dependencies with low refinement threshold
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter with low refinement threshold
	adapter := NewAdapterWithConfig(Config{
		Playbook:  pb,
		Reflector: refl,
		Curator:   cur,
		MemoryConfig: MemoryConfig{
			MaxBullets:     10,
			RefinementAt:   2,   // Trigger at 2 bullets
			PruneThreshold: 0.5, // High threshold to prune bullets
		},
	})

	// Add some low-utility bullets to playbook
	for i := 0; i < 3; i++ {
		b, _ := bullet.New("Low utility bullet")
		emb, _ := embedder.Embed(ctx, b.Content)
		b.Embedding = emb
		pb.Add(ctx, b)
		// Low utility: no helpful/harmful marks
	}

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// Create skip signal (won't add bullets, but will check refinement)
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Context:    "Test passed",
		Outcome:    OutcomeSuccess,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
	}

	// Adapt online - should trigger refinement
	result, err := adapter.AdaptOnline(ctx, signal)
	require.NoError(t, err)

	// Should have triggered refinement
	assert.True(t, result.RefinementTriggered)
	assert.Contains(t, result.Reason, "pruned")

	// Playbook should have been pruned
	assert.Less(t, pb.Stats().TotalBullets, 3)
}

func TestAdapter_AdaptOnline_SessionNotFound(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Create signal with non-existent session
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Context:    "Test failed",
		Outcome:    OutcomeFailure,
		SessionID:  "non-existent",
		Timestamp:  time.Now(),
	}

	// Should error
	result, err := adapter.AdaptOnline(ctx, signal)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "session not found")
}

func TestAdapter_AdaptOnline_MultipleSignals(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm", llm.WithResponse("[]"))
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// Process multiple signals
	signals := []ExecutionSignal{
		{SignalType: SignalTypeTest, Outcome: OutcomeSuccess, Context: "Test 1", SessionID: sessionID, Timestamp: time.Now()},
		{SignalType: SignalTypeBuild, Outcome: OutcomeFailure, Context: "Build error", SessionID: sessionID, Timestamp: time.Now()},
		{SignalType: SignalTypeTest, Outcome: OutcomeFailure, Context: "Test 2 failed", SessionID: sessionID, Timestamp: time.Now()},
	}

	for _, sig := range signals {
		_, err := adapter.AdaptOnline(ctx, sig)
		require.NoError(t, err)
	}

	// Session should track all signals
	session, err := adapter.GetSession(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 3, session.SignalCount)
	assert.Len(t, session.RecentSignals, 3)
}

func TestAdapter_AdaptOnline_ReflectWithNoInsights(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// LLM returns empty insights array
	llmProvider := llm.NewMockProvider("test-llm", llm.WithResponse("[]"))
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// Create test failure signal
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Context:    "TestFoo failed",
		Outcome:    OutcomeFailure,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
	}

	// Adapt online
	result, err := adapter.AdaptOnline(ctx, signal)
	require.NoError(t, err)

	// Should use Reflect action but add no bullets (no insights)
	assert.Equal(t, ActionReflect, result.Action)
	assert.Equal(t, 0, result.BulletsAdded)
	assert.Equal(t, 0, pb.Stats().TotalBullets)
}

func TestAdapter_AdaptOnline_QuickAddWithoutGenerator(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter WITHOUT generator
	adapter := NewAdapter(pb, refl, cur)

	// Start session
	sessionID, err := adapter.StartSession(ctx)
	require.NoError(t, err)

	// Create build failure signal
	signal := ExecutionSignal{
		SignalType: SignalTypeBuild,
		Context:    "Build failed",
		Outcome:    OutcomeFailure,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
	}

	// Adapt online
	result, err := adapter.AdaptOnline(ctx, signal)
	require.NoError(t, err)

	// Should use QuickAdd action but add no bullets (no generator)
	assert.Equal(t, ActionQuickAdd, result.Action)
	assert.Equal(t, 0, result.BulletsAdded)
}

func TestAdapter_EndSession_NotFound(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	llmProvider := llm.NewMockProvider("test-llm")
	refl := reflector.NewReflector(llmProvider)
	cur := curator.NewCurator(pb, embedder)

	// Create adapter
	adapter := NewAdapter(pb, refl, cur)

	// Try to end non-existent session
	err := adapter.EndSession(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestDecideAction_UnknownSignalType(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalType("unknown"),
		Outcome:    OutcomeFailure,
		Context:    "Unknown signal",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionSkip, action)
	assert.Contains(t, reason, "Unknown")
}
