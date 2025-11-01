package adapter

// decideAction determines what action to take for a given signal
func decideAction(signal ExecutionSignal) (AdaptationAction, string) {
	// Skip success signals (not high priority for learning)
	if signal.Outcome == OutcomeSuccess {
		return ActionSkip, "Skipping success signal (low priority)"
	}

	// Skip neutral signals
	if signal.Outcome == OutcomeNeutral {
		return ActionSkip, "Skipping low-priority signal"
	}

	// Handle failure signals based on type
	switch signal.SignalType {
	case SignalTypeTest:
		// Test failures warrant full reflection
		return ActionReflect, "Test failure detected - extracting insights"

	case SignalTypeBuild:
		// Build errors are quick to extract
		return ActionQuickAdd, "Build error - generating prevention bullet"

	case SignalTypeError:
		// Runtime errors are quick to extract
		return ActionQuickAdd, "Runtime error - generating prevention bullet"

	case SignalTypeLint:
		// Lint issues are quick to extract
		return ActionQuickAdd, "Lint error - generating fix bullet"

	case SignalTypeUser:
		// User corrections are high-value, deserve reflection
		return ActionReflect, "User correction - extracting lesson"

	default:
		return ActionSkip, "Unknown signal type"
	}
}
