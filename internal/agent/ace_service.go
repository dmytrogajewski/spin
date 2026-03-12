package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/adapter"
	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/curator"
	"github.com/dmytrogajewski/spin/internal/ace/delta"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/feedback"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/prompt"
	"github.com/dmytrogajewski/spin/internal/ace/refine"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
	"github.com/dmytrogajewski/spin/internal/ace/retrieval"
	"github.com/dmytrogajewski/spin/internal/llm"
)

var ErrConfigIsRequired = errors.New("config is required")

// ErrACEDisabled is returned when an ACE operation is called but ACE is disabled.
var ErrACEDisabled = errors.New("ACE is disabled")

// ACEService provides Agentic Context Engineering functionality to the Agent.
// It manages the playbook, retrieves relevant bullets, builds prompts, parses feedback,
// and updates bullet counters based on ItemizedLearning workflow.
// It now includes the complete ACE pipeline: Adapter, Reflector, Curator, Delta, and Refine.
type ACEService struct {
	config         *ACEConfig
	playbook       *playbook.Playbook
	retriever      retrieval.Retriever
	generator      generator.Generator   // Legacy quick generation.
	reflector      reflector.Reflector   // Deep analysis.
	curator        curator.Curator       // Quality control & deduplication.
	adapter        adapter.Adapter       // Online learning orchestration.
	deltaHistory   *delta.History   // Change tracking.
	growthMonitor  *refine.GrowthMonitor // Playbook growth management.
	feedbackParser *feedback.RegexParser
	embedder       embedding.Embedder
	logger         *slog.Logger
	workDir        string
	enabled        bool
	llm            llm.Provider
	modelName      string // LLM model name for generation.
}

// NewACEService creates a new ACE service with the given configuration.
// If ACE is disabled in config, returns a no-op service that returns empty results.
// The llm parameter is optional - if nil, bullet generation is disabled.
// The modelName parameter specifies which LLM model to use for generation.
// The maxTokens parameter sets the max tokens for LLM calls (0 = use default).
func NewACEService(cfg *ACEConfig, workDir string, llm llm.Provider, modelName string, maxTokens int) (*ACEService, error) {
	if cfg == nil {
		return nil, ErrConfigIsRequired
	}

	// Return no-op service if disabled.
	if !cfg.Enabled {
		return &ACEService{
			config:  cfg,
			enabled: false,
			logger:  slog.Default(),
		}, nil
	}

	logger := slog.Default()

	// Expand home directory in paths.
	playbookPath := expandPath(cfg.PlaybookPath)

	// Load or create playbook.
	var (
		pb  *playbook.Playbook
		err error
	)

	// Create embedder - try Ollama first, fall back to mock.
	var embedder embedding.Embedder

	ollamaConfig := embedding.DefaultOllamaEmbedderConfig()

	ollamaEmbedder, err := embedding.NewOllamaEmbedder(ollamaConfig)
	if err != nil {
		logger.Warn("Failed to create Ollama embedder, using mock embedder", "error", err)

		embedder = embedding.NewMockEmbedder(768) // Match nomic-embed-text dimension.
	} else {
		embedder = ollamaEmbedder

		logger.Info(" Using Ollama embedder", "model", ollamaConfig.Model, "dimension", ollamaConfig.Dimension)
	}

	// Check if playbook file exists.
	_, statErr := os.Stat(playbookPath)
	if statErr == nil {
		// Load existing playbook.
		pb, err = playbook.Load(playbookPath, nil, embedder)
		if err != nil {
			// If load fails, create new playbook.
			pb = playbook.New(nil, embedder)
		}
	} else {
		// Create new playbook with seed bullets.
		pb = playbook.New(nil, embedder)

		// Ensure directory exists.
		dir := filepath.Dir(playbookPath)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create playbook directory: %w", err)
		}

		// Seed with initial bullets for Go/coding best practices.
		err = seedInitialBullets(pb, embedder)
		if err != nil {
			// Log warning but continue - ACE can still learn from scratch.
			logger.Warn("Failed to seed initial bullets", "error", err)
		}

		// Save the seeded playbook.
		err = pb.Save(playbookPath)
		if err != nil {
			// Log warning but continue.
			logger.Warn("Failed to save initial playbook", "error", err)
		}
	}

	// Create components
	// Use HNSW retriever by default for better performance, but fall back to semantic retriever if needed.
	var retriever retrieval.Retriever
	if embedder != nil {
		retriever = retrieval.NewHNSWRetriever(pb, embedder)
	} else {
		// Fallback to semantic retriever when no embedder available.
		retriever = retrieval.NewSemanticRetriever(pb, embedder)
	}

	feedbackParser := feedback.NewRegexParser()

	// Create generator if LLM is provided and generation is enabled.
	var gen generator.Generator
	if llm != nil && cfg.Generation.Enabled {
		gen, err = generator.NewGenerator(generator.Config{
			LLM:       llm,
			Playbook:  pb,
			Retriever: retriever,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create generator: %w", err)
		}
	}

	// Create reflector for deep analysis (if LLM available).
	var refl reflector.Reflector

	if llm != nil {
		reflectorOpts := []reflector.Option{}
		if maxTokens > 0 {
			reflectorOpts = append(reflectorOpts, reflector.WithMaxTokens(maxTokens))
		}

		refl = reflector.NewReflector(llm, reflectorOpts...)

		logger.Debug("Created reflector for deep insight analysis", "max_tokens", maxTokens)
	}

	// Create curator for quality control and deduplication.
	var cur curator.Curator

	if embedder != nil {
		// Configure curator based on config.
		curatorOpts := []curator.Option{
			curator.WithSimilarityThreshold(0.85), // High threshold for deduplication.
		}

		// Set max tokens for LLM calls.
		if maxTokens > 0 {
			curatorOpts = append(curatorOpts, curator.WithMaxTokens(maxTokens))
		}

		// Enable LLM-based curation if LLM is available and AutoReflect is enabled
		// LLM-based curation provides intelligent filtering vs simple deduplication.
		if llm != nil && cfg.Generation.AutoReflect {
			curatorOpts = append(curatorOpts, curator.WithLLMProvider(llm))

			logger.Debug("Enabled LLM-based intelligent curation")
		}

		// Set refinement mode based on config.
		if cfg.Refine.Enabled {
			// Enable merge engine for semantic deduplication (similarity-based merging).
			curatorOpts = append(curatorOpts, curator.WithMergeEngine(0.90))

			logger.Debug("Enabled merge engine for advanced bullet deduplication")

			if cfg.Refine.Mode == "proactive" {
				curatorOpts = append(curatorOpts, curator.WithRefinementMode(
					curator.RefinementModeProactive,
					curator.ProactiveRefinementConfig{
						MaxBullets:      cfg.Refine.MaxBullets,
						MaxSizeBytes:    int64(cfg.Refine.MaxTokens),
						MinUtilityScore: cfg.Refine.MinUtilityScore,
					},
				))
			} else if cfg.Refine.Mode == "lazy" {
				curatorOpts = append(curatorOpts, curator.WithRefinementMode(
					curator.RefinementModeLazy,
					curator.LazyRefinementConfig{
						MinUtilityScore: cfg.Refine.MinUtilityScore,
					},
				))
			}
		} else {
			curatorOpts = append(curatorOpts, curator.WithRefinementMode(curator.RefinementModeNone, nil))
		}

		cur = curator.NewCurator(pb, embedder, curatorOpts...)

		logger.Debug("Created curator for quality control")
	}

	// Create adapter for online learning orchestration (if reflector and curator available).
	var adp adapter.Adapter

	if refl != nil && cur != nil {
		// Check if we need custom configuration.
		needsCustomConfig := cfg.Adapter.MaxMemorySize > 0 || cfg.Adapter.UtilityThreshold > 0 || gen != nil

		if needsCustomConfig {
			// Use default memory config as base, then override with user settings.
			memConfig := adapter.DefaultMemoryConfig()
			if cfg.Adapter.MaxMemorySize > 0 {
				memConfig.MaxBullets = cfg.Adapter.MaxMemorySize
				memConfig.RefinementAt = int(float64(cfg.Adapter.MaxMemorySize) * 0.9) // 90% of max.
			}

			if cfg.Adapter.UtilityThreshold > 0 {
				memConfig.PruneThreshold = cfg.Adapter.UtilityThreshold
			}

			adapterConfig := adapter.Config{
				Playbook:     pb,
				Reflector:    refl,
				Curator:      cur,
				Generator:    gen,
				MemoryConfig: memConfig,
			}
			adp = adapter.NewAdapterWithConfig(adapterConfig)
		} else {
			// Use simple constructor with defaults.
			adp = adapter.NewAdapter(pb, refl, cur)
		}

		logger.Info("Created adapter for online learning orchestration")
	}

	// Create delta history for tracking bullet changes.
	deltaHist := delta.NewHistory()

	logger.Debug("Created delta history for change tracking")

	// Create growth monitor for playbook management.
	var growthMon *refine.GrowthMonitor

	if cfg.Refine.Enabled {
		// Start with defaults and override with config values.
		thresholds := refine.DefaultGrowthThresholds()
		if cfg.Refine.MaxBullets > 0 {
			thresholds.MaxBullets = cfg.Refine.MaxBullets
		}

		if cfg.Refine.MaxTokens > 0 {
			thresholds.MaxTokens = cfg.Refine.MaxTokens
		}

		if cfg.Refine.MinUtilityScore > 0 {
			thresholds.MinUtility = cfg.Refine.MinUtilityScore
		}

		if cfg.Refine.CheckInterval > 0 {
			thresholds.CheckInterval = time.Duration(cfg.Refine.CheckInterval) * time.Second
		}

		growthMon = refine.NewGrowthMonitor(pb, thresholds)
		logger.Info("Created growth monitor", "max_bullets", thresholds.MaxBullets, "max_tokens", thresholds.MaxTokens)
	}

	return &ACEService{
		config:         cfg,
		playbook:       pb,
		retriever:      retriever,
		generator:      gen,
		reflector:      refl,
		curator:        cur,
		adapter:        adp,
		deltaHistory:   deltaHist,
		growthMonitor:  growthMon,
		feedbackParser: feedbackParser,
		embedder:       embedder,
		logger:         slog.Default(),
		workDir:        workDir,
		enabled:        true,
		llm:            llm,
		modelName:      modelName,
	}, nil
}

// Retrieve fetches top-K relevant bullets for the given query.
// Returns empty slice if ACE is disabled.
func (s *ACEService) Retrieve(ctx context.Context, query string) ([]*bullet.Bullet, error) {
	if !s.enabled {
		return nil, nil
	}

	// Use RetrieveWithScores to get scores for filtering.
	results, err := s.retriever.RetrieveWithScores(ctx, query, s.config.Retrieval.TopK)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// Filter by minimum score threshold.
	filtered := make([]*bullet.Bullet, 0, len(results))
	for _, result := range results {
		if result.Score >= s.config.Retrieval.MinScore {
			filtered = append(filtered, result.Bullet)
		}
	}

	return filtered, nil
}

// BuildPrompt constructs a system prompt with bullets.
// If ACE is disabled, returns the systemPrompt unchanged.
// If ItemizedLearning is enabled, includes feedback instructions.
func (s *ACEService) BuildPrompt(_ context.Context, systemPrompt string, bullets []*bullet.Bullet) (string, error) {
	if !s.enabled {
		return systemPrompt, nil
	}

	// Build prompt builder with options.
	opts := []prompt.Option{
		prompt.WithSystemPrompt(systemPrompt),
	}

	// Include ItemizedLearning instructions if enabled.
	if s.config.ItemizedLearning.Enabled && len(bullets) > 0 {
		opts = append(opts, prompt.WithItemizedLearning())
	}

	builder := prompt.NewBuilder(opts...)

	return builder.BuildSystemPrompt(bullets), nil
}

// ParseFeedback extracts HELPFUL/HARMFUL markers from LLM response.
// Returns ErrACEDisabled if ACE is disabled.
func (s *ACEService) ParseFeedback(response string) (*feedback.BulletFeedback, error) {
	if !s.enabled {
		return nil, ErrACEDisabled
	}

	return s.feedbackParser.Parse(response)
}

// UpdateBullets increments bullet counters based on feedback markers.
// The bullets slice corresponds to the bullets used in the prompt (B0, B1, B2, ...).
// Feedback markers like "B0", "B2" are indices into this slice.
// If config.ItemizedLearning.UpdateAsync is true, saves playbook asynchronously.
func (s *ACEService) UpdateBullets(ctx context.Context, bullets []*bullet.Bullet, fb *feedback.BulletFeedback) error {
	if !s.enabled || fb == nil {
		return nil
	}

	// Build feedback map for batch delta application.
	feedbackMap := make(map[string]string)

	// Map bullet markers (B0, B1, ...) to actual bullet IDs.
	for _, marker := range fb.HelpfulBullets {
		idx := parseBulletIndex(marker)
		if idx >= 0 && idx < len(bullets) {
			b := bullets[idx]
			feedbackMap[b.ID] = "helpful"
		}
	}

	for _, marker := range fb.HarmfulBullets {
		idx := parseBulletIndex(marker)
		if idx >= 0 && idx < len(bullets) {
			b := bullets[idx]
			feedbackMap[b.ID] = "harmful"
		}
	}

	// Use curator's batch delta application for parallel updates.
	if s.curator != nil {
		err := s.curator.ApplyBulletFeedback(ctx, feedbackMap)
		if err != nil {
			return fmt.Errorf("failed to apply bullet feedback: %w", err)
		}
	}

	// Save playbook.
	if s.config.ItemizedLearning.UpdateAsync {
		go func() { _ = s.SavePlaybook() }()
	} else {
		err := s.SavePlaybook()
		if err != nil {
			return fmt.Errorf("failed to save playbook: %w", err)
		}
	}

	return nil
}

// SavePlaybook saves the playbook to disk.
// Returns nil if ACE is disabled.
func (s *ACEService) SavePlaybook() error {
	if !s.enabled {
		return nil
	}

	playbookPath := expandPath(s.config.PlaybookPath)

	return s.playbook.Save(playbookPath)
}

// RestoreBullet creates a bullet with a specific ID for restoration/migration scenarios.
// This is useful when importing bullets from backups or migrating from other systems.
// Returns ErrACEDisabled if ACE is disabled.
func (s *ACEService) RestoreBullet(ctx context.Context, id, content string) (*bullet.Bullet, error) {
	if !s.enabled {
		return nil, ErrACEDisabled
	}

	// Create bullet with custom ID.
	b, err := bullet.New(content, bullet.WithID(id))
	if err != nil {
		return nil, fmt.Errorf("failed to create bullet: %w", err)
	}

	// Get embedding if embedder available.
	if s.embedder != nil {
		var emb []float32
		emb, err = s.embedder.Embed(ctx, content)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}

		b.Embedding = emb
	}

	// Add to playbook.
	err = s.playbook.Add(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("failed to add bullet to playbook: %w", err)
	}

	return b, nil
}

// GenerateBullets creates new bullets from execution context and adds them to the playbook.
// Returns the successfully added bullets and an error if generation fails.
// Returns (nil, nil) if ACE is disabled or generation is not enabled.
func (s *ACEService) GenerateBullets(ctx context.Context, input string, sourceType string) ([]*bullet.Bullet, error) {
	if !s.enabled || s.generator == nil {
		s.logger.DebugContext(ctx, "GenerateBullets skipped", "enabled", s.enabled, "has_generator", s.generator != nil)

		return nil, nil
	}

	s.logger.InfoContext(ctx, "Generating bullets", "source_type", sourceType, "input_len", len(input))

	// Generate bullets using the generator.
	req := generator.BulletGenerationRequest{
		Input:      input,
		SourceType: sourceType,
		MaxBullets: 0,           // No limit - let the model decide.
		Model:      s.modelName, // Use same model as agent.
		Tags: map[string]string{
			"source":    sourceType,
			"generated": "true",
		},
	}

	bullets, err := s.generator.GenerateBullets(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate bullets: %w", err)
	}

	generatedCount := len(bullets)
	s.logger.InfoContext(ctx, "Generated bullets", "count", generatedCount)

	// Generate embeddings and add bullets to playbook.
	addedBullets := make([]*bullet.Bullet, 0, len(bullets))
	for i, b := range bullets {
		s.logger.DebugContext(ctx, "Processing bullet", "index", i+1, "content", b.Content)

		// Generate embedding for the bullet if embedder is available.
		if s.embedder != nil {
			var embVec []float32
			embVec, err = s.embedder.Embed(ctx, b.Content)
			if err != nil {
				s.logger.WarnContext(ctx, "Failed to generate embedding for bullet", "error", err, "content", b.Content)
				// Continue without embedding - bullet will still be added but won't be retrievable.
			} else {
				// Update bullet with embedding.
				b.Embedding = embVec
				s.logger.DebugContext(ctx, "Generated embedding", "index", i+1, "embedding_len", len(embVec))
			}
		}

		// Add bullet to playbook.
		err = s.playbook.Add(ctx, b)
		if err != nil {
			// Log warning but continue with other bullets.
			s.logger.WarnContext(ctx, "Failed to add bullet to playbook", "error", err)
		} else {
			addedBullets = append(addedBullets, b)
		}
	}

	// Count bullets in playbook.
	bulletCount := 0
	for range s.playbook.List(func(_ *bullet.Bullet) bool { return true }) {
		bulletCount++
	}

	s.logger.InfoContext(ctx, "Playbook updated", "total_bullets", bulletCount)

	// Save playbook with new bullets.
	err = s.SavePlaybook()
	if err != nil {
		return nil, fmt.Errorf("failed to save playbook: %w", err)
	}

	s.logger.DebugContext(ctx, "Playbook saved successfully")

	return addedBullets, nil
}

// GenerateBulletsWithReflectionFromTrajectory uses the Reflector+Curator pipeline with a full trajectory.
// This is the preferred method as it provides detailed execution trace for high-quality reflection.
func (s *ACEService) GenerateBulletsWithReflectionFromTrajectory(ctx context.Context, trajectory *generator.Trajectory) ([]*bullet.Bullet, error) {
	if !s.enabled || s.reflector == nil || s.curator == nil {
		s.logger.DebugContext(ctx, "Reflection pipeline not available", "enabled", s.enabled, "has_reflector", s.reflector != nil, "has_curator", s.curator != nil)

		return nil, nil
	}

	s.logger.InfoContext(ctx, "Using Reflector+Curator pipeline with full trajectory",
		"steps", len(trajectory.Steps),
		"success", trajectory.Success,
		"retrieved_bullets", len(trajectory.RetrievedBullets))

	// Debug: Log full trajectory details.
	s.logger.DebugContext(ctx, "Trajectory details",
		"query", trajectory.Query,
		"output", trajectory.Output,
		"success", trajectory.Success,
		"num_steps", len(trajectory.Steps))

	for i, step := range trajectory.Steps {
		s.logger.DebugContext(ctx, "Trajectory step",
			"index", i,
			"step_number", step.StepNumber,
			"type", step.Type,
			"content", step.Content)
	}

	// Reflect on trajectory to extract insights.
	reflectionReq := reflector.ReflectionRequest{
		Trajectories: []*generator.Trajectory{trajectory},
	}

	reflectionResp, err := s.reflector.Reflect(ctx, reflectionReq)
	if err != nil {
		s.logger.WarnContext(ctx, "Reflection failed", "error", err)

		return nil, err
	}

	s.logger.InfoContext(ctx, "Reflector extracted insights", "count", len(reflectionResp.Insights))

	// Debug: Log all extracted insights.
	for i, insight := range reflectionResp.Insights {
		s.logger.DebugContext(ctx, "Extracted insight",
			"index", i,
			"content", insight.Content,
			"category", insight.Category,
			"confidence", insight.Confidence,
			"evidence_count", len(insight.Evidence))
	}

	if len(reflectionResp.Insights) == 0 {
		s.logger.DebugContext(ctx, "No insights extracted from reflection")

		return nil, nil
	}

	// Curate insights into bullets with deduplication.
	mergeReq := curator.MergeRequest{
		Insights:            reflectionResp.Insights,
		SimilarityThreshold: 0.85,
	}

	s.logger.DebugContext(ctx, "Starting curation", "num_insights", len(mergeReq.Insights), "threshold", mergeReq.SimilarityThreshold)

	mergeResp, err := s.curator.Curate(ctx, mergeReq)
	if err != nil {
		s.logger.WarnContext(ctx, "Curation failed", "error", err)

		return nil, err
	}

	s.logger.InfoContext(ctx, "Curator processed insights",
		"added", mergeResp.Added,
		"updated", mergeResp.Updated,
		"duplicates", len(mergeResp.Duplicates))

	// Debug: Log added bullets.
	for i, b := range mergeResp.AddedBullets {
		s.logger.DebugContext(ctx, "Added bullet",
			"index", i,
			"id", b.ID,
			"content", b.Content,
			"helpful_count", b.HelpfulCount,
			"harmful_count", b.HarmfulCount)
	}

	// Debug: Log duplicates.
	if len(mergeResp.Duplicates) > 0 {
		s.logger.DebugContext(ctx, "Found duplicates", "duplicate_ids", mergeResp.Duplicates)
	}

	// Save playbook.
	err = s.SavePlaybook()
	if err != nil {
		return nil, fmt.Errorf("failed to save playbook: %w", err)
	}

	// Return the new bullets that were added.
	return mergeResp.AddedBullets, nil
}

// ProcessExecutionSignal handles online learning via the Adapter.
// This is called after each tool execution to adapt context in real-time.
func (s *ACEService) ProcessExecutionSignal(ctx context.Context, signal *adapter.ExecutionSignal) (*adapter.AdaptationResult, error) {
	if !s.enabled || s.adapter == nil || !s.config.Adapter.Enabled {
		return nil, ErrACEDisabled
	}

	// Debug: Log incoming signal.
	s.logger.DebugContext(ctx, "Processing execution signal",
		"signal_type", signal.SignalType,
		"outcome", signal.Outcome,
		"context", signal.Context,
		"session_id", signal.SessionID)

	// Process signal through adapter.
	result, err := s.adapter.AdaptOnline(ctx, *signal)
	if err != nil {
		s.logger.WarnContext(ctx, "Adapter failed to process signal", "error", err, "signal_type", signal.SignalType)

		return nil, err
	}

	s.logger.DebugContext(ctx, "Adapter processed signal",
		"action", result.Action,
		"bullets_added", result.BulletsAdded,
		"bullets_updated", result.BulletsUpdated,
		"latency_ms", result.LatencyMs)

	// Debug: Log adaptation details.
	if result.BulletsAdded > 0 {
		s.logger.DebugContext(ctx, "Adaptation created new bullets", "count", result.BulletsAdded)
	}

	// Track delta if bullets were modified.
	if result.BulletsAdded > 0 || result.BulletsUpdated > 0 {
		s.trackBulletChanges(ctx, result)
	}

	// Check growth and trigger refinement if needed.
	if s.growthMonitor != nil {
		s.checkGrowthAndRefine(ctx)
	}

	return result, nil
}

// trackBulletChanges records bullet modifications in delta history.
func (s *ACEService) trackBulletChanges(ctx context.Context, result *adapter.AdaptationResult) {
	if s.deltaHistory == nil {
		return
	}

	// Record adaptation event in delta history
	// This is a simplified version - in production you'd track specific bullet changes.
	s.logger.DebugContext(ctx, "Tracking bullet changes",
		"added", result.BulletsAdded,
		"updated", result.BulletsUpdated)
}

// checkGrowthAndRefine monitors playbook growth and triggers refinement if needed.
func (s *ACEService) checkGrowthAndRefine(ctx context.Context) {
	if s.growthMonitor == nil || s.curator == nil {
		return
	}

	// Check if refinement is needed.
	metrics, shouldRefine := s.growthMonitor.CheckGrowth(ctx)

	if shouldRefine {
		s.logger.InfoContext(ctx, "Growth threshold reached, triggering refinement",
			"bullet_count", metrics.BulletCount,
			"estimated_tokens", metrics.EstimatedTokens)

		// Trigger refinement asynchronously.
		go func() {
			bgCtx := context.Background()

			result, err := s.curator.Refine(bgCtx)
			if err != nil {
				s.logger.WarnContext(bgCtx, "Refinement failed", "error", err)

				return
			}

			s.logger.InfoContext(bgCtx, "Refinement completed",
				"pruned", result.Pruned,
				"reason", result.Reason)

			// Mark refinement in growth monitor.
			s.growthMonitor.MarkRefinement()

			// Save playbook after refinement.
			err = s.SavePlaybook()
			if err != nil {
				s.logger.WarnContext(bgCtx, "Failed to save playbook after refinement", "error", err)
			}
		}()
	}
}

// StartSession begins a new online learning session.
// Returns the session ID for tracking.
func (s *ACEService) StartSession(ctx context.Context) (string, error) {
	if !s.enabled || s.adapter == nil {
		return "", nil
	}

	sessionID, err := s.adapter.StartSession(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start adapter session: %w", err)
	}

	s.logger.InfoContext(ctx, "Started online learning session", "session_id", sessionID)

	return sessionID, nil
}

// EndSession finalizes an online learning session.
func (s *ACEService) EndSession(ctx context.Context, sessionID string) error {
	if !s.enabled || s.adapter == nil || sessionID == "" {
		return nil
	}

	err := s.adapter.EndSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to end adapter session: %w", err)
	}

	s.logger.InfoContext(ctx, "Ended online learning session", "session_id", sessionID)

	return nil
}

// parseBulletIndex parses "B0", "B1", etc. to integer index.
// Returns -1 if invalid format.
func parseBulletIndex(marker string) int {
	if len(marker) < 2 || marker[0] != 'B' {
		return -1
	}

	var idx int

	_, err := fmt.Sscanf(marker[1:], "%d", &idx)
	if err != nil {
		return -1
	}

	return idx
}

// expandPath expands ~ to home directory in file paths.
func expandPath(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if len(path) == 1 {
		return home
	}

	return filepath.Join(home, path[1:])
}

// seedInitialBullets seeds the playbook with initial Go/coding best practices.
// This provides a starting point for ACE to learn from.
func seedInitialBullets(pb *playbook.Playbook, embedder embedding.Embedder) error {
	ctx := context.Background()

	// Seed bullets covering common Go patterns and best practices.
	seeds := []struct {
		content  string
		category string
	}{}

	for _, seed := range seeds {
		// Compute embedding for the bullet content.
		emb, err := embedder.Embed(ctx, seed.content)
		if err != nil {
			return fmt.Errorf("failed to compute embedding: %w", err)
		}

		// Create bullet with embedding.
		b, err := bullet.New(seed.content, bullet.WithEmbedding(emb))
		if err != nil {
			return fmt.Errorf("failed to create seed bullet: %w", err)
		}

		// Initialize Tags map if needed.
		if b.Tags == nil {
			b.Tags = make(map[string]string)
		}

		b.Tags["category"] = seed.category
		b.Tags["source"] = "initial-seed"

		err = pb.Add(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to add seed bullet: %w", err)
		}
	}

	return nil
}
