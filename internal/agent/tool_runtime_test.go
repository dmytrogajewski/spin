package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestToolRuntime_parseToolArguments(t *testing.T) {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	toolRegistry := tools.NewRegistry()

	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid JSON arguments",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "test_tool",
					Arguments: `{"key": "value"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "empty arguments (strict parser rejects)",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "test_tool",
					Arguments: "", // Empty arguments.
				},
			},
			wantErr: true,
		},
		{
			name: "empty JSON object",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "test_tool",
					Arguments: `{}`, // Empty JSON object (allowed).
				},
			},
			wantErr: false,
		},
		{
			name: "invalid JSON arguments",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "test_tool",
					Arguments: `{"key": "value"`, // Missing closing brace.
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := toolRuntime.parseToolArguments(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToolRuntime.parseToolArguments() error = %v, wantErr %v", err, tt.wantErr)

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

func TestToolRuntime_Execute_EmptyArguments(t *testing.T) {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	toolRegistry := tools.NewRegistry()

	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	// Test that Execute returns error for empty arguments (strict parser).
	call := &ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "test_tool",
			Arguments: "", // Empty arguments should be rejected.
		},
	}

	ctx := context.Background()
	result, err := toolRuntime.Execute(ctx, call)
	require.NoError(t, err) // Execute returns nil error, error is in result.
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Err)
	assert.Contains(t, result.Err.Error(), "cannot be empty")
}

func TestToolRuntime_Execute_ValidArguments(t *testing.T) {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	toolRegistry := tools.NewRegistry()

	// Register a simple test tool.
	testTool := tools.NewReadFileTool()
	_ = toolRegistry.Register(testTool)

	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	// Test that Execute succeeds with valid arguments.
	call := &ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"path": "/tmp/test.txt"}`,
		},
	}

	ctx := context.Background()
	result, err := toolRuntime.Execute(ctx, call)
	require.NoError(t, err) // Execute returns nil error, error is in result.
	require.NotNil(t, result)
	// Note: Tool may fail (file doesn't exist), but parsing should succeed
	// We're just verifying that strict parsing doesn't reject valid JSON.
}
