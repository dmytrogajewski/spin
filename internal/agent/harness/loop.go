package harness

import (
	"context"
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
)

// Execute runs the ReAct loop: call LLM, check guards, dispatch tools, repeat.
// It implements Phases 2-3 of the Extended ReAct algorithm.
func (e *Executor) Execute(
	ctx context.Context, query string, history []message.Message,
) (*Response, error) {
	start := time.Now()

	iterCtx := e.buildIterationContext(query, history)
	resp := &Response{}

	err := e.runLoop(ctx, iterCtx, resp)

	resp.Duration = time.Since(start)
	resp.Messages = iterCtx.Messages

	e.runAfterExecution(ctx, iterCtx, resp)

	return resp, err
}

// buildIterationContext creates the initial iteration context from query and history.
func (e *Executor) buildIterationContext(
	query string, history []message.Message,
) *IterationContext {
	messages := make([]message.Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, message.Message{
		Role:    message.RoleUser,
		Content: query,
	})

	iterCtx := NewIterationContext(messages)
	iterCtx.TrajectoryCtx = trajectory.NewContext(query)

	return iterCtx
}

// runLoop executes the core iteration loop with bounded turns.
func (e *Executor) runLoop(
	ctx context.Context, iterCtx *IterationContext, resp *Response,
) error {
	for turn := range e.maxTurns {
		if err := ctx.Err(); err != nil {
			resp.FinishReason = FinishReasonTimeout

			return fmt.Errorf("harness loop canceled: %w", err)
		}

		iterCtx.Turn = turn

		if iterCtx.TrajectoryCtx != nil {
			iterCtx.TrajectoryCtx.CurrentTurn = turn
		}

		e.runBeforeTurn(ctx, iterCtx)

		if err := e.phaseCompaction(ctx, iterCtx); err != nil {
			return err
		}

		e.phaseReminders(ctx, iterCtx)

		content, toolCalls, finished, err := e.phaseAction(ctx, iterCtx)
		if err != nil {
			return err
		}

		if finished {
			resp.Output = content
			resp.FinishReason = FinishReasonStop

			return nil
		}

		halt, err := e.phaseGuards(ctx, iterCtx, content, toolCalls)
		if err != nil {
			return err
		}

		if halt {
			resp.Output = content
			resp.FinishReason = FinishReasonGuard

			return nil
		}

		// Record message count before dispatch so observation only
		// summarizes tool results that the LLM has already seen.
		preDispatchLen := len(iterCtx.Messages)
		e.phaseDispatch(ctx, iterCtx, content, toolCalls)
		e.phaseObservation(iterCtx, preDispatchLen)

		resp.ToolCalls = append(resp.ToolCalls, toolCalls...)
	}

	resp.FinishReason = FinishReasonMaxTurn

	return nil
}

// phaseAction calls the LLM and determines if the loop should finish.
// Returns (content, toolCalls, finished, error).
func (e *Executor) phaseAction(
	ctx context.Context, iterCtx *IterationContext,
) (string, []message.ToolCall, bool, error) {
	content, toolCalls, _, err := e.caller.Call(
		ctx, iterCtx.Messages, e.currentToolSchemas(), iterCtx.Turn,
	)
	if err != nil {
		return "", nil, false, fmt.Errorf("caller failed at turn %d: %w", iterCtx.Turn, err)
	}

	// Empty response (no content, no tool calls) is not a valid completion —
	// the LLM likely returned nothing after retries were exhausted.
	if content == "" && len(toolCalls) == 0 {
		return "", nil, false, fmt.Errorf("caller returned empty response at turn %d: %w", iterCtx.Turn, ErrEmptyResponse)
	}

	// Implicit completion: text response with no tool calls.
	if len(toolCalls) == 0 {
		return content, nil, true, nil
	}

	return content, toolCalls, false, nil
}

// phaseGuards runs all registered guards and returns whether to halt.
func (e *Executor) phaseGuards(
	ctx context.Context, iterCtx *IterationContext,
	content string, toolCalls []message.ToolCall,
) (bool, error) {
	for _, guard := range e.guards {
		injected, halt, err := guard.Check(ctx, iterCtx, content, toolCalls)
		if err != nil {
			return false, fmt.Errorf("guard check failed: %w", err)
		}

		if len(injected) > 0 {
			iterCtx.Messages = append(iterCtx.Messages, injected...)
		}

		if halt {
			return true, nil
		}
	}

	return false, nil
}

// phaseDispatch dispatches tool calls and appends results to messages.
func (e *Executor) phaseDispatch(
	ctx context.Context, iterCtx *IterationContext,
	content string, toolCalls []message.ToolCall,
) {
	updated := e.dispatcher.Dispatch(ctx, iterCtx.Messages, content, toolCalls)
	iterCtx.Messages = updated
}

// runBeforeTurn calls BeforeTurn on all middlewares.
func (e *Executor) runBeforeTurn(ctx context.Context, iterCtx *IterationContext) {
	for _, mw := range e.middlewares {
		mw.BeforeTurn(ctx, iterCtx)
	}
}

// runAfterExecution calls AfterExecution on all middlewares.
func (e *Executor) runAfterExecution(
	ctx context.Context, iterCtx *IterationContext, resp *Response,
) {
	for _, mw := range e.middlewares {
		mw.AfterExecution(ctx, iterCtx, resp)
	}
}

// phaseCompaction runs context compaction (Phase 0) if a compactor is configured.
func (e *Executor) phaseCompaction(
	ctx context.Context, iterCtx *IterationContext,
) error {
	if e.compactor == nil {
		return nil
	}

	compacted, changed, err := e.compactor.Compact(ctx, iterCtx.Messages)
	if err != nil {
		return fmt.Errorf("compaction failed at turn %d: %w", iterCtx.Turn, err)
	}

	if changed {
		iterCtx.Messages = compacted
		e.logger.InfoContext(ctx, "context compacted", "turn", iterCtx.Turn)

		e.emit(events.Event{
			Type: events.EventCompactionTriggered,
			Data: events.CompactionTriggeredData{
				Turn:  iterCtx.Turn,
				Stage: "compacted",
			},
		})
	}

	return nil
}

// phaseReminders runs reminder injection if an injector is configured.
func (e *Executor) phaseReminders(
	ctx context.Context, iterCtx *IterationContext,
) {
	if e.reminderInjector == nil {
		return
	}

	reminders := e.reminderInjector.InjectReminders(
		ctx, iterCtx.Messages, iterCtx.Turn,
	)

	if len(reminders) > 0 {
		iterCtx.Messages = append(iterCtx.Messages, reminders...)

		e.emit(events.Event{
			Type: events.EventReminderInjected,
			Data: events.ReminderInjectedData{
				Turn:  iterCtx.Turn,
				Count: len(reminders),
			},
		})
	}
}

// phaseObservation applies observation summarization to tool results that
// the LLM has already seen (i.e. messages before preDispatchLen).
// New tool results from the current dispatch are left raw so the LLM
// can consume them on the next turn.
func (e *Executor) phaseObservation(iterCtx *IterationContext, preDispatchLen int) {
	if e.observationSummarizer == nil {
		return
	}

	if preDispatchLen <= 0 || preDispatchLen > len(iterCtx.Messages) {
		return
	}

	// Only summarize the portion the LLM has already seen.
	older := iterCtx.Messages[:preDispatchLen]
	older = e.observationSummarizer.SummarizeToolResults(older)
	iterCtx.Messages = append(older, iterCtx.Messages[preDispatchLen:]...)
}

// emit sends an event through the emitter if one is configured.
func (e *Executor) emit(event events.Event) {
	if e.emitter == nil {
		return
	}

	e.emitter.Emit(event)
}
