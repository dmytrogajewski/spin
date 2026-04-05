package harness

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/message"
)

// Standard finish reasons for loop termination.
const (
	FinishReasonStop    = "stop"
	FinishReasonLength  = "length"
	FinishReasonTimeout = "timeout"
	FinishReasonMaxTurn = "max_turns"
	FinishReasonGuard   = "guard_halt"
)

// Response is the result of a harness execution.
type Response struct {
	// Output is the agent's final text response.
	Output string

	// FinishReason indicates why the loop terminated.
	FinishReason string

	// Messages contains all messages produced during execution.
	Messages []message.Message

	// ToolCalls records completed tool invocations.
	ToolCalls []message.ToolCall

	// Duration of the execution.
	Duration time.Duration
}
