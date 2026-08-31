package overlay

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

const hookVetoPrefix = "HOOK veto: "

// HookVetoDialog shows a blocking hook's reason (approval-style overlay).
type HookVetoDialog struct {
	reason string
}

// NewHookVetoDialog creates a veto overlay with the script reason.
func NewHookVetoDialog(reason string) *HookVetoDialog {
	return &HookVetoDialog{reason: reason}
}

// Render returns the visible veto text.
func (d *HookVetoDialog) Render() string {
	if d == nil || d.reason == "" {
		return "HOOK veto"
	}

	return hookVetoPrefix + d.reason
}
