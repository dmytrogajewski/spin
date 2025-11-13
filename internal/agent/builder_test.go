package agent

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/config"
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

func TestBuilder_WithUnifiedConfig_FluentInterface(t *testing.T) {
	cfg := &config.ConfigV2{
		LLM: config.LLMConfigV2{Model: "test"},
	}

	builder := NewBuilder().
		WithUnifiedConfig(cfg).
		WithWorkingDir("/test").
		WithEmitter(nil).
		WithApprovalHandler(nil)

	if builder == nil {
		t.Error("Fluent interface broke chain")
	}
	if builder.unifiedConfig != cfg {
		t.Error("Config not set")
	}
	if builder.workingDir != "/test" {
		t.Error("WorkingDir not set")
	}
}

func TestBuilder_BuildExecutor(t *testing.T) {
	cfg := &config.ConfigV2{
		Agent: config.AgentConfigV2{
			Timeout:       30 * time.Second,
			CacheCommands: false,
		},
	}

	emitter := events.NewEventEmitter(10)

	builder := NewBuilder().
		WithUnifiedConfig(cfg).
		WithWorkingDir("/tmp/test").
		WithEmitter(emitter)

	exec := builder.BuildExecutor()

	if exec == nil {
		t.Fatal("BuildExecutor() returned nil")
	}
}

func TestBuilder_BuildEnvironment(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.ConfigV2{
		Agent: config.AgentConfigV2{
			MaxFiles: 100,
			MaxDepth: 5,
			SkipGit:  false,
		},
	}

	builder := NewBuilder().
		WithUnifiedConfig(cfg).
		WithWorkingDir(tmpDir)

	env := builder.BuildEnvironment()

	if env == nil {
		t.Fatal("BuildEnvironment() returned nil")
	}
	if env.WorkDir != tmpDir {
		t.Errorf("WorkDir = %s, want %s", env.WorkDir, tmpDir)
	}
}

func TestBuilder_BuildHelpers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock LLM provider
	mockLLM := &mockProvider{}

	cfg := &config.ConfigV2{
		LLM: config.LLMConfigV2{
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   1000,
		},
		Agent: config.AgentConfigV2{
			MaxTurns: 10,
			Timeout:  30 * time.Second,
		},
	}

	emitter := events.NewEventEmitter(10)

	builder := NewBuilder().
		WithUnifiedConfig(cfg).
		WithProvider(mockLLM).
		WithWorkingDir(tmpDir).
		WithEmitter(emitter)

	// Test individual builders
	secSvc := builder.BuildSecurityService()
	if secSvc == nil {
		t.Fatal("BuildSecurityService() returned nil")
	}

	detSvc := builder.BuildDetectionService()
	if detSvc == nil {
		t.Fatal("BuildDetectionService() returned nil")
	}

	opts := builder.BuildAgentOptions()
	if len(opts) == 0 {
		t.Fatal("BuildAgentOptions() returned empty options")
	}
}

func TestBuilder_Build(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock LLM provider
	mockLLM := &mockProvider{}

	cfg := &config.ConfigV2{
		LLM: config.LLMConfigV2{
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   1000,
		},
		Agent: config.AgentConfigV2{
			MaxTurns: 10,
			Timeout:  30 * time.Second,
		},
	}

	emitter := events.NewEventEmitter(10)

	agent, err := NewBuilder().
		WithUnifiedConfig(cfg).
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

// TestBuilder_BuildACEService tests ACE service creation.
func TestBuilder_BuildACEService(t *testing.T) {
	tmpDir := t.TempDir()
	mockLLM := &mockProvider{}

	// Use config.ConfigV2 with ACE enabled
	cfg := &config.ConfigV2{
		LLM: config.LLMConfigV2{
			Provider:    "openai",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   1000,
		},
		Agent: config.AgentConfigV2{
			MaxTurns: 10,
			Timeout:  30 * time.Second,
			WorkDir:  tmpDir,
		},
		ACE: config.ACEConfigV2{
			Enabled:        true,
			PlaybookPath:   tmpDir + "/playbook.json",
			TrajectoryPath: tmpDir + "/trajectories/",
			TopK:           5,
			MinScore:       0.3,
		},
	}

	builder := NewBuilder().
		WithUnifiedConfig(cfg).
		WithProvider(mockLLM).
		WithWorkingDir(tmpDir)

	aceSvc, err := builder.BuildACEService()
	if err != nil {
		t.Fatalf("BuildACEService() error = %v", err)
	}
	if aceSvc == nil {
		t.Fatal("BuildACEService() returned nil")
	}
}
