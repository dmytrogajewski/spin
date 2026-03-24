package adapter

// decisionValue holds the action and reason for a decision.
type decisionValue struct {
	action AdaptationAction
	reason string
}

var outcomeDecisions = map[SignalOutcome]decisionValue{
	OutcomeSuccess: {ActionSkip, "Skipping success signal (low priority)"},
	OutcomeFailure: {action: "", reason: ""}, // Sentinel: failures use signal-type lookup.
	OutcomeNeutral: {ActionSkip, "Skipping low-priority signal"},
}

var failureDecisions = map[SignalType]decisionValue{
	SignalTypeTest:    {ActionReflect, "Test failure detected - extracting insights"},
	SignalTypeUser:    {ActionReflect, "User correction - extracting lesson"},
	SignalTypeBuild:   {ActionQuickAdd, "Build error - generating prevention bullet"},
	SignalTypeError:   {ActionQuickAdd, "Runtime error - generating prevention bullet"},
	SignalTypeLint:    {ActionQuickAdd, "Lint error - generating fix bullet"},
	SignalTypeToolUse: {ActionSkip, "Tool use signal - no action needed"},
}

var defaultDecision = decisionValue{ActionSkip, "Unknown signal type"}

// decideAction determines what action to take for a given signal.
func decideAction(signal ExecutionSignal) (action AdaptationAction, reason string) {
	decision := lookupDecision(signal.Outcome, signal.SignalType)

	return decision.action, decision.reason
}

// lookupDecision finds the appropriate decision for the given outcome and signal type.
func lookupDecision(outcome SignalOutcome, signalType SignalType) decisionValue {
	if decision, ok := outcomeDecisions[outcome]; ok && decision.action != "" {
		return decision
	}

	return lookupFailureDecision(signalType)
}

// lookupFailureDecision handles failure outcome lookups.
func lookupFailureDecision(signalType SignalType) decisionValue {
	if decision, ok := failureDecisions[signalType]; ok {
		return decision
	}

	return defaultDecision
}
