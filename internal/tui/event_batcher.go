package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/tui/ui"
)

// BatchedEventsMsg contains multiple core events to process together.
type BatchedEventsMsg struct {
	Events []core.Event
}

// waitForBatchedEvents returns a command that collects multiple events within a time window.
// This improves streaming performance by processing multiple content deltas together.
func waitForBatchedEvents(events <-chan core.Event, batchTimeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		var batch []core.Event
		timer := time.NewTimer(batchTimeout)
		defer timer.Stop()

		for {
			select {
			case event, ok := <-events:
				if !ok {
					// Channel closed
					if len(batch) > 0 {
						return BatchedEventsMsg{Events: batch}
					}
					return nil
				}
				batch = append(batch, event)

				// For content deltas, keep collecting more without waiting
				if event.Type == core.EventContentDelta {
					// Try to collect more events immediately
					for {
						select {
						case e, ok := <-events:
							if !ok {
								return BatchedEventsMsg{Events: batch}
							}
							batch = append(batch, e)
							// Break if we get a different event type
							if e.Type != core.EventContentDelta {
								return BatchedEventsMsg{Events: batch}
							}
							// Limit batch size to prevent UI freezing
							if len(batch) >= 10 {
								return BatchedEventsMsg{Events: batch}
							}
						default:
							// No more events immediately available
							if len(batch) > 0 {
								return BatchedEventsMsg{Events: batch}
							}
							// Continue outer loop to wait with timeout
							goto continueWaiting
						}
					}
				} else {
					// For non-content events, return immediately
					return BatchedEventsMsg{Events: batch}
				}

			case <-timer.C:
				// Timeout reached, return what we have
				if len(batch) > 0 {
					return BatchedEventsMsg{Events: batch}
				}
				// Reset timer for next batch
				timer.Reset(batchTimeout)
			}
			continueWaiting:
		}
	}
}

// handleBatchedEvents processes multiple events at once.
func (m Model) handleBatchedEvents(msg BatchedEventsMsg) (Model, tea.Cmd) {
	for _, event := range msg.Events {
		m = m.processSingleEvent(event)
	}

	// Continue listening for more events
	return m, waitForBatchedEvents(m.events, time.Millisecond*8)
}

// processSingleEvent handles a single event without returning a command.
func (m Model) processSingleEvent(event core.Event) Model {
	switch event.Type {
	case core.EventTurnStart:
		m.state = StateWaitingResponse

	case core.EventContentDelta:
		// Stop spinner when we start receiving content
		if m.spinner.IsActive() {
			m.spinner.Stop()
		}

		// Process content delta
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			if content, ok := dataMap["content"].(string); ok {
				m.chat.AppendDelta(content)
			}
		} else if data, ok := event.Data.(*core.ContentDeltaData); ok {
			m.chat.AppendDelta(data.Content)
		}

	case core.EventToolCallStart:
		// Add tool call message to chat
		if data, ok := event.Data.(map[string]interface{}); ok {
			toolName, _ := data["tool_name"].(string)
			toolID, _ := data["tool_id"].(string)

			if toolName != "" {
				// Create a tool call message
				toolCall := ui.ToolCall{
					Name:      toolName,
					Arguments: make(map[string]interface{}),
					ID:        toolID,
				}

				// Extract arguments if present
				if args, ok := data["arguments"].(map[string]interface{}); ok {
					toolCall.Arguments = args
				}

				// Add message with tool call
				msg := ui.Message{
					Role:      ui.RoleAssistant,
					ToolCall:  &toolCall,
					Timestamp: time.Now(),
				}
				m.chat.AddMessage(msg)
			}
		}

	case core.EventToolCallComplete:
		// Add tool result message to chat
		if data, ok := event.Data.(map[string]interface{}); ok {
			toolResult := ui.ToolResult{}

			// Extract output and error
			if output, ok := data["output"].(string); ok {
				toolResult.Output = output
			}
			if errStr, ok := data["error"].(string); ok {
				toolResult.Error = errStr
			}

			// Add message with tool result
			msg := ui.Message{
				Role:       ui.RoleTool,
				ToolResult: &toolResult,
				Timestamp:  time.Now(),
			}
			m.chat.AddMessage(msg)
		}

	case core.EventCommandApproval:
		if req, ok := event.Data.(*core.ApprovalRequest); ok {
			m.state = StateToolApproval
			m.approval.SetRequest(req)
		}

	case core.EventCommandApproved, core.EventCommandDenied:
		m.state = StateWaitingResponse

	case core.EventTurnComplete, core.EventTurnFailed:
		m.spinner.Stop()
		if data, ok := event.Data.(map[string]interface{}); ok {
			if tokensUsed, ok := data["tokens_used"].(int); ok {
				m.statusBar.SetTokens(tokensUsed, 0)
			}
		}
		m.state = StateIdle
		m.chat.FinalizeMessage()

	case core.EventTurnPaused:
		m.state = StateIdle

	case core.EventTurnResumed:
		m.state = StateWaitingResponse

	case core.EventError:
		if data, ok := event.Data.(*core.ErrorData); ok {
			errDisplay := NewErrorDisplay(*data)
			if errDisplay.ShouldShowInModal() {
				uiErr := convertToUIError(errDisplay)
				m.errorModal.Show(uiErr)
			} else if errDisplay.ShouldShowInStatusBar() {
				uiErr := convertToUIError(errDisplay)
				m.statusBar.ShowError(uiErr)
			} else if errDisplay.ShouldShowInTranscript() {
				uiErr := convertToUIError(errDisplay)
				m.chat.AddError(uiErr)
			}
		}
		m.state = StateIdle
	}

	return m
}