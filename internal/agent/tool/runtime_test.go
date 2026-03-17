package tool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func newTestRuntime(registry *tools.Registry) *Runtime {
	validator := safety.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := safety.NewApprovalServiceWithConfig(safety.ApprovalServiceConfig{
		Handler: nil, Emitter: emitter, Validator: validator,
	})

	if registry == nil {
		registry = tools.NewRegistry()
	}

	return NewRuntime(RuntimeConfig{
		Registry:        registry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})
}

func TestRuntime_parseToolArguments(t *testing.T) {
	t.Parallel()

	runtime := newTestRuntime(tools.NewRegistry())

	tests := []struct {
		name    string
		call    *message.ToolCall
		wantErr bool
	}{
		{
			name: "valid JSON arguments",
			call: &message.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "test_tool",
					Arguments: `{"key": "value"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "empty arguments (strict parser rejects)",
			call: &message.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "test_tool",
					Arguments: "", // Empty arguments.
				},
			},
			wantErr: true,
		},
		{
			name: "empty JSON object",
			call: &message.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "test_tool",
					Arguments: `{}`, // Empty JSON object (allowed).
				},
			},
			wantErr: false,
		},
		{
			name: "invalid JSON arguments",
			call: &message.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "test_tool",
					Arguments: `{"key": "value"`, // Missing closing brace.
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args, err := runtime.parseToolArguments(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("Runtime.parseToolArguments() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				assert.NotNil(t, args)
			} else {
				assert.Error(t, err)

				if tt.name == "empty arguments (strict parser rejects)" {
					assert.Contains(t, err.Error(), "cannot be empty")
				}
			}
		})
	}
}

func TestRuntime_Execute_EmptyArguments(t *testing.T) {
	t.Parallel()

	runtime := newTestRuntime(tools.NewRegistry())

	call := &message.ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "test_tool",
			Arguments: "", // Empty arguments should be rejected.
		},
	}

	ctx := context.Background()
	result, err := runtime.Execute(ctx, call)
	require.NoError(t, err) // Execute returns nil error, error is in result.
	require.NotNil(t, result)
	assert.False(t, result.Success)
	require.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "cannot be empty")
}

func TestRuntime_Execute_ValidArguments(t *testing.T) {
	t.Parallel()

	toolRegistry := tools.NewRegistry()
	testTool := tools.NewReadFileTool()
	_ = toolRegistry.Register(testTool)

	runtime := newTestRuntime(toolRegistry)

	call := &message.ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"path": "/tmp/test.txt"}`,
		},
	}

	ctx := context.Background()
	result, err := runtime.Execute(ctx, call)
	require.NoError(t, err) // Execute returns nil error, error is in result.
	require.NotNil(t, result)
	// Note: Tool may fail (file doesn't exist), but parsing should succeed
	// We're just verifying that strict parsing doesn't reject valid JSON.
}
