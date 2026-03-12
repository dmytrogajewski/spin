package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/task"
)

const (
	tokensPerChar             = 4 // approximate chars per token.
	escalateSeverityThreshold = 3
	llmDefaultTimeout         = 5 * time.Minute
	shortConversationTurns    = 10
	mediumConversationTurns   = 30
)

// ErrAgentCallllmContextCanceledContextDeadline is a sentinel error.
var ErrAgentCallllmContextCanceledContextDeadline = errors.New("Agent.callLLM: context canceled: context deadline exceeded")

// estimateTokenCount provides a rough token count estimate for messages.
// Uses simple heuristic: ~4 characters per token, plus overhead for structure.
func estimateTokenCount(messages []message.Message) int {
	total := 0
	for _, msg := range messages {
		// Content tokens (roughly 1 token per 4 characters).
		total += len(msg.Content) / tokensPerChar

		// Tool call tokens.
		for _, tc := range msg.ToolCalls {
			total += len(tc.Function.Name) / tokensPerChar
			total += len(tc.Function.Arguments) / tokensPerChar
			total += 8 // overhead per tool call.
		}

		// Message overhead.
		total += 4
	}

	return total
}

// executeAgentLoop runs the main agent execution loop.
//
// This method orchestrates the agent's turn-based interaction with the LLM:
// 1. Calls the LLM to get a response
// 2. Checks for cycle detection if enabled
// 3. Processes any tool calls in the response
// 4. Continues until completion or max turns reached
//
// The loop respects context cancellation and enforces the MaxTurns limit
// from the agent configuration.
//
// If trajCtx is provided and progressive context is enabled, uses progressive
// retrieval with caching. Otherwise falls back to simple retrieval.
func (a *Agent) executeAgentLoop(
	ctx context.Context, messages []message.Message,
	t task.Task, resp *Response, trajCtx *trajectory.Context,
) ([]message.Message, *Response, error) {
	maxTurns := a.maxTurns
	allRetrievedBullets := make([]*bullet.Bullet, 0)

	var lastErr error

	for turn := range maxTurns {
		if ctxErr := ctx.Err(); ctxErr != nil {
			resp.FinishReason = "timeout"

			return messages, resp, fmt.Errorf("agent loop context canceled: %w", ctxErr)
		}

		a.emitTurnStart(turn + 1)
		a.performDynamicToolSelection(ctx, trajCtx, turn)
		a.updateTrajectoryContext(trajCtx, messages, turn)

		currentTurnBullets := a.performProgressiveRetrieval(ctx, trajCtx, turn)

		llmResp, content, toolCalls, finishReason, err := a.callLLMWithRetries(ctx, messages, t, currentTurnBullets, turn, resp)
		if err != nil {
			lastErr = err

			return messages, resp, err
		}

		// If we exhausted retries and still have empty/error response, break.
		if llmResp == nil || (content == "" && len(toolCalls) == 0) {
			break
		}

		lastErr = nil

		a.logger.DebugContext(ctx, "LLM response received",
			"turn", turn+1, "content_len", len(content),
			"tool_calls", len(toolCalls), "finish_reason", finishReason)

		messages, shouldStop, err := a.runCycleDetectionIfEnabled(ctx, messages, llmResp, turn+1, resp)
		if err != nil {
			return messages, resp, err
		}

		if shouldStop {
			return messages, resp, nil
		}

		shouldContinue, msgs := a.processLLMResponse(ctx, messages, llmResp, content, toolCalls, finishReason, turn, resp)
		messages = msgs

		if !shouldContinue {
			break
		}
	}

	resp.RetrievedBullets = allRetrievedBullets

	return messages, resp, lastErr
}

// performDynamicToolSelection runs dynamic tool selection for a turn.
func (a *Agent) performDynamicToolSelection(ctx context.Context, trajCtx *trajectory.Context, turn int) {
	if a.toolSelector == nil || trajCtx == nil {
		return
	}

	_, err := a.toolSelector.SelectToolsForTurn(ctx, trajCtx.Query, turn)
	if err != nil {
		a.logger.WarnContext(ctx, "dynamic tool selection failed", "turn", turn, "error", err)
	}
}

// updateTrajectoryContext updates the trajectory with new steps from messages.
func (a *Agent) updateTrajectoryContext(trajCtx *trajectory.Context, messages []message.Message, turn int) {
	if trajCtx == nil {
		return
	}

	trajCtx.CurrentTurn = turn
	newSteps := extractNewSteps(messages, len(trajCtx.Steps))
	trajCtx.AppendSteps(newSteps)
}

// performProgressiveRetrieval retrieves ACE bullets using progressive context.
func (a *Agent) performProgressiveRetrieval(ctx context.Context, trajCtx *trajectory.Context, turn int) []*bullet.Bullet {
	if !a.isProgressiveRetrievalEnabled(trajCtx) {
		return nil
	}

	shouldRetrieve, trigger := a.shouldRetrieveProgressive(trajCtx)
	if shouldRetrieve {
		a.retrieveAndRecordBullets(ctx, trajCtx, trigger, turn)
	}

	return trajCtx.GetActiveBullets()
}

// isProgressiveRetrievalEnabled checks if progressive retrieval is configured and enabled.
func (a *Agent) isProgressiveRetrievalEnabled(trajCtx *trajectory.Context) bool {
	return a.aceService != nil && trajCtx != nil && a.aceConfig != nil && a.aceConfig.Retrieval.ProgressiveContext.Enabled
}

// retrieveAndRecordBullets retrieves bullets and records the retrieval event.
func (a *Agent) retrieveAndRecordBullets(ctx context.Context, trajCtx *trajectory.Context, trigger trajectory.TriggerType, turn int) {
	query := a.buildQueryFromContext(trajCtx, trigger)
	a.logger.DebugContext(ctx, "Progressive retrieval triggered", "trigger", trigger, "query", query, "turn", turn+1)

	retrievedBullets, err := a.aceService.Retrieve(ctx, query)
	if err != nil {
		a.logger.WarnContext(ctx, "ACE retrieval failed", "error", err, "turn", turn+1)

		return
	}

	event := trajectory.RetrievalEvent{
		Turn:         turn,
		Trigger:      trigger,
		Query:        query,
		BulletsAdded: extractBulletIDs(retrievedBullets),
		Timestamp:    time.Now(),
	}
	trajCtx.RecordRetrieval(event, retrievedBullets)

	a.logger.InfoContext(ctx, "Retrieved bullets",
		"count", len(retrievedBullets), "trigger", trigger,
		"cached", len(trajCtx.BulletCache), "hits", trajCtx.CacheHits, "misses", trajCtx.CacheMisses)

	if a.aceConfig.Retrieval.ProgressiveContext.EmitACEEvents {
		a.emitACERetrievalEvent(trajCtx, trigger, query, retrievedBullets, turn)
	}
}

// callLLMWithRetries calls the LLM with retry logic for transient errors.
func (a *Agent) callLLMWithRetries(
	ctx context.Context, messages []message.Message,
	t task.Task, bullets []*bullet.Bullet, turn int, resp *Response,
) (completion *openai.ChatCompletion, content string, toolCalls []ToolCall, finishReason string, err error) {
	const maxRetries = 3

	for retry := 0; retry <= maxRetries; retry++ {
		llmResp, err := a.callLLMWithTimeout(ctx, messages, t, bullets)
		if err != nil {
			shouldReturn, retErr := a.handleLLMError(ctx, err, retry, maxRetries, turn, resp)
			if shouldReturn {
				return nil, "", nil, "", retErr
			}

			continue
		}

		content := getContent(llmResp)
		toolCalls := getToolCalls(llmResp)
		finishReason := getFinishReason(llmResp)

		if llmResp != nil && (content != "" || len(toolCalls) > 0) {
			a.logRetrySuccess(ctx, retry, turn)

			return llmResp, content, toolCalls, finishReason, nil
		}

		if done, retErr := a.handleEmptyResponse(ctx, retry, maxRetries, turn, resp, llmResp, content, toolCalls); done {
			return nil, "", nil, "", retErr
		}
	}

	return nil, "", nil, "", nil
}

// logRetrySuccess logs if a retry was needed and succeeded.
func (a *Agent) logRetrySuccess(ctx context.Context, retry, turn int) {
	if retry > 0 {
		a.logger.InfoContext(ctx, "LLM retry succeeded", "turn", turn+1, "retry", retry)
	}
}

// handleEmptyResponse handles an empty LLM response during retries.
// Returns (true, err) if retries are done, (false, nil) to continue retrying.
func (a *Agent) handleEmptyResponse(
	ctx context.Context, retry, maxRetries, turn int,
	resp *Response, llmResp *openai.ChatCompletion,
	content string, toolCalls []ToolCall,
) (bool, error) {
	if retry < maxRetries {
		if err := a.waitWithBackoff(ctx, retry, turn, resp, "Received empty response from LLM, retrying"); err != nil {
			return true, err
		}

		return false, nil
	}

	a.emitEmptyResponseWarning(ctx, turn, maxRetries, llmResp, content, toolCalls)

	resp.FinishReason = "empty_response"

	return true, nil
}

// handleLLMError handles an error from the LLM call, deciding whether to retry.
func (a *Agent) handleLLMError(
	ctx context.Context, err error, retry, maxRetries, turn int,
	resp *Response,
) (shouldReturn bool, retErr error) {
	if ctx.Err() != nil {
		a.logger.ErrorContext(ctx, "LLM call failed (context canceled)", "turn", turn+1, "error", err)
		resp.Error = fmt.Errorf("llm call failed: %w", err)
		resp.FinishReason = "error"

		return true, err
	}

	if retry < maxRetries {
		if waitErr := a.waitWithBackoff(ctx, retry, turn, resp, "LLM call failed, retrying"); waitErr != nil {
			return true, waitErr
		}

		return false, nil
	}

	a.logger.ErrorContext(ctx, "LLM call failed after retries", "turn", turn+1, "retries", maxRetries, "error", err)
	resp.Error = fmt.Errorf("llm call failed: %w", err)
	resp.FinishReason = "error"

	return true, err
}

// waitWithBackoff waits with exponential backoff, respecting context cancellation.
func (a *Agent) waitWithBackoff(ctx context.Context, retry, turn int, resp *Response, logMsg string) error {
	retryUint := uint(0)
	if retry > 0 {
		retryUint = uint(retry)
	}

	backoff := time.Duration(1<<retryUint) * time.Second
	a.logger.WarnContext(ctx, logMsg, "turn", turn+1, "retry", retry+1, "backoff", backoff)

	select {
	case <-ctx.Done():
		resp.FinishReason = "timeout"

		return fmt.Errorf("retry context canceled: %w", ctx.Err())
	case <-time.After(backoff):
		return nil
	}
}

// emitEmptyResponseWarning emits a warning event for empty LLM responses after retries.
func (a *Agent) emitEmptyResponseWarning(
	ctx context.Context, turn, maxRetries int,
	llmResp *openai.ChatCompletion, content string, toolCalls []ToolCall,
) {
	a.logger.WarnContext(ctx, "Received empty response from LLM after retries, breaking loop",
		"turn", turn+1, "retries_exhausted", maxRetries,
		"llm_resp_nil", llmResp == nil, "content_len", len(content), "tool_calls", len(toolCalls))

	a.emitter.Emit(events.Event{
		Type:      events.EventWarning,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "warning",
			Message: "LLM returned empty response after retries",
			Details: fmt.Sprintf("turn=%d, retries=%d", turn+1, maxRetries),
		},
	})
}

// processLLMResponse processes a valid LLM response, handling tool calls, truncation, and final messages.
// Returns whether the loop should continue and the updated messages.
func (a *Agent) processLLMResponse(
	ctx context.Context, messages []message.Message,
	llmResp *openai.ChatCompletion, content string,
	toolCalls []ToolCall, finishReason string,
	turn int, resp *Response,
) (bool, []message.Message) {
	if len(toolCalls) > 0 {
		a.logger.DebugContext(ctx, "processing tool calls", "count", len(toolCalls), "turn", turn+1)
		messages = a.processToolCallsFromCompletion(ctx, messages, llmResp, resp)

		estimatedTokens := estimateTokenCount(messages)
		a.emitter.Emit(events.Event{
			Type:      events.EventTurnProgress,
			Timestamp: time.Now(),
			Data: events.TurnEventData{
				Turn:       turn + 1,
				TokensUsed: estimatedTokens,
			},
		})
		a.logger.DebugContext(ctx, "emitted token progress", "turn", turn+1, "estimated_tokens", estimatedTokens)

		return true, messages
	}

	if finishReason == string(openai.ChatCompletionChoicesFinishReasonLength) {
		return a.handleTruncatedResponse(ctx, messages, content, turn)
	}

	messages = a.addFinalMessage(messages, content)

	resp.FinishReason = finishReason
	if resp.FinishReason == "" {
		resp.FinishReason = "stop"
	}

	return false, messages
}

// handleTruncatedResponse handles a truncated LLM response by adding continuation messages.
func (a *Agent) handleTruncatedResponse(
	ctx context.Context, messages []message.Message,
	content string, turn int,
) (bool, []message.Message) {
	a.logger.WarnContext(ctx, "LLM response truncated (finish_reason=length), continuing",
		"turn", turn+1, "content_len", len(content))

	if content != "" {
		messages = append(messages, message.Message{
			Role:      message.RoleAssistant,
			Content:   content,
			Timestamp: time.Now(),
		})
	}

	messages = append(messages, message.Message{
		Role:      message.RoleUser,
		Content:   "Your previous response was truncated. Please continue where you left off.",
		Timestamp: time.Now(),
	})

	return true, messages
}

// runCycleDetectionIfEnabled runs cycle detection if enabled, returning whether to stop.
func (a *Agent) runCycleDetectionIfEnabled(
	ctx context.Context, messages []message.Message,
	llmResp *openai.ChatCompletion, turn int, resp *Response,
) ([]message.Message, bool, error) {
	if !a.cycleDetection {
		return messages, false, nil
	}

	messages, shouldStop, err := a.handleCycleDetection(ctx, messages, llmResp, turn, resp)
	if err != nil {
		a.logger.ErrorContext(ctx, "cycle detection failed", "turn", turn, "error", err)

		return messages, false, err
	}

	if shouldStop {
		a.logger.InfoContext(ctx, "cycle detected, stopping agent", "turn", turn)
	}

	return messages, shouldStop, nil
}

// handleCycleDetection processes cycle detection and interventions via detection service.
//
// This method:
// 1. Records the current turn snapshot
// 2. Checks for cycles using the detection service
// 3. Selects and applies appropriate intervention if a cycle is detected
// 4. Emits cycle detection events
//
// Returns the modified messages (with intervention added if applicable),
// whether to stop the agent loop, and any error.
func (a *Agent) handleCycleDetection(
	ctx context.Context, messages []message.Message,
	llmResp *openai.ChatCompletion, turn int, resp *Response,
) ([]message.Message, bool, error) {
	content := getContent(llmResp)
	toolCalls := getToolCalls(llmResp)

	snapshot := detection.Snapshot{
		Turn:      turn,
		Response:  content,
		ToolCalls: extractToolNamesFromOrchestration(toolCalls),
		Error:     "",
		Timestamp: time.Now(),
	}
	a.detection.RecordSnapshot(snapshot)

	cycleResult, checkErr := a.detection.CheckCycle()
	if checkErr != nil {
		// Cycle detection errors are non-fatal; log and continue agent loop.
		a.logger.WarnContext(ctx, "cycle detection error", "error", checkErr)

		return messages, false, nil
	}

	if cycleResult.Type == detection.CycleNone {
		return messages, false, nil
	}

	intervention := a.selectIntervention(cycleResult.Type, turn)
	if intervention == nil {
		return messages, false, nil
	}

	// Convert messages to detection.Message interface for the intervention.
	detectionMessages := make([]detection.Message, len(messages))
	for i, msg := range messages {
		detectionMessages[i] = &messageAdapter{msg: msg}
	}

	modifiedDetectionMessages, applyErr := intervention.Apply(ctx, detectionMessages)
	if applyErr != nil {
		a.logger.WarnContext(ctx, "cycle intervention failed", "error", applyErr, "cycle_type", cycleResult.Type)

		return messages, false, nil
	}

	messages = reconstructMessagesFromIntervention(messages, modifiedDetectionMessages)

	// Emit cycle detection event.
	a.emitter.Emit(events.Event{
		Type:      events.EventWarning,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "warning",
			Message: fmt.Sprintf("Cycle detected: %s. Applied intervention: %s", cycleResult.Type, intervention.Name()),
			Details: cycleResult.Details,
		},
	})

	// If this was an escalation intervention, pause the agent.
	if intervention.Severity() >= escalateSeverityThreshold {
		resp.FinishReason = "cycle_intervention"

		return messages, true, nil
	}

	return messages, false, nil
}

// reconstructMessagesFromIntervention rebuilds the message list from detection messages,
// preserving original data (ToolCalls, ToolCallID, Metadata) where possible.
func reconstructMessagesFromIntervention(originals []message.Message, modified []detection.Message) []message.Message {
	if len(modified) >= len(originals) {
		return reconstructWithAppendedMessages(originals, modified)
	}

	return reconstructWithRemovedMessages(originals, modified)
}

// reconstructWithAppendedMessages keeps originals intact and converts only new appended messages.
func reconstructWithAppendedMessages(originals []message.Message, modified []detection.Message) []message.Message {
	result := make([]message.Message, len(originals), len(modified))
	copy(result, originals)

	for i := len(originals); i < len(modified); i++ {
		dm := modified[i]
		result = append(result, message.Message{
			Role:      message.Role(dm.GetRole()),
			Content:   dm.GetContent(),
			Timestamp: dm.GetTimestamp(),
		})
	}

	return result
}

// reconstructWithRemovedMessages attempts to match originals by index to preserve full data.
func reconstructWithRemovedMessages(originals []message.Message, modified []detection.Message) []message.Message {
	result := make([]message.Message, len(modified))

	for i, dm := range modified {
		if i < len(originals) && originals[i].Role == message.Role(dm.GetRole()) && originals[i].Content == dm.GetContent() {
			result[i] = originals[i]
		} else {
			result[i] = message.Message{
				Role:      message.Role(dm.GetRole()),
				Content:   dm.GetContent(),
				Timestamp: dm.GetTimestamp(),
			}
		}
	}

	return result
}

// emitTurnStart emits a turn start event.
func (a *Agent) emitTurnStart(turn int) {
	a.emitter.Emit(events.Event{
		Type:      events.EventTurnStart,
		Timestamp: time.Now(),
		Data: events.TurnEventData{
			Turn: turn,
		},
	})
}

// callLLMWithTimeout calls the LLM provider with timeout protection to prevent getting stuck.
func (a *Agent) callLLMWithTimeout(
	ctx context.Context, messages []message.Message,
	t task.Task, bullets []*bullet.Bullet,
) (*openai.ChatCompletion, error) {
	// Use a reasonable timeout for LLM calls (5 minutes)
	// Don't use agent timeout which may be very long for multi-step tasks.
	llmTimeout := llmDefaultTimeout

	// Check if parent context has a deadline that's already expired.
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		// If parent is about to expire, return error immediately.
		if remaining <= 0 {
			return nil, ErrAgentCallllmContextCanceledContextDeadline
		}
		// If parent has less time remaining, use that instead.
		if remaining < llmTimeout {
			llmTimeout = remaining
		}
	}

	a.logger.DebugContext(ctx, "calling LLM with timeout",
		"timeout", llmTimeout, "message_count", len(messages),
		"bullets_count", len(bullets))

	// Create context with LLM timeout, derived from parent context
	// This ensures cancellation propagates from parent context.
	llmCtx, cancel := context.WithTimeout(ctx, llmTimeout)
	defer cancel()

	// Call the actual LLM method.
	resp, err := a.callLLM(llmCtx, messages, t, bullets)
	if err != nil {
		a.logger.ErrorContext(ctx, "LLM call error", "error", err, "timeout", llmTimeout)
	}

	return resp, err
}

// addFinalMessage adds the final assistant message to the messages array.
// This is called when the agent is done (no more tool calls).
func (a *Agent) addFinalMessage(messages []message.Message, content string) []message.Message {
	if content != "" {
		messages = append(messages, message.Message{
			Role:      message.RoleAssistant,
			Content:   content,
			Timestamp: time.Now(),
		})
	}

	return messages
}

// selectIntervention chooses the appropriate intervention based on cycle type and turn count.
//
// The intervention escalation ladder:
// - Early cycles (< 10 turns): Soft intervention (reflection)
// - Mid-stage cycles (< 30 turns): Medium intervention (context summarization)
// - Late-stage/persistent cycles (>= 30 turns): Escalate to user.
func (a *Agent) selectIntervention(_ detection.CycleType, turnCount int) detection.Intervention {
	// Escalation ladder based on turn count.
	switch {
	case turnCount < shortConversationTurns:
		// Early cycles: Use soft intervention (reflection).
		return &detection.ReflectionIntervention{}

	case turnCount < mediumConversationTurns:
		// Mid-stage cycles: Use medium intervention (context summarization)
		// Using reflection intervention (context summarization requires compressor integration).
		return &detection.ReflectionIntervention{}

	default:
		// Late-stage/persistent cycles: Escalate to user.
		return &detection.EscalateIntervention{
			Emitter: &eventEmitterAdapter{emitter: a.emitter},
		}
	}
}

// extractToolNames extracts tool calls with parameters from LLM tool calls for cycle detection.
//
// Returns strings in format "tool_name(arguments_json)" to enable parameter-aware cycle detection.
// This prevents false positives when same tool is called with different params.
// e.g., "list_directory(.)" vs "list_directory(advanced-features-20251012)"
// extractToolNamesFromOrchestration extracts tool names from ToolCall slice.
func extractToolNamesFromOrchestration(toolCalls []ToolCall) []string {
	calls := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		// Include both name and arguments for accurate cycle detection.
		calls[i] = tc.Function.Name + "(" + tc.Function.Arguments + ")"
	}

	return calls
}

// eventEmitterAdapter adapts events.EventEmitter to detection.EventEmitter interface.
type eventEmitterAdapter struct {
	emitter *events.EventEmitter
}

// Emit implements the Emit operation.
func (a *eventEmitterAdapter) Emit(event detection.Event) {
	// Convert detection.Event to events.Event
	// Map event type based on string value.
	var eventType events.EventType

	switch event.GetType() {
	case "turn_paused":
		eventType = events.EventTurnPaused
	default:
		eventType = events.EventWarning // fallback.
	}

	coreEvent := events.Event{
		Type:      eventType,
		Timestamp: event.GetTimestamp(),
		Data:      event.GetData(),
	}
	a.emitter.Emit(coreEvent)
}
