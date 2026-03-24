package conversation

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	agentexec "github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/safety"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
)

// testConfig creates a valid test configuration.
func testConfig() *config.V2 {
	cfg := config.DefaultV2()
	cfg.LLM.Provider = "mock"
	cfg.LLM.Model = "test-model"

	return cfg
}

// createTestRuntime creates a builtin runtime for testing with auto-approve handler.
func createTestRuntime(t *testing.T, workDir string) (*agentexec.BuiltinRuntime, *events.EventEmitter, *llm.MockProvider) {
	t.Helper()

	emitter := events.NewEventEmitter(100)
	provider := llm.NewMockProvider("test")

	// Auto-approve handler for tests.
	approvalHandler := func(_ context.Context, req safety.ApprovalRequest) safety.ApprovalResponse {
		return safety.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "test auto-approve",
		}
	}

	executor, err := agent.NewExecutor(workDir)
	require.NoError(t, err)

	validator := safety.NewValidator()

	builtinRuntime, err := agentexec.NewBuiltinRuntime(agentexec.BuiltinRuntimeConfig{
		WorkDir:         workDir,
		Emitter:         emitter,
		Storage:         nil, // No persistence in tests.
		SessionID:       fmt.Sprintf("test-%d", time.Now().UnixNano()),
		Executor:        agent.NewExecutorRuntimeAdapter(executor),
		Validator:       validator,
		UI:              nil, // No UI in tests.
		ApprovalHandler: approvalHandler,
		Logger:          slog.Default(),
	})
	require.NoError(t, err)

	return builtinRuntime, emitter, provider
}

func TestNewBuilder(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)

	b := NewBuilder(cfg, workDir, rt, emitter, provider)
	require.NotNil(t, b)
	assert.Equal(t, cfg, b.cfg)
	assert.Equal(t, workDir, b.workDir)
	assert.Equal(t, rt, b.runtime)
	assert.Equal(t, emitter, b.emitter)
	assert.Equal(t, provider, b.llm)
}

func TestBuilder_WithGit(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)
	b := NewBuilder(cfg, workDir, rt, emitter, provider)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	gitSvc, err := gitpkg.NewService(context.Background(), false, "/tmp", logger)
	require.NoError(t, err)

	defer gitSvc.Close()

	result := b.WithGit(gitSvc)
	assert.Equal(t, b, result) // Fluent API.
	assert.Equal(t, gitSvc, b.gitService)
}

func TestBuilder_WithShell(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)
	b := NewBuilder(cfg, workDir, rt, emitter, provider)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	shellSvc, err := shellpkg.NewService(context.Background(), false, workDir, logger, 30*time.Second)
	require.NoError(t, err)

	defer shellSvc.Close()

	result := b.WithShell(shellSvc)
	assert.Equal(t, b, result) // Fluent API.
	assert.Equal(t, shellSvc, b.shellService)
}

func TestBuilder_WithMCP(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)
	b := NewBuilder(cfg, workDir, rt, emitter, provider)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	mcpSvc := mcppkg.NewService(mcppkg.NewDefaultRegistryManager(logger))
	defer mcpSvc.Close()

	result := b.WithMCP(mcpSvc)
	assert.Equal(t, b, result) // Fluent API.
	assert.Equal(t, mcpSvc, b.mcpService)
}

func TestBuilder_Build_Minimal(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	tempDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, tempDir)

	b := NewBuilder(cfg, tempDir, rt, emitter, provider)

	conv, err := b.Build(context.Background())
	require.NoError(t, err)

	require.NotNil(t, conv)
	defer conv.Close(context.Background())

	// Verify conversation structure.
	assert.Equal(t, tempDir, conv.workDir)
	assert.Equal(t, "regular", conv.taskMode)
	assert.NotEmpty(t, conv.GetSessionID())
	assert.NotNil(t, conv.agent)
	assert.NotNil(t, conv.history)
	assert.NotNil(t, conv.emitter)

	// Services should be nil (not provided).
	assert.Nil(t, conv.gitService)
	assert.Nil(t, conv.shellService)
	assert.Nil(t, conv.mcpService)
}

func TestBuilder_Build_WithServices(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create services.
	gitSvc, err := gitpkg.NewService(context.Background(), false, tempDir, logger)
	require.NoError(t, err)

	defer gitSvc.Close()

	shellSvc, err := shellpkg.NewService(context.Background(), false, tempDir, logger, 30*time.Second)
	require.NoError(t, err)

	defer shellSvc.Close()

	mcpSvc := mcppkg.NewService(mcppkg.NewDefaultRegistryManager(logger))
	defer mcpSvc.Close()

	// Build conversation with all services.
	rt, emitter, provider := createTestRuntime(t, tempDir)
	conv, err := NewBuilder(cfg, tempDir, rt, emitter, provider).
		WithGit(gitSvc).
		WithShell(shellSvc).
		WithMCP(mcpSvc).
		Build(context.Background())

	require.NoError(t, err)

	require.NotNil(t, conv)
	defer conv.Close(context.Background())

	// Verify all services are set.
	assert.NotNil(t, conv.gitService)
	assert.NotNil(t, conv.shellService)
	assert.NotNil(t, conv.mcpService)
	assert.Equal(t, gitSvc, conv.gitService)
	assert.Equal(t, shellSvc, conv.shellService)
	assert.Equal(t, mcpSvc, conv.mcpService)
}

func TestBuilder_Build_NilConfig(t *testing.T) {
	t.Parallel()

	// Panics are caught by require.Panics.
	require.Panics(t, func() {
		NewBuilder(nil, "/tmp", nil, nil, nil)
	})
}

func TestBuilder_Build_EmptyWorkDir(t *testing.T) {
	t.Parallel()

	// Panics are caught by require.Panics.
	require.Panics(t, func() {
		NewBuilder(testConfig(), "", nil, nil, nil)
	})
}

func TestBuilder_FluentAPI(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test fluent API chaining.
	gitSvc, _ := gitpkg.NewService(context.Background(), false, tempDir, logger)
	defer gitSvc.Close()

	shellSvc, _ := shellpkg.NewService(context.Background(), false, tempDir, logger, 30*time.Second)
	defer shellSvc.Close()

	rt, emitter, provider := createTestRuntime(t, tempDir)

	conv, err := NewBuilder(cfg, tempDir, rt, emitter, provider).
		WithGit(gitSvc).
		WithShell(shellSvc).
		Build(context.Background())

	require.NoError(t, err)

	require.NotNil(t, conv)
	defer conv.Close(context.Background())

	assert.NotNil(t, conv.gitService)
	assert.NotNil(t, conv.shellService)
}

func TestBuilder_ServiceReuse(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a single Git service.
	gitSvc, err := gitpkg.NewService(context.Background(), false, tempDir, logger)
	require.NoError(t, err)

	defer gitSvc.Close()

	// Create two conversations sharing the same service.
	rt1, emitter1, provider1 := createTestRuntime(t, tempDir)
	conv1, err := NewBuilder(cfg, tempDir, rt1, emitter1, provider1).
		WithGit(gitSvc).
		Build(context.Background())
	require.NoError(t, err)

	defer conv1.Close(context.Background())

	rt2, emitter2, provider2 := createTestRuntime(t, tempDir)
	conv2, err := NewBuilder(cfg, tempDir, rt2, emitter2, provider2).
		WithGit(gitSvc).
		Build(context.Background())
	require.NoError(t, err)

	defer conv2.Close(context.Background())

	// Both should reference the same service.
	assert.Equal(t, conv1.gitService, conv2.gitService)

	// Closing conv1 should not close the service.
	err = conv1.Close(context.Background())
	require.NoError(t, err)

	// Service should still be usable by conv2.
	assert.NotNil(t, conv2.gitService)
	info := conv2.gitService.GetContextInfo()
	assert.NotNil(t, info)
}
