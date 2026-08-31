package child

import (
	"sync"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/frame"
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

// Harness is the child's isolated conversation plus TaskFrame.
// It is constructed from a Spec only — parent history is never accepted.
type Harness struct {
	frame       frame.TaskFrame
	history     []a2a.Message
	hookScripts []hooks.PluginScript
	mu          sync.Mutex
}

// NewHarness builds an isolated child harness from spec. Parent history is not copied.
func NewHarness(spec *subagent.Spec) *Harness {
	taskFrame := frame.FromMode(phaseForSpec(spec.Name))
	taskFrame.Tools = append([]string{}, spec.AllowedTools...)

	return &Harness{frame: taskFrame}
}

// Frame returns the child's TaskFrame.
func (childHarness *Harness) Frame() frame.TaskFrame {
	return childHarness.frame
}

// History returns the child's own conversation messages.
func (childHarness *Harness) History() []a2a.Message {
	childHarness.mu.Lock()
	defer childHarness.mu.Unlock()

	return append([]a2a.Message{}, childHarness.history...)
}

// InheritHookScripts copies parent extra scripts onto the child. Drops nothing.
func (childHarness *Harness) InheritHookScripts(scripts []hooks.PluginScript) {
	childHarness.mu.Lock()
	defer childHarness.mu.Unlock()

	childHarness.hookScripts = hooks.CopyScripts(scripts)
}

// HookScripts returns extra scripts registered on this child.
func (childHarness *Harness) HookScripts() []hooks.PluginScript {
	childHarness.mu.Lock()
	defer childHarness.mu.Unlock()

	return hooks.CopyScripts(childHarness.hookScripts)
}

// Record appends a message received on this child's stream.
func (childHarness *Harness) Record(message a2a.Message) {
	childHarness.mu.Lock()
	defer childHarness.mu.Unlock()

	childHarness.history = append(childHarness.history, message)
}

func phaseForSpec(name string) string {
	switch name {
	case subagent.NamePlanner:
		return agent.ModePlanning
	case subagent.NameReviewer:
		return agent.ModeReview
	default:
		return agent.ModeRegular
	}
}
