package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/tui/ui"
)

// CoreEventMsg wraps core events as Bubble Tea messages.
type CoreEventMsg struct {
	Event core.Event
}

// waitForCoreEvent returns a Bubble Tea command that waits for the next core event.
// It converts core events into Bubble Tea messages.
func waitForCoreEvent(events <-chan core.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return nil // Channel closed
		}
		return CoreEventMsg{Event: event}
	}
}

// handleCoreEvent processes core events and updates the model.
// It routes events to specific handlers based on event type.
func (m Model) handleCoreEvent(msg CoreEventMsg) (Model, tea.Cmd) {
	switch msg.Event.Type {
	case core.EventTurnStart:
		return m.handleTurnStart(msg.Event), waitForCoreEvent(m.events)

	case core.EventContentDelta:
		return m.handleStreamDelta(msg.Event), waitForCoreEvent(m.events)

	case core.EventCommandApproval:
		return m.handleApprovalRequest(msg.Event), waitForCoreEvent(m.events)

	case core.EventCommandApproved:
		return m.handleApprovalResult(msg.Event, true), waitForCoreEvent(m.events)

	case core.EventCommandDenied:
		return m.handleApprovalResult(msg.Event, false), waitForCoreEvent(m.events)

	case core.EventTurnComplete, core.EventTurnFailed:
		return m.handleTurnComplete(msg.Event), waitForCoreEvent(m.events)

	case core.EventTurnPaused:
		return m.handleTurnPaused(msg.Event), waitForCoreEvent(m.events)

	case core.EventTurnResumed:
		return m.handleTurnResumed(msg.Event), waitForCoreEvent(m.events)

	case core.EventError:
		return m.handleError(msg.Event), waitForCoreEvent(m.events)

	default:
		// Unknown event type - just continue listening
		return m, waitForCoreEvent(m.events)
	}
}

// Event-specific handlers

// handleTurnStart processes turn start events.
func (m Model) handleTurnStart(event core.Event) Model {
	m.state = StateWaitingResponse
	return m
}

// handleStreamDelta processes streaming content deltas.
func (m Model) handleStreamDelta(event core.Event) Model {
	if data, ok := event.Data.(*core.ContentDeltaData); ok {
		m.chat.AppendDelta(data.Content)
	}
	return m
}

// handleApprovalRequest processes command approval requests.
func (m Model) handleApprovalRequest(event core.Event) Model {
	if req, ok := event.Data.(*core.ApprovalRequest); ok {
		m.state = StateToolApproval
		m.approval.SetRequest(req)
	}
	return m
}

// handleApprovalResult processes approval decision results.
func (m Model) handleApprovalResult(event core.Event, approved bool) Model {
	m.state = StateWaitingResponse
	// TODO: Add approval result to transcript
	return m
}

// handleTurnComplete processes turn completion events.
func (m Model) handleTurnComplete(event core.Event) Model {
	// Extract tokens from event data (map[string]interface{})
	if data, ok := event.Data.(map[string]interface{}); ok {
		if tokensUsed, ok := data["tokens_used"].(int); ok {
			m.statusBar.SetTokens(tokensUsed, 0) // Set turn tokens, session total TBD
		}
	}

	m.state = StateIdle
	m.chat.FinalizeMessage()
	return m
}

// handleTurnPaused processes turn pause events.
func (m Model) handleTurnPaused(event core.Event) Model {
	// Can interact while paused
	m.state = StateIdle
	// TODO: Show pause indicator in UI
	return m
}

// handleTurnResumed processes turn resume events.
func (m Model) handleTurnResumed(event core.Event) Model {
	m.state = StateWaitingResponse
	return m
}

// handleError processes error events (Phase 3.12).
func (m Model) handleError(event core.Event) Model {
	if data, ok := event.Data.(*core.ErrorData); ok {
		// Convert core.ErrorData to tui.ErrorDisplay
		errDisplay := NewErrorDisplay(*data)

		// Route error based on severity
		if errDisplay.ShouldShowInModal() {
			// Critical errors show in modal
			uiErr := convertToUIError(errDisplay)
			m.errorModal.Show(uiErr)
			// Don't change state - modal doesn't block
		} else if errDisplay.ShouldShowInStatusBar() {
			// Info/Warning errors show in status bar
			uiErr := convertToUIError(errDisplay)
			m.statusBar.ShowError(uiErr)
		} else if errDisplay.ShouldShowInTranscript() {
			// Error/Critical errors show inline in chat
			uiErr := convertToUIError(errDisplay)
			m.chat.AddError(uiErr)
		}
	}

	m.state = StateIdle
	return m
}

// convertToUIError converts tui.ErrorDisplay to ui.ErrorDisplay for UI components.
func convertToUIError(err ErrorDisplay) ui.ErrorDisplay {
	return ui.ErrorDisplay{
		Message:     err.Message,
		Code:        err.Code,
		Details:     err.Details,
		Operation:   err.Operation,
		Severity:    int(err.Severity),
		Timestamp:   err.Timestamp.Format("2006-01-02 15:04:05"),
		Dismissible: err.Dismissible,
		Dismissed:   err.Dismissed,
		AutoDismiss: int(err.AutoDismiss.Seconds()),
	}
}
