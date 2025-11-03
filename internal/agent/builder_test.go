package agent

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/openai/openai-go"
)

func TestNewBuilder_CreatesBuilder(t *testing.T) {
	builder := NewBuilder()

	if builder == nil {
		t.Fatal("NewBuilder() returned nil")
	}
}

func TestBuilder_WithConfig(t *testing.T) {
	cfg := &Config{
		Model: "test-model",
	}

	builder := NewBuilder().WithConfig(cfg)

	if builder.config != cfg {
		t.Error("WithConfig() did not store config")
	}
}

func TestBuilder_FluentInterface(t *testing.T) {
	cfg := &Config{Model: "test"}

	builder := NewBuilder().
		WithConfig(cfg).
		WithWorkingDir("/test").
		WithEmitter(nil).
		WithApprovalHandler(nil)

	if builder == nil {
		t.Error("Fluent interface broke chain")
	}
	if builder.config != cfg {
		t.Error("Config not set")
	}
	if builder.workingDir != "/test" {
		t.Error("WorkingDir not set")
	}
}

func TestBuilder_BuildExecutor(t *testing.T) {
	cfg := &Config{
		Timeout:       30,
		CacheCommands: false,
	}

	builder := NewBuilder().
		WithConfig(cfg).
		WithWorkingDir("/tmp/test")

	exec := builder.buildExecutor()

	if exec == nil {
		t.Fatal("buildExecutor() returned nil")
	}
}

func TestBuilder_BuildEnvironment(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		MaxFiles: 100,
		MaxDepth: 5,
		SkipGit:  false,
	}

	builder := NewBuilder().
		WithConfig(cfg).
		WithWorkingDir(tmpDir)

	env := builder.buildEnvironment()

	if env == nil {
		t.Fatal("buildEnvironment() returned nil")
	}
	if env.WorkDir != tmpDir {
		t.Errorf("WorkDir = %s, want %s", env.WorkDir, tmpDir)
	}
}

func TestBuilder_Build(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock LLM provider
	mockLLM := &mockProvider{}

	cfg := &Config{
		Model:       "test-model",
		MaxTurns:    10,
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     30,
	}

	emitter := events.NewEventEmitter(10)

	agent, err := NewBuilder().
		WithConfig(cfg).
		WithProvider(mockLLM).
		WithWorkingDir(tmpDir).
		WithEmitter(emitter).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

// mockProvider is a simple mock LLM provider for testing
type mockProvider struct{}

func (m *mockProvider) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return &openai.ChatCompletion{
		ID:    "test-completion",
		Model: "test-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatCompletionMessageRoleAssistant,
					Content: "test response",
				},
				FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
			},
		},
	}, nil
}

func (m *mockProvider) Stream(ctx context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	ch := make(chan openai.ChatCompletionChunk)
	close(ch)
	return ch, nil
}

func (m *mockProvider) Models(ctx context.Context) ([]openai.Model, error) {
	return []openai.Model{{ID: "test-model"}}, nil
}

func (m *mockProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,
	}
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Close() error {
	return nil
}
