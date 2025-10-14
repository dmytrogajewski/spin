package cycle

import (
	"context"
	"fmt"
	"time"
)

// Message represents a conversation message for interventions
type Message interface {
	GetRole() string
	GetContent() string
	GetTimestamp() time.Time
}

// EventEmitter defines the interface for emitting events
type EventEmitter interface {
	Emit(event Event)
}

// Event defines the interface for events
type Event interface {
	GetType() string
	GetTimestamp() time.Time
	GetData() interface{}
}

// Intervention defines the interface for cycle-breaking strategies.
// Each intervention type implements a different approach to breaking
// the detected cycle and getting the agent back on track.
type Intervention interface {
	// Apply executes the intervention strategy
	// Returns modified messages and any error that occurred
	Apply(ctx context.Context, messages []Message) ([]Message, error)

	// Name returns a human-readable name for this intervention
	Name() string

	// Description returns a detailed description of what this intervention does
	Description() string

	// Severity returns the severity level of this intervention (1-3)
	// 1=Soft (minimal disruption), 2=Medium (moderate disruption), 3=Hard (significant disruption)
	Severity() int
}

// ReflectionIntervention implements a soft intervention that injects
// a reflection prompt to help the agent recognize and break out of cycles.
type ReflectionIntervention struct{}

// Apply adds a reflection prompt to the conversation to help break the cycle
func (i *ReflectionIntervention) Apply(ctx context.Context, messages []Message) ([]Message, error) {
	// Create a reflection prompt that acknowledges the potential cycle
	// and encourages the agent to try a different approach
	reflectionMsg := &messageImpl{
		role:      "user",
		content:   "I notice you may be repeating similar responses or approaches. Let's take a step back and try a different perspective. What other angles or methods could we explore for this task?",
		timestamp: time.Now(),
	}

	// Add the reflection message to the conversation
	result := append(messages, reflectionMsg)

	return result, nil
}

// Name returns the intervention name
func (i *ReflectionIntervention) Name() string {
	return "Reflection"
}

// Description returns a detailed description
func (i *ReflectionIntervention) Description() string {
	return "Injects a reflection prompt to help the agent recognize repetitive patterns and consider alternative approaches"
}

// Severity returns the severity level (soft)
func (i *ReflectionIntervention) Severity() int {
	return 1
}

// SummarizeIntervention implements a medium intervention that compresses
// the conversation context to help the agent refocus and break cycles.
type SummarizeIntervention struct {
	// compressor is used to compress the conversation history
	compressor interface {
		Compress(messages []Message, targetTokens int) ([]Message, error)
	}
}

// Apply compresses the conversation context to help break the cycle
func (i *SummarizeIntervention) Apply(ctx context.Context, messages []Message) ([]Message, error) {
	if i.compressor == nil {
		return messages, fmt.Errorf("compressor not configured for summarization intervention")
	}

	// Compress to approximately 50% of current message count
	// This provides a significant reduction while preserving key information
	targetCount := len(messages) / 2
	if targetCount < 1 {
		targetCount = 1
	}

	compressed, err := i.compressor.Compress(messages, targetCount)
	if err != nil {
		return messages, fmt.Errorf("failed to compress context for intervention: %w", err)
	}

	// Add a system message explaining the summarization
	systemMsg := &messageImpl{
		role:      "system",
		content:   "Context has been summarized to help focus on the current task and break potential reasoning loops. Key information has been preserved while reducing redundancy.",
		timestamp: time.Now(),
	}

	// Return compressed messages plus the system explanation
	result := append(compressed, systemMsg)

	return result, nil
}

// Name returns the intervention name
func (i *SummarizeIntervention) Name() string {
	return "Context Summarization"
}

// Description returns a detailed description
func (i *SummarizeIntervention) Description() string {
	return "Compresses conversation context to approximately 50% of original size to help the agent refocus and break cycles"
}

// Severity returns the severity level (medium)
func (i *SummarizeIntervention) Severity() int {
	return 2
}

// EscalateIntervention implements a hard intervention that pauses the agent
// and requests user guidance when automated interventions have failed.
type EscalateIntervention struct {
	// Emitter is used to send events to the UI
	Emitter EventEmitter
}

// Apply pauses the agent and requests user guidance
func (i *EscalateIntervention) Apply(ctx context.Context, messages []Message) ([]Message, error) {
	// Emit a turn paused event to notify the UI
	if i.Emitter != nil {
		i.Emitter.Emit(&eventImpl{
			eventType: "turn_paused",
			timestamp: time.Now(),
			data: map[string]interface{}{
				"status":  "paused",
				"message": "Agent appears stuck in a reasoning cycle. Please provide guidance or restart the conversation.",
			},
		})
	}

	// Return messages unchanged - the agent execution should be paused
	// by the calling code when this intervention is applied
	return messages, nil
}

// Name returns the intervention name
func (i *EscalateIntervention) Name() string {
	return "User Escalation"
}

// Description returns a detailed description
func (i *EscalateIntervention) Description() string {
	return "Pauses agent execution and requests user guidance when automated interventions fail to break the cycle"
}

// Severity returns the severity level (hard)
func (i *EscalateIntervention) Severity() int {
	return 3
}

// InterventionSelector chooses the appropriate intervention based on
// cycle type, conversation length, and previous interventions.
type InterventionSelector struct {
	// Track previous interventions to implement escalation
	previousInterventions []InterventionResult
}

// NewInterventionSelector creates a new intervention selector
func NewInterventionSelector() *InterventionSelector {
	return &InterventionSelector{
		previousInterventions: make([]InterventionResult, 0),
	}
}

// SelectIntervention chooses the best intervention for the given cycle and context
func (is *InterventionSelector) SelectIntervention(cycleType CycleType, turnCount int) Intervention {
	// Base selection on cycle type and turn count (escalation ladder)
	switch {
	case turnCount < 10:
		// Early cycles: Use soft intervention
		return &ReflectionIntervention{}

	case turnCount < 30:
		// Mid-stage cycles: Use medium intervention
		return &SummarizeIntervention{}

	default:
		// Late-stage/persistent cycles: Escalate to user
		return &EscalateIntervention{}
	}
}

// RecordIntervention records that an intervention was applied
// This helps track escalation and prevents repeated soft interventions
func (is *InterventionSelector) RecordIntervention(result InterventionResult) {
	is.previousInterventions = append(is.previousInterventions, result)

	// Keep only recent interventions (last 10)
	if len(is.previousInterventions) > 10 {
		is.previousInterventions = is.previousInterventions[1:]
	}
}

// GetPreviousInterventions returns the history of applied interventions
func (is *InterventionSelector) GetPreviousInterventions() []InterventionResult {
	// Return a copy to prevent external modification
	results := make([]InterventionResult, len(is.previousInterventions))
	copy(results, is.previousInterventions)
	return results
}

// messageImpl implements the Message interface for intervention use
type messageImpl struct {
	role      string
	content   string
	timestamp time.Time
}

// GetRole returns the message role
func (m *messageImpl) GetRole() string {
	return m.role
}

// GetContent returns the message content
func (m *messageImpl) GetContent() string {
	return m.content
}

// GetTimestamp returns the message timestamp
func (m *messageImpl) GetTimestamp() time.Time {
	return m.timestamp
}

// eventImpl implements the Event interface for intervention use
type eventImpl struct {
	eventType string
	timestamp time.Time
	data      interface{}
}

// GetType returns the event type
func (e *eventImpl) GetType() string {
	return e.eventType
}

// GetTimestamp returns the event timestamp
func (e *eventImpl) GetTimestamp() time.Time {
	return e.timestamp
}

// GetData returns the event data
func (e *eventImpl) GetData() interface{} {
	return e.data
}
