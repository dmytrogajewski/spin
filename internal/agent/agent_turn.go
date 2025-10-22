package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// executeAgentLoop runs the main agent execution loop.
func (a *Agent) executeAgentLoop(ctx context.Context, messages []Message, task Task, resp *AgentResponse) ([]Message, *AgentResponse, error) {
	maxTurns := a.config.MaxTurns

	for turn := 0; turn < maxTurns; turn++ {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			resp.FinishReason = "timeout"
			return messages, resp, err
		}

		a.emitTurnStart(turn + 1)

		// Call LLM
		llmResp, err := a.callLLM(ctx, messages, task)
		if err != nil {
			resp.Error = fmt.Errorf("LLM call failed: %w", err)
			resp.FinishReason = "error"
			return messages, resp, err
		}

		// Handle cycle detection via detection service
		if a.config.CycleDetection.Enabled {
			var shouldStop bool
			var err error
			messages, shouldStop, err = a.handleCycleDetection(ctx, messages, llmResp, turn+1, resp)
			if err != nil {
				return messages, resp, err
			}
			if shouldStop {
				return messages, resp, nil
			}
		}

		// Process tool calls or finish
		if len(llmResp.ToolCalls) > 0 {
			slog.Debug("processing tool calls", "count", len(llmResp.ToolCalls), "turn", turn+1)
			messages = a.processToolCalls(ctx, messages, llmResp, resp)
			continue
		}

		messages = a.addFinalMessage(messages, llmResp.Content)
		resp.FinishReason = llmResp.FinishReason
		if resp.FinishReason == "" {
			resp.FinishReason = "stop"
		}
		break
	}

	return messages, resp, nil
}

// handleCycleDetection processes cycle detection and interventions via detection service.
// Returns the modified messages (with intervention added if applicable), whether to stop, and any error.
func (a *Agent) handleCycleDetection(ctx context.Context, messages []Message, llmResp *llm.CompletionResponse, turn int, resp *AgentResponse) ([]Message, bool, error) {
	snapshot := cycle.Snapshot{
		Turn:      turn,
		Response:  llmResp.Content,
		ToolCalls: extractToolNames(llmResp.ToolCalls),
		Error:     "",
		Timestamp: time.Now(),
	}
	a.detection.RecordSnapshot(snapshot)

	cycleResult, err := a.detection.CheckCycle()
	if err != nil || cycleResult.Type == cycle.CycleNone {
		return messages, false, nil
	}

	intervention := a.selectIntervention(cycleResult.Type, turn)
	if intervention == nil {
		return messages, false, nil
	}

	// Convert messages to cycle.Message interface
	cycleMessages := make([]cycle.Message, len(messages))
	for i, msg := range messages {
		cycleMessages[i] = &messageAdapter{msg: msg}
	}

	modifiedCycleMessages, err := intervention.Apply(ctx, cycleMessages)
	if err != nil {
		slog.Warn("cycle intervention failed", "error", err, "cycle_type", cycleResult.Type)
		return messages, false, nil
	}

	// Convert back to Message slice
	messages = make([]Message, len(modifiedCycleMessages))
	for i, msg := range modifiedCycleMessages {
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

// callLLM calls the LLM provider with the given messages and filtered tools based on task mode.
// The task parameter controls both tool filtering and token budget:
//   - Tools: Only tools in task.AllowedTools() are included
//   - Tokens: Uses task.MaxTokens() if > 0, otherwise agent.config.MaxTokens
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
	// Convert messages to LLM format
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMsg := llm.Message{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		// Convert tool calls if present
		if len(msg.ToolCalls) > 0 {
			llmMsg.ToolCalls = make([]llm.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				llmMsg.ToolCalls[j] = llm.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: llm.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		llmMessages[i] = llmMsg
	}

	// Build filtered tool list for this task mode
	tools, err := a.BuildToolsForTask(task)
	if err != nil {
		return nil, fmt.Errorf("failed to build tools: %w", err)
	}

	// Determine token budget: task overrides agent config
	maxTokens := a.config.MaxTokens
	if task != nil {
		taskMaxTokens := task.MaxTokens()
		if taskMaxTokens > 0 {
			maxTokens = taskMaxTokens
		}
	}

	// Build LLM request with filtered tools
	req := llm.CompletionRequest{
		Messages:    llmMessages,
		Temperature: a.config.Temperature,
		MaxTokens:   maxTokens,
		Tools:       tools,
	}
	slog.Debug("calling LLM", "tool_count", len(tools), "message_count", len(llmMessages))

	// Call LLM with streaming
	chunks, err := a.llm.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start LLM stream: %w", err)
	}

	// Accumulate response from streaming chunks
	response := &llm.CompletionResponse{
		Content:      "",
		ToolCalls:    []llm.ToolCall{},
		Usage:        llm.Usage{},
		FinishReason: "",
	}

	for chunk := range chunks {
		if chunk.Error != nil {
			return nil, fmt.Errorf("stream error: %w", chunk.Error)
		}

		// Accumulate content
		response.Content += chunk.Content

		// Emit content delta immediately for real-time streaming
		if chunk.Content != "" {
			a.emitter.Emit(events.Event{
				Type:      events.EventContentDelta,
				Timestamp: time.Now(),
				Data: events.ContentDeltaData{
					Content: chunk.Content,
					Role:    "assistant",
				},
			})
		}

		// Accumulate tool calls
		if chunk.ToolCall != nil {
			response.ToolCalls = append(response.ToolCalls, *chunk.ToolCall)
		}

		// Update finish reason
		if chunk.FinishReason != "" {
			response.FinishReason = chunk.FinishReason
		}
	}

	return response, nil
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

// messageAdapter adapts agent.Message to cycle.Message interface
type messageAdapter struct {
	msg Message
}

func (m *messageAdapter) GetRole() string {
	return string(m.msg.Role)
}

func (m *messageAdapter) GetContent() string {
	return m.msg.Content
}

func (m *messageAdapter) GetTimestamp() time.Time {
	return m.msg.Timestamp
}
