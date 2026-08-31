package overlay

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

const harnessCategory = "Agent"

// RegisterHarnessCommands adds Skills, Tasks, and Agents palette entries.
func RegisterHarnessCommands(r *CommandRegistry) {
	if r == nil {
		return
	}

	r.Register(NewSimpleCommand("Skills", "List discovered agent skills", harnessCategory, 'S', nil))
	r.Register(NewSimpleCommand("Tasks", "List agent and shell tasks", harnessCategory, 'T', nil))
	r.Register(NewSimpleCommand("Agents", "List builtin specs and A2A peers", harnessCategory, 'A', nil))
}
