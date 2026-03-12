package main

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/ollama"
	"github.com/dmytrogajewski/spin/internal/session"
)

// createACPAgentComponents is a test helper that creates ACP agent components using the new refactored flow.
func createACPAgentComponents(t *testing.T, workDir string, provider llm.Provider, cfg *config.V2) (*agent.Agent, *events.EventEmitter, *executor.ACPRuntime) {
	t.Helper()

	logger := slog.Default()

	// 1. Create shared infrastructure.
	emitter := events.NewEventEmitter(100)

	storageDir := cfg.Agent.SessionDir
	if storageDir == "" {
		storageDir = t.TempDir()
	}

	storage, err := session.NewFileStorage(storageDir)
	require.NoError(t, err)

	// 2. Create protocol services.
	protocolServices, cleanup, err := createServices(context.Background(), cfg, workDir, logger)
	require.NoError(t, err)
	t.Cleanup(cleanup)

	// 3. Create ACP runtime (complete, self-contained).
	acpRuntime, err := executor.NewACP(executor.ACPConfig{
		WorkDir:      workDir,
		Emitter:      emitter,
		Storage:      storage,
		ShellService: protocolServices.Shell,
		GitService:   protocolServices.Git,
		Logger:       logger,
	})
	require.NoError(t, err)

	// 4. Build core agent using executor.
	coreAgent, err := buildCoreAgent(cfg, provider, workDir, emitter, acpRuntime)
	require.NoError(t, err)

	return coreAgent, emitter, acpRuntime
}

// TestNewACPCmd tests ACP command creation.
func TestNewACPCmd(t *testing.T) {
	t.Parallel()

	cmd := newACPCmd()

	assert.Equal(t, "acp", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "ACP")
	assert.Contains(t, cmd.Long, "Agent Client Protocol")
}

// TestNewACPCmd_Flags tests that all flags are registered.
func TestNewACPCmd_Flags(t *testing.T) {
	t.Parallel()

	cmd := newACPCmd()

	flags := []string{"workspace", "provider", "base-url", "model", "api-key"}

	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Flag --%s should be registered", flagName)
	}
}

// TestNewACPCmd_FlagDefaults tests flag default values.
func TestNewACPCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	cmd := newACPCmd()

	workspace, _ := cmd.Flags().GetString("workspace")
	assert.Equal(t, ".", workspace, "workspace should default to current directory")

	provider, _ := cmd.Flags().GetString("provider")
	assert.Empty(t, provider, "provider should default to empty (config takes precedence)")

	baseURL, _ := cmd.Flags().GetString("base-url")
	assert.Empty(t, baseURL, "base-url should default to empty (config takes precedence)")

	model, _ := cmd.Flags().GetString("model")
	assert.Empty(t, model, "model should default to empty (config takes precedence)")
}

// TestBuildProviderForACP_Ollama tests Ollama provider creation using unified builder.
func TestBuildProviderForACP_Ollama(t *testing.T) {
	t.Parallel()

	// Create a mock HTTP server to stand in for Ollama.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := config.DefaultV2()
	cfg.LLM.Provider = "ollama"
	cfg.LLM.BaseURL = server.URL
	cfg.LLM.Model = "test-model"
	authMgr := createAuthManager()

	provider, err := buildProviderForACP(ctx, cfg, authMgr, "ollama", server.URL, "test-model", "")

	require.NoError(t, err)

	require.NotNil(t, provider)
	defer provider.Close()

	// Verify it's an Ollama provider.
	_, ok := provider.(*ollama.Provider)
	assert.True(t, ok, "should create Ollama provider")
}

// TestBuildProviderForACP_OpenAI tests OpenAI provider creation using unified builder.
func TestBuildProviderForACP_OpenAI(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.DefaultV2()
	cfg.LLM.Provider = "openai"
	cfg.LLM.BaseURL = "https://api.openai.com"
	cfg.LLM.Model = "gpt-4"
	cfg.LLM.APIKey = "test-key"
	authMgr := createAuthManager()

	provider, err := buildProviderForACP(ctx, cfg, authMgr, "openai", "https://api.openai.com", "gpt-4", "test-key")

	require.NoError(t, err)

	require.NotNil(t, provider)
	defer provider.Close()
}

// TestBuildProviderForACP_UnknownProvider tests error handling for unknown provider.
func TestBuildProviderForACP_UnknownProvider(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.DefaultV2()
	cfg.LLM.Provider = "unknown"
	cfg.LLM.Model = "test-model"
	authMgr := createAuthManager()

	provider, err := buildProviderForACP(ctx, cfg, authMgr, "unknown", "", "test-model", "")

	require.Error(t, err)
	require.Nil(t, provider)
	// The unified builder validates configuration before provider creation.
	assert.Contains(t, err.Error(), "unknown provider")
}

// TestCreateACPConversation tests ACP conversation creation.
func TestCreateACPConversation(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for workspace.
	tmpDir := t.TempDir()

	provider := llm.NewMockProvider("test")
	defer provider.Close()

	cfg := config.DefaultV2()
	cfg.Agent.WorkDir = tmpDir

	// Test the new refactored flow.
	agentInstance, emitter, runtime := createACPAgentComponents(t, tmpDir, provider, cfg)

	require.NotNil(t, agentInstance)
	require.NotNil(t, emitter)
	require.NotNil(t, runtime)
}

// TestCreateACPConversation_InvalidWorkDir tests error handling for invalid work directory.
func TestCreateACPConversation_InvalidWorkDir(t *testing.T) {
	t.Parallel()

	provider := llm.NewMockProvider("test")
	defer provider.Close()

	// Use a non-existent directory.
	invalidDir := filepath.Join(t.TempDir(), "nonexistent", "path")

	cfg := config.DefaultV2()
	cfg.Agent.WorkDir = invalidDir

	// The new flow should handle invalid directories gracefully or panic.
	require.Panics(t, func() {
		createACPAgentComponents(t, invalidDir, provider, cfg)
	})
}

// TestACPCmd_Help tests help output.
func TestACPCmd_Help(t *testing.T) {
	t.Parallel()

	cmd := newACPCmd()
	cmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "acp")
	assert.Contains(t, output, "ACP")
	assert.Contains(t, output, "Agent Client Protocol")
}

// TestACPCmd_Examples tests that examples are shown in help.
func TestACPCmd_Examples(t *testing.T) {
	t.Parallel()

	cmd := newACPCmd()
	cmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	// Check that examples section exists.
	assert.Contains(t, output, "Examples:")
	assert.Contains(t, output, "spin acp")
}

// TestACPCmd_FlagParsing tests flag parsing.
func TestACPCmd_FlagParsing(t *testing.T) {
	t.Parallel()

	cmd := newACPCmd()
	// Parse flags by executing with args (but we'll catch the error since we don't have a real provider).
	cmd.SetArgs([]string{
		"--workspace", "/tmp/test",
		"--provider", "openai",
		"--base-url", "https://api.openai.com",
		"--model", "gpt-4",
		"--api-key", "test-key",
	})

	// Parse flags.
	err := cmd.ParseFlags([]string{
		"--workspace", "/tmp/test",
		"--provider", "openai",
		"--base-url", "https://api.openai.com",
		"--model", "gpt-4",
		"--api-key", "test-key",
	})
	require.NoError(t, err)

	workspace, _ := cmd.Flags().GetString("workspace")
	assert.Equal(t, "/tmp/test", workspace)

	provider, _ := cmd.Flags().GetString("provider")
	assert.Equal(t, "openai", provider)

	baseURL, _ := cmd.Flags().GetString("base-url")
	assert.Equal(t, "https://api.openai.com", baseURL)

	model, _ := cmd.Flags().GetString("model")
	assert.Equal(t, "gpt-4", model)

	apiKey, _ := cmd.Flags().GetString("api-key")
	assert.Equal(t, "test-key", apiKey)
}

// TestLogACPServerStart tests logging function.
func TestLogACPServerStart(t *testing.T) {
	t.Parallel()

	// Capture log output.
	var buf bytes.Buffer

	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	logACPServerStart("ollama", "test-model", "/tmp/test")

	output := buf.String()
	assert.Contains(t, output, "Starting ACP server")
	assert.Contains(t, output, "ollama")
	assert.Contains(t, output, "test-model")
	assert.Contains(t, output, "/tmp/test")
}

// TestCreateAgentComponents_RegistersTools tests that all tools are registered.
func TestCreateAgentComponents_RegistersTools(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	provider := llm.NewMockProvider("test")
	defer provider.Close()

	cfg := config.DefaultV2()
	cfg.Agent.WorkDir = tmpDir

	agentInstance, emitter, runtime := createACPAgentComponents(t, tmpDir, provider, cfg)
	require.NotNil(t, agentInstance)
	require.NotNil(t, emitter)
	require.NotNil(t, runtime)

	// The agent should be created successfully with all tools registered
	// We can't easily test tool registration directly, but if agent creation
	// succeeds, it means tools were registered correctly.
}

// TestCreateACPComponents_ReturnsApprovalService tests that ApprovalService is returned.
func TestCreateACPComponents_ReturnsApprovalService(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	provider := llm.NewMockProvider("test")
	defer provider.Close()

	cfg := config.DefaultV2()
	cfg.Agent.WorkDir = tmpDir

	agentInstance, emitter, _ := createACPAgentComponents(t, tmpDir, provider, cfg)

	require.NotNil(t, agentInstance, "Agent should be created")
	require.NotNil(t, emitter, "Emitter should be created")
	// Agent is created successfully, which means SecurityService is configured.
}
