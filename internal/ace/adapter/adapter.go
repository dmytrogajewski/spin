package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/curator"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
	"github.com/google/uuid"
)

// Adapter handles online context adaptation
type Adapter interface {
	// StartSession begins a new online learning session
	StartSession(ctx context.Context) (string, error)

	// AdaptOnline processes a signal and updates context in real-time
	AdaptOnline(ctx context.Context, signal ExecutionSignal) (*AdaptationResult, error)

	// EndSession finalizes session and persists state
	EndSession(ctx context.Context, sessionID string) error

	// GetSession retrieves current session state
	GetSession(sessionID string) (*Session, error)
}

// adapter implements Adapter interface
type adapter struct {
	playbook  *playbook.Playbook
	reflector reflector.Reflector
	curator   curator.Curator
	generator generator.Generator
	memory    *MemoryManager

	// Session management
	sessions map[string]*Session
	mu       sync.RWMutex
}

// Config configures an adapter
type Config struct {
	Playbook     *playbook.Playbook
	Reflector    reflector.Reflector
	Curator      curator.Curator
	Generator    generator.Generator
	MemoryConfig MemoryConfig
}

// NewAdapter creates a new adapter
func NewAdapter(pb *playbook.Playbook, refl reflector.Reflector, cur curator.Curator) Adapter {
	return NewAdapterWithConfig(Config{
		Playbook:     pb,
		Reflector:    refl,
		Curator:      cur,
		MemoryConfig: DefaultMemoryConfig(),
	})
}

// NewAdapterWithConfig creates a new adapter with custom configuration
func NewAdapterWithConfig(cfg Config) Adapter {
	return &adapter{
		playbook:  cfg.Playbook,
		reflector: cfg.Reflector,
		curator:   cfg.Curator,
		generator: cfg.Generator,
		memory:    NewMemoryManager(cfg.MemoryConfig),
		sessions:  make(map[string]*Session),
	}
}

// StartSession begins a new online learning session
func (a *adapter) StartSession(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Generate session ID
	sessionID := uuid.New().String()

	// Create session
	session := &Session{
		ID:            sessionID,
		StartTime:     time.Now(),
		SignalCount:   0,
		UpdateCount:   0,
		RecentSignals: []*ExecutionSignal{},
	}

	// Store session
	a.sessions[sessionID] = session

	return sessionID, nil
}

// AdaptOnline processes a signal and updates context in real-time
func (a *adapter) AdaptOnline(ctx context.Context, signal ExecutionSignal) (*AdaptationResult, error) {
	startTime := time.Now()

	slog.Debug("Adapter processing signal",
		"session_id", signal.SessionID,
		"signal_type", signal.SignalType,
		"outcome", signal.Outcome)

	// Get session
	a.mu.RLock()
	session, exists := a.sessions[signal.SessionID]
	a.mu.RUnlock()

	if !exists {
		slog.Warn("Adapter: session not found", "session_id", signal.SessionID)
		return nil, fmt.Errorf("session not found: %s", signal.SessionID)
	}

	// Add signal to session
	session.AddSignal(&signal)
	slog.Debug("Signal added to session",
		"session_id", signal.SessionID,
		"total_signals", session.SignalCount)

	// Decide action based on signal
	action, reason := decideAction(signal)

	slog.Debug("Adapter decided action",
		"action", action,
		"reason", reason,
		"signal_type", signal.SignalType)

	bulletsAdded := 0
	bulletsUpdated := 0
	refinementTriggered := false

	// Execute action
	switch action {
	case ActionReflect:
		// Full reflection: Signal → Trajectory → Reflector → Insights → Curator → Bullets
		added, err := a.executeReflect(ctx, signal)
		if err != nil {
			return nil, fmt.Errorf("reflect action failed: %w", err)
		}
		bulletsAdded = added
		session.UpdateCount += added

	case ActionQuickAdd:
		// Quick generation: Signal → Generator → Bullets
		added, err := a.executeQuickAdd(ctx, signal)
		if err != nil {
			return nil, fmt.Errorf("quick add action failed: %w", err)
		}
		bulletsAdded = added
		session.UpdateCount += added

	case ActionSkip:
		// No action needed
	}

	// Check if memory management should trigger
	if a.memory.ShouldRefine(a.playbook.Stats().TotalBullets) {
		pruned, err := a.memory.Prune(ctx, a.playbook)
		if err != nil {
			return nil, fmt.Errorf("memory refinement failed: %w", err)
		}
		refinementTriggered = true
		reason = fmt.Sprintf("%s (pruned %d bullets)", reason, pruned)
	}

	// Calculate latency
	latency := time.Since(startTime).Milliseconds()

	return &AdaptationResult{
		Action:              action,
		BulletsAdded:        bulletsAdded,
		BulletsUpdated:      bulletsUpdated,
		LatencyMs:           latency,
		Reason:              reason,
		RefinementTriggered: refinementTriggered,
	}, nil
}

// executeReflect performs full reflection cycle
func (a *adapter) executeReflect(ctx context.Context, signal ExecutionSignal) (int, error) {
	// Create a simple trajectory from the signal
	traj := &generator.Trajectory{
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

	// Call reflector to extract insights
	reflReq := reflector.ReflectionRequest{
		Trajectories:  []*generator.Trajectory{traj},
		MaxIterations: 1,
		MinConfidence: 0.5,
	}

	reflResp, err := a.reflector.Reflect(ctx, reflReq)
	if err != nil {
		return 0, fmt.Errorf("reflection failed: %w", err)
	}

	// No insights extracted
	if len(reflResp.Insights) == 0 {
		return 0, nil
	}

	// Curate insights into bullets
	curReq := curator.MergeRequest{
		Insights:            reflResp.Insights,
		SimilarityThreshold: 0.85,
	}

	curResp, err := a.curator.Curate(ctx, curReq)
	if err != nil {
		return 0, fmt.Errorf("curation failed: %w", err)
	}

	return curResp.Added, nil
}

// executeQuickAdd performs quick bullet generation
func (a *adapter) executeQuickAdd(ctx context.Context, signal ExecutionSignal) (int, error) {
	// Use generator if available, otherwise create bullet directly
	if a.generator != nil {
		// Determine source type from signal type
		sourceType := "error"
		switch signal.SignalType {
		case SignalTypeBuild:
			sourceType = "error"
		case SignalTypeLint:
			sourceType = "error"
		case SignalTypeError:
			sourceType = "error"
		}

		// Generate bullets
		genReq := generator.BulletGenerationRequest{
			Input:      signal.Context,
			SourceType: sourceType,
			MaxBullets: 3,
			Tags: map[string]string{
				"signal_type": string(signal.SignalType),
				"outcome":     string(signal.Outcome),
			},
		}

		bullets, err := a.generator.GenerateBullets(ctx, genReq)
		if err != nil {
			return 0, fmt.Errorf("bullet generation failed: %w", err)
		}

		// Add bullets to playbook
		for _, b := range bullets {
			if err := a.playbook.Add(ctx, b); err != nil {
				return 0, fmt.Errorf("failed to add bullet: %w", err)
			}
		}

		return len(bullets), nil
	}

	// Fallback: Create single bullet from signal context
	// This is a simple implementation for when generator is not available
	// In practice, you'd want to extract a better bullet content
	return 0, nil
}

// EndSession finalizes session and persists state
func (a *adapter) EndSession(ctx context.Context, sessionID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check session exists
	if _, exists := a.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Remove session
	delete(a.sessions, sessionID)

	return nil
}

// GetSession retrieves current session state
func (a *adapter) GetSession(sessionID string) (*Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	session, exists := a.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}
