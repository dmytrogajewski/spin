// Package compactor implements staged context compaction for the harness loop.
// It applies progressively more aggressive compaction strategies based on
// context pressure thresholds derived from the OpenDev paper.
package compactor

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

// Stage represents the compaction stage applied.
type Stage int

const (
	// StageNone indicates no compaction was needed (pressure < warning threshold).
	StageNone Stage = iota

	// StageWarning indicates pressure is elevated but no modification is made (>= 70%).
	StageWarning

	// StageObservationMask replaces old tool outputs with compact references (>= 80%).
	StageObservationMask

	// StageFastPrune walks backward and replaces old results with pruned markers (>= 85%).
	StageFastPrune
)

// Default thresholds as fractions of max context.
const (
	DefaultWarningThreshold     = 0.70
	DefaultObservationThreshold = 0.80
	DefaultPruneThreshold       = 0.85
)

// Default configuration values.
const (
	defaultRecentProtected = 6
	overheadPerMessage     = 4
	prunedMarker           = "[pruned]"
	observationMaskPrefix  = "[observation: "
	observationMaskSuffix  = "]"
)

// String returns the human-readable name of the stage.
func (s Stage) String() string {
	switch s {
	case StageNone:
		return "none"
	case StageWarning:
		return "warning"
	case StageObservationMask:
		return "observation_mask"
	case StageFastPrune:
		return "fast_prune"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// Compactor applies staged context compaction based on pressure thresholds.
type Compactor struct {
	tokenizer        tokenizer.Tokenizer
	maxContext       int
	warningThreshold float64
	observeThreshold float64
	pruneThreshold   float64
	recentProtected  int
}

// NewCompactor creates a Compactor with the given tokenizer and max context window.
func NewCompactor(tok tokenizer.Tokenizer, maxContext int, opts ...Option) *Compactor {
	comp := &Compactor{
		tokenizer:        tok,
		maxContext:       maxContext,
		warningThreshold: DefaultWarningThreshold,
		observeThreshold: DefaultObservationThreshold,
		pruneThreshold:   DefaultPruneThreshold,
		recentProtected:  defaultRecentProtected,
	}

	for _, opt := range opts {
		opt(comp)
	}

	return comp
}

// Option configures a Compactor.
type Option func(*Compactor)

// WithThresholds overrides the default compaction thresholds.
func WithThresholds(warning, observe, prune float64) Option {
	return func(c *Compactor) {
		c.warningThreshold = warning
		c.observeThreshold = observe
		c.pruneThreshold = prune
	}
}

// WithRecentProtected sets how many recent messages are protected from pruning.
func WithRecentProtected(n int) Option {
	return func(c *Compactor) {
		c.recentProtected = n
	}
}

// Pressure returns the current context pressure as a ratio (0.0 to 1.0+).
func (c *Compactor) Pressure(messages []message.Message) float64 {
	if c.maxContext == 0 {
		return 0
	}

	tokens := c.countTokens(messages)

	return float64(tokens) / float64(c.maxContext)
}

// Compact applies the minimum needed compaction stage and returns the result.
func (c *Compactor) Compact(
	_ context.Context, messages []message.Message,
) ([]message.Message, Stage, error) {
	pressure := c.Pressure(messages)

	if pressure < c.warningThreshold {
		return messages, StageNone, nil
	}

	if pressure < c.observeThreshold {
		return messages, StageWarning, nil
	}

	if pressure < c.pruneThreshold {
		return c.applyObservationMask(messages), StageObservationMask, nil
	}

	return c.applyFastPrune(messages), StageFastPrune, nil
}

// countTokens estimates the total token count for all messages.
func (c *Compactor) countTokens(messages []message.Message) int {
	total := 0

	for _, msg := range messages {
		total += c.tokenizer.Count(msg.Content) + overheadPerMessage
	}

	return total
}

// applyObservationMask replaces old tool outputs with compact references.
// Recent messages within the protection window are preserved.
func (c *Compactor) applyObservationMask(messages []message.Message) []message.Message {
	result := make([]message.Message, len(messages))
	copy(result, messages)

	protectFrom := max(len(result)-c.recentProtected, 0)

	for idx := range result {
		if idx >= protectFrom {
			break
		}

		if result[idx].Role == message.RoleTool && result[idx].Content != "" {
			tokenCount := c.tokenizer.Count(result[idx].Content)
			result[idx].Content = fmt.Sprintf(
				"%s%d tokens%s",
				observationMaskPrefix, tokenCount, observationMaskSuffix,
			)
		}
	}

	return result
}

// applyFastPrune walks backward and replaces old tool results with pruned markers.
// Recent messages within the protection window are preserved.
func (c *Compactor) applyFastPrune(messages []message.Message) []message.Message {
	result := make([]message.Message, len(messages))
	copy(result, messages)

	protectFrom := max(len(result)-c.recentProtected, 0)

	for idx := range result {
		if idx >= protectFrom {
			break
		}

		if result[idx].Role == message.RoleTool {
			result[idx].Content = prunedMarker
		}
	}

	return result
}
