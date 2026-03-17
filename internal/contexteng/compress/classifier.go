package compress

import (
	"strings"

	"github.com/dmytrogajewski/spin/internal/message"
)

const (
	defaultVerboseThreshold = 1000
	minCodeBlockLines       = 3
)

// Importance represents message priority level for compression.
type Importance int

const (
	// ImportanceLow is for verbose output, thinking content (compress first).
	ImportanceLow Importance = iota

	// ImportanceMedium is for regular assistant responses.
	ImportanceMedium

	// ImportanceHigh is for code changes, decisions.
	ImportanceHigh

	// ImportanceCritical is for user messages, tool results, errors (always keep).
	ImportanceCritical
)

// String returns the importance level name.
func (i Importance) String() string {
	switch i {
	case ImportanceLow:
		return "low"
	case ImportanceMedium:
		return "medium"
	case ImportanceHigh:
		return "high"
	case ImportanceCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Classifier assigns importance levels to messages.
type Classifier struct {
	// verboseThreshold is the content length above which
	// assistant responses without code are considered verbose.
	verboseThreshold int
}

// NewClassifier creates a new message classifier with default settings.
func NewClassifier() *Classifier {
	return &Classifier{
		verboseThreshold: defaultVerboseThreshold,
	}
}

// ClassifierOption configures classifier behavior.
type ClassifierOption func(*Classifier)

// WithVerboseThreshold sets the threshold for verbose content detection.
func WithVerboseThreshold(threshold int) ClassifierOption {
	return func(c *Classifier) {
		c.verboseThreshold = threshold
	}
}

// NewClassifierWithOptions creates a classifier with custom options.
func NewClassifierWithOptions(opts ...ClassifierOption) *Classifier {
	c := NewClassifier()
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Classify assigns importance to a message.
func (c *Classifier) Classify(msg message.Message) Importance {
	// Critical: System messages (always preserve system prompt).
	if msg.Role == message.RoleSystem {
		return ImportanceCritical
	}

	// Critical: User messages (always preserve user intent).
	if msg.Role == message.RoleUser {
		return ImportanceCritical
	}

	// Critical: Tool results (must pair with tool calls).
	if msg.Role == message.RoleTool {
		return ImportanceCritical
	}

	// Critical: Tool calls (must pair with tool results).
	if len(msg.ToolCalls) > 0 {
		return ImportanceCritical
	}

	// Critical: Error messages.
	if c.isErrorMessage(msg) {
		return ImportanceCritical
	}

	// High: Code blocks (patches, diffs, code examples).
	if c.hasCodeBlock(msg) {
		return ImportanceHigh
	}

	// Assistant messages.
	if msg.Role == message.RoleAssistant {
		// Low: Verbose "thinking" content (long responses without code).
		if len(msg.Content) > c.verboseThreshold && !c.hasCodeBlock(msg) {
			return ImportanceLow
		}
		// Medium: Regular assistant responses.
		return ImportanceMedium
	}

	// Default: Low for any unknown content.
	return ImportanceLow
}

// isErrorMessage checks if a message contains error content.
func (c *Classifier) isErrorMessage(msg message.Message) bool {
	content := strings.ToLower(msg.Content)

	// Check for common error indicators.
	errorIndicators := []string{
		"error:",
		"error ",
		"failed:",
		"failed ",
		"exception:",
		"panic:",
		"fatal:",
		"cannot ",
		"could not ",
		"unable to ",
	}

	for _, indicator := range errorIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}

	// Check metadata for error flag (Metadata is map[string]string).
	if msg.Metadata != nil {
		if isError, ok := msg.Metadata["is_error"]; ok && isError == "true" {
			return true
		}
	}

	return false
}

// hasCodeBlock checks if a message contains code blocks.
func (c *Classifier) hasCodeBlock(msg message.Message) bool {
	// Check for fenced code blocks.
	if strings.Contains(msg.Content, "```") {
		return true
	}

	// Check for diff markers.
	if strings.Contains(msg.Content, "@@") {
		return true
	}

	// Check for indented code (4+ spaces at line start).
	lines := strings.Split(msg.Content, "\n")
	codeLineCount := 0

	for _, line := range lines {
		if len(line) >= 4 && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
			codeLineCount++
			if codeLineCount >= minCodeBlockLines {
				return true
			}
		}
	}

	return false
}

// ClassifiedMessage pairs a message with its importance level.
type ClassifiedMessage struct {
	Message    message.Message
	Importance Importance
	Tokens     int
	Index      int // Original index for preserving chronological order.
}
