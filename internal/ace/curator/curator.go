package curator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/delta"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/refine"
	"github.com/dmytrogajewski/spin/internal/llm"
)

const (
	defaultMinUtility       = 0.1
	defaultMaxBullets       = 1000
	defaultMinUtilityScore  = 0.1
	defaultSimilarityThresh = 0.85
	defaultMaxTokens        = 4096
	minQualityThreshold     = 0.3
)

var (
	// ErrDeltaApplierNotInitialized is a sentinel error.
	ErrDeltaApplierNotInitialized = errors.New("delta applier not initialized")
	// ErrDeltaApplierNotInitialized2 is a sentinel error.
	ErrDeltaApplierNotInitialized2 = errors.New("delta applier not initialized")
	// ErrDeltaApplierNotInitialized3 is a sentinel error.
	ErrDeltaApplierNotInitialized3 = errors.New("delta applier not initialized")
	// ErrDeltaApplierNotInitialized4 is a sentinel error.
	ErrDeltaApplierNotInitialized4 = errors.New("delta applier not initialized")
	// ErrDeltaApplierNotInitialized5 is a sentinel error.
	ErrDeltaApplierNotInitialized5 = errors.New("delta applier not initialized")
)

// BulletMerger handles insight-to-bullet conversion and playbook merging.
// Use this interface when only curation/deduplication functionality is needed.
type BulletMerger interface {
	// Curate converts insights to bullets and merges into playbook.
	Curate(ctx context.Context, req MergeRequest) (*MergeResult, error)

	// CurateBatch processes multiple merge requests in parallel.
	CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error)

	// FindDuplicates detects semantic duplicates using cosine similarity.
	FindDuplicates(ctx context.Context, newBullets []*bullet.Bullet) (map[string]string, error)
}

// BulletRefiner handles playbook quality maintenance through pruning.
// Use this interface when only refinement/pruning functionality is needed.
type BulletRefiner interface {
	// Refine explicitly prunes low-utility bullets (for lazy mode).
	Refine(ctx context.Context) (*RefinementResult, error)
}

// BulletUpdater handles individual bullet modifications via delta operations.
// Use this interface when only bullet update functionality is needed.
type BulletUpdater interface {
	// ApplyBulletFeedback applies helpful/harmful feedback using batch delta operations.
	ApplyBulletFeedback(ctx context.Context, feedback map[string]string) error

	// UpdateBulletContent updates bullet content using delta operation.
	UpdateBulletContent(ctx context.Context, bulletID, newContent string) error

	// AddBulletTag adds or updates a tag on a bullet using delta operation.
	AddBulletTag(ctx context.Context, bulletID, key, value string) error

	// RemoveBulletTag removes a tag from a bullet using delta operation.
	RemoveBulletTag(ctx context.Context, bulletID, key string) error

	// UpdateBulletEmbedding updates bullet embedding using delta operation.
	UpdateBulletEmbedding(ctx context.Context, bulletID string, embedding []float32) error
}

// Curator is the composite interface combining all bullet management capabilities.
// Use the specific interfaces (BulletMerger, BulletRefiner, BulletUpdater) when
// only a subset of functionality is needed. This interface maintains backward
// compatibility with existing code that depends on the full Curator interface.
type Curator interface {
	BulletMerger
	BulletRefiner
	BulletUpdater
}

// curator implements Curator interface.
type curator struct {
	playbook           *playbook.Playbook
	embedder           embedding.Embedder
	llmProvider        llm.Provider
	promptBuilder      *PromptBuilder
	deltaApplier       *delta.Applier
	mergeEngine        *refine.MergeEngine
	orchestrator       *refine.RefinementOrchestrator
	logger             *slog.Logger
	threshold          float64
	refinementStrategy RefinementStrategy
	useLLMCuration     bool
	useMergeEngine     bool
	maxTokens          int
}

// Option configures a Curator.
type Option func(*curator)

// WithSimilarityThreshold sets the similarity threshold for duplicate detection.
func WithSimilarityThreshold(threshold float64) Option {
	return func(c *curator) {
		c.threshold = threshold
	}
}

// WithLLMProvider enables LLM-based curation using the provided LLM.
func WithLLMProvider(llmProvider llm.Provider) Option {
	return func(c *curator) {
		c.llmProvider = llmProvider
		c.promptBuilder = NewPromptBuilder()
		c.useLLMCuration = true
	}
}

// WithMergeEngine enables merge-based refinement using semantic similarity.
func WithMergeEngine(similarityThreshold float64) Option {
	return func(c *curator) {
		c.mergeEngine = refine.NewMergeEngine(c.embedder, similarityThreshold)
		c.useMergeEngine = true
	}
}

// WithRefinementMode sets the refinement strategy.
func WithRefinementMode(mode RefinementMode, config any) Option {
	return func(c *curator) {
		switch mode {
		case RefinementModeNone:
			c.refinementStrategy = &noRefinementStrategy{}
		case RefinementModeLazy:
			cfg, ok := config.(LazyRefinementConfig)
			if !ok {
				cfg = LazyRefinementConfig{MinUtilityScore: defaultMinUtilityScore}
			}

			c.refinementStrategy = newLazyRefinementStrategy(cfg)
		case RefinementModeProactive:
			cfg, ok := config.(ProactiveRefinementConfig)
			if !ok {
				cfg = ProactiveRefinementConfig{
					MaxBullets:      defaultMaxBullets,
					MinUtilityScore: defaultMinUtilityScore,
				}
			}

			c.refinementStrategy = newProactiveRefinementStrategy(cfg)
		default:
			c.refinementStrategy = &noRefinementStrategy{}
		}
	}
}

// WithMaxTokens sets the maximum tokens for LLM calls.
func WithMaxTokens(maxTokens int) Option {
	return func(c *curator) {
		c.maxTokens = maxTokens
	}
}

// NewCurator creates a new curator.
func NewCurator(pb *playbook.Playbook, emb embedding.Embedder, opts ...Option) Curator {
	c := &curator{
		playbook:           pb,
		embedder:           emb,
		deltaApplier:       delta.NewApplier(pb),
		logger:             slog.Default(),
		threshold:          defaultSimilarityThresh,
		refinementStrategy: &noRefinementStrategy{}, // Default: no refinement.
		maxTokens:          defaultMaxTokens,        // Default max tokens for LLM calls.
	}

	for _, opt := range opts {
		opt(c)
	}

	// Create orchestrator if merge engine is enabled.
	if c.useMergeEngine && c.mergeEngine != nil {
		// Create prune function that calls basic refinement.
		pruneFunc := func(ctx context.Context) (int, []string, error) {
			// Use basic pruning logic from refinement strategies.
			bullets := pb.List(nil)
			prunedIDs := make([]string, 0)
			minUtilityScore := 0.1 // Default threshold.

			for _, b := range bullets {
				score := b.Score()
				if score < minUtilityScore {
					err := pb.Delete(ctx, b.ID)
					if err != nil {
						return 0, nil, err
					}

					prunedIDs = append(prunedIDs, b.ID)
				}
			}

			return len(prunedIDs), prunedIDs, nil
		}

		// Create archive for storing removed bullets.
		archive := refine.NewArchive()
		c.orchestrator = refine.NewRefinementOrchestrator(pb, c.mergeEngine, archive, pruneFunc)
		// Pass orchestrator to refinement strategy.
		c.refinementStrategy.SetOrchestrator(c.orchestrator)
	}

	return c
}

// Curate converts insights to bullets and merges into playbook.
func (c *curator) Curate(ctx context.Context, req MergeRequest) (*MergeResult, error) {
	c.logger.DebugContext(ctx, "Curator starting",
		"num_insights", len(req.Insights),
		"use_llm", c.useLLMCuration,
		"threshold", req.SimilarityThreshold)

	// Use LLM-based curation if enabled, otherwise use deduplication.
	if c.useLLMCuration && c.llmProvider != nil && c.promptBuilder != nil {
		c.logger.DebugContext(ctx, "Using LLM-based curation")

		return c.curateLLMBased(ctx, req)
	}

	c.logger.DebugContext(ctx, "Using deduplication-based curation")

	return c.curateDeduplicationBased(ctx, req)
}

// curateLLMBased uses LLM to intelligently decide which insights to add to playbook.
func (c *curator) curateLLMBased(ctx context.Context, req MergeRequest) (*MergeResult, error) {
	c.logger.DebugContext(ctx, "LLM-based curation starting", "num_insights", len(req.Insights))

	prompt := c.buildCurationPrompt(ctx, req)

	curationResp, err := c.callLLMForCuration(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return c.applyCurationOperations(ctx, curationResp)
}

// buildCurationPrompt formats playbook and insights into a curation prompt.
func (c *curator) buildCurationPrompt(ctx context.Context, req MergeRequest) string {
	bullets := c.playbook.List(nil)
	c.logger.DebugContext(ctx, "Current playbook state", "num_bullets", len(bullets))

	var playbookBuilder strings.Builder
	for _, b := range bullets {
		fmt.Fprintf(&playbookBuilder, "[%s] %s\n", b.ID, b.Content)
	}

	var reflectionBuilder strings.Builder

	for i, insight := range req.Insights {
		c.logger.DebugContext(ctx, "Formatting insight for curation",
			"index", i, "content", insight.Content, "category", insight.Category)
		reflectionBuilder.WriteString(FormatReflectionForCurator(insight))
		reflectionBuilder.WriteString("\n---\n")
	}

	curationReq := CurationRequest{
		TaskContext:     "Processing new insights from agent execution",
		CurrentPlaybook: playbookBuilder.String(),
		Reflection:      reflectionBuilder.String(),
	}

	return c.promptBuilder.BuildCurationPrompt(curationReq)
}

// callLLMForCuration calls the LLM and parses the curation response.
func (c *curator) callLLMForCuration(ctx context.Context, prompt string) (*CurationResponse, error) {
	c.logger.DebugContext(ctx, "Curation prompt built", "length", len(prompt))

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		Temperature: openai.F(minQualityThreshold),
	}

	if c.maxTokens > 0 {
		params.MaxTokens = openai.F(int64(c.maxTokens))
	}

	completion, err := c.llmProvider.Complete(ctx, params)
	if err != nil {
		c.logger.WarnContext(ctx, "LLM call failed during curation", "error", err)

		return nil, err
	}

	responseText := cleanJSONResponse(completion.Choices[0].Message.Content)
	c.logger.DebugContext(ctx, "LLM curation response received",
		"length", len(responseText), "tokens", completion.Usage.TotalTokens)

	var curationResp CurationResponse
	if err = json.Unmarshal([]byte(responseText), &curationResp); err != nil {
		c.logger.WarnContext(ctx, "Failed to parse curation response", "error", err, "response", responseText)

		return nil, fmt.Errorf("unmarshaling curation response: %w", err)
	}

	return &curationResp, nil
}

// applyCurationOperations applies ADD operations from the curation response.
func (c *curator) applyCurationOperations(ctx context.Context, resp *CurationResponse) (*MergeResult, error) {
	addedBullets := make([]*bullet.Bullet, 0, len(resp.Operations))
	for _, op := range resp.Operations {
		if op.Type != "ADD" {
			continue
		}

		newBullet, err := c.createAndAddBullet(ctx, op.Content)
		if err != nil {
			return nil, err
		}

		addedBullets = append(addedBullets, newBullet)
	}

	return &MergeResult{
		Added:        len(addedBullets),
		AddedBullets: addedBullets,
	}, nil
}

// createAndAddBullet creates a bullet, generates its embedding, and adds it to the playbook.
func (c *curator) createAndAddBullet(ctx context.Context, content string) (*bullet.Bullet, error) {
	newBullet, err := bullet.New(content)
	if err != nil {
		c.logger.WarnContext(ctx, "Failed to create bullet", "error", err, "content", content)

		return nil, err
	}

	emb, err := c.embedder.Embed(ctx, newBullet.Content)
	if err != nil {
		c.logger.WarnContext(ctx, "Failed to generate embedding", "error", err)

		return nil, err
	}

	newBullet.Embedding = emb

	if err = c.playbook.Add(ctx, newBullet); err != nil {
		c.logger.WarnContext(ctx, "Failed to add bullet to playbook", "error", err)

		return nil, err
	}

	c.logger.DebugContext(ctx, "Added bullet via LLM curation", "id", newBullet.ID, "content", newBullet.Content)

	return newBullet, nil
}

// curateDeduplicationBased uses simple deduplication-based curation.
func (c *curator) curateDeduplicationBased(ctx context.Context, req MergeRequest) (*MergeResult, error) {
	c.logger.DebugContext(ctx, "Deduplication-based curation starting", "num_insights", len(req.Insights))

	bullets, err := c.convertAndEmbedInsights(ctx, req)
	if err != nil {
		return nil, err
	}

	duplicates, err := c.FindDuplicates(ctx, bullets)
	if err != nil {
		c.logger.WarnContext(ctx, "Failed to find duplicates", "error", err)

		return nil, err
	}

	c.logger.DebugContext(ctx, "Duplicate detection complete", "num_duplicates", len(duplicates))

	result := c.processBulletsWithDuplicates(ctx, bullets, duplicates)

	return c.maybeRefine(ctx, result)
}

// convertAndEmbedInsights converts insights to bullets and generates embeddings.
func (c *curator) convertAndEmbedInsights(ctx context.Context, req MergeRequest) ([]*bullet.Bullet, error) {
	bullets, err := ConvertInsights(req.Insights)
	if err != nil {
		c.logger.WarnContext(ctx, "Failed to convert insights to bullets", "error", err)

		return nil, err
	}

	for _, b := range bullets {
		emb, embErr := c.embedder.Embed(ctx, b.Content)
		if embErr != nil {
			c.logger.WarnContext(ctx, "Failed to generate embedding", "error", embErr, "bullet", b.Content)

			return nil, embErr
		}

		b.Embedding = emb
	}

	return bullets, nil
}

// processBulletsWithDuplicates adds new bullets and updates duplicates, returning the merge result.
func (c *curator) processBulletsWithDuplicates(ctx context.Context, bullets []*bullet.Bullet, duplicates map[string]string) *MergeResult {
	addedBullets := make([]*bullet.Bullet, 0, len(bullets))
	duplicateIDs := make([]string, 0, len(duplicates))
	skipped, updated := 0, 0

	for _, b := range bullets {
		existingID, isDuplicate := duplicates[b.ID]
		if !isDuplicate {
			if err := c.playbook.Add(ctx, b); err != nil {
				c.logger.WarnContext(ctx, "Failed to add bullet", "error", err, "bullet", b.Content)

				continue
			}

			addedBullets = append(addedBullets, b)

			continue
		}

		deltaOp := delta.NewIncrementHelpful(existingID, delta.Metadata{
			Source: "curator", Reason: "duplicate insight detected",
		})
		if _, err := c.deltaApplier.Apply(ctx, *deltaOp); err == nil {
			updated++
		}

		duplicateIDs = append(duplicateIDs, existingID)
		skipped++
	}

	return &MergeResult{
		Added: len(addedBullets), Skipped: skipped, Updated: updated,
		Duplicates: duplicateIDs, AddedBullets: addedBullets,
	}
}

// maybeRefine checks if proactive refinement should be triggered and applies it.
func (c *curator) maybeRefine(ctx context.Context, result *MergeResult) (*MergeResult, error) {
	shouldRefine, err := c.refinementStrategy.ShouldRefine(ctx, c.playbook)
	if err != nil {
		return nil, err
	}

	if !shouldRefine {
		return result, nil
	}

	refinement, err := c.refinementStrategy.Refine(ctx, c.playbook)
	if err != nil {
		return nil, err
	}

	result.Refined = true
	result.Refinement = refinement

	return result, nil
}

// Refine explicitly prunes low-utility bullets.
func (c *curator) Refine(ctx context.Context) (*RefinementResult, error) {
	return c.refinementStrategy.Refine(ctx, c.playbook)
}

// CurateBatch processes multiple merge requests in parallel.
func (c *curator) CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error) {
	results := make([]MergeResult, len(req.Requests))
	errs := make([]error, len(req.Requests))

	// For empty requests, return empty results.
	if len(req.Requests) == 0 {
		return &BatchMergeResult{
			Results: results,
			Errors:  errs,
		}, nil
	}

	// For single request, process sequentially.
	if len(req.Requests) == 1 {
		result, err := c.Curate(ctx, req.Requests[0])
		if err != nil {
			errs[0] = err
		} else {
			results[0] = *result
		}

		return &BatchMergeResult{
			Results: results,
			Errors:  errs,
		}, nil
	}

	// For multiple requests, process in parallel with worker pool.
	return c.curateBatchParallel(ctx, req.Requests, req.MaxWorkers)
}

// cleanJSONResponse extracts JSON content from markdown code blocks.
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// Handle markdown code blocks.
	if after, ok := strings.CutPrefix(response, "```json"); ok {
		response = after
		response = strings.TrimSpace(response)
	} else if afterPlain, okPlain := strings.CutPrefix(response, "```"); okPlain {
		response = afterPlain
		response = strings.TrimSpace(response)
	}

	// Remove trailing code block markers.
	if before, ok := strings.CutSuffix(response, "```"); ok {
		response = before
		response = strings.TrimSpace(response)
	}

	return response
}

// ApplyBulletFeedback applies feedback to multiple bullets using batch delta operations.
// This is used for itemized learning where the LLM provides helpful/harmful feedback.
func (c *curator) ApplyBulletFeedback(ctx context.Context, feedback map[string]string) error {
	if c.deltaApplier == nil {
		return ErrDeltaApplierNotInitialized
	}

	// Build delta operations for each feedback item.
	deltas := make([]delta.Delta, 0, len(feedback))
	for bulletID, feedbackType := range feedback {
		var deltaOp *delta.Delta

		switch feedbackType {
		case "helpful", "HELPFUL":
			deltaOp = delta.NewIncrementHelpful(bulletID, delta.Metadata{
				Source: "itemized_learning",
				Reason: "marked helpful by LLM",
			})
		case "harmful", "HARMFUL":
			deltaOp = delta.NewIncrementHarmful(bulletID, delta.Metadata{
				Source: "itemized_learning",
				Reason: "marked harmful by LLM",
			})
		default:
			continue // Skip unknown feedback types.
		}

		if deltaOp != nil {
			deltas = append(deltas, *deltaOp)
		}
	}

	if len(deltas) == 0 {
		return nil // No feedback to apply.
	}

	// Apply batch deltas in parallel.
	req := delta.BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 0,     // Use NumCPU.
		Atomic:     false, // Best-effort application.
	}

	_, err := c.deltaApplier.ApplyBatch(ctx, req)

	return err
}

// UpdateBulletContent updates bullet content using delta operation.
func (c *curator) UpdateBulletContent(ctx context.Context, bulletID, newContent string) error {
	if c.deltaApplier == nil {
		return ErrDeltaApplierNotInitialized2
	}

	deltaOp := delta.NewContentUpdate(bulletID, newContent, delta.Metadata{
		Source: "curator",
		Reason: "content update",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)

	return err
}

// AddBulletTag adds or updates a tag on a bullet using delta operation.
func (c *curator) AddBulletTag(ctx context.Context, bulletID, key, value string) error {
	if c.deltaApplier == nil {
		return ErrDeltaApplierNotInitialized3
	}

	deltaOp := delta.NewAddTag(bulletID, key, value, delta.Metadata{
		Source: "curator",
		Reason: "tag added",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)

	return err
}

// RemoveBulletTag removes a tag from a bullet using delta operation.
func (c *curator) RemoveBulletTag(ctx context.Context, bulletID, key string) error {
	if c.deltaApplier == nil {
		return ErrDeltaApplierNotInitialized4
	}

	deltaOp := delta.NewRemoveTag(bulletID, key, delta.Metadata{
		Source: "curator",
		Reason: "tag removed",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)

	return err
}

// UpdateBulletEmbedding updates bullet embedding using delta operation.
func (c *curator) UpdateBulletEmbedding(ctx context.Context, bulletID string, vec []float32) error {
	if c.deltaApplier == nil {
		return ErrDeltaApplierNotInitialized5
	}

	deltaOp := delta.NewUpdateEmbedding(bulletID, vec, delta.Metadata{
		Source: "curator",
		Reason: "embedding updated",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)

	return err
}
