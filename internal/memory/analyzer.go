package memory

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	codeBlockThreshold       = 500
	toolOutputThreshold      = 1000
	priorityCritical         = 100
	priorityHigh             = 80
	priorityMedium           = 50
	charsPerTokenEstimate    = 4
)

// OffloadCandidate represents content that can be offloaded from context.
type OffloadCandidate struct {
	// MessageIndex is the index of the message in the conversation.
	MessageIndex int

	// Content is the content to offload.
	Content string

	// Reason explains why this should be offloaded.
	Reason string

	// Destination indicates where to offload (session or persistent).
	Destination Scope

	// Key is the suggested storage key.
	Key string

	// Priority determines offload order (higher = offload first).
	Priority int
}

// ContextAnalyzer identifies offloadable content in messages.
type ContextAnalyzer interface {
	// Analyze identifies offloadable content in messages.
	Analyze(messages []AnalyzableMessage) []OffloadCandidate
}

// AnalyzableMessage is a simplified message structure for analysis.
type AnalyzableMessage struct {
	Role    string
	Content string
}

// DefaultContextAnalyzer provides default analysis for context offloading.
type DefaultContextAnalyzer struct {
	// CodeBlockThreshold is the minimum token count for offloading code blocks.
	CodeBlockThreshold int

	// ToolOutputThreshold is the minimum token count for offloading tool outputs.
	ToolOutputThreshold int
}

// NewDefaultContextAnalyzer creates an analyzer with sensible defaults.
func NewDefaultContextAnalyzer() *DefaultContextAnalyzer {
	return &DefaultContextAnalyzer{
		CodeBlockThreshold:  codeBlockThreshold,
		ToolOutputThreshold: toolOutputThreshold,
	}
}

// Analyze identifies offloadable content in messages.
func (a *DefaultContextAnalyzer) Analyze(messages []AnalyzableMessage) []OffloadCandidate {
	candidates := make([]OffloadCandidate, 0)

	for i, msg := range messages {
		// Large code blocks -> scratchpad.
		codeBlocks := extractCodeBlocks(msg.Content)
		for j, block := range codeBlocks {
			tokens := estimateTokens(block)
			if tokens > a.CodeBlockThreshold {
				candidates = append(candidates, OffloadCandidate{
					MessageIndex: i,
					Content:      block,
					Reason:       "Large code block",
					Destination:  ScopeSession,
					Key:          generateKey("code", i, j),
					Priority:     priorityCritical,
				})
			}
		}

		// Long tool outputs -> scratchpad.
		if msg.Role == "tool" {
			tokens := estimateTokens(msg.Content)
			if tokens > a.ToolOutputThreshold {
				candidates = append(candidates, OffloadCandidate{
					MessageIndex: i,
					Content:      msg.Content,
					Reason:       "Large tool output",
					Destination:  ScopeSession,
					Key:          generateKey("tool_output", i, 0),
					Priority:     priorityHigh,
				})
			}
		}

		// Decisions -> persistent.
		if decision := extractDecision(msg.Content); decision != "" {
			candidates = append(candidates, OffloadCandidate{
				MessageIndex: i,
				Content:      decision,
				Reason:       "Important decision",
				Destination:  ScopePersistent,
				Key:          generateKey("decision", i, 0),
				Priority:     priorityMedium,
			})
		}
	}

	return candidates
}

// extractCodeBlocks finds code blocks in markdown content.
func extractCodeBlocks(content string) []string {
	// Match fenced code blocks (```...```).
	re := regexp.MustCompile("(?s)```[a-zA-Z]*\n?(.*?)```")
	matches := re.FindAllStringSubmatch(content, -1)

	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, match[1])
		}
	}

	return blocks
}

// extractDecision attempts to find decision statements in content.
func extractDecision(content string) string {
	lower := strings.ToLower(content)

	// Decision indicators.
	indicators := []string{
		"decided to ",
		"decision: ",
		"will use ",
		"chose to ",
		"going with ",
		"selected ",
	}

	for _, indicator := range indicators {
		idx := strings.Index(lower, indicator)
		if idx == -1 {
			continue
		}

		// Extract sentence containing the decision.
		start := idx
		// Find start of sentence.
		for start > 0 && content[start-1] != '.' && content[start-1] != '\n' {
			start--
		}
		// Find end of sentence.
		end := idx + len(indicator)
		for end < len(content) && content[end] != '.' && content[end] != '\n' {
			end++
		}

		if end < len(content) {
			end++ // Include the period.
		}

		return strings.TrimSpace(content[start:end])
	}

	return ""
}

// estimateTokens provides a rough estimate of token count.
// This is a simple heuristic: ~4 characters per token on average.
func estimateTokens(content string) int {
	return len(content) / charsPerTokenEstimate
}

// generateKey creates a unique key for offloaded content.
func generateKey(prefix string, messageIdx, blockIdx int) string {
	return strings.ToLower(strings.ReplaceAll(
		strings.TrimSpace(prefix)+"_"+
			strconv.Itoa(messageIdx)+"_"+
			strconv.Itoa(blockIdx),
		" ", "_"))
}
