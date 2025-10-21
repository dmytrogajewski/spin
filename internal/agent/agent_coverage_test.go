package agent

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// newTestAgentMinimal creates a minimal agent for testing with services
func newTestAgentMinimal(toolRegistry *tools.Registry, taskRegistry *orchestration.Registry) *Agent {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	if toolRegistry == nil {
		toolRegistry = tools.NewRegistry()
	}
	if taskRegistry == nil {
		taskRegistry = orchestration.NewRegistry()
	}

	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	agent, _ := NewAgent(
		&mockLLMProvider{},
		securityService,
		detectionService,
		orchestrationService,
		&Environment{WorkDir: "/tmp"},
		emitter,
	)
	return agent
}

func TestAgent_processToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		agent     *Agent
		messages  []Message
		llmResp   *llm.CompletionResponse
		resp      *AgentResponse
		wantCount int
	}{
		{
			name:     "successful tool processing",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "Let me use a tool to help you.",
			},
			resp:      &AgentResponse{},
			wantCount: 1, // One message added
		},
		{
			name:     "empty response",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "",
			},
			resp:      &AgentResponse{},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.processToolCalls(context.Background(), tt.messages, tt.llmResp, tt.resp)

			if len(result) != tt.wantCount {
				t.Errorf("Agent.processToolCalls() result length = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestAgent_processToolCalls_WithToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		agent     *Agent
		messages  []Message
		llmResp   *llm.CompletionResponse
		resp      *AgentResponse
		wantCount int
	}{
		{
			name:     "response with tool calls",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "I'll help you with that.",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "read_file",
							Arguments: `{"path": "test.txt"}`,
						},
					},
				},
			},
			resp:      &AgentResponse{},
			wantCount: 2, // Assistant message + tool result message
		},
		{
			name:     "tool call with error",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "Let me try this tool.",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_456",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "invalid_tool",
							Arguments: `{}`,
						},
					},
				},
			},
			resp:      &AgentResponse{},
			wantCount: 2, // Assistant message + error message
		},
		{
			name:     "multiple tool calls",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "I'll use multiple tools.",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "read_file",
							Arguments: `{"path": "file1.txt"}`,
						},
					},
					{
						ID:   "call_2",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "read_file",
							Arguments: `{"path": "file2.txt"}`,
						},
					},
				},
			},
			resp:      &AgentResponse{},
			wantCount: 3, // Assistant message + 2 tool result messages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.processToolCalls(context.Background(), tt.messages, tt.llmResp, tt.resp)

			if len(result) != tt.wantCount {
				t.Errorf("processToolCalls() got %d messages, want %d", len(result), tt.wantCount)
			}

			// Verify assistant message was added first
			if len(result) > 0 {
				firstMsg := result[0]
				if firstMsg.Role != RoleAssistant {
					t.Errorf("processToolCalls() first message role = %v, want %v", firstMsg.Role, RoleAssistant)
				}
			}
		})
	}
}

func TestAgent_CreatePlan(t *testing.T) {
	tests := []struct {
		name    string
		agent   *Agent
		input   string
		wantErr bool
	}{
		{
			name:    "successful plan creation",
			agent:   newTestAgentMinimal(nil, nil),
			input:   "Create a plan for refactoring",
			wantErr: false,
		},
		{
			name:    "empty input",
			agent:   newTestAgentMinimal(nil, nil),
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := tt.agent.CreatePlan(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Agent.CreatePlan() expected error, got nil")
				}
				if plan != nil {
					t.Errorf("Agent.CreatePlan() expected nil plan on error, got %v", plan)
				}
			} else {
				if err != nil {
					t.Errorf("Agent.CreatePlan() unexpected error: %v", err)
				}
				if plan == nil {
					t.Errorf("Agent.CreatePlan() expected plan, got nil")
				}
			}
		})
	}
}

func TestAgent_selectIntervention(t *testing.T) {
	tests := []struct {
		name      string
		agent     *Agent
		cycleType cycle.CycleType
		turnCount int
		want      cycle.Intervention
	}{
		{
			name:      "soft intervention for early cycles",
			agent:     &Agent{},
			cycleType: cycle.CycleNone,
			turnCount: 5,
			want:      &cycle.ReflectionIntervention{},
		},
		{
			name:      "soft intervention",
			agent:     &Agent{},
			cycleType: cycle.CycleSimilarResponses,
			turnCount: 5,
			want:      &cycle.ReflectionIntervention{},
		},
		{
			name:      "hard intervention",
			agent:     &Agent{},
			cycleType: cycle.CycleSameError,
			turnCount: 15,
			want:      &cycle.EscalateIntervention{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.selectIntervention(tt.cycleType, tt.turnCount)

			if tt.want == nil {
				if result != nil {
					t.Errorf("Agent.selectIntervention() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("Agent.selectIntervention() = nil, want %v", tt.want)
				}
			}
		})
	}
}

// Mock LLM provider for testing
type mockLLMProvider struct{}

func (m *mockLLMProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		Content: `{"steps": [{"id": "step1", "description": "Mock step"}]}`,
	}, nil
}

func (m *mockLLMProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{
		Type:    llm.ChunkTypeContentDelta,
		Content: "Mock stream response",
	}
	close(ch)
	return ch, nil
}

func (m *mockLLMProvider) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{}, nil
}

func (m *mockLLMProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{}
}

func (m *mockLLMProvider) Name() string {
	return "mock"
}

func (m *mockLLMProvider) Close() error {
	return nil
}

// Test agent option functions
func TestAgent_WithMaxTurns(t *testing.T) {
	tests := []struct {
		name     string
		maxTurns int
		wantErr  bool
	}{
		{
			name:     "valid max turns",
			maxTurns: 10,
			wantErr:  false,
		},
		{
			name:     "zero max turns",
			maxTurns: 0,
			wantErr:  true,
		},
		{
			name:     "negative max turns",
			maxTurns: -1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				config: &Config{},
			}
			option := WithMaxTurns(tt.maxTurns)
			err := option(agent)

			if tt.wantErr {
				if err == nil {
					t.Error("WithMaxTurns() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("WithMaxTurns() unexpected error: %v", err)
				}
				if agent.config.MaxTurns != tt.maxTurns {
					t.Errorf("WithMaxTurns() MaxTurns = %v, want %v", agent.config.MaxTurns, tt.maxTurns)
				}
			}
		})
	}
}

func TestAgent_WithAgentTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "valid timeout",
			timeout: 30 * time.Second,
			wantErr: false,
		},
		{
			name:    "zero timeout",
			timeout: 0,
			wantErr: true,
		},
		{
			name:    "negative timeout",
			timeout: -1 * time.Second,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				config: &Config{},
			}
			option := WithAgentTimeout(tt.timeout)
			err := option(agent)

			if tt.wantErr {
				if err == nil {
					t.Error("WithAgentTimeout() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("WithAgentTimeout() unexpected error: %v", err)
				}
				if agent.config.Timeout != tt.timeout {
					t.Errorf("WithAgentTimeout() Timeout = %v, want %v", agent.config.Timeout, tt.timeout)
				}
			}
		})
	}
}

func TestAgent_WithTemperature(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		wantErr     bool
	}{
		{
			name:        "valid temperature",
			temperature: 0.7,
			wantErr:     false,
		},
		{
			name:        "zero temperature",
			temperature: 0.0,
			wantErr:     false,
		},
		{
			name:        "maximum temperature",
			temperature: 2.0,
			wantErr:     false,
		},
		{
			name:        "negative temperature",
			temperature: -0.1,
			wantErr:     true,
		},
		{
			name:        "too high temperature",
			temperature: 2.1,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				config: &Config{},
			}
			option := WithTemperature(tt.temperature)
			err := option(agent)

			if tt.wantErr {
				if err == nil {
					t.Error("WithTemperature() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("WithTemperature() unexpected error: %v", err)
				}
				if agent.config.Temperature != tt.temperature {
					t.Errorf("WithTemperature() Temperature = %v, want %v", agent.config.Temperature, tt.temperature)
				}
			}
		})
	}
}

func TestAgent_WithMaxTokens(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
		wantErr   bool
	}{
		{
			name:      "valid max tokens",
			maxTokens: 4096,
			wantErr:   false,
		},
		{
			name:      "zero max tokens",
			maxTokens: 0,
			wantErr:   true,
		},
		{
			name:      "negative max tokens",
			maxTokens: -1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				config: &Config{},
			}
			option := WithMaxTokens(tt.maxTokens)
			err := option(agent)

			if tt.wantErr {
				if err == nil {
					t.Error("WithMaxTokens() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("WithMaxTokens() unexpected error: %v", err)
				}
				if agent.config.MaxTokens != tt.maxTokens {
					t.Errorf("WithMaxTokens() MaxTokens = %v, want %v", agent.config.MaxTokens, tt.maxTokens)
				}
			}
		})
	}
}

func TestAgent_ShouldApprove_Coverage(t *testing.T) {
	tests := []struct {
		name              string
		agent             *Agent
		cmd               *security.Command
		wantNeedsApproval bool
		wantReason        string
	}{
		{
			name: "approval disabled",
			agent: func() *Agent {
				a := newTestAgentMinimal(nil, nil)
				a.config.RequireApproval = false
				return a
			}(),
			cmd:               &security.Command{Program: "ls"},
			wantNeedsApproval: false,
			wantReason:        "",
		},
		{
			name: "safe command",
			agent: func() *Agent {
				a := newTestAgentMinimal(nil, nil)
				a.config.RequireApproval = true
				return a
			}(),
			cmd:               &security.Command{Program: "ls"},
			wantNeedsApproval: false,
			wantReason:        "",
		},
		{
			name: "interactive command",
			agent: func() *Agent {
				a := newTestAgentMinimal(nil, nil)
				a.config.RequireApproval = true
				return a
			}(),
			cmd:               &security.Command{Program: "mkdir", Args: []string{"testdir"}},
			wantNeedsApproval: true,
			wantReason:        "This command may modify files or system state",
		},
		{
			name: "dangerous command",
			agent: func() *Agent {
				a := newTestAgentMinimal(nil, nil)
				a.config.RequireApproval = true
				return a
			}(),
			cmd:               &security.Command{Program: "rm", Args: []string{"-rf", "testdir"}},
			wantNeedsApproval: true,
			wantReason:        "WARNING: Dangerous operation - Recursive force delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsApproval, reason := tt.agent.ShouldApprove(tt.cmd)

			if needsApproval != tt.wantNeedsApproval {
				t.Errorf("ShouldApprove() needsApproval = %v, want %v", needsApproval, tt.wantNeedsApproval)
			}
			if reason != tt.wantReason {
				t.Errorf("ShouldApprove() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestAgent_SelectIntervention(t *testing.T) {
	tests := []struct {
		name      string
		agent     *Agent
		cycleType cycle.CycleType
		turnCount int
		wantType  string
	}{
		{
			name:      "early cycle - reflection",
			agent:     newTestAgentMinimal(nil, nil),
			cycleType: cycle.CycleRepeatedTool,
			turnCount: 5,
			wantType:  "Reflection",
		},
		{
			name:      "mid cycle - reflection",
			agent:     newTestAgentMinimal(nil, nil),
			cycleType: cycle.CycleRepeatedTool,
			turnCount: 20,
			wantType:  "Reflection",
		},
		{
			name:      "late cycle - escalation",
			agent:     newTestAgentMinimal(nil, nil),
			cycleType: cycle.CycleRepeatedTool,
			turnCount: 35,
			wantType:  "User Escalation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intervention := tt.agent.selectIntervention(tt.cycleType, tt.turnCount)

			if intervention == nil {
				t.Error("selectIntervention() returned nil intervention")
				return
			}

			// Check intervention type by checking the name
			interventionType := intervention.Name()
			if interventionType != tt.wantType {
				t.Errorf("selectIntervention() intervention type = %q, want %q", interventionType, tt.wantType)
			}
		})
	}
}

func TestExtractToolNames(t *testing.T) {
	tests := []struct {
		name      string
		toolCalls []llm.ToolCall
		wantNames []string
	}{
		{
			name:      "empty tool calls",
			toolCalls: []llm.ToolCall{},
			wantNames: []string{},
		},
		{
			name: "single tool call",
			toolCalls: []llm.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: llm.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path": "test.txt"}`,
					},
				},
			},
			wantNames: []string{"read_file"},
		},
		{
			name: "multiple tool calls",
			toolCalls: []llm.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: llm.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path": "test.txt"}`,
					},
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: llm.FunctionCall{
						Name:      "write_file",
						Arguments: `{"path": "output.txt", "content": "hello"}`,
					},
				},
				{
					ID:   "call_3",
					Type: "function",
					Function: llm.FunctionCall{
						Name:      "execute_command",
						Arguments: `{"command": "ls"}`,
					},
				},
			},
			wantNames: []string{"read_file", "write_file", "execute_command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := extractToolNames(tt.toolCalls)

			if len(names) != len(tt.wantNames) {
				t.Errorf("extractToolNames() got %d names, want %d", len(names), len(tt.wantNames))
				return
			}

			for i, name := range names {
				if name != tt.wantNames[i] {
					t.Errorf("extractToolNames()[%d] = %q, want %q", i, name, tt.wantNames[i])
				}
			}
		})
	}
}
