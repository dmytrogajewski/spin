package cycle

import (
	"fmt"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/pkg/alg/collections"
	"github.com/dmytrogajewski/spin/pkg/alg/search"
	"github.com/dmytrogajewski/spin/pkg/alg/similarity"
)

const (
	minHistoryForDetection = 2
	exactMatchConfidence   = 0.9
	groupSimilarityDivisor = 2.0
	exactErrorConfidence   = 0.95
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
// Returns the type of cycle detected (or None) along with
// confidence and details about the detection.
func (d *Detector) Check() (Result, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Need minimum history for meaningful analysis.
	if len(d.history) < minHistoryForDetection {
		return Result{
			Type:       None,
			Confidence: 0.0,
			Timestamp:  time.Now(),
		}, nil
	}

	// Check each pattern type (order matters - more specific patterns first).
	if result := d.checkRepeatedTool(); result.Type != None {
		return result, nil
	}

	if result := d.checkSameError(); result.Type != None {
		return result, nil
	}

	if result := d.checkOscillation(); result.Type != None {
		return result, nil
	}

	if result := d.checkSimilarResponses(); result.Type != None {
		return result, nil
	}

	// No cycle detected.
	return Result{
		Type:       None,
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
func (d *Detector) checkSimilarResponses() Result {
	if !d.hasMinimumSnapshots() {
		return Result{Type: None}
	}

	recent := d.getRecentSnapshots()
	similarities := d.calculateResponseSimilarities(recent)

	if d.isSimilarResponsePattern(similarities) {
		return d.createSimilarResponseResult(similarities)
	}

	return Result{Type: None}
}

// hasMinimumSnapshots checks if we have enough snapshots for detection.
func (d *Detector) hasMinimumSnapshots() bool {
	minSnapshots := 3

	return d.config.Enabled && len(d.history) >= minSnapshots
}

// getRecentSnapshots returns recent snapshots up to WindowSize.
func (d *Detector) getRecentSnapshots() []Snapshot {
	return collections.TailNOrAll(d.history, d.config.WindowSize)
}

// calculateResponseSimilarities calculates similarities between consecutive responses.
func (d *Detector) calculateResponseSimilarities(recent []Snapshot) []float64 {
	similarities := make([]float64, 0, len(recent)-1)

	for i := 1; i < len(recent); i++ {
		if !d.hasValidResponses(recent[i-1], recent[i]) {
			break
		}

		sim := similarity.JaccardSimilarity(recent[i-1].Response, recent[i].Response)
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
func (d *Detector) createSimilarResponseResult(similarities []float64) Result {
	return Result{
		Type:       SimilarResponses,
		Confidence: collections.Mean(similarities),
		Details:    fmt.Sprintf("detected %d similar consecutive responses", len(similarities)+1),
		Timestamp:  time.Now(),
	}
}

// checkRepeatedTool detects when the same tool is called repeatedly.
func (d *Detector) checkRepeatedTool() Result {
	return d.detectRepeatPattern(repeatSpec{
		minCount: d.config.ToolRepeatLimit,
		isValid:  func(s Snapshot) bool { return len(s.ToolCalls) > 0 },
		areSame: func(a, b Snapshot) bool {
			return len(a.ToolCalls) > 0 && len(b.ToolCalls) > 0 && a.ToolCalls[0] == b.ToolCalls[0]
		},
		resultType: RepeatedTool,
		confidence: exactMatchConfidence,
		details: func(recent []Snapshot) string {
			return fmt.Sprintf("tool '%s' called %d times consecutively", recent[0].ToolCalls[0], len(recent))
		},
	})
}

// checkOscillation detects A→B→A→B oscillation patterns in responses.
// This indicates the agent is alternating between two states without progress.
func (d *Detector) checkOscillation() Result {
	// Need at least 4 snapshots for oscillation detection.
	if !d.config.Enabled || len(d.history) < 4 {
		return Result{Type: None}
	}

	recent := d.history[len(d.history)-4:]

	// Check for A→B→A→B pattern where A responses are similar to each other
	// and B responses are similar to each other, but A and B are different
	// recent[0] = A1, recent[1] = B1, recent[2] = A2, recent[3] = B2.
	simAA := similarity.JaccardSimilarity(recent[0].Response, recent[2].Response) // A1 vs A2.
	simBB := similarity.JaccardSimilarity(recent[1].Response, recent[3].Response) // B1 vs B2.
	simAB := similarity.JaccardSimilarity(recent[0].Response, recent[1].Response) // A1 vs B1.

	// Average similarity within same groups (A's and B's).
	withinGroupSimilarity := (simAA + simBB) / groupSimilarityDivisor

	// Pattern: High similarity within groups (A's together, B's together),
	// but low similarity between groups (A vs B).
	if withinGroupSimilarity >= d.config.SimilarityThresh && simAB < 0.5 {
		return Result{
			Type:       Oscillation,
			Confidence: withinGroupSimilarity,
			Details:    fmt.Sprintf("detected oscillation pattern with within-group similarity %.2f", withinGroupSimilarity),
			Timestamp:  time.Now(),
		}
	}

	return Result{Type: None}
}

// checkSameError detects when the same error occurs repeatedly.
func (d *Detector) checkSameError() Result {
	return d.detectRepeatPattern(repeatSpec{
		minCount:   d.config.ErrorRepeatLimit,
		isValid:    func(s Snapshot) bool { return s.Error != "" },
		areSame:    func(a, b Snapshot) bool { return a.Error == b.Error },
		resultType: SameError,
		confidence: exactErrorConfidence,
		details: func(recent []Snapshot) string {
			return fmt.Sprintf("error '%s' occurred %d times consecutively", recent[0].Error, len(recent))
		},
	})
}

// repeatSpec describes a repeat-detection pattern that checks if the last N items all match.
type repeatSpec struct {
	minCount   int
	isValid    func(Snapshot) bool
	areSame    func(Snapshot, Snapshot) bool
	resultType Type
	confidence float64
	details    func([]Snapshot) string
}

// detectRepeatPattern is the shared protocol for checkRepeatedTool and checkSameError:
// check enabled → check minimum count → get tail → validate first → check all same → build result.
func (d *Detector) detectRepeatPattern(spec repeatSpec) Result {
	if !d.config.Enabled || len(d.history) < spec.minCount {
		return Result{Type: None}
	}

	recent := collections.TailN(d.history, spec.minCount)
	if len(recent) == 0 || !spec.isValid(recent[0]) {
		return Result{Type: None}
	}

	if !search.DetectRepeat(recent, spec.areSame) {
		return Result{Type: None}
	}

	return Result{
		Type:       spec.resultType,
		Confidence: spec.confidence,
		Details:    spec.details(recent),
		Timestamp:  time.Now(),
	}
}
