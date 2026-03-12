package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// AutoOffloader automatically offloads context when approaching limits.
type AutoOffloader struct {
	scratchpad *Scratchpad
	persistent *PersistentStore
	analyzer   ContextAnalyzer
	threshold  float64 // Token usage threshold (e.g., 0.7 = 70%).
	mu         sync.Mutex
}

// AutoOffloaderConfig configures the auto-offloader.
type AutoOffloaderConfig struct {
	// Scratchpad is the session-scoped memory store.
	Scratchpad *Scratchpad

	// Persistent is the cross-session memory store.
	Persistent *PersistentStore

	// Analyzer identifies offloadable content.
	Analyzer ContextAnalyzer

	// Threshold is the token usage percentage that triggers offloading (0.0-1.0).
	Threshold float64
}

// NewAutoOffloader creates a new auto-offloader.
func NewAutoOffloader(cfg AutoOffloaderConfig) *AutoOffloader {
	analyzer := cfg.Analyzer
	if analyzer == nil {
		analyzer = NewDefaultContextAnalyzer()
	}

	threshold := cfg.Threshold
	if threshold <= 0 || threshold > 1.0 {
		threshold = 0.7 // Default: 70%.
	}

	return &AutoOffloader{
		scratchpad: cfg.Scratchpad,
		persistent: cfg.Persistent,
		analyzer:   analyzer,
		threshold:  threshold,
	}
}

// ShouldOffload determines if offloading should occur based on token usage.
func (o *AutoOffloader) ShouldOffload(currentTokens, maxTokens int) bool {
	if maxTokens <= 0 {
		return false
	}

	return float64(currentTokens)/float64(maxTokens) > o.threshold
}

// Offload analyzes messages and offloads content to appropriate stores.
// Returns the modified messages with offloaded content replaced by references.
func (o *AutoOffloader) Offload(ctx context.Context, messages []AnalyzableMessage) ([]AnalyzableMessage, []OffloadResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Analyze messages for offloadable content.
	candidates := o.analyzer.Analyze(messages)
	if len(candidates) == 0 {
		return messages, nil, nil
	}

	// Sort by priority (higher priority first).
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})

	// Track offloaded content.
	results := make([]OffloadResult, 0, len(candidates))
	offloadedIndices := make(map[int][]OffloadCandidate)

	for _, candidate := range candidates {
		// Store to appropriate destination.
		var err error

		switch candidate.Destination {
		case ScopeSession:
			if o.scratchpad != nil {
				err = o.scratchpad.Put(ctx, candidate.Key, candidate.Content, PutOptions{
					Overwrite: true,
				})
			}
		case ScopePersistent:
			if o.persistent != nil {
				err = o.persistent.Put(ctx, candidate.Key, candidate.Content, PutOptions{
					Overwrite: true,
					Namespace: "offloaded",
				})
			}
		}

		if err != nil {
			// Log but continue with other candidates.
			results = append(results, OffloadResult{
				Key:     candidate.Key,
				Success: false,
				Error:   err.Error(),
			})

			continue
		}

		results = append(results, OffloadResult{
			Key:         candidate.Key,
			Success:     true,
			Destination: candidate.Destination,
			Reason:      candidate.Reason,
			TokensSaved: estimateTokens(candidate.Content),
		})

		offloadedIndices[candidate.MessageIndex] = append(
			offloadedIndices[candidate.MessageIndex],
			candidate,
		)
	}

	// Replace offloaded content with references.
	modified := make([]AnalyzableMessage, len(messages))
	copy(modified, messages)

	for idx, candidates := range offloadedIndices {
		msg := modified[idx]

		for _, candidate := range candidates {
			reference := fmt.Sprintf("[Content offloaded to %s as '%s': %s]",
				candidate.Destination, candidate.Key, candidate.Reason)
			// For simple implementation, append reference
			// A more sophisticated version would replace the exact content.
			msg.Content = msg.Content + "\n" + reference
		}

		modified[idx] = msg
	}

	return modified, results, nil
}

// OffloadResult describes the outcome of an offload operation.
type OffloadResult struct {
	// Key is the storage key used.
	Key string

	// Success indicates if the offload succeeded.
	Success bool

	// Error contains any error message.
	Error string

	// Destination is where the content was stored.
	Destination Scope

	// Reason explains why this was offloaded.
	Reason string

	// TokensSaved is the estimated token reduction.
	TokensSaved int
}

// Recall retrieves previously offloaded content by key.
func (o *AutoOffloader) Recall(ctx context.Context, key string) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Try scratchpad first.
	if o.scratchpad != nil {
		entry, err := o.scratchpad.Get(ctx, key)
		if err == nil {
			return entry.Value, nil
		}
	}

	// Try persistent store.
	if o.persistent != nil {
		entry, err := o.persistent.Get(ctx, key)
		if err == nil {
			return entry.Value, nil
		}
	}

	return "", ErrNotFound
}

// GetThreshold returns the current offload threshold.
func (o *AutoOffloader) GetThreshold() float64 {
	return o.threshold
}

// SetThreshold updates the offload threshold.
func (o *AutoOffloader) SetThreshold(threshold float64) {
	if threshold > 0 && threshold <= 1.0 {
		o.threshold = threshold
	}
}
