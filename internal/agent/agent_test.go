package agent

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAgent tests the refactored agent creation with services
func TestNewAgent(t *testing.T) {
	tests := []struct {
		name          string
		provider      llm.Provider
		security      *security.SecurityService
		detection     *detection.DetectionService
		orchestration *orchestration.OrchestrationService
		environment   *Environment
		emitter       *events.EventEmitter
		wantErr       bool
		errContains   string
	}{
		{
			name:     "valid agent",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       false,
		},
		{
			name:     "nil provider",
			provider: nil,
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "LLM provider cannot be nil",
		},
		{
			name:          "nil security",
			provider:      llm.NewMockProvider("test"),
			security:      nil,
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "security service cannot be nil",
		},
		{
			name:     "nil detection",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     nil,
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "detection service cannot be nil",
		},
		{
			name:     "nil orchestration",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: nil,
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "orchestration service cannot be nil",
		},
		{
			name:     "nil environment",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   nil,
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "context cannot be nil",
		},
		{
			name:     "nil emitter",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       nil,
			wantErr:       true,
			errContains:   "event emitter cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(tt.provider, tt.security, tt.detection, tt.orchestration, tt.environment, tt.emitter)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, agent)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, agent)
			}
		})
	}
}

// TestAgent_ListTaskModes tests the ListTaskModes method
func TestAgent_ListTaskModes(t *testing.T) {
	agent := createTestAgentWithServices(t)

	modes := agent.ListTaskModes()

	assert.NotNil(t, modes)
	assert.GreaterOrEqual(t, len(modes), 4) // At least 4 built-in modes
	assert.Contains(t, modes, "regular")
	assert.Contains(t, modes, "review")
	assert.Contains(t, modes, "compact")
	assert.Contains(t, modes, "planning")
}

// TestAgent_Execute_Integration is a minimal integration test
func TestAgent_Execute_Integration(t *testing.T) {
	t.Skip("Integration test - requires full setup")

	agent := createTestAgentWithServices(t)

	req := &AgentRequest{
		Input:    "Hello, how are you?",
		TaskName: "regular",
	}

	ctx := context.Background()
	resp, err := agent.Execute(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}
