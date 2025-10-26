// Package testutil provides test helpers and utilities for Spin tests.
//
// This package contains common test fixtures, mocks, and helper functions
// to reduce duplication across test files.
package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// TestTimeout is the default timeout for tests
const TestTimeout = 5 * time.Second

// AgentBuilder helps build Agent instances for testing with sensible defaults.
type AgentBuilder struct {
	t             *testing.T
	provider      llm.Provider
	security      *security.SecurityService
	detection     *detection.DetectionService
	orchestration *orchestration.OrchestrationService
	environment   *agent.Environment
	emitter       *events.EventEmitter
	maxTurns      int
	timeout       time.Duration
}

// NewAgentBuilder creates a new AgentBuilder with default test configuration.
func NewAgentBuilder(t *testing.T) *AgentBuilder {
	t.Helper()

	return &AgentBuilder{
		t:             t,
		provider:      NewMockLLMProvider("test-provider"),
		security:      NewSecurityService(t),
		detection:     NewDetectionService(t, false),
		orchestration: NewOrchestrationService(t),
		environment:   NewEnvironment(t, "/tmp/test"),
		emitter:       NewEventEmitter(t),
		maxTurns:      10,
		timeout:       TestTimeout,
	}
}

// WithProvider sets a custom LLM provider.
func (b *AgentBuilder) WithProvider(p llm.Provider) *AgentBuilder {
	b.provider = p
	return b
}

// WithSecurity sets a custom security service.
func (b *AgentBuilder) WithSecurity(s *security.SecurityService) *AgentBuilder {
	b.security = s
	return b
}

// WithDetection sets a custom detection service.
func (b *AgentBuilder) WithDetection(d *detection.DetectionService) *AgentBuilder {
	b.detection = d
	return b
}

// WithOrchestration sets a custom orchestration service.
func (b *AgentBuilder) WithOrchestration(o *orchestration.OrchestrationService) *AgentBuilder {
	b.orchestration = o
	return b
}

// WithEnvironment sets a custom environment.
func (b *AgentBuilder) WithEnvironment(e *agent.Environment) *AgentBuilder {
	b.environment = e
	return b
}

// WithEmitter sets a custom event emitter.
func (b *AgentBuilder) WithEmitter(em *events.EventEmitter) *AgentBuilder {
	b.emitter = em
	return b
}

// WithMaxTurns sets the maximum number of turns.
func (b *AgentBuilder) WithMaxTurns(max int) *AgentBuilder {
	b.maxTurns = max
	return b
}

// WithTimeout sets the execution timeout.
func (b *AgentBuilder) WithTimeout(timeout time.Duration) *AgentBuilder {
	b.timeout = timeout
	return b
}

// Build creates the Agent instance.
func (b *AgentBuilder) Build() *agent.Agent {
	b.t.Helper()

	a, err := agent.NewAgent(
		b.provider,
		b.security,
		b.detection,
		b.orchestration,
		b.environment,
		b.emitter,
		agent.WithMaxTurns(b.maxTurns),
	)
	if err != nil {
		b.t.Fatalf("Failed to create agent: %v", err)
	}

	return a
}

// NewMockLLMProvider creates a mock LLM provider for testing.
func NewMockLLMProvider(name string) llm.Provider {
	return llm.NewMockProvider(name)
}

// NewEventEmitter creates an event emitter for testing.
func NewEventEmitter(t *testing.T) *events.EventEmitter {
	t.Helper()
	return events.NewEventEmitter(100)
}

// NewSecurityService creates a security service for testing.
func NewSecurityService(t *testing.T) *security.SecurityService {
	t.Helper()

	validator := security.NewValidator()
	emitter := NewEventEmitter(t)
	approvalService := security.NewApprovalService(nil, emitter, validator)
	return security.NewSecurityService(validator, approvalService)
}

// NewDetectionService creates a detection service for testing.
func NewDetectionService(t *testing.T, cycleDetectionEnabled bool) *detection.DetectionService {
	t.Helper()

	detector := cycle.NewDetector(cycle.Config{
		Enabled:          cycleDetectionEnabled,
		WindowSize:       10,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 3,
	})

	return detection.NewDetectionService(detector, nil)
}

// NewOrchestrationService creates an orchestration service for testing.
func NewOrchestrationService(t *testing.T) *orchestration.OrchestrationService {
	t.Helper()

	toolRegistry := tools.NewRegistry()
	taskRegistry := orchestration.NewRegistry()

	return orchestration.NewOrchestrationService(nil, toolRegistry, taskRegistry)
}

// NewEnvironment creates an environment for testing.
func NewEnvironment(t *testing.T, workDir string) *agent.Environment {
	t.Helper()

	return &agent.Environment{
		WorkDir: workDir,
		OS: agent.OSInfo{
			OS:   "linux",
			Arch: "amd64",
		},
	}
}

// ContextWithTimeout creates a test context with timeout.
func ContextWithTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), TestTimeout)
}

// RequireNoError is a test helper that fails the test if err is not nil.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("%v: %v", msgAndArgs[0], err)
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// RequireError is a test helper that fails the test if err is nil.
func RequireError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("%v: expected error but got nil", msgAndArgs[0])
		} else {
			t.Fatal("expected error but got nil")
		}
	}
}

// AssertEqual is a test helper that fails the test if got != want.
func AssertEqual(t *testing.T, want, got interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if want != got {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v: want %v, got %v", msgAndArgs[0], want, got)
		} else {
			t.Errorf("want %v, got %v", want, got)
		}
	}
}

// AssertNotNil is a test helper that fails the test if value is nil.
func AssertNotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if value == nil {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v: expected non-nil value", msgAndArgs[0])
		} else {
			t.Error("expected non-nil value")
		}
	}
}

// AssertContains is a test helper that fails if substring is not in str.
func AssertContains(t *testing.T, str, substring string, msgAndArgs ...interface{}) {
	t.Helper()
	if len(str) == 0 || len(substring) == 0 {
		t.Error("both string and substring must be non-empty")
		return
	}

	// Simple contains check
	found := false
	for i := 0; i <= len(str)-len(substring); i++ {
		if str[i:i+len(substring)] == substring {
			found = true
			break
		}
	}

	if !found {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v: '%s' does not contain '%s'", msgAndArgs[0], str, substring)
		} else {
			t.Errorf("'%s' does not contain '%s'", str, substring)
		}
	}
}

// TableTest represents a single test case in a table-driven test.
type TableTest struct {
	Name    string
	Setup   func(t *testing.T) interface{}
	Execute func(t *testing.T, fixture interface{}) (interface{}, error)
	Assert  func(t *testing.T, result interface{}, err error)
}

// RunTableTests executes a slice of table tests.
func RunTableTests(t *testing.T, tests []TableTest) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var fixture interface{}
			if tt.Setup != nil {
				fixture = tt.Setup(t)
			}

			result, err := tt.Execute(t, fixture)

			if tt.Assert != nil {
				tt.Assert(t, result, err)
			}
		})
	}
}
