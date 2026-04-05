package hooks

// Event identifies a lifecycle point where hooks can execute.
type Event string

// eventCount is the total number of defined lifecycle events.
const eventCount = 10

// Lifecycle events matching SPEC2.md §3.3.3 Layer 5.
const (
	EventSessionStart       Event = "SESSION_START"
	EventUserPromptSubmit   Event = "USER_PROMPT_SUBMIT"
	EventPreToolUse         Event = "PRE_TOOL_USE"
	EventPostToolUse        Event = "POST_TOOL_USE"
	EventPostToolUseFailure Event = "POST_TOOL_USE_FAILURE"
	EventSubagentStart      Event = "SUBAGENT_START"
	EventSubagentStop       Event = "SUBAGENT_STOP"
	EventPreCompact         Event = "PRE_COMPACT"
	EventStop               Event = "STOP"
	EventSessionEnd         Event = "SESSION_END"
)

// AllEvents returns all defined lifecycle events in specification order.
func AllEvents() []Event {
	return []Event{
		EventSessionStart,
		EventUserPromptSubmit,
		EventPreToolUse,
		EventPostToolUse,
		EventPostToolUseFailure,
		EventSubagentStart,
		EventSubagentStop,
		EventPreCompact,
		EventStop,
		EventSessionEnd,
	}
}

// blockingEvents contains events where exit code 2 blocks the operation.
var blockingEvents = map[Event]bool{
	EventSessionStart:       false,
	EventUserPromptSubmit:   true,
	EventPreToolUse:         true,
	EventPostToolUse:        false,
	EventPostToolUseFailure: false,
	EventSubagentStart:      true,
	EventSubagentStop:       false,
	EventPreCompact:         false,
	EventStop:               false,
	EventSessionEnd:         false,
}

// IsBlocking reports whether the event supports blocking semantics.
// Blocking events can veto the operation via exit code 2.
func (e Event) IsBlocking() bool {
	return blockingEvents[e]
}

// ScriptName returns the hook script filename for this event.
// Example: EventPreToolUse → "pre-tool-use".
func (e Event) ScriptName() string {
	return eventScriptNames[e]
}

// eventScriptNames maps events to their hook script filenames.
var eventScriptNames = map[Event]string{
	EventSessionStart:       "session-start",
	EventUserPromptSubmit:   "user-prompt-submit",
	EventPreToolUse:         "pre-tool-use",
	EventPostToolUse:        "post-tool-use",
	EventPostToolUseFailure: "post-tool-use-failure",
	EventSubagentStart:      "subagent-start",
	EventSubagentStop:       "subagent-stop",
	EventPreCompact:         "pre-compact",
	EventStop:               "stop",
	EventSessionEnd:         "session-end",
}
