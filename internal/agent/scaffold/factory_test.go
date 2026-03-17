package scaffold

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Journey: specs/journeys/JOURNEY-1.2.md.

const (
	testMaxTurns    = 50
	testTimeout     = 60 * time.Minute
	testTemperature = 0.7
	testMaxTokens   = 8192
)

func validTestConfig() *config.V2 {
	return &config.V2{
		Version: "2.0",
		LLM: config.LLMV2{
			Provider:    "ollama",
			Model:       "qwen2.5-coder:7b",
			Temperature: testTemperature,
			MaxTokens:   testMaxTokens,
			Timeout:     5 * time.Minute,
		},
		Agent: config.AgentV2{
			MaxTurns: testMaxTurns,
			Timeout:  testTimeout,
			WorkDir:  "/tmp",
		},
	}
}

// TestNewFactory_NilConfig tests that nil config is rejected.
// Kills mutant: removing nil check would make this test fail.
func TestNewFactory_NilConfig(t *testing.T) {
	t.Parallel()

	registry := tools.NewRegistry()

	_, err := NewFactory(nil, registry, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilConfig)
}

// TestNewFactory_NilRegistry tests that nil registry is rejected.
// Kills mutant: removing nil check would make this test fail.
func TestNewFactory_NilRegistry(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()

	_, err := NewFactory(cfg, nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilRegistry)
}

// TestNewFactory_Valid tests successful factory creation.
// Kills mutant: returning error for valid inputs would make this test fail.
func TestNewFactory_Valid(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()
	registry := tools.NewRegistry()

	factory, err := NewFactory(cfg, registry, nil)
	require.NoError(t, err)
	assert.NotNil(t, factory)
}

// TestCompile_Main_ToolSchemas tests that Compile("main") includes all built-in tools.
// Kills mutant: returning empty schemas would make this test fail.
func TestCompile_Main_ToolSchemas(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()
	registry := tools.NewRegistryWithBuiltins()

	factory, err := NewFactory(cfg, registry, nil)
	require.NoError(t, err)

	spec, err := factory.Compile(AgentTypeMain)
	require.NoError(t, err)

	// Main agent should have all builtin tools.
	assert.NotEmpty(t, spec.ToolSchemas, "main agent should have tool schemas")

	assert.Len(t, spec.ToolSchemas, len(tools.BuiltinTools),
		"main agent should have all builtin tools")
}

// TestCompile_Main_SystemPrompt tests that Compile("main") produces a non-empty system prompt.
// Kills mutant: returning empty prompt would make this test fail.
func TestCompile_Main_SystemPrompt(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()
	registry := tools.NewRegistry()

	factory, err := NewFactory(cfg, registry, nil)
	require.NoError(t, err)

	spec, err := factory.Compile(AgentTypeMain)
	require.NoError(t, err)
	assert.NotEmpty(t, spec.SystemPrompt, "main agent should have a system prompt")
}

// TestCompile_Main_Config tests that Compile("main") populates config from V2.
// Kills mutant: ignoring config values would make this test fail.
func TestCompile_Main_Config(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()
	registry := tools.NewRegistry()

	factory, err := NewFactory(cfg, registry, nil)
	require.NoError(t, err)

	spec, err := factory.Compile(AgentTypeMain)
	require.NoError(t, err)

	assert.Equal(t, testMaxTurns, spec.Config.MaxTurns)
	assert.Equal(t, testTimeout, spec.Config.Timeout)
	assert.InDelta(t, testTemperature, spec.Config.Temperature, 1e-9)
	assert.Equal(t, testMaxTokens, spec.Config.MaxTokens)
}

// TestCompile_Main_NotSubagent tests that the main agent is not marked as a subagent.
// Kills mutant: setting IsSubagent=true would make this test fail.
func TestCompile_Main_NotSubagent(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()
	registry := tools.NewRegistry()

	factory, err := NewFactory(cfg, registry, nil)
	require.NoError(t, err)

	spec, err := factory.Compile(AgentTypeMain)
	require.NoError(t, err)

	assert.False(t, spec.IsSubagent, "main agent should not be a subagent")
	assert.Nil(t, spec.AllowedTools, "main agent should have no tool restrictions")
}

// TestCompile_UnknownType tests that Compile with unknown type returns error.
// Kills mutant: accepting unknown types would make this test fail.
func TestCompile_UnknownType(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()
	registry := tools.NewRegistry()

	factory, err := NewFactory(cfg, registry, nil)
	require.NoError(t, err)

	_, err = factory.Compile("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownAgentType)
}
