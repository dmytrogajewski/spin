// Package adapter provides adapters between ACE components and external interfaces.
package adapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/curator"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

const (
	defaultDecayRate       = 0.5
	highRelevanceThreshold = 0.85
	defaultMaxRetainCount  = 3
)

var (
	// ErrSessionNotFound is a sentinel error.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionNotFound2 is a sentinel error.
	ErrSessionNotFound2 = errors.New("session not found")
	// ErrSessionNotFound3 is a sentinel error.
	ErrSessionNotFound3 = errors.New("session not found")
)

// Adapter handles online context adaptation.
type Adapter interface {
	// StartSession begins a new online learning session.
	StartSession(ctx context.Context) (string, error)

	// AdaptOnline processes a signal and updates context in real-time.
	AdaptOnline(ctx context.Context, signal ExecutionSignal) (*AdaptationResult, error)

	// EndSession finalizes session and persists state.
	EndSession(ctx context.Context, sessionID string) error

	// GetSession retrieves current session state.
	GetSession(sessionID string) (*Session, error)
}

// adapter implements Adapter interface.
type adapter struct {
	playbook  *playbook.Playbook
	reflector reflector.Reflector
	curator   curator.Curator
	generator generator.Generator
	memory    *MemoryManager
	logger    *slog.Logger

	sessions map[string]*Session
	mu       sync.RWMutex
}

// Config configures an adapter.
type Config struct {
	Playbook     *playbook.Playbook
	Reflector    reflector.Reflector
	Curator      curator.Curator
	Generator    generator.Generator
	MemoryConfig MemoryConfig
}

// NewAdapter creates a new adapter with default configuration.
func NewAdapter(pb *playbook.Playbook, refl reflector.Reflector, cur curator.Curator) Adapter {
	return NewAdapterWithConfig(Config{
		Playbook:     pb,
		Reflector:    refl,
		Curator:      cur,
		MemoryConfig: DefaultMemoryConfig(),
	})
}

// NewAdapterWithConfig creates a new adapter with custom configuration.
func NewAdapterWithConfig(cfg Config) Adapter {
	return &adapter{
		playbook:  cfg.Playbook,
		reflector: cfg.Reflector,
		curator:   cfg.Curator,
		generator: cfg.Generator,
		memory:    NewMemoryManager(cfg.MemoryConfig),
		logger:    slog.Default(),
		sessions:  make(map[string]*Session),
	}
}

// StartSession begins a new online learning session.
func (a *adapter) StartSession(_ context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sessionID := uuid.New().String()
	a.sessions[sessionID] = newSession(sessionID)

	return sessionID, nil
}

// newSession creates a new session with the given ID.
func newSession(id string) *Session {
	return &Session{
		ID:            id,
		StartTime:     time.Now(),
		SignalCount:   0,
		UpdateCount:   0,
		RecentSignals: []*ExecutionSignal{},
	}
}

// AdaptOnline processes a signal and updates context in real-time.
func (a *adapter) AdaptOnline(ctx context.Context, signal ExecutionSignal) (*AdaptationResult, error) {
	startTime := time.Now()

	a.logger.DebugContext(ctx, "Adapter processing signal",
		"session_id", signal.SessionID,
		"signal_type", signal.SignalType,
		"outcome", signal.Outcome)

	session, err := a.getAndUpdateSession(signal)
	if err != nil {
		return nil, err
	}

	action, reason := decideAction(signal)
	a.logger.DebugContext(ctx, "Adapter decided action", "action", action, "reason", reason)

	bulletsAdded, err := a.executeAction(ctx, action, signal, session)
	if err != nil {
		return nil, err
	}

	refinementTriggered, reason := a.maybeRefine(ctx, reason)

	return &AdaptationResult{
		Action:              action,
		BulletsAdded:        bulletsAdded,
		BulletsUpdated:      0,
		LatencyMs:           time.Since(startTime).Milliseconds(),
		Reason:              reason,
		RefinementTriggered: refinementTriggered,
	}, nil
}

// getAndUpdateSession retrieves session and adds signal to it.
func (a *adapter) getAndUpdateSession(signal ExecutionSignal) (*Session, error) {
	a.mu.RLock()
	session, exists := a.sessions[signal.SessionID]
	a.mu.RUnlock()

	if !exists {
		a.logger.Warn("Adapter: session not found", "session_id", signal.SessionID)

		return nil, fmt.Errorf("session not found: %s: %w", signal.SessionID, ErrSessionNotFound)
	}

	session.AddSignal(&signal)
	a.logger.Debug("Signal added to session",
		"session_id", signal.SessionID,
		"total_signals", session.SignalCount)

	return session, nil
}

// executeAction dispatches to the appropriate action handler.
func (a *adapter) executeAction(ctx context.Context, action AdaptationAction, signal ExecutionSignal, session *Session) (int, error) {
	added, err := a.dispatchAction(ctx, action, signal)
	if err != nil {
		return 0, err
	}

	session.UpdateCount += added

	return added, nil
}

// dispatchAction routes to the correct handler based on action type.
func (a *adapter) dispatchAction(ctx context.Context, action AdaptationAction, signal ExecutionSignal) (int, error) {
	switch action {
	case ActionReflect:
		return a.executeReflect(ctx, signal)
	case ActionQuickAdd:
		return a.executeQuickAdd(ctx, signal)
	default:
		return 0, nil
	}
}

// maybeRefine checks if memory management should trigger and performs it.
func (a *adapter) maybeRefine(ctx context.Context, reason string) (refined bool, detail string) {
	if !a.memory.ShouldRefine(a.playbook.Stats().TotalBullets) {
		return false, reason
	}

	pruned, err := a.memory.Prune(ctx, a.playbook)
	if err != nil {
		a.logger.ErrorContext(ctx, "Memory refinement failed", "error", err)

		return false, reason
	}

	return true, fmt.Sprintf("%s (pruned %d bullets)", reason, pruned)
}

// executeReflect performs full reflection cycle.
func (a *adapter) executeReflect(ctx context.Context, signal ExecutionSignal) (int, error) {
	traj := buildTrajectory(signal)

	insights, err := a.extractInsights(ctx, traj)
	if err != nil {
		return 0, err
	}

	if len(insights) == 0 {
		return 0, nil
	}

	return a.curateInsights(ctx, insights)
}

// extractInsights calls the reflector to get insights from a trajectory.
func (a *adapter) extractInsights(ctx context.Context, traj *generator.Trajectory) ([]*reflector.Insight, error) {
	resp, err := a.reflector.Reflect(ctx, reflector.ReflectionRequest{
		Trajectories:  []*generator.Trajectory{traj},
		MaxIterations: 1,
		MinConfidence: defaultDecayRate,
	})
	if err != nil {
		return nil, fmt.Errorf("reflection failed: %w", err)
	}

	return resp.Insights, nil
}

// curateInsights processes insights through the curator.
func (a *adapter) curateInsights(ctx context.Context, insights []*reflector.Insight) (int, error) {
	resp, err := a.curator.Curate(ctx, curator.MergeRequest{
		Insights:            insights,
		SimilarityThreshold: highRelevanceThreshold,
	})
	if err != nil {
		return 0, fmt.Errorf("curation failed: %w", err)
	}

	return resp.Added, nil
}

// buildTrajectory creates a trajectory from an execution signal.
func buildTrajectory(signal ExecutionSignal) *generator.Trajectory {
	return &generator.Trajectory{
		ID:    uuid.New().String(),
		Query: signal.Context,
		Steps: []generator.TrajectoryStep{
			{
				StepNumber: 0,
				Type:       "execution",
				Content:    signal.Context,
				Timestamp:  signal.Timestamp,
			},
		},
		Output:    signal.Context,
		Success:   signal.Outcome == OutcomeSuccess,
		CreatedAt: signal.Timestamp,
		Metadata: generator.TrajectoryMetadata{
			Duration: 0,
			Turns:    1,
		},
	}
}

// executeQuickAdd performs quick bullet generation.
func (a *adapter) executeQuickAdd(ctx context.Context, signal ExecutionSignal) (int, error) {
	if a.generator == nil {
		return 0, nil
	}

	bullets, err := a.generator.GenerateBullets(ctx, generator.BulletGenerationRequest{
		Input:      signal.Context,
		SourceType: mapSignalTypeToSource(signal.SignalType),
		MaxBullets: defaultMaxRetainCount,
		Tags: map[string]string{
			"signal_type": string(signal.SignalType),
			"outcome":     string(signal.Outcome),
		},
	})
	if err != nil {
		return 0, fmt.Errorf("bullet generation failed: %w", err)
	}

	return a.addBulletsToPlaybook(ctx, bullets)
}

// mapSignalTypeToSource maps signal type to generator source type.
func mapSignalTypeToSource(st SignalType) string {
	switch st {
	case SignalTypeBuild, SignalTypeLint, SignalTypeError:
		return "error"
	default:
		return "error"
	}
}

// addBulletsToPlaybook adds bullets to the playbook and returns count added.
func (a *adapter) addBulletsToPlaybook(ctx context.Context, bullets []*bullet.Bullet) (int, error) {
	for _, b := range bullets {
		err := a.playbook.Add(ctx, b)
		if err != nil {
			return 0, fmt.Errorf("failed to add bullet: %w", err)
		}
	}

	return len(bullets), nil
}

// EndSession finalizes session and persists state.
func (a *adapter) EndSession(_ context.Context, sessionID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found: %s: %w", sessionID, ErrSessionNotFound2)
	}

	delete(a.sessions, sessionID)

	return nil
}

// GetSession retrieves current session state.
func (a *adapter) GetSession(sessionID string) (*Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	session, exists := a.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s: %w", sessionID, ErrSessionNotFound3)
	}

	return session, nil
}
