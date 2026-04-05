package cycle

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/search"
	"github.com/dmytrogajewski/spin/pkg/alg/similarity"
)

const (
	minSnapshotsForAnalysis = 3
	minPatternsForTool      = 3
	toolCycleConfidence     = 0.7
	minPatternsForError     = 3
	errorCycleConfidence    = 0.8
	minPatternsForOscill    = 2
	minPatternsForABBA      = 4
	ngramSize               = 3
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
// This implements the PatternAnalyzer interface.
func (pd *PatternDetector) AnalyzePatterns(snapshots []Snapshot) []PatternResult {
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
	case PatternNone:
		return noneLabel
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
		return noneLabel
	}
}

// analyzeResponsePatterns detects more sophisticated response patterns.
func (pd *PatternDetector) analyzeResponsePatterns(snapshots []Snapshot) PatternResult {
	if len(snapshots) < minSnapshotsForAnalysis {
		return PatternResult{Type: PatternNone}
	}

	// Collect all 3-word phrases across responses.
	var allPhrases []string

	for _, snapshot := range snapshots {
		if snapshot.Response == "" {
			continue
		}

		words := similarity.ExtractWords(snapshot.Response)
		allPhrases = append(allPhrases, similarity.NGrams(words, ngramSize)...)
	}

	mostFrequentPhrase, maxCount := similarity.MaxByFrequency(allPhrases)

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
			Confidence: toolCycleConfidence,
			Details:    "detected circular reasoning patterns in responses",
			Suggestion: "The agent appears to be going in circles. Try providing a concrete example or breaking down the problem.",
		}
	}

	return PatternResult{Type: PatternNone}
}

// analyzeToolPatterns detects problematic tool usage patterns.
func (pd *PatternDetector) analyzeToolPatterns(snapshots []Snapshot) PatternResult {
	if len(snapshots) < minSnapshotsForAnalysis {
		return PatternResult{Type: PatternNone}
	}

	// Collect all tool calls and find the dominant tool.
	var allTools []string

	for _, snapshot := range snapshots {
		allTools = append(allTools, snapshot.ToolCalls...)
	}

	mostUsedTool, maxCount := similarity.MaxByFrequency(allTools)

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
			Confidence: errorCycleConfidence,
			Details:    "detected oscillating tool usage patterns",
			Suggestion: "Agent is alternating between tools without clear progress. Try providing more specific direction.",
		}
	}

	return PatternResult{Type: PatternNone}
}

// analyzeErrorPatterns detects problematic error patterns.
func (pd *PatternDetector) analyzeErrorPatterns(snapshots []Snapshot) PatternResult {
	if len(snapshots) < minPatternsForOscill {
		return PatternResult{Type: PatternNone}
	}

	// Collect errors and affected turns.
	var allErrors []string

	var errorTurns []int

	for _, snapshot := range snapshots {
		if snapshot.Error != "" {
			allErrors = append(allErrors, snapshot.Error)
			errorTurns = append(errorTurns, snapshot.Turn)
		}
	}

	mostFrequentError, maxCount := similarity.MaxByFrequency(allErrors)

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
	if len(snapshots) < minPatternsForABBA {
		return false
	}

	// Extract first tool name from each snapshot.
	toolNames := make([]string, minPatternsForABBA)
	for idx := range minPatternsForABBA {
		if len(snapshots[idx].ToolCalls) > 0 {
			toolNames[idx] = snapshots[idx].ToolCalls[0]
		}
	}

	// Empty tool names cannot form a valid pattern.
	if slices.Contains(toolNames, "") {
		return false
	}

	return search.DetectAlternating(toolNames)
}
