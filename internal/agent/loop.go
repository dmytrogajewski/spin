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

var ErrAgentCallllmContextCanceledContextDeadline = errors.New("Agent.callLLM: context canceled: context deadline exceeded")

// estimateTokenCount provides a rough token count estimate for messages.
// Uses simple heuristic: ~4 characters per token, plus overhead for structure.
func estimateTokenCount(messages []message.Message) int {
	total := 0
	for _, msg := range messages {
		// Content tokens (roughly 1 token per 4 characters).
		total += len(msg.Content) / 4

		// Tool call tokens.
		for _, tc := range msg.ToolCalls {
			total += len(tc.Function.Name) / 4
			total += len(tc.Function.Arguments) / 4
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
func (a *Agent) executeAgentLoop(ctx context.Context, messages []message.Message, t task.Task, resp *Response, trajCtx *trajectory.Context) ([]message.Message, *Response, error) {
	maxTurns := a.maxTurns

	// Initialize retrieved bullets slice to accumulate across turns.
	allRetrievedBullets := make([]*bullet.Bullet, 0)

	var lastErr error

	for turn := range maxTurns {
		// Check context cancellation.
		if ctxErr := ctx.Err(); ctxErr != nil {
			resp.FinishReason = "timeout"

			return messages, resp, fmt.Errorf("agent loop context canceled: %w", ctxErr)
		}

		a.emitTurnStart(turn + 1)

		// Dynamic tool selection on each turn based on current context.
		if a.toolSelector != nil && trajCtx != nil {
			query := trajCtx.Query // Use trajectory context's query.
			_, err := a.toolSelector.SelectToolsForTurn(ctx, query, turn)
			if err != nil {
				a.logger.WarnContext(ctx, "dynamic tool selection failed", "turn", turn, "error", err)
			}
		}

		// Update trajectory context.
		if trajCtx != nil {
			trajCtx.CurrentTurn = turn

			// Extract new steps from messages since last extraction.
			newSteps := extractNewSteps(messages, len(trajCtx.Steps))
			trajCtx.AppendSteps(newSteps)
		}

		// ACE: Progressive retrieval with caching.
		var currentTurnBullets []*bullet.Bullet

		if a.aceService != nil && trajCtx != nil && a.aceConfig != nil && a.aceConfig.Retrieval.ProgressiveContext.Enabled {
			// Progressive retrieval path.
			shouldRetrieve, trigger := a.shouldRetrieveProgressive(trajCtx)

			if shouldRetrieve {
				// Build dynamic query based on context and trigger.
				query := a.buildQueryFromContext(trajCtx, trigger)
				a.logger.DebugContext(ctx, "Progressive retrieval triggered",
					"trigger", trigger,
					"query", query,
					"turn", turn+1)

				// Retrieve bullets.
				retrievedBullets, err := a.aceService.Retrieve(ctx, query)
				if err != nil {
					a.logger.WarnContext(ctx, "ACE retrieval failed", "error", err, "turn", turn+1)
				} else {
					// Record retrieval event.
					event := trajectory.RetrievalEvent{
						Turn:         turn,
						Trigger:      trigger,
						Query:        query,
						BulletsAdded: extractBulletIDs(retrievedBullets),
						Timestamp:    time.Now(),
					}
					trajCtx.RecordRetrieval(event, retrievedBullets)

					a.logger.InfoContext(ctx, "Retrieved bullets",
						"count", len(retrievedBullets),
						"trigger", trigger,
						"cached", len(trajCtx.BulletCache),
						"hits", trajCtx.CacheHits,
						"misses", trajCtx.CacheMisses)

					// Emit ACE retrieval event for TUI.
					if a.aceConfig.Retrieval.ProgressiveContext.EmitACEEvents {
						a.emitACERetrievalEvent(trajCtx, trigger, query, retrievedBullets, turn)
					}
				}
			}

			// Get active bullets for this turn (TTL-filtered, from cache).
			currentTurnBullets = trajCtx.GetActiveBullets()
		}

		// Call LLM with timeout protection and retry on transient errors or empty responses.
		const maxRetries = 3

		var (
			llmResp      *openai.ChatCompletion
			content      string
			toolCalls    []ToolCall
			finishReason string
		)

		for retry := 0; retry <= maxRetries; retry++ {
			var err error

			llmResp, err = a.callLLMWithTimeout(ctx, messages, t, currentTurnBullets)
			if err != nil {
				lastErr = err
				// Check if context was canceled (non-retryable).
				if ctx.Err() != nil {
					a.logger.ErrorContext(ctx, "LLM call failed (context canceled)", "turn", turn+1, "error", err)
					resp.Error = fmt.Errorf("llm call failed: %w", err)
					resp.FinishReason = "error"

					return messages, resp, err
				}
				// Transient error (e.g. HTTP 500, connection error) - retry.
				if retry < maxRetries {
					backoff := time.Duration(1<<uint(retry)) * time.Second // 1s, 2s, 4s.
					a.logger.WarnContext(ctx, "LLM call failed, retrying",
						"turn", turn+1, "retry", retry+1, "max_retries", maxRetries,
						"backoff", backoff, "error", err)

					select {
					case <-ctx.Done():
						resp.FinishReason = "timeout"

						return messages, resp, fmt.Errorf("llm retry context canceled: %w", ctx.Err())
					case <-time.After(backoff):
					}

					continue
				}

				a.logger.ErrorContext(ctx, "LLM call failed after retries", "turn", turn+1, "retries", maxRetries, "error", err)
				resp.Error = fmt.Errorf("llm call failed: %w", err)
				resp.FinishReason = "error"

				return messages, resp, err
			}

			lastErr = nil

			// Extract response data using helper functions.
			content = getContent(llmResp)
			toolCalls = getToolCalls(llmResp)
			finishReason = getFinishReason(llmResp)

			// Check for empty response.
			if llmResp != nil && (content != "" || len(toolCalls) > 0) {
				// Got a valid response, break out of retry loop.
				if retry > 0 {
					a.logger.InfoContext(ctx, "LLM retry succeeded", "turn", turn+1, "retry", retry)
				}

				break
			}

			// Empty response - retry if we have attempts left.
			if retry < maxRetries {
				backoff := time.Duration(1<<uint(retry)) * time.Second // 1s, 2s, 4s.
				a.logger.WarnContext(ctx, "Received empty response from LLM, retrying",
					"turn", turn+1, "retry", retry+1, "max_retries", maxRetries,
					"backoff", backoff,
					"llm_resp_nil", llmResp == nil, "content_len", len(content), "tool_calls", len(toolCalls))

				select {
				case <-ctx.Done():
					resp.FinishReason = "timeout"

					return messages, resp, fmt.Errorf("empty response retry context canceled: %w", ctx.Err())
				case <-time.After(backoff):
				}

				continue
			}

			// All retries exhausted - break the agent loop.
			a.logger.WarnContext(ctx, "Received empty response from LLM after retries, breaking loop",
				"turn", turn+1, "retries_exhausted", maxRetries,
				"llm_resp_nil", llmResp == nil, "content_len", len(content), "tool_calls", len(toolCalls))

			resp.FinishReason = "empty_response"

			// Emit warning event so UI can show the error state.
			a.emitter.Emit(events.Event{
				Type:      events.EventWarning,
				Timestamp: time.Now(),
				Data: events.SystemEventData{
					Level:   "warning",
					Message: "LLM returned empty response after retries",
					Details: fmt.Sprintf("turn=%d, retries=%d", turn+1, maxRetries),
				},
			})

			break
		}

		// If we exhausted retries and still have empty/error response, break outer loop.
		if lastErr != nil || llmResp == nil || (content == "" && len(toolCalls) == 0) {
			break
		}

		a.logger.DebugContext(ctx, "LLM response received", "turn", turn+1, "content_len", len(content), "tool_calls", len(toolCalls), "finish_reason", finishReason)

		// Handle cycle detection via detection service.
		if a.cycleDetection {
			var (
				shouldStop bool
				err        error
			)

			messages, shouldStop, err = a.handleCycleDetection(ctx, messages, llmResp, turn+1, resp)
			if err != nil {
				a.logger.ErrorContext(ctx, "cycle detection failed", "turn", turn+1, "error", err)

				return messages, resp, err
			}

			if shouldStop {
				a.logger.InfoContext(ctx, "cycle detected, stopping agent", "turn", turn+1)

				return messages, resp, nil
			}
		}

		// Process tool calls or finish.
		if len(toolCalls) > 0 {
			a.logger.DebugContext(ctx, "processing tool calls", "count", len(toolCalls), "turn", turn+1)

			messages = a.processToolCallsFromCompletion(ctx, messages, llmResp, resp)

			// Emit estimated token count for the UI to show progress
			// This helps display accurate context percentage during execution.
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

			continue
		}

		// Handle truncated response: if finish_reason=length, the model was cut off
		// by token/context limits. Add partial content as assistant message and continue
		// so the model can finish its response on the next turn.
		if finishReason == string(openai.ChatCompletionChoicesFinishReasonLength) {
			a.logger.WarnContext(ctx, "LLM response truncated (finish_reason=length), continuing",
				"turn", turn+1, "content_len", len(content))

			if content != "" {
				messages = append(messages, message.Message{
					Role:      message.RoleAssistant,
					Content:   content,
					Timestamp: time.Now(),
				})
			}
			// Add a user message to prompt continuation.
			messages = append(messages, message.Message{
				Role:      message.RoleUser,
				Content:   "Your previous response was truncated. Please continue where you left off.",
				Timestamp: time.Now(),
			})

			continue
		}

		messages = a.addFinalMessage(messages, content)

		resp.FinishReason = finishReason
		if resp.FinishReason == "" {
			resp.FinishReason = "stop"
		}

		break
	}

	// Store accumulated retrieved bullets in response for trajectory building.
	resp.RetrievedBullets = allRetrievedBullets

	return messages, resp, lastErr
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
func (a *Agent) handleCycleDetection(ctx context.Context, messages []message.Message, llmResp *openai.ChatCompletion, turn int, resp *Response) ([]message.Message, bool, error) {
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

	// Reconstruct messages preserving original data (ToolCalls, ToolCallID, Metadata).
	// Interventions typically only append new messages, so we keep originals intact
	// and only convert genuinely new messages added by the intervention.
	if len(modifiedDetectionMessages) >= len(messages) {
		// Intervention appended message(s) — keep originals, convert only new ones.
		newMessages := make([]message.Message, len(messages), len(modifiedDetectionMessages))
		copy(newMessages, messages)

		for i := len(messages); i < len(modifiedDetectionMessages); i++ {
			dm := modifiedDetectionMessages[i]
			newMessages = append(newMessages, message.Message{
				Role:      message.Role(dm.GetRole()),
				Content:   dm.GetContent(),
				Timestamp: dm.GetTimestamp(),
			})
		}

		messages = newMessages
	} else {
		// Intervention removed messages — fall back to full conversion but
		// attempt to match originals by index to preserve ToolCalls/ToolCallID.
		newMessages := make([]message.Message, len(modifiedDetectionMessages))
		for i, dm := range modifiedDetectionMessages {
			if i < len(messages) && messages[i].Role == message.Role(dm.GetRole()) && messages[i].Content == dm.GetContent() {
				newMessages[i] = messages[i] // Preserve original with full data.
			} else {
				newMessages[i] = message.Message{
					Role:      message.Role(dm.GetRole()),
					Content:   dm.GetContent(),
					Timestamp: dm.GetTimestamp(),
				}
			}
		}

		messages = newMessages
	}

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
	if intervention.Severity() >= 3 {
		resp.FinishReason = "cycle_intervention"

		return messages, true, nil
	}

	return messages, false, nil
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
func (a *Agent) callLLMWithTimeout(ctx context.Context, messages []message.Message, t task.Task, bullets []*bullet.Bullet) (*openai.ChatCompletion, error) {
	// Use a reasonable timeout for LLM calls (5 minutes)
	// Don't use agent timeout which may be very long for multi-step tasks.
	llmTimeout := 5 * time.Minute

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

	a.logger.DebugContext(ctx, "calling LLM with timeout", "timeout", llmTimeout, "message_count", len(messages), "bullets_count", len(bullets))

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
	case turnCount < 10:
		// Early cycles: Use soft intervention (reflection).
		return &detection.ReflectionIntervention{}

	case turnCount < 30:
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
