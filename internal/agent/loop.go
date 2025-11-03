package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/openai/openai-go"
)

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
func (a *Agent) executeAgentLoop(ctx context.Context, messages []Message, task Task, resp *AgentResponse, trajCtx *trajectory.TrajectoryContext) ([]Message, *AgentResponse, error) {
	maxTurns := a.config.MaxTurns

	// Initialize retrieved bullets slice to accumulate across turns
	allRetrievedBullets := make([]*bullet.Bullet, 0)

	for turn := 0; turn < maxTurns; turn++ {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			resp.FinishReason = "timeout"
			return messages, resp, err
		}

		a.emitTurnStart(turn + 1)

		// Update trajectory context
		if trajCtx != nil {
			trajCtx.CurrentTurn = turn

			// Extract new steps from messages since last extraction
			newSteps := extractNewSteps(messages, len(trajCtx.Steps))
			trajCtx.AppendSteps(newSteps)
		}

		// ACE: Progressive retrieval with caching
		var currentTurnBullets []*bullet.Bullet
		if a.aceService != nil && trajCtx != nil && a.config.ACE.Retrieval.ProgressiveContext.Enabled {
			// Progressive retrieval path
			shouldRetrieve, trigger := a.shouldRetrieveProgressive(trajCtx)

			if shouldRetrieve {
				// Build dynamic query based on context and trigger
				query := a.buildQueryFromContext(trajCtx, trigger)
				slog.Debug("Progressive retrieval triggered",
					"trigger", trigger,
					"query", query,
					"turn", turn+1)

				// Retrieve bullets
				retrievedBullets, err := a.aceService.Retrieve(ctx, query)
				if err != nil {
					slog.Warn("ACE retrieval failed", "error", err, "turn", turn+1)
				} else {
					// Record retrieval event
					event := trajectory.RetrievalEvent{
						Turn:         turn,
						Trigger:      trigger,
						Query:        query,
						BulletsAdded: extractBulletIDs(retrievedBullets),
						Timestamp:    time.Now(),
					}
					trajCtx.RecordRetrieval(event, retrievedBullets)

					slog.Info("Retrieved bullets",
						"count", len(retrievedBullets),
						"trigger", trigger,
						"cached", len(trajCtx.BulletCache),
						"hits", trajCtx.CacheHits,
						"misses", trajCtx.CacheMisses)

					// Emit ACE retrieval event for TUI
					if a.config.ACE.Retrieval.ProgressiveContext.EmitACEEvents {
						a.emitACERetrievalEvent(trajCtx, trigger, query, len(retrievedBullets), turn)
					}
				}
			}

			// Get active bullets for this turn (TTL-filtered, from cache)
			currentTurnBullets = trajCtx.GetActiveBullets()

		} else if a.aceService != nil {
			// Simple retrieval mode (when progressive context is disabled)
			query := extractQueryFromMessages(messages)
			if query != "" {
				retrievedBullets, err := a.aceService.Retrieve(ctx, query)
				if err != nil {
					slog.Warn("ACE retrieval failed", "error", err, "turn", turn+1)
				} else {
					currentTurnBullets = retrievedBullets
					// Accumulate bullets from all turns (deduplicate by ID)
					for _, newBullet := range retrievedBullets {
						alreadyRetrieved := false
						for _, existing := range allRetrievedBullets {
							if existing.ID == newBullet.ID {
								alreadyRetrieved = true
								break
							}
						}
						if !alreadyRetrieved {
							allRetrievedBullets = append(allRetrievedBullets, newBullet)
						}
					}
					slog.Debug("ACE retrieved bullets", "count", len(retrievedBullets), "total", len(allRetrievedBullets), "turn", turn+1)
				}
			}
		}

		// Call LLM with timeout protection, passing retrieved bullets
		llmResp, err := a.callLLMWithTimeout(ctx, messages, task, currentTurnBullets)
		if err != nil {
			slog.Error("LLM call failed", "turn", turn+1, "error", err)
			resp.Error = fmt.Errorf("llm call failed: %w", err)
			resp.FinishReason = "error"
			return messages, resp, err
		}

		// Extract response data using helper functions
		content := getContent(llmResp)
		toolCalls := getToolCalls(llmResp)
		finishReason := getFinishReason(llmResp)

		// Check for empty response to prevent getting stuck
		if llmResp == nil || (content == "" && len(toolCalls) == 0) {
			slog.Warn("Received empty response from LLM, breaking loop to prevent stuck state",
				"turn", turn+1, "llm_resp_nil", llmResp == nil, "content_len", len(content), "tool_calls", len(toolCalls))
			resp.FinishReason = "empty_response"
			break
		}

		slog.Debug("LLM response received", "turn", turn+1, "content_len", len(content), "tool_calls", len(toolCalls), "finish_reason", finishReason)

		// Handle cycle detection via detection service
		if a.config.CycleDetection.Enabled {
			var shouldStop bool
			var err error
			messages, shouldStop, err = a.handleCycleDetection(ctx, messages, llmResp, turn+1, resp)
			if err != nil {
				slog.Error("cycle detection failed", "turn", turn+1, "error", err)
				return messages, resp, err
			}
			if shouldStop {
				slog.Info("cycle detected, stopping agent", "turn", turn+1)
				return messages, resp, nil
			}
		}

		// Process tool calls or finish
		if len(toolCalls) > 0 {
			slog.Debug("processing tool calls", "count", len(toolCalls), "turn", turn+1)
			messages = a.processToolCallsFromCompletion(ctx, messages, llmResp, resp)
			continue
		}

		messages = a.addFinalMessage(messages, content)
		resp.FinishReason = finishReason
		if resp.FinishReason == "" {
			resp.FinishReason = "stop"
		}
		break
	}

	// Store accumulated retrieved bullets in response for trajectory building
	resp.RetrievedBullets = allRetrievedBullets

	return messages, resp, nil
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
func (a *Agent) handleCycleDetection(ctx context.Context, messages []Message, llmResp *openai.ChatCompletion, turn int, resp *AgentResponse) ([]Message, bool, error) {
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

	cycleResult, err := a.detection.CheckCycle()
	if err != nil || cycleResult.Type == detection.CycleNone {
		return messages, false, nil
	}

	intervention := a.selectIntervention(cycleResult.Type, turn)
	if intervention == nil {
		return messages, false, nil
	}

	// Convert messages to detection.Message interface
	detectionMessages := make([]detection.Message, len(messages))
	for i, msg := range messages {
		detectionMessages[i] = &messageAdapter{msg: msg}
	}

	modifiedDetectionMessages, err := intervention.Apply(ctx, detectionMessages)
	if err != nil {
		slog.Warn("cycle intervention failed", "error", err, "cycle_type", cycleResult.Type)
		return messages, false, nil
	}

	// Convert back to Message slice
	messages = make([]Message, len(modifiedDetectionMessages))
	for i, msg := range modifiedDetectionMessages {
		messages[i] = Message{
			Role:      msg.GetRole(),
			Content:   msg.GetContent(),
			Timestamp: msg.GetTimestamp(),
		}
	}

	// Emit cycle detection event
	a.emitter.Emit(events.Event{
		Type:      events.EventWarning,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "warning",
			Message: fmt.Sprintf("Cycle detected: %s. Applied intervention: %s", cycleResult.Type, intervention.Name()),
			Details: cycleResult.Details,
		},
	})

	// If this was an escalation intervention, pause the agent
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
func (a *Agent) callLLMWithTimeout(ctx context.Context, messages []Message, task Task, bullets []*bullet.Bullet) (*openai.ChatCompletion, error) {
	// Use a reasonable timeout for LLM calls (5 minutes)
	// Don't use agent timeout which may be very long for multi-step tasks
	llmTimeout := 5 * time.Minute

	// Check if parent context has a deadline that's already expired
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		// If parent is about to expire, return error immediately
		if remaining <= 0 {
			return nil, fmt.Errorf("Agent.callLLM: context cancelled: context deadline exceeded")
		}
		// If parent has less time remaining, use that instead
		if remaining < llmTimeout {
			llmTimeout = remaining
		}
	}

	slog.Debug("calling LLM with timeout", "timeout", llmTimeout, "message_count", len(messages), "bullets_count", len(bullets))

	// Create fresh context with LLM timeout
	llmCtx, cancel := context.WithTimeout(context.Background(), llmTimeout)
	defer cancel()

	// Call the actual LLM method
	resp, err := a.callLLM(llmCtx, messages, task, bullets)
	if err != nil {
		slog.Error("LLM call error", "error", err, "timeout", llmTimeout)
	}
	return resp, err
}

// addFinalMessage adds the final assistant message to the messages array.
// This is called when the agent is done (no more tool calls).
func (a *Agent) addFinalMessage(messages []Message, content string) []Message {
	if content != "" {
		messages = append(messages, Message{
			Role:      RoleAssistant,
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
// - Late-stage/persistent cycles (>= 30 turns): Escalate to user
func (a *Agent) selectIntervention(cycleType detection.CycleType, turnCount int) detection.Intervention {
	// Escalation ladder based on turn count
	switch {
	case turnCount < 10:
		// Early cycles: Use soft intervention (reflection)
		return &detection.ReflectionIntervention{}

	case turnCount < 30:
		// Mid-stage cycles: Use medium intervention (context summarization)
		// For now, use reflection as fallback since summarization needs compressor integration
		return &detection.ReflectionIntervention{}

	default:
		// Late-stage/persistent cycles: Escalate to user
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
// extractToolNamesFromOrchestration extracts tool names from orchestration.ToolCall slice.
func extractToolNamesFromOrchestration(toolCalls []orchestration.ToolCall) []string {
	calls := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		// Include both name and arguments for accurate cycle detection
		calls[i] = tc.Function.Name + "(" + tc.Function.Arguments + ")"
	}
	return calls
}

// eventEmitterAdapter adapts events.EventEmitter to detection.EventEmitter interface.
type eventEmitterAdapter struct {
	emitter *events.EventEmitter
}

func (a *eventEmitterAdapter) Emit(event detection.Event) {
	// Convert detection.Event to events.Event
	// Map event type based on string value
	var eventType events.EventType
	switch event.GetType() {
	case "turn_paused":
		eventType = events.EventTurnPaused
	default:
		eventType = events.EventWarning // fallback
	}

	coreEvent := events.Event{
		Type:      eventType,
		Timestamp: event.GetTimestamp(),
		Data:      event.GetData(),
	}
	a.emitter.Emit(coreEvent)
}
