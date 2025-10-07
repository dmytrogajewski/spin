package tui

import (
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
)

// ApprovalBridge connects the core agent approval requests to the TUI.
// It provides a channel-based mechanism for the agent to wait for user decisions.
type ApprovalBridge struct {
	mu       sync.Mutex
	pending  map[string]chan core.ApprovalResponse
	emitter  *core.EventEmitter
}

// NewApprovalBridge creates a new approval bridge.
func NewApprovalBridge(emitter *core.EventEmitter) *ApprovalBridge {
	return &ApprovalBridge{
		pending:  make(map[string]chan core.ApprovalResponse),
		emitter:  emitter,
	}
}

// Handler returns an ApprovalHandler function for the agent.
// This function blocks until the TUI sends a response.
func (ab *ApprovalBridge) Handler() core.ApprovalHandler {
	return func(req core.ApprovalRequest) core.ApprovalResponse {
		// Create a channel for this request
		respChan := make(chan core.ApprovalResponse, 1)

		ab.mu.Lock()
		ab.pending[req.ID] = respChan
		ab.mu.Unlock()

		// Clean up on exit
		defer func() {
			ab.mu.Lock()
			delete(ab.pending, req.ID)
			ab.mu.Unlock()
		}()

		// Emit the approval request event for the TUI to handle
		// The TUI already listens for EventCommandApproval events
		// (this is redundant since agent already emits it, but keeping for clarity)

		// Wait for response with timeout
		select {
		case resp := <-respChan:
			return resp
		case <-time.After(60 * time.Second):
			// Timeout - return denied
			return core.ApprovalResponse{
				RequestID: req.ID,
				Approved:  false,
				Reason:    "approval timeout",
				Timestamp: time.Now(),
			}
		}
	}
}

// SendResponse sends an approval response from the TUI to the waiting agent.
func (ab *ApprovalBridge) SendResponse(resp core.ApprovalResponse) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ch, ok := ab.pending[resp.RequestID]; ok {
		select {
		case ch <- resp:
			// Response sent successfully
		default:
			// Channel full or closed, ignore
		}
	}
}