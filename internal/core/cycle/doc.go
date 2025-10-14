// Package cycle provides automatic detection of agent reasoning loops
// and intelligent intervention strategies to break cycles and maintain productivity.
//
// # Overview
//
// The cycle package implements comprehensive cycle detection for autonomous agents.
// It identifies when agents get stuck in infinite loops and applies appropriate
// interventions to restore productive behavior.
//
// # Detection Methods
//
// The package supports multiple detection methods:
//
//   - Response Similarity: Detects when LLM responses are too similar (>80% Jaccard similarity)
//   - Repeated Tool Calls: Identifies when the same tool is called repeatedly
//   - State Oscillation: Finds A→B→A→B alternating patterns in agent behavior
//   - Error Loops: Catches when identical errors occur multiple times
//
// # Intervention Strategies
//
// When cycles are detected, the package applies interventions based on severity:
//
//   - Reflection (Soft): Injects prompts encouraging alternative approaches
//   - Summarization (Medium): Compresses context to help refocus
//   - Escalation (Hard): Pauses execution and requests user guidance
//
// # Usage Example
//
//	// Create detector with default configuration
//	detector := cycle.NewDetector(cycle.DefaultConfig())
//
//	// Record agent behavior after each turn
//	detector.Record(cycle.Snapshot{
//		Turn:      turn,
//		Response:  llmResponse.Content,
//		ToolCalls: extractToolNames(llmResponse.ToolCalls),
//		Error:     errorMessage,
//		Timestamp: time.Now(),
//	})
//
//	// Check for cycles
//	result, err := detector.Check()
//	if err != nil {
//		// handle error
//	}
//
//	if result.Type != cycle.CycleNone {
//		// Select and apply intervention
//		selector := cycle.NewInterventionSelector()
//		intervention := selector.SelectIntervention(result.Type, turnCount)
//
//		modifiedMessages, err := intervention.Apply(ctx, messages)
//		if err != nil {
//			// handle intervention error
//		}
//
//		// Use modifiedMessages for next LLM call
//	}
//
// # Configuration
//
// The detector can be configured with custom thresholds:
//
//	config := cycle.Config{
//		WindowSize:       5,     // Look at last 5 snapshots
//		SimilarityThresh: 0.85,  // Higher threshold for similarity
//		ToolRepeatLimit:  4,     // Allow more tool repetitions
//		ErrorRepeatLimit: 2,     // Stricter error detection
//		Enabled:         true,   // Enable/disable detection
//	}
//
// # Integration
//
// The cycle package integrates with the core agent loop in internal/core/agent.go.
// It hooks into the Execute() method to check for cycles after each turn and
// apply interventions when needed.
//
// # Performance
//
// Cycle detection is designed to be fast (<10ms per check) with minimal memory
// overhead (<1MB for typical usage). Detection can be disabled entirely if needed.
package cycle
