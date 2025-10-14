package cycle

import (
	"fmt"
	"sync"
	"time"
)

// Detector implements cycle detection for agent reasoning loops.
// It maintains a rolling history of snapshots and detects various
// patterns that indicate the agent may be stuck in a cycle.
type Detector struct {
	// history stores recent snapshots for pattern analysis
	history []Snapshot

	// config contains detection parameters
	config Config

	// mu protects concurrent access to history
	mu sync.RWMutex
}

// NewDetector creates a new cycle detector with the given configuration.
func NewDetector(config Config) *Detector {
	return &Detector{
		history: make([]Snapshot, 0),
		config:  config,
	}
}

// Record adds a new snapshot to the detection history.
// This should be called after each agent turn to maintain
// the rolling window of recent behavior.
func (d *Detector) Record(snapshot Snapshot) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add new snapshot
	d.history = append(d.history, snapshot)

	// Maintain rolling window (keep only the most recent snapshots)
	maxSize := d.config.WindowSize
	if maxSize <= 0 {
		maxSize = 3 // fallback to default
	}

	if len(d.history) > maxSize {
		// Remove oldest snapshots to maintain window size
		d.history = d.history[len(d.history)-maxSize:]
	}
}

// Check analyzes the current history for cycle patterns.
// Returns the type of cycle detected (or CycleNone) along with
// confidence and details about the detection.
func (d *Detector) Check() (CycleResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Need minimum history for meaningful analysis
	if len(d.history) < 2 {
		return CycleResult{
			Type:       CycleNone,
			Confidence: 0.0,
			Timestamp:  time.Now(),
		}, nil
	}

	// Check each pattern type (order matters - more specific patterns first)
	if result := d.checkRepeatedTool(); result.Type != CycleNone {
		return result, nil
	}

	if result := d.checkSameError(); result.Type != CycleNone {
		return result, nil
	}

	if result := d.checkOscillation(); result.Type != CycleNone {
		return result, nil
	}

	if result := d.checkSimilarResponses(); result.Type != CycleNone {
		return result, nil
	}

	// No cycle detected
	return CycleResult{
		Type:       CycleNone,
		Confidence: 0.0,
		Timestamp:  time.Now(),
	}, nil
}

// GetHistory returns a copy of the current snapshot history.
// This is primarily for testing and debugging purposes.
func (d *Detector) GetHistory() []Snapshot {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]Snapshot, len(d.history))
	copy(history, d.history)
	return history
}

// Reset clears the detection history.
// This can be useful for testing or when starting a new conversation.
func (d *Detector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.history = d.history[:0]
}

// checkSimilarResponses detects when consecutive responses are too similar.
// This uses text similarity to identify when the LLM is repeating itself.
func (d *Detector) checkSimilarResponses() CycleResult {
	if !d.config.Enabled || len(d.history) < d.config.WindowSize {
		return CycleResult{Type: CycleNone}
	}

	recent := d.history[len(d.history)-d.config.WindowSize:]

	// Check if this is actually a similar responses pattern
	// by ensuring the responses are similar but tools and errors are not the issue
	hasSimilarResponses := true
	similarities := make([]float64, 0, len(recent)-1)

	for i := 1; i < len(recent); i++ {
		if recent[i].Response == "" || recent[i-1].Response == "" {
			hasSimilarResponses = false
			break
		}

		sim := calculateSimilarity(recent[i-1].Response, recent[i].Response)
		similarities = append(similarities, sim)

		// Early exit if any pair is dissimilar
		if sim < d.config.SimilarityThresh {
			hasSimilarResponses = false
			break
		}
	}

	// Only return similar responses if we have similar responses AND
	// no other patterns are more specific (checked by caller)
	if hasSimilarResponses && len(similarities) >= d.config.WindowSize-1 {
		return CycleResult{
			Type:       CycleSimilarResponses,
			Confidence: calculateAverageSimilarity(similarities),
			Details:    fmt.Sprintf("detected %d similar consecutive responses", len(similarities)+1),
			Timestamp:  time.Now(),
		}
	}

	return CycleResult{Type: CycleNone}
}

// checkRepeatedTool detects when the same tool is called repeatedly.
// This indicates the agent may be stuck trying the same approach.
func (d *Detector) checkRepeatedTool() CycleResult {
	if !d.config.Enabled || len(d.history) < d.config.ToolRepeatLimit {
		return CycleResult{Type: CycleNone}
	}

	recent := d.history[len(d.history)-d.config.ToolRepeatLimit:]

	// Check if all recent snapshots called the same tool
	if len(recent[0].ToolCalls) == 0 {
		return CycleResult{Type: CycleNone}
	}

	firstTool := recent[0].ToolCalls[0]
	allSame := true

	for i := 1; i < len(recent); i++ {
		if len(recent[i].ToolCalls) == 0 || recent[i].ToolCalls[0] != firstTool {
			allSame = false
			break
		}
	}

	if allSame {
		return CycleResult{
			Type:       CycleRepeatedTool,
			Confidence: 0.9, // High confidence for exact tool name matches
			Details:    fmt.Sprintf("tool '%s' called %d times consecutively", firstTool, len(recent)),
			Timestamp:  time.Now(),
		}
	}

	return CycleResult{Type: CycleNone}
}

// checkOscillation detects A→B→A→B oscillation patterns in responses.
// This indicates the agent is alternating between two states without progress.
func (d *Detector) checkOscillation() CycleResult {
	// Need at least 4 snapshots for oscillation detection
	if !d.config.Enabled || len(d.history) < 4 {
		return CycleResult{Type: CycleNone}
	}

	recent := d.history[len(d.history)-4:]

	// Check for A→B→A→B pattern where A and B are similar within themselves
	// but different from each other
	simAB := calculateSimilarity(recent[0].Response, recent[1].Response)
	simBA := calculateSimilarity(recent[1].Response, recent[2].Response)
	simAA := calculateSimilarity(recent[2].Response, recent[3].Response)

	// Check if first pair (A→B) and second pair (B→A) are similar within pairs
	// but the pairs are different from each other
	withinPairSimilarity := (simAB + simBA + simAA) / 3.0
	betweenPairSimilarity := calculateSimilarity(recent[0].Response, recent[2].Response)

	// Pattern: High similarity within pairs, low similarity between pairs
	if withinPairSimilarity >= d.config.SimilarityThresh && betweenPairSimilarity < 0.5 {
		return CycleResult{
			Type:       CycleOscillation,
			Confidence: withinPairSimilarity,
			Details:    fmt.Sprintf("detected A→B→A→B oscillation pattern (within-pair sim: %.2f)", withinPairSimilarity),
			Timestamp:  time.Now(),
		}
	}

	return CycleResult{Type: CycleNone}
}

// checkSameError detects when the same error occurs repeatedly.
// This indicates the agent is stuck in a failure loop.
func (d *Detector) checkSameError() CycleResult {
	if !d.config.Enabled || len(d.history) < d.config.ErrorRepeatLimit {
		return CycleResult{Type: CycleNone}
	}

	recent := d.history[len(d.history)-d.config.ErrorRepeatLimit:]

	// Check if all recent snapshots had the same error
	if recent[0].Error == "" {
		return CycleResult{Type: CycleNone}
	}

	firstError := recent[0].Error
	allSame := true

	for i := 1; i < len(recent); i++ {
		if recent[i].Error != firstError {
			allSame = false
			break
		}
	}

	if allSame {
		return CycleResult{
			Type:       CycleSameError,
			Confidence: 0.95, // Very high confidence for exact error matches
			Details:    fmt.Sprintf("error '%s' occurred %d times consecutively", firstError, len(recent)),
			Timestamp:  time.Now(),
		}
	}

	return CycleResult{Type: CycleNone}
}

// calculateAverageSimilarity computes the average of similarity values
func calculateAverageSimilarity(similarities []float64) float64 {
	if len(similarities) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, sim := range similarities {
		sum += sim
	}

	return sum / float64(len(similarities))
}
