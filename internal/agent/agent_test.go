package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// newTestSecurityService creates a security service for testing.
func newTestSecurityService() *safety.Service {
	validator := safety.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := safety.NewApprovalServiceWithConfig(safety.ApprovalServiceConfig{
		Handler: nil, Emitter: emitter, Validator: validator,
	})

	return safety.NewService(validator, approvalService)
}

// newAgentTestDeps creates a full set of valid dependencies for NewAgent tests.
type newAgentTestDeps struct {
	provider    llm.Provider
	security    *safety.Service
	detection   *cycle.Service
	toolRuntime *tool.Runtime
	environment *Environment
	emitter     *events.EventEmitter
}

func validAgentDeps() newAgentTestDeps {
	return newAgentTestDeps{
		provider:    llm.NewMockProvider("test"),
		security:    newTestSecurityService(),
		detection:   cycle.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
		toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
		environment: &Environment{WorkDir: "/tmp"},
		emitter:     events.NewEventEmitter(100),
	}
}

func newTestToolRuntime(_ any, registry *tools.Registry) *tool.Runtime {
	validator := safety.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := safety.NewApprovalServiceWithConfig(safety.ApprovalServiceConfig{
		Handler: nil, Emitter: emitter, Validator: validator,
	})

	if registry == nil {
		registry = tools.NewRegistry()
	}

	return tool.NewRuntime(tool.RuntimeConfig{
		Registry:        registry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})
}

func TestNewAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modify      func(*newAgentTestDeps)
		wantErr     bool
		errContains string
	}{
		{name: "valid agent"},
		{
			name: "nil provider", wantErr: true, errContains: "LLM provider cannot be nil",
			modify: func(d *newAgentTestDeps) { d.provider = nil },
		},
		{
			name: "nil security", wantErr: true, errContains: "security service cannot be nil",
			modify: func(d *newAgentTestDeps) { d.security = nil },
		},
		{
			name: "nil detection", wantErr: true, errContains: "detection service cannot be nil",
			modify: func(d *newAgentTestDeps) { d.detection = nil },
		},
		{
			name: "nil tool runtime", wantErr: true, errContains: "tool runtime cannot be nil",
			modify: func(d *newAgentTestDeps) { d.toolRuntime = nil },
		},
		{
			name: "nil environment", wantErr: true, errContains: "context cannot be nil",
			modify: func(d *newAgentTestDeps) { d.environment = nil },
		},
		{
			name: "nil emitter", wantErr: true, errContains: "event emitter cannot be nil",
			modify: func(d *newAgentTestDeps) { d.emitter = nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := validAgentDeps()
			if tt.modify != nil {
				tt.modify(&deps)
			}

			agent, err := NewAgent(
				deps.provider, deps.security, deps.detection,
				deps.toolRuntime, deps.environment, deps.emitter,
			)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, agent)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, agent)
			}
		})
	}
}

func TestAgent_WithACEService(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfg := &ace.Config{
		Enabled:      true,
		PlaybookPath: tmpDir + "/test-playbook.json",
		Retrieval: ace.RetrievalConfig{
			TopK:     5,
			MinScore: 0.3,
		},
	}

	mockLLM := llm.NewMockProvider("test")
	aceService, err := ace.NewService(context.Background(), cfg, tmpDir, mockLLM, "test-model", 0)
	require.NoError(t, err)

	agent, err := NewAgent(
		mockLLM,
		newTestSecurityService(),
		cycle.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
		newTestToolRuntime(nil, tools.NewRegistry()),
		&Environment{WorkDir: "/tmp"},
		events.NewEventEmitter(100),
		WithACEService(aceService),
	)

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.NotNil(t, agent.aceService)
}

func TestAgent_Options(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		opt         Option
		wantErr     bool
		errContains string
		verify      func(t *testing.T, a *Agent)
	}{
		{
			name: "WithMaxTurns valid",
			opt:  WithMaxTurns(20),
			verify: func(t *testing.T, a *Agent) {
				t.Helper()
				assert.Equal(t, 20, a.maxTurns)
			},
		},
		{
			name:        "WithMaxTurns zero",
			opt:         WithMaxTurns(0),
			wantErr:     true,
			errContains: "max turns must be positive",
		},
		{
			name:        "WithMaxTurns negative",
			opt:         WithMaxTurns(-1),
			wantErr:     true,
			errContains: "max turns must be positive",
		},
		{
			name: "WithAgentTimeout valid",
			opt:  WithAgentTimeout(5 * time.Second),
			verify: func(t *testing.T, a *Agent) {
				t.Helper()
				assert.Equal(t, 5*time.Second, a.timeout)
			},
		},
		{
			name:        "WithAgentTimeout zero",
			opt:         WithAgentTimeout(0),
			wantErr:     true,
			errContains: "timeout must be positive",
		},
		{
			name: "WithTemperature valid",
			opt:  WithTemperature(1.0),
			verify: func(t *testing.T, a *Agent) {
				t.Helper()
				assert.InDelta(t, 1.0, a.temperature, 0.001)
			},
		},
		{
			name:        "WithTemperature too high",
			opt:         WithTemperature(3.0),
			wantErr:     true,
			errContains: "temperature must be between 0 and 2",
		},
		{
			name:        "WithTemperature negative",
			opt:         WithTemperature(-0.1),
			wantErr:     true,
			errContains: "temperature must be between 0 and 2",
		},
		{
			name: "WithMaxTokens valid",
			opt:  WithMaxTokens(4096),
			verify: func(t *testing.T, a *Agent) {
				t.Helper()
				assert.Equal(t, 4096, a.maxTokens)
			},
		},
		{
			name:        "WithMaxTokens zero",
			opt:         WithMaxTokens(0),
			wantErr:     true,
			errContains: "max tokens must be positive",
		},
		{
			name: "WithRequireApproval is no-op",
			opt:  WithRequireApproval(true),
		},
		{
			name: "WithACEConfig",
			opt:  WithACEConfig(&ace.Config{Enabled: true}),
			verify: func(t *testing.T, a *Agent) {
				t.Helper()
				assert.NotNil(t, a.aceConfig)
				assert.True(t, a.aceConfig.Enabled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := validAgentDeps()

			agent, err := NewAgent(
				deps.provider, deps.security, deps.detection,
				deps.toolRuntime, deps.environment, deps.emitter,
				tt.opt,
			)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)

				return
			}

			require.NoError(t, err)

			if tt.verify != nil {
				tt.verify(t, agent)
			}
		})
	}
}

func TestAgent_Accessors(t *testing.T) {
	t.Parallel()

	deps := validAgentDeps()

	agent, err := NewAgent(
		deps.provider, deps.security, deps.detection,
		deps.toolRuntime, deps.environment, deps.emitter,
	)
	require.NoError(t, err)

	t.Run("SecurityService", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, deps.security, agent.SecurityService())
	})

	t.Run("ToolRuntime", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, deps.toolRuntime, agent.ToolRuntime())
	})

	t.Run("ApprovalService does not panic on nil", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			agent.ApprovalService(nil)
		})
	})
}

func TestAgent_DefaultValues(t *testing.T) {
	t.Parallel()

	deps := validAgentDeps()

	agent, err := NewAgent(
		deps.provider, deps.security, deps.detection,
		deps.toolRuntime, deps.environment, deps.emitter,
	)
	require.NoError(t, err)

	assert.Equal(t, DefaultMaxTurns, agent.maxTurns)
	assert.Equal(t, DefaultAgentTimeout, agent.timeout)
	assert.InDelta(t, DefaultTemperature, agent.temperature, 0.001)
	assert.Equal(t, DefaultMaxTokens, agent.maxTokens)
}

func TestAgent_ValidateToolCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil tool call",
			call:    nil,
			wantErr: true,
		},
		{
			name: "empty ID",
			call: &ToolCall{
				ID:   "",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: true,
		},
		{
			name: "empty function name",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tools.ValidateToolCall(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgent_ParseToolArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{
			name:    "valid JSON arguments",
			args:    `{"path": "/tmp"}`,
			wantErr: false,
		},
		{
			name:    "empty arguments",
			args:    "",
			wantErr: true,
		},
		{
			name:    "invalid JSON arguments",
			args:    `{"path": "/tmp"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := tools.NewStrictArgumentParser()

			args, err := parser.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(args.Keys()) == 0 {
				t.Error("Parse() returned empty args for valid input")
			}
		})
	}
}

func TestValidateMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "regular mode", mode: "regular"},
		{name: "review mode", mode: "review"},
		{name: "compact mode", mode: "compact"},
		{name: "planning mode", mode: "planning"},
		{name: "invalid mode", mode: "invalid", wantErr: true},
		{name: "empty mode is valid", mode: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateMode(tt.mode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
