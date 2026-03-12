package generator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/retrieval"
	"github.com/dmytrogajewski/spin/internal/llm"
)

func TestNewGenerator_Success(t *testing.T) {
	t.Parallel()

	// Setup.
	mockLLM := llm.NewMockProvider("test-provider")
	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	// Create generator.
	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})

	require.NoError(t, err)
	assert.NotNil(t, gen)
}

func TestNewGenerator_MissingLLM(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		Playbook:  pb,
		Retriever: ret,
	})

	require.Error(t, err)
	assert.Nil(t, gen)
	assert.Contains(t, err.Error(), "LLM provider is required")
}

func TestNewGenerator_MissingPlaybook(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test-provider")
	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Retriever: ret,
	})

	require.Error(t, err)
	assert.Nil(t, gen)
	assert.Contains(t, err.Error(), "playbook is required")
}

func TestNewGenerator_MissingRetriever(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test-provider")
	pb := playbook.New(nil, nil)

	gen, err := NewGenerator(Config{
		LLM:      mockLLM,
		Playbook: pb,
	})

	require.Error(t, err)
	assert.Nil(t, gen)
	assert.Contains(t, err.Error(), "retriever is required")
}

func TestItemizedLearning_WithEmptyPlaybook(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup with empty playbook.
	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("The answer is 42")

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	// Execute ItemizedLearning.
	req := ItemizedLearningRequest{
		Query:       "What is the answer?",
		TopK:        5,
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	resp, err := gen.ItemizedLearning(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "The answer is 42", resp.Output)
	assert.NotNil(t, resp.Trajectory)
	assert.Equal(t, "What is the answer?", resp.Trajectory.Query)
	assert.Empty(t, resp.Trajectory.RetrievedBullets)
}

func TestItemizedLearning_WithBullets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup playbook with bullets.
	embedder := embedding.NewMockEmbedder(1536)
	pb := playbook.New(nil, embedder)

	// Add bullets with embeddings.
	emb1, err := embedder.Embed(ctx, "Always validate input")
	require.NoError(t, err)
	b1, err := bullet.New("Always validate input", bullet.WithEmbedding(emb1))
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b1))

	emb2, err := embedder.Embed(ctx, "Use context.Context")
	require.NoError(t, err)
	b2, err := bullet.New("Use context.Context", bullet.WithEmbedding(emb2))
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b2))

	// Setup mock LLM with feedback.
	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("Use proper validation.\n\nHELPFUL: [B0]\nHARMFUL: []")

	ret := retrieval.NewSemanticRetriever(pb, embedder)
	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	// Execute ItemizedLearning.
	req := ItemizedLearningRequest{
		Query:       "How to validate?",
		TopK:        2,
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	resp, err := gen.ItemizedLearning(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.Output, "validation")
	assert.NotNil(t, resp.Trajectory)
	assert.Len(t, resp.Trajectory.RetrievedBullets, 2)
	assert.NotNil(t, resp.Feedback)
	assert.Len(t, resp.Feedback.HelpfulBullets, 1)
	assert.Equal(t, "B0", resp.Feedback.HelpfulBullets[0])

	// Verify bullet counters were updated.
	updatedBullet, found := pb.Get(b1.ID)
	require.True(t, found)
	assert.Equal(t, 1, updatedBullet.HelpfulCount)
	assert.Equal(t, 0, updatedBullet.HarmfulCount)
}

func TestItemizedLearning_WithGroundTruth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("The answer is 42")

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	req := ItemizedLearningRequest{
		Query:       "What is the answer?",
		GroundTruth: "42",
		TopK:        5,
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	resp, err := gen.ItemizedLearning(ctx, req)

	require.NoError(t, err)
	assert.True(t, resp.Success) // Has ground truth and non-empty output.
	assert.True(t, resp.Trajectory.Success)
}

func TestItemizedLearning_TrajectoryMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("Response")

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	req := ItemizedLearningRequest{
		Query:       "Test query",
		TopK:        5,
		Model:       "gpt-4",
		Temperature: 0.8,
		MaxTokens:   2000,
	}

	resp, err := gen.ItemizedLearning(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "gpt-4", resp.Trajectory.Metadata.Model)
	assert.InDelta(t, 0.8, resp.Trajectory.Metadata.Temperature, 1e-9)
	assert.Equal(t, 2000, resp.Trajectory.Metadata.MaxTokens)
	assert.GreaterOrEqual(t, resp.Trajectory.Metadata.TotalTokens, 0) // MockProvider generates usage.
	assert.Equal(t, 1, resp.Trajectory.Metadata.Turns)
	assert.GreaterOrEqual(t, resp.Trajectory.Metadata.Duration.Milliseconds(), int64(0))
}

func TestItemizedLearning_HarmfulBulletUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup playbook with bullets.
	embedder := embedding.NewMockEmbedder(1536)
	pb := playbook.New(nil, embedder)

	emb, err := embedder.Embed(ctx, "Bad advice")
	require.NoError(t, err)
	b, err := bullet.New("Bad advice", bullet.WithEmbedding(emb))
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b))

	// Mock LLM marks bullet as harmful.
	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("Don't use that advice.\n\nHELPFUL: []\nHARMFUL: [B0]")

	ret := retrieval.NewSemanticRetriever(pb, embedder)
	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	req := ItemizedLearningRequest{
		Query:       "Should I use this?",
		TopK:        1,
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	resp, err := gen.ItemizedLearning(ctx, req)

	require.NoError(t, err)
	assert.Len(t, resp.Feedback.HarmfulBullets, 1)

	// Verify harmful counter was incremented.
	updatedBullet, found := pb.Get(b.ID)
	require.True(t, found)
	assert.Equal(t, 0, updatedBullet.HelpfulCount)
	assert.Equal(t, 1, updatedBullet.HarmfulCount)
}

func TestItemizedLearning_InvalidFeedbackGraceful(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Mock LLM with malformed feedback.
	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("Just a response with no feedback markers")

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	req := ItemizedLearningRequest{
		Query:       "Test",
		TopK:        5,
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	resp, err := gen.ItemizedLearning(ctx, req)

	// Should succeed even with invalid feedback.
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Feedback.HelpfulBullets)
	assert.Empty(t, resp.Feedback.HarmfulBullets)
}

func TestGenerateBullets_FromTask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse(`1. Always validate user input before processing
2. Use parameterized queries to prevent SQL injection
3. Implement proper error handling for database operations`)

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	req := BulletGenerationRequest{
		Input:      "Build a secure user authentication system",
		SourceType: "task",
		MaxBullets: 3,
		Tags: map[string]string{
			"category": "security",
			"source":   "task",
		},
	}

	bullets, err := gen.GenerateBullets(ctx, req)

	require.NoError(t, err)
	assert.Len(t, bullets, 3)
	assert.Contains(t, bullets[0].Content, "validate")
	assert.Contains(t, bullets[1].Content, "SQL injection")
	assert.Contains(t, bullets[2].Content, "error handling")

	// Check tags were applied.
	assert.Equal(t, "security", bullets[0].Tags["category"])
	assert.Equal(t, "task", bullets[0].Tags["source"])
}

func TestGenerateBullets_FromTrajectory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse(`1. Break down complex problems into smaller steps
2. Test each component independently before integration
3. Document assumptions and edge cases`)

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	req := BulletGenerationRequest{
		Input:      "Trajectory showing successful problem solving approach...",
		SourceType: "trajectory",
		MaxBullets: 5,
	}

	bullets, err := gen.GenerateBullets(ctx, req)

	require.NoError(t, err)
	assert.Len(t, bullets, 3)
	assert.Contains(t, bullets[0].Content, "Break down")
	assert.Contains(t, bullets[1].Content, "Test")
	assert.Contains(t, bullets[2].Content, "Document")
}

func TestGenerateBullets_FromFeedback(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("1. Add more comprehensive input validation\n2. Include timeout mechanisms for external calls")

	gen := newTestGenerator(t, mockLLM)

	req := BulletGenerationRequest{
		Input:      "Users reported timeout issues during high load...",
		SourceType: "feedback",
		MaxBullets: 2,
	}

	bullets, err := gen.GenerateBullets(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, bullets, 2)
}

func TestGenerateBullets_FromError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse(`1. Always check for nil pointers before dereferencing
2. Use defer to ensure cleanup happens even on panic
3. Add context deadlines to prevent indefinite blocking`)

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	req := BulletGenerationRequest{
		Input:      "panic: runtime error: invalid memory address or nil pointer dereference",
		SourceType: "error",
		MaxBullets: 3,
		Tags: map[string]string{
			"type": "runtime-error",
		},
	}

	bullets, err := gen.GenerateBullets(ctx, req)

	require.NoError(t, err)
	assert.Len(t, bullets, 3)
	assert.Contains(t, bullets[0].Content, "nil")
	assert.Equal(t, "runtime-error", bullets[0].Tags["type"])
}

// newTestGenerator creates a generator with a mock LLM for testing.
func newTestGenerator(t *testing.T, mockLLM *llm.MockProvider) Generator {
	t.Helper()

	pb := playbook.New(nil, nil)
	ret := retrieval.NewSemanticRetriever(pb, embedding.NewMockEmbedder(1536))

	gen, err := NewGenerator(Config{
		LLM:       mockLLM,
		Playbook:  pb,
		Retriever: ret,
	})
	require.NoError(t, err)

	return gen
}

func TestGenerateBullets_EmptyInput(t *testing.T) {
	t.Parallel()

	gen := newTestGenerator(t, llm.NewMockProvider("test-provider"))

	req := BulletGenerationRequest{
		Input:      "",
		SourceType: "task",
		MaxBullets: 5,
	}

	bullets, err := gen.GenerateBullets(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, bullets)
	assert.Contains(t, err.Error(), "input is required")
}

func TestGenerateBullets_UnknownSourceType(t *testing.T) {
	t.Parallel()

	gen := newTestGenerator(t, llm.NewMockProvider("test-provider"))

	req := BulletGenerationRequest{
		Input:      "Some input",
		SourceType: "unknown",
		MaxBullets: 5,
	}

	bullets, err := gen.GenerateBullets(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, bullets)
	assert.Contains(t, err.Error(), "unknown source type")
}

func TestGenerateBullets_NoLimit(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("1. First strategy\n2. Second strategy\n3. Third strategy\n4. Fourth strategy\n5. Fifth strategy\n6. Sixth strategy")

	gen := newTestGenerator(t, mockLLM)

	req := BulletGenerationRequest{
		Input:      "Test input",
		SourceType: "task",
		MaxBullets: 0, // No limit - model decides.
	}

	bullets, err := gen.GenerateBullets(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, bullets, 6) // All 6 bullets returned, no truncation.
}

func TestGenerateBullets_AllReturned(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test-provider")
	mockLLM.SetResponse("1. Strategy one\n2. Strategy two\n3. Strategy three\n4. Strategy four\n5. Strategy five\n6. Strategy six\n7. Strategy seven")

	gen := newTestGenerator(t, mockLLM)

	req := BulletGenerationRequest{
		Input:      "Test input",
		SourceType: "task",
		MaxBullets: 0, // No limit.
	}

	bullets, err := gen.GenerateBullets(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, bullets, 7) // All 7 bullets returned.
}

func TestParseBulletCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "numbered with dot",
			input:    "1. First\n2. Second\n3. Third",
			expected: []string{"First", "Second", "Third"},
		},
		{
			name:     "numbered with paren",
			input:    "1) First\n2) Second",
			expected: []string{"First", "Second"},
		},
		{
			name:     "dashed list",
			input:    "- First\n- Second",
			expected: []string{"First", "Second"},
		},
		{
			name:     "asterisk list",
			input:    "* First\n* Second",
			expected: []string{"First", "Second"},
		},
		{
			name:     "mixed format",
			input:    "1. First\n- Second\n3. Third",
			expected: []string{"First", "Second", "Third"},
		},
		{
			name:     "with empty lines",
			input:    "1. First\n\n2. Second\n\n",
			expected: []string{"First", "Second"},
		},
		{
			name:     "no prefix",
			input:    "First\nSecond",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseBulletCandidates(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
