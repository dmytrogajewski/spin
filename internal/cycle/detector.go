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
	// history stores recent snapshots for pattern analysis.
	history []Snapshot

	// config contains detection parameters.
	config Config

	// mu protects concurrent access to history.
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

	// Add new snapshot.
	d.history = append(d.history, snapshot)

	// Maintain rolling window (keep only the most recent snapshots).
	maxSize := d.config.WindowSize
	if maxSize <= 0 {
		maxSize = 3 // fallback to default.
	}

	if len(d.history) > maxSize {
		// Remove oldest snapshots to maintain window size.
		d.history = d.history[len(d.history)-maxSize:]
	}
}

// Check analyzes the current history for cycle patterns.
// Returns the type of cycle detected (or CycleNone) along with
// confidence and details about the detection.
func (d *Detector) Check() (CycleResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Need minimum history for meaningful analysis.
	if len(d.history) < 2 {
		return CycleResult{
			Type:       CycleNone,
			Confidence: 0.0,
			Timestamp:  time.Now(),
		}, nil
	}

	// Check each pattern type (order matters - more specific patterns first).
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

	// No cycle detected.
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

	// Return a copy to prevent external modification.
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
	if !d.hasMinimumSnapshots() {
		return CycleResult{Type: CycleNone}
	}

	recent := d.getRecentSnapshots()
	similarities := d.calculateResponseSimilarities(recent)

	if d.isSimilarResponsePattern(similarities) {
		return d.createSimilarResponseResult(similarities)
	}

	return CycleResult{Type: CycleNone}
}

// hasMinimumSnapshots checks if we have enough snapshots for detection.
func (d *Detector) hasMinimumSnapshots() bool {
	minSnapshots := 3

	return d.config.Enabled && len(d.history) >= minSnapshots
}

// getRecentSnapshots returns recent snapshots up to WindowSize.
func (d *Detector) getRecentSnapshots() []Snapshot {
	maxToCheck := d.config.WindowSize
	if maxToCheck <= 0 || maxToCheck > len(d.history) {
		maxToCheck = len(d.history)
	}

	return d.history[len(d.history)-maxToCheck:]
}

// calculateResponseSimilarities calculates similarities between consecutive responses.
func (d *Detector) calculateResponseSimilarities(recent []Snapshot) []float64 {
	similarities := make([]float64, 0, len(recent)-1)

	for i := 1; i < len(recent); i++ {
		if !d.hasValidResponses(recent[i-1], recent[i]) {
			break
		}

		sim := calculateSimilarity(recent[i-1].Response, recent[i].Response)
		similarities = append(similarities, sim)

		if sim < d.config.SimilarityThresh {
			break
		}
	}

	return similarities
}

// hasValidResponses checks if both snapshots have valid responses.
func (d *Detector) hasValidResponses(s1, s2 Snapshot) bool {
	return s1.Response != "" && s2.Response != ""
}

// isSimilarResponsePattern checks if similarities indicate a pattern.
func (d *Detector) isSimilarResponsePattern(similarities []float64) bool {
	minSnapshots := 3

	return len(similarities) >= minSnapshots-1
}

// createSimilarResponseResult creates a cycle result for similar responses.
func (d *Detector) createSimilarResponseResult(similarities []float64) CycleResult {
	return CycleResult{
		Type:       CycleSimilarResponses,
		Confidence: calculateAverageSimilarity(similarities),
		Details:    fmt.Sprintf("detected %d similar consecutive responses", len(similarities)+1),
		Timestamp:  time.Now(),
	}
}

// checkRepeatedTool detects when the same tool is called repeatedly.
// This indicates the agent may be stuck trying the same approach.
func (d *Detector) checkRepeatedTool() CycleResult {
	if !d.hasEnoughHistoryForToolCheck() {
		return CycleResult{Type: CycleNone}
	}

	recent := d.getRecentSnapshotsForToolCheck()
	if !d.hasValidToolCalls(recent) {
		return CycleResult{Type: CycleNone}
	}

	if d.allToolsAreSame(recent) {
		return d.createRepeatedToolResult(recent)
	}

	return CycleResult{Type: CycleNone}
}

// hasEnoughHistoryForToolCheck checks if we have enough history for tool analysis.
func (d *Detector) hasEnoughHistoryForToolCheck() bool {
	return d.config.Enabled && len(d.history) >= d.config.ToolRepeatLimit
}

// getRecentSnapshotsForToolCheck returns recent snapshots for tool analysis.
func (d *Detector) getRecentSnapshotsForToolCheck() []Snapshot {
	return d.history[len(d.history)-d.config.ToolRepeatLimit:]
}

// hasValidToolCalls checks if the first snapshot has valid tool calls.
func (d *Detector) hasValidToolCalls(recent []Snapshot) bool {
	return len(recent) > 0 && len(recent[0].ToolCalls) > 0
}

// allToolsAreSame checks if all recent snapshots use the same tool.
func (d *Detector) allToolsAreSame(recent []Snapshot) bool {
	firstTool := recent[0].ToolCalls[0]

	for i := 1; i < len(recent); i++ {
		if !d.snapshotUsesTool(recent[i], firstTool) {
			return false
		}
	}

	return true
}

// snapshotUsesTool checks if a snapshot uses the specified tool.
func (d *Detector) snapshotUsesTool(snapshot Snapshot, tool string) bool {
	return len(snapshot.ToolCalls) > 0 && snapshot.ToolCalls[0] == tool
}

// createRepeatedToolResult creates a cycle result for repeated tool usage.
func (d *Detector) createRepeatedToolResult(recent []Snapshot) CycleResult {
	firstTool := recent[0].ToolCalls[0]

	return CycleResult{
		Type:       CycleRepeatedTool,
		Confidence: 0.9, // High confidence for exact tool name matches.
		Details:    fmt.Sprintf("tool '%s' called %d times consecutively", firstTool, len(recent)),
		Timestamp:  time.Now(),
	}
}

// checkOscillation detects A→B→A→B oscillation patterns in responses.
// This indicates the agent is alternating between two states without progress.
func (d *Detector) checkOscillation() CycleResult {
	// Need at least 4 snapshots for oscillation detection.
	if !d.config.Enabled || len(d.history) < 4 {
		return CycleResult{Type: CycleNone}
	}

	recent := d.history[len(d.history)-4:]

	// Check for A→B→A→B pattern where A responses are similar to each other
	// and B responses are similar to each other, but A and B are different
	// recent[0] = A1, recent[1] = B1, recent[2] = A2, recent[3] = B2.
	simAA := calculateSimilarity(recent[0].Response, recent[2].Response) // A1 vs A2.
	simBB := calculateSimilarity(recent[1].Response, recent[3].Response) // B1 vs B2.
	simAB := calculateSimilarity(recent[0].Response, recent[1].Response) // A1 vs B1.

	// Average similarity within same groups (A's and B's).
	withinGroupSimilarity := (simAA + simBB) / 2.0

	// Pattern: High similarity within groups (A's together, B's together),
	// but low similarity between groups (A vs B).
	if withinGroupSimilarity >= d.config.SimilarityThresh && simAB < 0.5 {
		return CycleResult{
			Type:       CycleOscillation,
			Confidence: withinGroupSimilarity,
			Details:    fmt.Sprintf("detected oscillation pattern with within-group similarity %.2f", withinGroupSimilarity),
			Timestamp:  time.Now(),
		}
	}

	return CycleResult{Type: CycleNone}
}

// checkSameError detects when the same error occurs repeatedly.
// This indicates the agent is stuck in a failure loop.
func (d *Detector) checkSameError() CycleResult {
	if !d.hasEnoughHistoryForErrorCheck() {
		return CycleResult{Type: CycleNone}
	}

	recent := d.getRecentSnapshotsForErrorCheck()
	if !d.hasValidError(recent) {
		return CycleResult{Type: CycleNone}
	}

	if d.allErrorsAreSame(recent) {
		return d.createSameErrorResult(recent)
	}

	return CycleResult{Type: CycleNone}
}

// hasEnoughHistoryForErrorCheck checks if we have enough history for error analysis.
func (d *Detector) hasEnoughHistoryForErrorCheck() bool {
	return d.config.Enabled && len(d.history) >= d.config.ErrorRepeatLimit
}

// getRecentSnapshotsForErrorCheck returns recent snapshots for error analysis.
func (d *Detector) getRecentSnapshotsForErrorCheck() []Snapshot {
	return d.history[len(d.history)-d.config.ErrorRepeatLimit:]
}

// hasValidError checks if the first snapshot has a valid error.
func (d *Detector) hasValidError(recent []Snapshot) bool {
	return len(recent) > 0 && recent[0].Error != ""
}

// allErrorsAreSame checks if all recent snapshots have the same error.
func (d *Detector) allErrorsAreSame(recent []Snapshot) bool {
	firstError := recent[0].Error

	for i := 1; i < len(recent); i++ {
		if recent[i].Error != firstError {
			return false
		}
	}

	return true
}

// createSameErrorResult creates a cycle result for repeated errors.
func (d *Detector) createSameErrorResult(recent []Snapshot) CycleResult {
	firstError := recent[0].Error

	return CycleResult{
		Type:       CycleSameError,
		Confidence: 0.95, // Very high confidence for exact error matches.
		Details:    fmt.Sprintf("error '%s' occurred %d times consecutively", firstError, len(recent)),
		Timestamp:  time.Now(),
	}
}

// calculateAverageSimilarity computes the average of similarity values.
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
