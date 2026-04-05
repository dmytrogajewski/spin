package bridge

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/message"
)

// TurnExecutor adapts harness.Executor to the conversation.HarnessTurnExecutor interface.
type TurnExecutor struct {
	executor *harness.Executor
}

// NewTurnExecutor creates a TurnExecutor wrapping the given harness executor.
func NewTurnExecutor(executor *harness.Executor) *TurnExecutor {
	return &TurnExecutor{executor: executor}
}

// Execute delegates to the harness executor and returns the output and messages.
func (t *TurnExecutor) Execute(
	ctx context.Context,
	query string,
	history []message.Message,
) (string, []message.Message, error) {
	resp, err := t.executor.Execute(ctx, query, history)
	if err != nil {
		return "", nil, err
	}

	return resp.Output, resp.Messages, nil
}
