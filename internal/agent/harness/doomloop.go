package harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
)

const (
	// DefaultWindowSize is the sliding window size for fingerprint tracking.
	DefaultWindowSize = 20

	// DefaultThreshold is the minimum fingerprint recurrence count to trigger halt.
	DefaultThreshold = 3
)

// DoomLoopGuard detects repetitive tool call patterns using fingerprinting.
// It maintains a sliding window of recent tool call fingerprints and halts
// execution when any fingerprint recurs at or above the configured threshold.
//
// To avoid false positives on build-fix-rebuild cycles, the guard tracks the
// last result for each fingerprint. If the result changes (e.g., different
// compilation errors), the counter resets — the agent is making progress.
type DoomLoopGuard struct {
	windowSize       int
	threshold        int
	window           []string
	lastFP           string
	consecutiveCount int
	emitter          *events.EventEmitter
}

// NewDoomLoopGuard creates a DoomLoopGuard with the given window size and threshold.
// Uses defaults if zero values are provided.
func NewDoomLoopGuard(windowSize, threshold int) *DoomLoopGuard {
	if windowSize <= 0 {
		windowSize = DefaultWindowSize
	}

	if threshold <= 0 {
		threshold = DefaultThreshold
	}

	return &DoomLoopGuard{
		windowSize: windowSize,
		threshold:  threshold,
		window:     make([]string, 0, windowSize),
	}
}

// SetEmitter configures the event emitter for doom-loop detection events.
func (d *DoomLoopGuard) SetEmitter(em *events.EventEmitter) {
	d.emitter = em
}

// Check inspects tool calls for doom-loop patterns.
// Returns injected warning messages and halt=true if threshold is exceeded.
//
// A tool call is only counted as repeated if no OTHER tool calls occurred
// between repetitions. This prevents false positives on build-fix-rebuild
// cycles (build → edit → build → edit → build), which are the most common
// agent workflow.
func (d *DoomLoopGuard) Check(
	_ context.Context, iterCtx *IterationContext,
	_ string, toolCalls []message.ToolCall,
) ([]message.Message, bool, error) {
	if len(toolCalls) == 0 {
		return nil, false, nil
	}

	for _, tc := range toolCalls {
		fp := fingerprint(tc)

		// If a different tool was called since last recording, clear consecutive count.
		if d.lastFP != "" && d.lastFP != fp {
			d.consecutiveCount = 0
		}

		if fp == d.lastFP {
			d.consecutiveCount++
		} else {
			d.consecutiveCount = 1
		}

		d.lastFP = fp
		d.record(fp)

		if d.consecutiveCount >= d.threshold {
			d.emitDetected(iterCtx.Turn, fp, d.consecutiveCount, tc.Function.Name)

			warning := fmt.Sprintf(
				"[SYSTEM WARNING] Doom-loop detected: tool %q called %d times consecutively with same arguments. Halting.",
				tc.Function.Name, d.consecutiveCount,
			)

			return []message.Message{{
				Role:    message.RoleSystem,
				Content: warning,
			}}, true, nil
		}
	}

	return nil, false, nil
}

// record adds a fingerprint to the sliding window, evicting the oldest if full.
func (d *DoomLoopGuard) record(fp string) {
	if len(d.window) >= d.windowSize {
		d.window = d.window[1:]
	}

	d.window = append(d.window, fp)
}

// emitDetected emits a doom-loop detection event if an emitter is configured.
func (d *DoomLoopGuard) emitDetected(
	turn int, fp string, count int, toolName string,
) {
	if d.emitter == nil {
		return
	}

	d.emitter.Emit(events.Event{
		Type: events.EventDoomLoopDetected,
		Data: events.DoomLoopDetectedData{
			Turn:        turn,
			Fingerprint: fp,
			Count:       count,
			ToolName:    toolName,
		},
	})
}

// fingerprint computes a SHA-256 hash of the tool call name and arguments.
func fingerprint(tc message.ToolCall) string {
	hash := sha256.Sum256([]byte(tc.Function.Name + "|" + tc.Function.Arguments))

	return hex.EncodeToString(hash[:])
}
