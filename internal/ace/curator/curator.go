package curator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/delta"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/refine"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/openai/openai-go"
)

// Curator transforms insights into bullets and merges into playbook.
type Curator interface {
	// Curate converts insights to bullets and merges into playbook
	Curate(ctx context.Context, req MergeRequest) (*MergeResult, error)

	// CurateBatch processes multiple merge requests in parallel
	CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error)

	// Refine explicitly prunes low-utility bullets (for lazy mode)
	Refine(ctx context.Context) (*RefinementResult, error)

	// FindDuplicates detects semantic duplicates using cosine similarity
	FindDuplicates(ctx context.Context, newBullets []*bullet.Bullet) (map[string]string, error)

	// ApplyBulletFeedback applies helpful/harmful feedback using batch delta operations
	ApplyBulletFeedback(ctx context.Context, feedback map[string]string) error

	// UpdateBulletContent updates bullet content using delta operation
	UpdateBulletContent(ctx context.Context, bulletID, newContent string) error

	// AddBulletTag adds or updates a tag on a bullet using delta operation
	AddBulletTag(ctx context.Context, bulletID, key, value string) error

	// RemoveBulletTag removes a tag from a bullet using delta operation
	RemoveBulletTag(ctx context.Context, bulletID, key string) error

	// UpdateBulletEmbedding updates bullet embedding using delta operation
	UpdateBulletEmbedding(ctx context.Context, bulletID string, embedding []float32) error
}

// curator implements Curator interface.
type curator struct {
	playbook           *playbook.Playbook
	embedder           embedding.Embedder
	llmProvider        llm.Provider
	promptBuilder      *PromptBuilder
	deltaApplier       *delta.DeltaApplier
	mergeEngine        *refine.MergeEngine
	orchestrator       *refine.RefinementOrchestrator
	threshold          float64
	refinementStrategy RefinementStrategy
	useLLMCuration     bool
	useMergeEngine     bool
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
func WithRefinementMode(mode RefinementMode, config interface{}) Option {
	return func(c *curator) {
		switch mode {
		case RefinementModeNone:
			c.refinementStrategy = &noRefinementStrategy{}
		case RefinementModeLazy:
			cfg, ok := config.(LazyRefinementConfig)
			if !ok {
				cfg = LazyRefinementConfig{MinUtilityScore: 0.1}
			}
			c.refinementStrategy = newLazyRefinementStrategy(cfg)
		case RefinementModeProactive:
			cfg, ok := config.(ProactiveRefinementConfig)
			if !ok {
				cfg = ProactiveRefinementConfig{
					MaxBullets:      1000,
					MinUtilityScore: 0.1,
				}
			}
			c.refinementStrategy = newProactiveRefinementStrategy(cfg)
		default:
			c.refinementStrategy = &noRefinementStrategy{}
		}
	}
}

// NewCurator creates a new curator.
func NewCurator(pb *playbook.Playbook, emb embedding.Embedder, opts ...Option) Curator {
	c := &curator{
		playbook:           pb,
		embedder:           emb,
		deltaApplier:       delta.NewDeltaApplier(pb),
		threshold:          0.85,
		refinementStrategy: &noRefinementStrategy{}, // Default: no refinement
	}

	for _, opt := range opts {
		opt(c)
	}

	// Create orchestrator if merge engine is enabled
	if c.useMergeEngine && c.mergeEngine != nil {
		// Create prune function that calls basic refinement
		pruneFunc := func(ctx context.Context) (int, []string, error) {
			// Use basic pruning logic from refinement strategies
			bullets := pb.List(nil)
			prunedIDs := make([]string, 0)
			minUtilityScore := 0.1 // Default threshold

			for _, b := range bullets {
				score := b.Score()
				if score < minUtilityScore {
					if err := pb.Delete(ctx, b.ID); err != nil {
						return 0, nil, err
					}
					prunedIDs = append(prunedIDs, b.ID)
				}
			}

			return len(prunedIDs), prunedIDs, nil
		}

		// Create archive for storing removed bullets
		archive := refine.NewArchive()
		c.orchestrator = refine.NewRefinementOrchestrator(pb, c.mergeEngine, archive, pruneFunc)
		// Pass orchestrator to refinement strategy
		c.refinementStrategy.SetOrchestrator(c.orchestrator)
	}

	return c
}

// Curate converts insights to bullets and merges into playbook.
func (c *curator) Curate(ctx context.Context, req MergeRequest) (*MergeResult, error) {
	slog.Debug("Curator starting",
		"num_insights", len(req.Insights),
		"use_llm", c.useLLMCuration,
		"threshold", req.SimilarityThreshold)

	// Use LLM-based curation if enabled, otherwise use deduplication
	if c.useLLMCuration && c.llmProvider != nil && c.promptBuilder != nil {
		slog.Debug("Using LLM-based curation")
		return c.curateLLMBased(ctx, req)
	}
	slog.Debug("Using deduplication-based curation")
	return c.curateDeduplicationBased(ctx, req)
}

// curateLLMBased uses LLM to intelligently decide which insights to add to playbook.
func (c *curator) curateLLMBased(ctx context.Context, req MergeRequest) (*MergeResult, error) {
	slog.Debug("LLM-based curation starting", "num_insights", len(req.Insights))

	// Format current playbook
	bullets := c.playbook.List(nil)
	slog.Debug("Current playbook state", "num_bullets", len(bullets))

	var playbookBuilder strings.Builder
	for _, b := range bullets {
		playbookBuilder.WriteString(fmt.Sprintf("[%s] %s\n", b.ID, b.Content))
	}
	currentPlaybook := playbookBuilder.String()

	// Format insights into reflection text
	var reflectionBuilder strings.Builder
	for i, insight := range req.Insights {
		slog.Debug("Formatting insight for curation",
			"index", i,
			"content", insight.Content,
			"category", insight.Category)
		reflectionBuilder.WriteString(FormatReflectionForCurator(insight))
		reflectionBuilder.WriteString("\n---\n")
	}

	// Build curation prompt
	curationReq := CurationRequest{
		TaskContext:     "Processing new insights from agent execution",
		CurrentPlaybook: currentPlaybook,
		Reflection:      reflectionBuilder.String(),
	}

	prompt := c.promptBuilder.BuildCurationPrompt(curationReq)

	slog.Debug("Curation prompt built", "length", len(prompt))
	slog.Debug("Curation prompt content", "prompt", prompt)

	// Call LLM
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		Temperature: openai.F(0.3),
	}

	slog.Debug("Calling LLM for curation", "temperature", 0.3)

	completion, err := c.llmProvider.Complete(ctx, params)
	if err != nil {
		slog.Warn("LLM call failed during curation", "error", err)
		return nil, err
	}

	// Parse response
	responseText := completion.Choices[0].Message.Content

	slog.Debug("LLM curation response received",
		"length", len(responseText),
		"tokens", completion.Usage.TotalTokens)
	slog.Debug("LLM curation response content", "response", responseText)

	responseText = cleanJSONResponse(responseText)

	var curationResp CurationResponse
	if err := json.Unmarshal([]byte(responseText), &curationResp); err != nil {
		slog.Warn("Failed to parse curation response", "error", err, "response", responseText)
		return nil, err
	}

	slog.Debug("Parsed curation response", "num_operations", len(curationResp.Operations))

	// Apply operations
	addedBullets := make([]*bullet.Bullet, 0, len(curationResp.Operations))
	for i, op := range curationResp.Operations {
		slog.Debug("Processing curation operation",
			"index", i,
			"type", op.Type,
			"section", op.Section,
			"content", op.Content)

		if op.Type == "ADD" {
			// Create bullet from operation
			newBullet, err := bullet.New(op.Content)
			if err != nil {
				slog.Warn("Failed to create bullet", "error", err, "content", op.Content)
				return nil, err
			}

			// Get embedding
			emb, err := c.embedder.Embed(ctx, newBullet.Content)
			if err != nil {
				slog.Warn("Failed to generate embedding", "error", err)
				return nil, err
			}
			newBullet.Embedding = emb

			// Add to playbook
			if err := c.playbook.Add(ctx, newBullet); err != nil {
				slog.Warn("Failed to add bullet to playbook", "error", err)
				return nil, err
			}

			slog.Debug("Added bullet via LLM curation", "id", newBullet.ID, "content", newBullet.Content)
			addedBullets = append(addedBullets, newBullet)
		}
	}

	return &MergeResult{
		Added:        len(addedBullets),
		AddedBullets: addedBullets,
	}, nil
}

// curateDeduplicationBased uses simple deduplication-based curation.
func (c *curator) curateDeduplicationBased(ctx context.Context, req MergeRequest) (*MergeResult, error) {
	slog.Debug("Deduplication-based curation starting", "num_insights", len(req.Insights))

	// Convert insights to bullets
	bullets, err := ConvertInsights(req.Insights)
	if err != nil {
		slog.Warn("Failed to convert insights to bullets", "error", err)
		return nil, err
	}

	slog.Debug("Converted insights to bullets", "count", len(bullets))

	// Get embeddings for all bullets
	for i, b := range bullets {
		slog.Debug("Generating embedding for bullet", "index", i, "content", b.Content)
		emb, err := c.embedder.Embed(ctx, b.Content)
		if err != nil {
			slog.Warn("Failed to generate embedding", "error", err, "bullet", b.Content)
			return nil, err
		}
		b.Embedding = emb
	}

	// Find duplicates
	slog.Debug("Searching for duplicates", "threshold", c.threshold)
	duplicates, err := c.FindDuplicates(ctx, bullets)
	if err != nil {
		slog.Warn("Failed to find duplicates", "error", err)
		return nil, err
	}

	slog.Debug("Duplicate detection complete", "num_duplicates", len(duplicates))

	// Process bullets: add new ones, update duplicates
	addedBullets := make([]*bullet.Bullet, 0, len(bullets))
	skipped := 0
	updated := 0
	duplicateIDs := make([]string, 0, len(duplicates))

	for i, b := range bullets {
		if existingID, isDuplicate := duplicates[b.ID]; isDuplicate {
			slog.Debug("Duplicate found",
				"index", i,
				"new_bullet", b.ID,
				"existing_bullet", existingID,
				"content", b.Content)

			// Duplicate found - update existing bullet's helpful count using delta operation
			deltaOp := delta.NewIncrementHelpful(existingID, delta.DeltaMetadata{
				Source: "curator",
				Reason: "duplicate insight detected",
			})

			_, err := c.deltaApplier.Apply(ctx, *deltaOp)
			if err == nil {
				updated++
				slog.Debug("Updated duplicate bullet helpful count", "bullet_id", existingID)
			} else {
				slog.Warn("Failed to update duplicate", "error", err, "bullet_id", existingID)
			}
			duplicateIDs = append(duplicateIDs, existingID)
			skipped++
		} else {
			slog.Debug("Adding new bullet",
				"index", i,
				"id", b.ID,
				"content", b.Content)

			// Not a duplicate - add to playbook
			if err := c.playbook.Add(ctx, b); err != nil {
				slog.Warn("Failed to add bullet", "error", err, "bullet", b.Content)
				return nil, err
			}
			addedBullets = append(addedBullets, b)
		}
	}

	slog.Debug("Deduplication complete",
		"added", len(addedBullets),
		"skipped", skipped,
		"updated", updated)

	result := &MergeResult{
		Added:        len(addedBullets),
		Skipped:      skipped,
		Updated:      updated,
		Duplicates:   duplicateIDs,
		AddedBullets: addedBullets,
	}

	// Check if refinement should be triggered (proactive mode)
	shouldRefine, err := c.refinementStrategy.ShouldRefine(ctx, c.playbook)
	if err != nil {
		return nil, err
	}

	if shouldRefine {
		refinement, err := c.refinementStrategy.Refine(ctx, c.playbook)
		if err != nil {
			return nil, err
		}
		result.Refined = true
		result.Refinement = refinement
	}

	return result, nil
}

// Refine explicitly prunes low-utility bullets.
func (c *curator) Refine(ctx context.Context) (*RefinementResult, error) {
	return c.refinementStrategy.Refine(ctx, c.playbook)
}

// CurateBatch processes multiple merge requests in parallel.
func (c *curator) CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error) {
	results := make([]MergeResult, len(req.Requests))
	errors := make([]error, len(req.Requests))

	// For empty requests, return empty results
	if len(req.Requests) == 0 {
		return &BatchMergeResult{
			Results: results,
			Errors:  errors,
		}, nil
	}

	// For single request, process sequentially
	if len(req.Requests) == 1 {
		result, err := c.Curate(ctx, req.Requests[0])
		if err != nil {
			errors[0] = err
		} else {
			results[0] = *result
		}
		return &BatchMergeResult{
			Results: results,
			Errors:  errors,
		}, nil
	}

	// For multiple requests, process in parallel with worker pool
	return c.curateBatchParallel(ctx, req.Requests, req.MaxWorkers)
}

// cleanJSONResponse extracts JSON content from markdown code blocks.
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// Handle markdown code blocks
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Remove trailing code block markers
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	return response
}

// ApplyBulletFeedback applies feedback to multiple bullets using batch delta operations.
// This is used for itemized learning where the LLM provides helpful/harmful feedback.
func (c *curator) ApplyBulletFeedback(ctx context.Context, feedback map[string]string) error {
	if c.deltaApplier == nil {
		return fmt.Errorf("delta applier not initialized")
	}

	// Build delta operations for each feedback item
	deltas := make([]delta.Delta, 0, len(feedback))
	for bulletID, feedbackType := range feedback {
		var deltaOp *delta.Delta
		switch feedbackType {
		case "helpful", "HELPFUL":
			deltaOp = delta.NewIncrementHelpful(bulletID, delta.DeltaMetadata{
				Source: "itemized_learning",
				Reason: "marked helpful by LLM",
			})
		case "harmful", "HARMFUL":
			deltaOp = delta.NewIncrementHarmful(bulletID, delta.DeltaMetadata{
				Source: "itemized_learning",
				Reason: "marked harmful by LLM",
			})
		default:
			continue // Skip unknown feedback types
		}

		if deltaOp != nil {
			deltas = append(deltas, *deltaOp)
		}
	}

	if len(deltas) == 0 {
		return nil // No feedback to apply
	}

	// Apply batch deltas in parallel
	req := delta.BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 0,     // Use NumCPU
		Atomic:     false, // Best-effort application
	}

	_, err := c.deltaApplier.ApplyBatch(ctx, req)
	return err
}

// UpdateBulletContent updates bullet content using delta operation.
func (c *curator) UpdateBulletContent(ctx context.Context, bulletID, newContent string) error {
	if c.deltaApplier == nil {
		return fmt.Errorf("delta applier not initialized")
	}

	deltaOp := delta.NewContentUpdate(bulletID, newContent, delta.DeltaMetadata{
		Source: "curator",
		Reason: "content update",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)
	return err
}

// AddBulletTag adds or updates a tag on a bullet using delta operation.
func (c *curator) AddBulletTag(ctx context.Context, bulletID, key, value string) error {
	if c.deltaApplier == nil {
		return fmt.Errorf("delta applier not initialized")
	}

	deltaOp := delta.NewAddTag(bulletID, key, value, delta.DeltaMetadata{
		Source: "curator",
		Reason: "tag added",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)
	return err
}

// RemoveBulletTag removes a tag from a bullet using delta operation.
func (c *curator) RemoveBulletTag(ctx context.Context, bulletID, key string) error {
	if c.deltaApplier == nil {
		return fmt.Errorf("delta applier not initialized")
	}

	deltaOp := delta.NewRemoveTag(bulletID, key, delta.DeltaMetadata{
		Source: "curator",
		Reason: "tag removed",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)
	return err
}

// UpdateBulletEmbedding updates bullet embedding using delta operation.
func (c *curator) UpdateBulletEmbedding(ctx context.Context, bulletID string, embedding []float32) error {
	if c.deltaApplier == nil {
		return fmt.Errorf("delta applier not initialized")
	}

	deltaOp := delta.NewUpdateEmbedding(bulletID, embedding, delta.DeltaMetadata{
		Source: "curator",
		Reason: "embedding updated",
	})

	_, err := c.deltaApplier.Apply(ctx, *deltaOp)
	return err
}
