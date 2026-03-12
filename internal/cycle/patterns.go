package cycle

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/detection"
)

// PatternDetector provides additional pattern detection methods
// that can be used in conjunction with the main detector.
type PatternDetector struct {
	config Config
}

// NewPatternDetector creates a new pattern detector.
func NewPatternDetector(config Config) *PatternDetector {
	return &PatternDetector{config: config}
}

// AnalyzePatterns performs comprehensive pattern analysis on snapshots.
// This implements the detection.PatternDetector interface.
func (pd *PatternDetector) AnalyzePatterns(snapshots []Snapshot) []detection.PatternResult {
	internalResults := pd.analyzeInternal(snapshots)

	// Convert internal results to detection.PatternResult.
	results := make([]detection.PatternResult, 0, len(internalResults))
	for _, r := range internalResults {
		results = append(results, detection.PatternResult{
			Type:       r.Type.String(),
			Confidence: r.Confidence,
			Details:    r.Details,
		})
	}

	return results
}

// analyzeInternal performs the actual analysis returning full internal PatternResult.
// This is used by tests and internal code that needs the full result.
func (pd *PatternDetector) analyzeInternal(snapshots []Snapshot) []PatternResult {
	var results []PatternResult

	// Analyze response patterns.
	if result := pd.analyzeResponsePatterns(snapshots); result.Type != PatternNone {
		results = append(results, result)
	}

	// Analyze tool call patterns.
	if result := pd.analyzeToolPatterns(snapshots); result.Type != PatternNone {
		results = append(results, result)
	}

	// Analyze error patterns.
	if result := pd.analyzeErrorPatterns(snapshots); result.Type != PatternNone {
		results = append(results, result)
	}

	return results
}

// PatternType represents more granular pattern types.
type PatternType int

const (
	// PatternNone defines a PatternNone constant.
	PatternNone PatternType = iota
	// PatternRepeatedPhrase defines a PatternRepeatedPhrase constant.
	PatternRepeatedPhrase
	// PatternCircularReasoning detects circular reasoning patterns.
	PatternCircularReasoning
	// PatternToolStuck detects stuck tool patterns.
	PatternToolStuck
	// PatternErrorLoop detects error loop patterns.
	PatternErrorLoop
	// PatternOscillatingTools detects oscillating tool patterns.
	PatternOscillatingTools
)

// PatternResult contains detailed pattern analysis results.
type PatternResult struct {
	Type          PatternType
	Confidence    float64
	Details       string
	AffectedTurns []int
	Suggestion    string
}

// String returns the string representation of the pattern type.
func (pt PatternType) String() string {
	switch pt {
	case PatternRepeatedPhrase:
		return "repeated_phrase"
	case PatternCircularReasoning:
		return "circular_reasoning"
	case PatternToolStuck:
		return "tool_stuck"
	case PatternErrorLoop:
		return "error_loop"
	case PatternOscillatingTools:
		return "oscillating_tools"
	default:
		return "none"
	}
}

// analyzeResponsePatterns detects more sophisticated response patterns.
func (pd *PatternDetector) analyzeResponsePatterns(snapshots []Snapshot) PatternResult {
	if len(snapshots) < 3 {
		return PatternResult{Type: PatternNone}
	}

	// Look for repeated phrases across responses.
	phraseCounts := make(map[string]int)
	totalWords := 0

	for _, snapshot := range snapshots {
		if snapshot.Response == "" {
			continue
		}

		words := extractWords(snapshot.Response)
		totalWords += len(words)

		// Count 3-word phrases (n-grams).
		for i := range len(words) - 2 {
			phrase := strings.Join(words[i:i+3], " ")
			phraseCounts[phrase]++
		}
	}

	// Find most frequent phrases.
	var (
		mostFrequentPhrase string
		maxCount           int
	)

	for phrase, count := range phraseCounts {
		if count > maxCount {
			maxCount = count
			mostFrequentPhrase = phrase
		}
	}

	// If a phrase appears in most responses, it might indicate repetition.
	if maxCount >= len(snapshots)/2 && maxCount >= 2 {
		confidence := float64(maxCount) / float64(len(snapshots))

		return PatternResult{
			Type:       PatternRepeatedPhrase,
			Confidence: confidence,
			Details:    fmt.Sprintf("phrase '%s' appears %d times across responses", mostFrequentPhrase, maxCount),
			Suggestion: "Consider asking the agent to try a different approach or provide more specific guidance",
		}
	}

	// Look for circular reasoning indicators.
	if pd.detectCircularReasoning(snapshots) {
		return PatternResult{
			Type:       PatternCircularReasoning,
			Confidence: 0.7,
			Details:    "detected circular reasoning patterns in responses",
			Suggestion: "The agent appears to be going in circles. Try providing a concrete example or breaking down the problem.",
		}
	}

	return PatternResult{Type: PatternNone}
}

// analyzeToolPatterns detects problematic tool usage patterns.
func (pd *PatternDetector) analyzeToolPatterns(snapshots []Snapshot) PatternResult {
	if len(snapshots) < 3 {
		return PatternResult{Type: PatternNone}
	}

	// Count tool usage frequency.
	toolCounts := make(map[string]int)

	for _, snapshot := range snapshots {
		for _, tool := range snapshot.ToolCalls {
			toolCounts[tool]++
		}
	}

	// Find most frequently used tool.
	var (
		mostUsedTool string
		maxCount     int
	)

	for tool, count := range toolCounts {
		if count > maxCount {
			maxCount = count
			mostUsedTool = tool
		}
	}

	// If one tool dominates, it might indicate being stuck.
	if maxCount >= len(snapshots) && maxCount >= pd.config.ToolRepeatLimit {
		return PatternResult{
			Type:       PatternToolStuck,
			Confidence: float64(maxCount) / float64(len(snapshots)),
			Details:    fmt.Sprintf("tool '%s' used %d times in recent turns", mostUsedTool, maxCount),
			Suggestion: "The agent seems stuck using the same tool. Consider suggesting alternative approaches.",
		}
	}

	// Check for oscillating tool usage.
	if pd.detectOscillatingTools(snapshots) {
		return PatternResult{
			Type:       PatternOscillatingTools,
			Confidence: 0.8,
			Details:    "detected oscillating tool usage patterns",
			Suggestion: "Agent is alternating between tools without clear progress. Try providing more specific direction.",
		}
	}

	return PatternResult{Type: PatternNone}
}

// analyzeErrorPatterns detects problematic error patterns.
func (pd *PatternDetector) analyzeErrorPatterns(snapshots []Snapshot) PatternResult {
	if len(snapshots) < 2 {
		return PatternResult{Type: PatternNone}
	}

	// Count error frequency.
	errorCounts := make(map[string]int)
	errorTurns := make([]int, 0)

	for _, snapshot := range snapshots {
		if snapshot.Error != "" {
			errorCounts[snapshot.Error]++
			errorTurns = append(errorTurns, snapshot.Turn)
		}
	}

	// Find most frequent error.
	var (
		mostFrequentError string
		maxCount          int
	)

	for error, count := range errorCounts {
		if count > maxCount {
			maxCount = count
			mostFrequentError = error
		}
	}

	// If errors are repeating, it indicates a loop.
	if maxCount >= pd.config.ErrorRepeatLimit {
		return PatternResult{
			Type:          PatternErrorLoop,
			Confidence:    float64(maxCount) / float64(len(snapshots)),
			Details:       fmt.Sprintf("error '%s' occurred %d times", mostFrequentError, maxCount),
			AffectedTurns: errorTurns,
			Suggestion:    "The agent is encountering repeated errors. This suggests a systematic issue that needs to be addressed.",
		}
	}

	return PatternResult{Type: PatternNone}
}

// detectCircularReasoning looks for signs of circular reasoning in responses.
func (pd *PatternDetector) detectCircularReasoning(snapshots []Snapshot) bool {
	// Look for phrases that indicate circular reasoning.
	circularIndicators := []string{
		"as i mentioned",
		"as previously stated",
		"going back to",
		"returning to",
		"revisiting",
		"as we saw",
		"as discussed",
	}

	for _, snapshot := range snapshots {
		response := strings.ToLower(snapshot.Response)
		for _, indicator := range circularIndicators {
			if strings.Contains(response, indicator) {
				return true
			}
		}
	}

	return false
}

// detectOscillatingTools checks if tools are being used in an oscillating pattern.
func (pd *PatternDetector) detectOscillatingTools(snapshots []Snapshot) bool {
	if len(snapshots) < 4 {
		return false
	}

	// Look for A→B→A→B pattern in tool usage
	// This is a simplified check - a more sophisticated version would
	// look for semantic oscillation rather than just different tools.
	toolPattern := make([]string, len(snapshots))
	for i, snapshot := range snapshots {
		if len(snapshot.ToolCalls) > 0 {
			toolPattern[i] = snapshot.ToolCalls[0]
		} else {
			toolPattern[i] = ""
		}
	}

	// Check for alternating pattern: toolA, toolB, toolA, toolB.
	if len(snapshots) >= 4 {
		pattern := []string{toolPattern[0], toolPattern[1], toolPattern[2], toolPattern[3]}
		if pattern[0] != "" && pattern[1] != "" && pattern[0] != pattern[1] &&
			pattern[2] != "" && pattern[3] != "" && pattern[2] == pattern[0] && pattern[3] == pattern[1] {
			return true
		}
	}

	return false
}
