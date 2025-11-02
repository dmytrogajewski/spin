package conversation

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/config"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig creates a valid test configuration
func testConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	return cfg
}

func TestNewBuilder(t *testing.T) {
	cfg := testConfig()
	workDir := "/tmp"

	b := NewBuilder(cfg, workDir)
	require.NotNil(t, b)
	assert.Equal(t, cfg, b.cfg)
	assert.Equal(t, workDir, b.workDir)
}

func TestBuilder_WithGit(t *testing.T) {
	cfg := testConfig()
	b := NewBuilder(cfg, "/tmp")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	gitSvc, err := gitpkg.NewService(false, "/tmp", logger)
	require.NoError(t, err)
	defer gitSvc.Close()

	result := b.WithGit(gitSvc)
	assert.Equal(t, b, result) // Fluent API
	assert.Equal(t, gitSvc, b.gitService)
}

func TestBuilder_WithShell(t *testing.T) {
	cfg := testConfig()
	b := NewBuilder(cfg, "/tmp")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	shellSvc, err := shellpkg.NewService(false, "/tmp", logger, 30*time.Second)
	require.NoError(t, err)
	defer shellSvc.Close()

	result := b.WithShell(shellSvc)
	assert.Equal(t, b, result) // Fluent API
	assert.Equal(t, shellSvc, b.shellService)
}

func TestBuilder_WithMCP(t *testing.T) {
	cfg := testConfig()
	b := NewBuilder(cfg, "/tmp")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mcpCfg := &mcppkg.Config{
		EnableMCP:  false,
		MCPServers: []mcppkg.MCPServerConfig{},
	}
	mcpSvc, err := mcppkg.NewService(mcpCfg, logger)
	require.NoError(t, err)
	defer mcpSvc.Close()

	result := b.WithMCP(mcpSvc)
	assert.Equal(t, b, result) // Fluent API
	assert.Equal(t, mcpSvc, b.mcpService)
}

func TestBuilder_WithLLM(t *testing.T) {
	cfg := testConfig()
	b := NewBuilder(cfg, "/tmp")

	provider := llm.NewMockProvider("test")

	result := b.WithLLM(provider)
	assert.Equal(t, b, result) // Fluent API
	assert.Equal(t, provider, b.llm)
}

func TestBuilder_Build_Minimal(t *testing.T) {
	cfg := testConfig()
	tempDir := t.TempDir()

	b := NewBuilder(cfg, tempDir)

	conv, err := b.Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conv)
	defer conv.Close()

	// Verify conversation structure
	assert.Equal(t, tempDir, conv.workDir)
	assert.Equal(t, "regular", conv.taskMode)
	assert.NotEmpty(t, conv.sessionID)
	assert.NotNil(t, conv.agent)
	assert.NotNil(t, conv.history)
	assert.NotNil(t, conv.emitter)

	// Services should be nil (not provided)
	assert.Nil(t, conv.gitService)
	assert.Nil(t, conv.shellService)
	assert.Nil(t, conv.mcpService)
}

func TestBuilder_Build_WithServices(t *testing.T) {
	cfg := testConfig()
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create services
	gitSvc, err := gitpkg.NewService(false, tempDir, logger)
	require.NoError(t, err)
	defer gitSvc.Close()

	shellSvc, err := shellpkg.NewService(false, tempDir, logger, 30*time.Second)
	require.NoError(t, err)
	defer shellSvc.Close()

	mcpCfg := &mcppkg.Config{
		EnableMCP:  false,
		MCPServers: []mcppkg.MCPServerConfig{},
	}
	mcpSvc, err := mcppkg.NewService(mcpCfg, logger)
	require.NoError(t, err)
	defer mcpSvc.Close()

	// Build conversation with all services
	conv, err := NewBuilder(cfg, tempDir).
		WithGit(gitSvc).
		WithShell(shellSvc).
		WithMCP(mcpSvc).
		Build(context.Background())

	require.NoError(t, err)
	require.NotNil(t, conv)
	defer conv.Close()

	// Verify all services are set
	assert.NotNil(t, conv.gitService)
	assert.NotNil(t, conv.shellService)
	assert.NotNil(t, conv.mcpService)
	assert.Equal(t, gitSvc, conv.gitService)
	assert.Equal(t, shellSvc, conv.shellService)
	assert.Equal(t, mcpSvc, conv.mcpService)
}

func TestBuilder_Build_NilConfig(t *testing.T) {
	b := NewBuilder(nil, "/tmp")

	_, err := b.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestBuilder_Build_EmptyWorkDir(t *testing.T) {
	cfg := testConfig()
	b := NewBuilder(cfg, "")

	_, err := b.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workDir cannot be empty")
}

func TestBuilder_FluentAPI(t *testing.T) {
	cfg := testConfig()
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test fluent API chaining
	gitSvc, _ := gitpkg.NewService(false, tempDir, logger)
	defer gitSvc.Close()

	shellSvc, _ := shellpkg.NewService(false, tempDir, logger, 30*time.Second)
	defer shellSvc.Close()

	customLLM := llm.NewMockProvider("test")

	conv, err := NewBuilder(cfg, tempDir).
		WithGit(gitSvc).
		WithShell(shellSvc).
		WithLLM(customLLM).
		Build(context.Background())

	require.NoError(t, err)
	require.NotNil(t, conv)
	defer conv.Close()

	assert.NotNil(t, conv.gitService)
	assert.NotNil(t, conv.shellService)
}

func TestBuilder_ServiceReuse(t *testing.T) {
	cfg := testConfig()
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a single Git service
	gitSvc, err := gitpkg.NewService(false, tempDir, logger)
	require.NoError(t, err)
	defer gitSvc.Close()

	// Create two conversations sharing the same service
	conv1, err := NewBuilder(cfg, tempDir).
		WithGit(gitSvc).
		Build(context.Background())
	require.NoError(t, err)
	defer conv1.Close()

	conv2, err := NewBuilder(cfg, tempDir).
		WithGit(gitSvc).
		Build(context.Background())
	require.NoError(t, err)
	defer conv2.Close()

	// Both should reference the same service
	assert.Equal(t, conv1.gitService, conv2.gitService)

	// Closing conv1 should not close the service
	err = conv1.Close()
	assert.NoError(t, err)

	// Service should still be usable by conv2
	assert.NotNil(t, conv2.gitService)
	info := conv2.gitService.GetContextInfo()
	assert.NotNil(t, info)
}
