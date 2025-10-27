package manager

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilder_NewBuilder tests builder creation
func TestBuilder_NewBuilder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	builder := NewBuilder(cfg)

	assert.NotNil(t, builder, "builder should not be nil")
	assert.Equal(t, cfg, builder.cfg, "builder should have config")
}

// TestBuilder_WithMethods tests builder fluent interface methods
func TestBuilder_WithMethods(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	mockLLM := llm.NewMockProvider("test")
	mockEmitter := events.NewEventEmitter(100)
	mockLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	toolRegistry := tools.NewRegistry()
	taskRegistry := orchestration.NewRegistry()

	builder := NewBuilder(cfg).
		WithLLM(mockLLM).
		WithEventEmitter(mockEmitter).
		WithLogger(mockLogger).
		WithApprovalHandler(mockApprovalHandler).
		WithToolRegistry(toolRegistry).
		WithTaskRegistry(taskRegistry)

	assert.Equal(t, mockLLM, builder.llm, "LLM should be set")
	assert.Equal(t, mockEmitter, builder.emitter, "emitter should be set")
	assert.Equal(t, mockLogger, builder.logger, "logger should be set")
	assert.NotNil(t, builder.approvalHandler, "approval handler should be set")
	assert.Equal(t, toolRegistry, builder.toolRegistry, "tool registry should be set")
	assert.Equal(t, taskRegistry, builder.taskRegistry, "task registry should be set")
}

// TestBuilder_Build_Success tests successful manager construction
func TestBuilder_Build_Success(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = false

	mockLLM := llm.NewMockProvider("test")
	builder := NewBuilder(cfg).WithLLM(mockLLM)

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	require.NotNil(t, mgr, "Manager should not be nil")

	assert.Equal(t, cfg, mgr.cfg, "Manager should have config")
	assert.Equal(t, mockLLM, mgr.llm, "Manager should have LLM")
	assert.NotNil(t, mgr.emitter, "Manager should have event emitter")
	assert.NotNil(t, mgr.storage, "Manager should have storage")
	assert.NotNil(t, mgr.authManager, "Manager should have auth manager")
}

// TestBuilder_Build_NilConfig tests validation failure
func TestBuilder_Build_NilConfig(t *testing.T) {
	builder := NewBuilder(nil)

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	assert.Error(t, err, "Build should fail with nil config")
	assert.Nil(t, mgr, "Manager should be nil on error")
	assert.Contains(t, err.Error(), "validation failed", "error should mention validation")
}

// TestBuilder_Build_InvalidConfig tests validation failure with invalid config
func TestBuilder_Build_InvalidConfig(t *testing.T) {
	cfg := &Config{
		Provider:  "", // Invalid: empty provider
		Model:     "test-model",
		MaxTurns:  -1, // Invalid: negative
		Timeout:   0,  // Invalid: zero
		MaxTokens: 0,  // Invalid: zero
	}

	builder := NewBuilder(cfg)
	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	assert.Error(t, err, "Build should fail with invalid config")
	assert.Nil(t, mgr, "Manager should be nil on error")
}

// TestBuilder_Build_WithDefaultLLM tests that a default LLM is created if none provided
func TestBuilder_Build_WithDefaultLLM(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = false

	builder := NewBuilder(cfg)
	// Don't set LLM

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	require.NotNil(t, mgr, "Manager should not be nil")
	assert.NotNil(t, mgr.llm, "Manager should have default LLM")
}

// TestBuilder_Build_WithCustomStorage tests custom storage
func TestBuilder_Build_WithCustomStorage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = false

	mockStorage, err := session.NewFileStorage(cfg.SessionDir)
	require.NoError(t, err)

	builder := NewBuilder(cfg).WithStorage(mockStorage)

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	assert.Equal(t, mockStorage, mgr.storage, "Manager should have custom storage")
}

// TestBuilder_Build_WithMCPIntegration tests MCP integration initialization
func TestBuilder_Build_WithMCPIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip this test - requires actual MCP server to work properly
	// The echo command is not a valid MCP server and causes hangs
	t.Skip("skipping MCP integration test - requires actual MCP server implementation")

	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = true
	cfg.MCPServers = []MCPServerConfig{
		{
			Name:    "test-server",
			Command: "echo",
			Args:    []string{"test"},
		},
	}
	cfg.EnableGit = false
	cfg.EnableShell = false

	builder := NewBuilder(cfg)

	// Use timeout context to prevent hang when MCP server doesn't respond
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	// MCP initialization may fail but shouldn't prevent manager creation
	// Just verify manager was created
	assert.NotNil(t, mgr, "Manager should be created")
}

// TestBuilder_Build_WithGitIntegration tests Git integration initialization
func TestBuilder_Build_WithGitIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = true
	cfg.EnableShell = false

	builder := NewBuilder(cfg)

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	assert.NotNil(t, mgr, "Manager should be created")
	// Git integration may fail if not a git repo, but manager should still be created
}

// TestBuilder_Build_WithShellContext tests Shell context initialization
func TestBuilder_Build_WithShellContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = true
	cfg.ShellTimeout = 5 * time.Minute

	builder := NewBuilder(cfg)

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	assert.NotNil(t, mgr, "Manager should be created")
	assert.NotNil(t, mgr.shellIntegration, "Shell context should be initialized")
}

// TestBuilder_Build_WithEventEmitter tests custom event emitter
func TestBuilder_Build_WithEventEmitter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = false

	customEmitter := events.NewEventEmitter(200)
	builder := NewBuilder(cfg).WithEventEmitter(customEmitter)

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	assert.Equal(t, customEmitter, mgr.emitter, "Manager should have custom emitter")
}

// TestBuilder_Build_WithStreamBufferConfig tests stream buffer configuration
func TestBuilder_Build_WithStreamBufferConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.StreamBuffer = 500
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = false

	builder := NewBuilder(cfg)

	ctx := context.Background()
	mgr, err := builder.Build(ctx)

	require.NoError(t, err, "Build should succeed")
	assert.NotNil(t, mgr.emitter, "Manager should have emitter")
	// Note: Can't directly verify buffer size, but emitter should be created
}

// TestNewManager_BackwardCompatibility tests that NewManager still works
func TestNewManager_BackwardCompatibility(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = false

	mockLLM := llm.NewMockProvider("test")
	toolRegistry := tools.NewRegistry()

	// Test old-style NewManager with functional options
	mgr, err := NewManager(cfg,
		WithLLM(mockLLM),
		WithManagerToolRegistry(toolRegistry),
		WithManagerApprovalHandler(mockApprovalHandler),
	)

	require.NoError(t, err, "NewManager should succeed")
	require.NotNil(t, mgr, "Manager should not be nil")

	assert.Equal(t, mockLLM, mgr.llm, "Manager should have custom LLM")
	assert.Equal(t, toolRegistry, mgr.toolRegistry, "Manager should have custom tool registry")
	assert.NotNil(t, mgr.approvalHandler, "Manager should have custom approval handler")
}

// TestBuilder_FluentInterface tests builder pattern chaining
func TestBuilder_FluentInterface(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()
	cfg.EnableMCP = false
	cfg.EnableGit = false
	cfg.EnableShell = false

	// Test that all With methods return the builder for chaining
	mgr, err := NewBuilder(cfg).
		WithLLM(llm.NewMockProvider("test")).
		WithEventEmitter(events.NewEventEmitter(100)).
		WithLogger(slog.Default()).
		WithApprovalHandler(mockApprovalHandler).
		WithToolRegistry(tools.NewRegistry()).
		WithTaskRegistry(orchestration.NewRegistry()).
		Build(context.Background())

	require.NoError(t, err, "Build should succeed")
	assert.NotNil(t, mgr, "Manager should not be nil")
}

// mockApprovalHandler is a test double for approval handler
func mockApprovalHandler(req security.ApprovalRequest) security.ApprovalResponse {
	return security.ApprovalResponse{
		Approved: true,
		Reason:   "auto-approved for testing",
	}
}
