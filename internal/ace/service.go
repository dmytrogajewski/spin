// Package ace provides Agentic Context Engineering functionality for managing
// playbooks, retrieving relevant bullets, and adapting context through online learning.
package ace

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

const (
	defaultEmbeddingDim      = 768
	highSimilarityThreshold  = 0.85
	veryHighSimilarityThresh = 0.90
	aceConfidenceThreshold   = 0.9
)

// ErrConfigIsRequired is a sentinel error.
var ErrConfigIsRequired = errors.New("config is required")

// ErrDisabled is returned when an ACE operation is called but ACE is disabled.
var ErrDisabled = errors.New("ACE is disabled")

// Service provides Agentic Context Engineering functionality to the Agent.
// It manages the playbook, retrieves relevant bullets, builds prompts, parses feedback,
// and updates bullet counters based on ItemizedLearning workflow.
// It now includes the complete ACE pipeline: Adapter, Reflector, Curator, Delta, and Refine.
type Service struct {
	config         *Config
	playbook       *playbook.Playbook
	retriever      retrieval.Retriever
	generator      generator.Generator   // Legacy quick generation.
	reflector      reflector.Reflector   // Deep analysis.
	curator        curator.Curator       // Quality control & deduplication.
	adapter        adapter.Adapter       // Online learning orchestration.
	deltaHistory   *delta.History        // Change tracking.
	growthMonitor  *refine.GrowthMonitor // Playbook growth management.
	feedbackParser *feedback.RegexParser
	embedder       embedding.Embedder
	logger         *slog.Logger
	workDir        string
	enabled        bool
	llm            llm.Provider
	modelName      string // LLM model name for generation.
}

// NewService creates a new ACE service with the given configuration.
// If ACE is disabled in config, returns a no-op service that returns empty results.
// The llm parameter is optional - if nil, bullet generation is disabled.
// The modelName parameter specifies which LLM model to use for generation.
// The maxTokens parameter sets the max tokens for LLM calls (0 = use default).
func NewService(
	ctx context.Context, cfg *Config, workDir string,
	llmProvider llm.Provider, modelName string, maxTokens int,
) (*Service, error) {
	if cfg == nil {
		return nil, ErrConfigIsRequired
	}

	// Return no-op service if disabled.
	if !cfg.Enabled {
		return &Service{
			config:  cfg,
			enabled: false,
			logger:  slog.Default(),
		}, nil
	}

	logger := slog.Default()
	playbookPath := expandPath(cfg.PlaybookPath)

	embedder := createEmbedder(logger)

	pb, err := loadOrCreatePlaybook(ctx, playbookPath, embedder, logger)
	if err != nil {
		return nil, err
	}

	retriever := createRetriever(pb, embedder)
	feedbackParser := feedback.NewRegexParser()

	gen, err := createGenerator(llmProvider, cfg, pb, retriever)
	if err != nil {
		return nil, err
	}

	refl := createReflector(llmProvider, maxTokens, logger)
	cur := createCurator(embedder, llmProvider, cfg, maxTokens, pb, logger)
	adp := createAdapter(refl, cur, gen, cfg, pb, logger)
	growthMon := createGrowthMonitor(cfg, pb, logger)

	return &Service{
		config:         cfg,
		playbook:       pb,
		retriever:      retriever,
		generator:      gen,
		reflector:      refl,
		curator:        cur,
		adapter:        adp,
		deltaHistory:   delta.NewHistory(),
		growthMonitor:  growthMon,
		feedbackParser: feedbackParser,
		embedder:       embedder,
		logger:         slog.Default(),
		workDir:        workDir,
		enabled:        true,
		llm:            llmProvider,
		modelName:      modelName,
	}, nil
}

// Config returns the ACE configuration.
func (s *Service) Config() *Config {
	return s.config
}

// Playbook returns the underlying playbook for direct access.
func (s *Service) Playbook() *playbook.Playbook {
	return s.playbook
}

// createEmbedder creates an embedder, trying Ollama first and falling back to mock.
func createEmbedder(logger *slog.Logger) embedding.Embedder {
	ollamaConfig := embedding.DefaultOllamaEmbedderConfig()

	ollamaEmbedder, err := embedding.NewOllamaEmbedder(ollamaConfig)
	if err != nil {
		logger.Warn("Failed to create Ollama embedder, using mock embedder", "error", err)

		return embedding.NewMockEmbedder(defaultEmbeddingDim)
	}

	logger.Info(" Using Ollama embedder", "model", ollamaConfig.Model, "dimension", ollamaConfig.Dimension)

	return ollamaEmbedder
}

// loadOrCreatePlaybook loads an existing playbook or creates a new one with seeds.
func loadOrCreatePlaybook(
	ctx context.Context, playbookPath string,
	embedder embedding.Embedder, logger *slog.Logger,
) (*playbook.Playbook, error) {
	_, statErr := os.Stat(playbookPath)
	if statErr == nil {
		pb, err := playbook.Load(playbookPath, nil, embedder)
		if err != nil {
			logger.Warn("Failed to load playbook, creating new one", "path", playbookPath, "error", err)

			return playbook.New(nil, embedder), nil
		}

		return pb, nil
	}

	pb := playbook.New(nil, embedder)

	dir := filepath.Dir(playbookPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create playbook directory: %w", err)
	}

	if err := seedInitialBullets(ctx, pb, embedder); err != nil {
		logger.Warn("Failed to seed initial bullets", "error", err)
	}

	if err := pb.Save(playbookPath); err != nil {
		logger.Warn("Failed to save initial playbook", "error", err)
	}

	return pb, nil
}

// createRetriever creates a retriever based on embedder availability.
func createRetriever(pb *playbook.Playbook, embedder embedding.Embedder) retrieval.Retriever {
	if embedder != nil {
		return retrieval.NewHNSWRetriever(pb, embedder)
	}

	return retrieval.NewSemanticRetriever(pb, embedder)
}

// createGenerator creates a generator if LLM is provided and generation is enabled.
func createGenerator(
	llmProvider llm.Provider, cfg *Config, pb *playbook.Playbook, retriever retrieval.Retriever,
) (generator.Generator, error) {
	if llmProvider == nil || !cfg.Generation.Enabled {
		return nil, nil //nolint:nilnil // intentional: nil generator means generation disabled
	}

	gen, err := generator.NewGenerator(generator.Config{
		LLM:       llmProvider,
		Playbook:  pb,
		Retriever: retriever,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	return gen, nil
}

// createReflector creates a reflector for deep analysis if LLM is available.
func createReflector(llmProvider llm.Provider, maxTokens int, logger *slog.Logger) reflector.Reflector {
	if llmProvider == nil {
		return nil
	}

	var reflectorOpts []reflector.Option
	if maxTokens > 0 {
		reflectorOpts = append(reflectorOpts, reflector.WithMaxTokens(maxTokens))
	}

	logger.Debug("Created reflector for deep insight analysis", "max_tokens", maxTokens)

	return reflector.NewReflector(llmProvider, reflectorOpts...)
}

// createCurator creates a curator for quality control and deduplication.
func createCurator(
	embedder embedding.Embedder, llmProvider llm.Provider,
	cfg *Config, maxTokens int,
	pb *playbook.Playbook, logger *slog.Logger,
) curator.Curator {
	if embedder == nil {
		return nil
	}

	curatorOpts := []curator.Option{
		curator.WithSimilarityThreshold(highSimilarityThreshold),
	}

	if maxTokens > 0 {
		curatorOpts = append(curatorOpts, curator.WithMaxTokens(maxTokens))
	}

	if llmProvider != nil && cfg.Generation.AutoReflect {
		curatorOpts = append(curatorOpts, curator.WithLLMProvider(llmProvider))

		logger.Debug("Enabled LLM-based intelligent curation")
	}

	curatorOpts = appendRefinementOpts(curatorOpts, cfg, logger)

	logger.Debug("Created curator for quality control")

	return curator.NewCurator(pb, embedder, curatorOpts...)
}

// appendRefinementOpts appends refinement configuration to curator options.
func appendRefinementOpts(opts []curator.Option, cfg *Config, logger *slog.Logger) []curator.Option {
	if !cfg.Refine.Enabled {
		return append(opts, curator.WithRefinementMode(curator.RefinementModeNone, nil))
	}

	opts = append(opts, curator.WithMergeEngine(veryHighSimilarityThresh))

	logger.Debug("Enabled merge engine for advanced bullet deduplication")

	switch cfg.Refine.Mode {
	case "proactive":
		opts = append(opts, curator.WithRefinementMode(
			curator.RefinementModeProactive,
			curator.ProactiveRefinementConfig{
				MaxBullets:      cfg.Refine.MaxBullets,
				MaxSizeBytes:    int64(cfg.Refine.MaxTokens),
				MinUtilityScore: cfg.Refine.MinUtilityScore,
			},
		))
	case "lazy":
		opts = append(opts, curator.WithRefinementMode(
			curator.RefinementModeLazy,
			curator.LazyRefinementConfig{
				MinUtilityScore: cfg.Refine.MinUtilityScore,
			},
		))
	}

	return opts
}

// createAdapter creates an adapter for online learning orchestration.
func createAdapter(
	refl reflector.Reflector, cur curator.Curator,
	gen generator.Generator, cfg *Config,
	pb *playbook.Playbook, logger *slog.Logger,
) adapter.Adapter {
	if refl == nil || cur == nil {
		return nil
	}

	needsCustomConfig := cfg.Adapter.MaxMemorySize > 0 || cfg.Adapter.UtilityThreshold > 0 || gen != nil
	if !needsCustomConfig {
		logger.Info("Created adapter for online learning orchestration")

		return adapter.NewAdapter(pb, refl, cur)
	}

	memConfig := adapter.DefaultMemoryConfig()
	if cfg.Adapter.MaxMemorySize > 0 {
		memConfig.MaxBullets = cfg.Adapter.MaxMemorySize
		memConfig.RefinementAt = int(float64(cfg.Adapter.MaxMemorySize) * aceConfidenceThreshold)
	}

	if cfg.Adapter.UtilityThreshold > 0 {
		memConfig.PruneThreshold = cfg.Adapter.UtilityThreshold
	}

	logger.Info("Created adapter for online learning orchestration")

	return adapter.NewAdapterWithConfig(adapter.Config{
		Playbook:     pb,
		Reflector:    refl,
		Curator:      cur,
		Generator:    gen,
		MemoryConfig: memConfig,
	})
}

// createGrowthMonitor creates a growth monitor for playbook management.
func createGrowthMonitor(cfg *Config, pb *playbook.Playbook, logger *slog.Logger) *refine.GrowthMonitor {
	if !cfg.Refine.Enabled {
		return nil
	}

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

	logger.Info("Created growth monitor", "max_bullets", thresholds.MaxBullets, "max_tokens", thresholds.MaxTokens)

	return refine.NewGrowthMonitor(pb, thresholds)
}

// Retrieve fetches top-K relevant bullets for the given query.
// Returns empty slice if ACE is disabled.
func (s *Service) Retrieve(ctx context.Context, query string) ([]*bullet.Bullet, error) {
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
func (s *Service) BuildPrompt(_ context.Context, systemPrompt string, bullets []*bullet.Bullet) (string, error) {
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
// Returns ErrDisabled if ACE is disabled.
func (s *Service) ParseFeedback(response string) (*feedback.BulletFeedback, error) {
	if !s.enabled {
		return nil, ErrDisabled
	}

	return s.feedbackParser.Parse(response)
}

// UpdateBullets increments bullet counters based on feedback markers.
// The bullets slice corresponds to the bullets used in the prompt (B0, B1, B2, ...).
// Feedback markers like "B0", "B2" are indices into this slice.
// If config.ItemizedLearning.UpdateAsync is true, saves playbook asynchronously.
func (s *Service) UpdateBullets(ctx context.Context, bullets []*bullet.Bullet, fb *feedback.BulletFeedback) error {
	if !s.enabled || fb == nil {
		return nil
	}

	feedbackMap := buildFeedbackMap(bullets, fb)

	if s.curator != nil {
		err := s.curator.ApplyBulletFeedback(ctx, feedbackMap)
		if err != nil {
			return fmt.Errorf("failed to apply bullet feedback: %w", err)
		}
	}

	return s.savePlaybookAfterUpdate()
}

// buildFeedbackMap maps bullet feedback markers to bullet IDs with their feedback type.
func buildFeedbackMap(bullets []*bullet.Bullet, fb *feedback.BulletFeedback) map[string]string {
	feedbackMap := make(map[string]string)

	mapMarkers(feedbackMap, bullets, fb.HelpfulBullets, "helpful")
	mapMarkers(feedbackMap, bullets, fb.HarmfulBullets, "harmful")

	return feedbackMap
}

// mapMarkers maps a list of bullet markers to their IDs with the given feedback type.
func mapMarkers(feedbackMap map[string]string, bullets []*bullet.Bullet, markers []string, feedbackType string) {
	for _, marker := range markers {
		idx := parseBulletIndex(marker)
		if idx >= 0 && idx < len(bullets) {
			feedbackMap[bullets[idx].ID] = feedbackType
		}
	}
}

// savePlaybookAfterUpdate saves the playbook synchronously or asynchronously based on config.
func (s *Service) savePlaybookAfterUpdate() error {
	if s.config.ItemizedLearning.UpdateAsync {
		go func() { _ = s.SavePlaybook() }()

		return nil
	}

	err := s.SavePlaybook()
	if err != nil {
		return fmt.Errorf("failed to save playbook: %w", err)
	}

	return nil
}

// SavePlaybook saves the playbook to disk.
// Returns nil if ACE is disabled.
func (s *Service) SavePlaybook() error {
	if !s.enabled {
		return nil
	}

	playbookPath := expandPath(s.config.PlaybookPath)

	return s.playbook.Save(playbookPath)
}

// RestoreBullet creates a bullet with a specific ID for restoration/migration scenarios.
// This is useful when importing bullets from backups or migrating from other systems.
// Returns ErrDisabled if ACE is disabled.
func (s *Service) RestoreBullet(ctx context.Context, id, content string) (*bullet.Bullet, error) {
	if !s.enabled {
		return nil, ErrDisabled
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
func (s *Service) GenerateBullets(ctx context.Context, input, sourceType string) ([]*bullet.Bullet, error) {
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
func (s *Service) GenerateBulletsWithReflectionFromTrajectory(
	ctx context.Context, trajectory *generator.Trajectory,
) ([]*bullet.Bullet, error) {
	if !s.enabled || s.reflector == nil || s.curator == nil {
		s.logger.DebugContext(ctx, "Reflection pipeline not available",
			"enabled", s.enabled, "has_reflector", s.reflector != nil,
			"has_curator", s.curator != nil)

		return nil, nil
	}

	s.logTrajectoryDetails(ctx, trajectory)

	insights, err := s.reflectOnTrajectory(ctx, trajectory)
	if err != nil {
		return nil, err
	}

	if len(insights) == 0 {
		s.logger.DebugContext(ctx, "No insights extracted from reflection")

		return nil, nil
	}

	mergeResp, err := s.curateInsights(ctx, insights)
	if err != nil {
		return nil, err
	}

	s.logCurationResults(ctx, mergeResp)

	err = s.SavePlaybook()
	if err != nil {
		return nil, fmt.Errorf("failed to save playbook: %w", err)
	}

	return mergeResp.AddedBullets, nil
}

// logTrajectoryDetails logs trajectory information at debug level.
func (s *Service) logTrajectoryDetails(ctx context.Context, trajectory *generator.Trajectory) {
	s.logger.InfoContext(ctx, "Using Reflector+Curator pipeline with full trajectory",
		"steps", len(trajectory.Steps),
		"success", trajectory.Success,
		"retrieved_bullets", len(trajectory.RetrievedBullets))

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
}

// reflectOnTrajectory runs the reflector on a trajectory and returns extracted insights.
func (s *Service) reflectOnTrajectory(ctx context.Context, trajectory *generator.Trajectory) ([]*reflector.Insight, error) {
	reflectionReq := reflector.ReflectionRequest{
		Trajectories: []*generator.Trajectory{trajectory},
	}

	reflectionResp, err := s.reflector.Reflect(ctx, reflectionReq)
	if err != nil {
		s.logger.WarnContext(ctx, "Reflection failed", "error", err)

		return nil, err
	}

	s.logger.InfoContext(ctx, "Reflector extracted insights", "count", len(reflectionResp.Insights))

	for i, insight := range reflectionResp.Insights {
		s.logger.DebugContext(ctx, "Extracted insight",
			"index", i,
			"content", insight.Content,
			"category", insight.Category,
			"confidence", insight.Confidence,
			"evidence_count", len(insight.Evidence))
	}

	return reflectionResp.Insights, nil
}

// curateInsights curates reflector insights into bullets with deduplication.
func (s *Service) curateInsights(ctx context.Context, insights []*reflector.Insight) (*curator.MergeResult, error) {
	mergeReq := curator.MergeRequest{
		Insights:            insights,
		SimilarityThreshold: highSimilarityThreshold,
	}

	s.logger.DebugContext(ctx, "Starting curation", "num_insights", len(mergeReq.Insights), "threshold", mergeReq.SimilarityThreshold)

	mergeResp, err := s.curator.Curate(ctx, mergeReq)
	if err != nil {
		s.logger.WarnContext(ctx, "Curation failed", "error", err)

		return nil, err
	}

	return mergeResp, nil
}

// logCurationResults logs the results of curation at appropriate levels.
func (s *Service) logCurationResults(ctx context.Context, mergeResp *curator.MergeResult) {
	s.logger.InfoContext(ctx, "Curator processed insights",
		"added", mergeResp.Added,
		"updated", mergeResp.Updated,
		"duplicates", len(mergeResp.Duplicates))

	for i, b := range mergeResp.AddedBullets {
		s.logger.DebugContext(ctx, "Added bullet",
			"index", i,
			"id", b.ID,
			"content", b.Content,
			"helpful_count", b.HelpfulCount,
			"harmful_count", b.HarmfulCount)
	}

	if len(mergeResp.Duplicates) > 0 {
		s.logger.DebugContext(ctx, "Found duplicates", "duplicate_ids", mergeResp.Duplicates)
	}
}

// ProcessExecutionSignal handles online learning via the Adapter.
// This is called after each tool execution to adapt context in real-time.
func (s *Service) ProcessExecutionSignal(ctx context.Context, signal *adapter.ExecutionSignal) (*adapter.AdaptationResult, error) {
	if !s.enabled || s.adapter == nil || !s.config.Adapter.Enabled {
		return nil, ErrDisabled
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
func (s *Service) trackBulletChanges(ctx context.Context, result *adapter.AdaptationResult) {
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
func (s *Service) checkGrowthAndRefine(ctx context.Context) {
	if s.growthMonitor == nil || s.curator == nil {
		return
	}

	// Check if refinement is needed.
	metrics, shouldRefine := s.growthMonitor.CheckGrowth(ctx)

	if shouldRefine {
		s.logger.InfoContext(ctx, "Growth threshold reached, triggering refinement",
			"bullet_count", metrics.BulletCount,
			"estimated_tokens", metrics.EstimatedTokens)

		// Trigger refinement asynchronously with a detached context.
		bgCtx := context.WithoutCancel(ctx)

		go func() {
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
func (s *Service) StartSession(ctx context.Context) (string, error) {
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
func (s *Service) EndSession(ctx context.Context, sessionID string) error {
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
	if path == "" || path[0] != '~' {
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
func seedInitialBullets(ctx context.Context, pb *playbook.Playbook, embedder embedding.Embedder) error {
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
