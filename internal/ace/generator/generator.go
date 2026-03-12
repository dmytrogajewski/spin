// Package generator provides content generation capabilities.
package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/feedback"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/prompt"
	"github.com/dmytrogajewski/spin/internal/ace/retrieval"
	"github.com/dmytrogajewski/spin/internal/llm"
)

var (
	ErrLlmProviderIsRequired = errors.New("LLM provider is required")
	ErrPlaybookIsRequired = errors.New("playbook is required")
	ErrRetrieverIsRequired = errors.New("retriever is required")
	ErrInputIsRequired = errors.New("input is required")
	ErrUnknownSourceType = errors.New("unknown source type")
)

// Generator produces reasoning trajectories with context bullets.
type Generator interface {
	// ItemizedLearning retrieves bullets, injects into prompt, executes task,
	// and collects feedback on bullet utility.
	ItemizedLearning(ctx context.Context, req ItemizedLearningRequest) (*ItemizedLearningResponse, error)

	// GenerateBullets creates new bullet candidates from input.
	GenerateBullets(ctx context.Context, req BulletGenerationRequest) ([]*bullet.Bullet, error)
}

// ItemizedLearningRequest is input for ItemizedLearning workflow.
type ItemizedLearningRequest struct {
	// Query is the task description or question.
	Query string

	// GroundTruth is the expected answer (optional, for labeled learning).
	GroundTruth string

	// TopK is number of bullets to retrieve.
	TopK int

	// Model is the LLM model to use.
	Model string

	// Temperature controls randomness.
	Temperature float64

	// MaxTokens limits response length.
	MaxTokens int
}

// ItemizedLearningResponse is output from ItemizedLearning.
type ItemizedLearningResponse struct {
	// Trajectory is the full execution trace.
	Trajectory *Trajectory

	// Feedback contains bullet utility annotations.
	Feedback *feedback.BulletFeedback

	// Output is the final answer.
	Output string

	// Success indicates if task succeeded.
	Success bool
}

// BulletGenerationRequest is input for bullet generation.
type BulletGenerationRequest struct {
	// Input is the source text (task, trajectory, feedback).
	Input string

	// SourceType indicates input type ("task", "trajectory", "feedback", "error").
	SourceType string

	// MaxBullets limits number of generated bullets.
	MaxBullets int

	// Tags to apply to generated bullets.
	Tags map[string]string

	// Model is the LLM model to use (optional, defaults to config).
	Model string
}

// generator is the concrete implementation of Generator interface.
type generator struct {
	llm            llm.Provider
	playbook       *playbook.Playbook
	retriever      retrieval.Retriever
	promptBuilder  *prompt.Builder
	feedbackParser feedback.Parser
	logger         *slog.Logger
}

// Config configures a generator.
type Config struct {
	LLM       llm.Provider
	Playbook  *playbook.Playbook
	Retriever retrieval.Retriever
}

// NewGenerator creates a new generator.
func NewGenerator(cfg Config) (Generator, error) {
	if cfg.LLM == nil {
		return nil, ErrLlmProviderIsRequired
	}

	if cfg.Playbook == nil {
		return nil, ErrPlaybookIsRequired
	}

	if cfg.Retriever == nil {
		return nil, ErrRetrieverIsRequired
	}

	return &generator{
		llm:            cfg.LLM,
		playbook:       cfg.Playbook,
		retriever:      cfg.Retriever,
		promptBuilder:  prompt.NewBuilder(prompt.WithItemizedLearning()),
		feedbackParser: feedback.NewRegexParser(),
		logger:         slog.Default(),
	}, nil
}

// ItemizedLearning implements Generator interface.
func (g *generator) ItemizedLearning(ctx context.Context, req ItemizedLearningRequest) (*ItemizedLearningResponse, error) {
	startTime := time.Now()

	// 1. Retrieve relevant bullets.
	bullets, err := g.retriever.Retrieve(ctx, req.Query, req.TopK)
	if err != nil {
		return nil, fmt.Errorf("retrieve bullets: %w", err)
	}

	// 2. Build system prompt with bullets and IL instructions.
	systemPrompt := g.promptBuilder.BuildSystemPrompt(bullets)

	// 3. Construct messages.
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(req.Query),
	}

	// 4. Call LLM.
	params := openai.ChatCompletionNewParams{
		Messages:    openai.F(messages),
		Model:       openai.F(openai.ChatModel(req.Model)),
		Temperature: openai.Float(req.Temperature),
		MaxTokens:   openai.Int(int64(req.MaxTokens)),
	}

	resp, err := g.llm.Complete(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("llm complete: %w", err)
	}

	// 5. Extract output.
	output := ""
	if len(resp.Choices) > 0 {
		output = resp.Choices[0].Message.Content
	}

	// 6. Parse feedback.
	bulletFeedback, err := g.feedbackParser.Parse(output)
	if err != nil {
		// Log error but don't fail - feedback is optional.
		bulletFeedback = &feedback.BulletFeedback{
			HelpfulBullets: []string{},
			HarmfulBullets: []string{},
		}
	}

	// 7. Update playbook with feedback.
	err = g.updatePlaybook(ctx, bullets, bulletFeedback)
	if err != nil {
		return nil, fmt.Errorf("update playbook: %w", err)
	}

	// 8. Build trajectory.
	duration := time.Since(startTime)
	trajectory := g.buildTrajectory(req, bullets, resp, bulletFeedback, output, duration)

	// 9. Determine success (if ground truth provided).
	success := false
	if req.GroundTruth != "" {
		success = g.checkSuccess(output, req.GroundTruth)
	}

	return &ItemizedLearningResponse{
		Trajectory: trajectory,
		Feedback:   bulletFeedback,
		Output:     output,
		Success:    success,
	}, nil
}

// updatePlaybook updates bullet counters based on feedback.
func (g *generator) updatePlaybook(ctx context.Context, bullets []*bullet.Bullet, fb *feedback.BulletFeedback) error {
	// Build marker to bullet ID map.
	markerToID := make(map[string]string)

	for i, b := range bullets {
		marker := fmt.Sprintf("B%d", i)
		markerToID[marker] = b.ID
	}

	// Update helpful bullets.
	for _, marker := range fb.HelpfulBullets {
		bulletID, ok := markerToID[marker]
		if !ok {
			continue // Skip unknown markers.
		}

		b, found := g.playbook.Get(bulletID)
		if !found {
			continue
		}

		b.IncrementHelpful()

		err := g.playbook.Update(ctx, b)
		if err != nil {
			return fmt.Errorf("update helpful bullet %s: %w", bulletID, err)
		}
	}

	// Update harmful bullets.
	for _, marker := range fb.HarmfulBullets {
		bulletID, ok := markerToID[marker]
		if !ok {
			continue // Skip unknown markers.
		}

		b, found := g.playbook.Get(bulletID)
		if !found {
			continue
		}

		b.IncrementHarmful()

		err := g.playbook.Update(ctx, b)
		if err != nil {
			return fmt.Errorf("update harmful bullet %s: %w", bulletID, err)
		}
	}

	return nil
}

// buildTrajectory constructs a trajectory from the execution.
func (g *generator) buildTrajectory(
	req ItemizedLearningRequest,
	bullets []*bullet.Bullet,
	resp *openai.ChatCompletion,
	bulletFeedback *feedback.BulletFeedback,
	output string,
	duration time.Duration,
) *Trajectory {
	// Create a single reasoning step.
	step := TrajectoryStep{
		StepNumber: 0,
		Type:       "reasoning",
		Content:    output,
		Timestamp:  time.Now(),
	}

	// Extract metadata.
	metadata := TrajectoryMetadata{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Duration:    duration,
		Turns:       1,
	}

	if resp.Usage.TotalTokens > 0 {
		metadata.TotalTokens = int(resp.Usage.TotalTokens)
	}

	return &Trajectory{
		ID:               resp.ID,
		Query:            req.Query,
		RetrievedBullets: bullets,
		Steps:            []TrajectoryStep{step},
		Output:           output,
		Success:          req.GroundTruth != "" && g.checkSuccess(output, req.GroundTruth),
		BulletFeedback:   bulletFeedback,
		Metadata:         metadata,
		CreatedAt:        time.Now(),
	}
}

// checkSuccess checks if output matches ground truth.
// Uses multiple strategies: exact match, normalized match, and substring matching.
func (g *generator) checkSuccess(output, groundTruth string) bool {
	if len(output) == 0 || len(groundTruth) == 0 {
		return false
	}

	// Strategy 1: Exact match.
	if output == groundTruth {
		return true
	}

	// Strategy 2: Normalized match (lowercase, trimmed).
	normalizedOutput := strings.ToLower(strings.TrimSpace(output))
	normalizedTruth := strings.ToLower(strings.TrimSpace(groundTruth))

	if normalizedOutput == normalizedTruth {
		return true
	}

	// Strategy 3: Contains match (for flexible matching).
	if strings.Contains(normalizedOutput, normalizedTruth) ||
		strings.Contains(normalizedTruth, normalizedOutput) {
		return true
	}

	// Strategy 4: Word-based similarity (check if key words match).
	outputWords := strings.Fields(normalizedOutput)
	truthWords := strings.Fields(normalizedTruth)

	if len(outputWords) == 0 || len(truthWords) == 0 {
		return false
	}

	// Count matching words.
	matchCount := 0

	for _, word := range truthWords {
		if slices.Contains(outputWords, word) {
			matchCount++
		}
	}

	// Require at least 70% word overlap.
	similarity := float64(matchCount) / float64(len(truthWords))

	return similarity >= 0.7
}

// GenerateBullets implements Generator interface.
func (g *generator) GenerateBullets(ctx context.Context, req BulletGenerationRequest) ([]*bullet.Bullet, error) {
	if req.Input == "" {
		return nil, ErrInputIsRequired
	}
	// Select prompt template based on source type.
	systemPrompt := bulletGenerationSystemPrompt

	var userPrompt string

	switch req.SourceType {
	case "task":
		userPrompt = fmt.Sprintf(taskBulletPrompt, req.Input)
	case "trajectory":
		userPrompt = fmt.Sprintf(trajectoryBulletPrompt, req.Input)
	case "feedback":
		userPrompt = fmt.Sprintf(feedbackBulletPrompt, req.Input)
	case "error":
		userPrompt = fmt.Sprintf(errorBulletPrompt, req.Input)
	default:
return nil, fmt.Errorf("unknown source type: %s: %w", req.SourceType, ErrUnknownSourceType)
	}

	// Call LLM.
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(userPrompt),
	}

	model := req.Model
	if model == "" {
		model = "gpt-4" // Default model.
	}

	g.logger.DebugContext(ctx, "ACE Generator: Calling LLM", "model", model, "system_prompt_len", len(systemPrompt), "user_prompt_len", len(userPrompt))

	params := openai.ChatCompletionNewParams{
		Messages:    openai.F(messages),
		Model:       openai.F(openai.ChatModel(model)),
		Temperature: openai.Float(0.7), // Some creativity for generation.
		MaxTokens:   openai.Int(2000),
	}

	resp, err := g.llm.Complete(ctx, params)
	if err != nil {
		g.logger.DebugContext(ctx, "ACE Generator: LLM call failed", "error", err)

		return nil, fmt.Errorf("llm complete: %w", err)
	}

	g.logger.DebugContext(ctx, "ACE Generator: LLM returned successfully", "choices", len(resp.Choices))

	// Extract output.
	output := ""
	if len(resp.Choices) > 0 {
		output = resp.Choices[0].Message.Content
		g.logger.DebugContext(ctx, "ACE Generator: LLM response", "length", len(output))

		if len(output) > 0 {
			preview := output
			if len(preview) > 200 {
				preview = preview[:200]
			}

			g.logger.DebugContext(ctx, "ACE Generator: LLM response preview", "preview", preview)
		}
	} else {
		g.logger.WarnContext(ctx, "ACE Generator: No choices in LLM response!")
	}

	// Parse bullet candidates from output.
	candidates := parseBulletCandidates(output)
	g.logger.DebugContext(ctx, "ACE Generator: Parsed candidates", "count", len(candidates))

	// Create bullet objects with tags.
	bullets := make([]*bullet.Bullet, 0, len(candidates))
	for _, content := range candidates {
		// Validate content length.
		if len(content) == 0 || len(content) > bullet.MaxContentLength {
			continue // Skip invalid candidates.
		}

		opts := []bullet.Option{}
		if len(req.Tags) > 0 {
			opts = append(opts, bullet.WithTags(req.Tags))
		}

		var b *bullet.Bullet
		b, err = bullet.New(content, opts...)
		if err != nil {
			// Log but continue with other bullets.
			continue
		}

		bullets = append(bullets, b)
	}

	g.logger.DebugContext(ctx, "ACE Generator: Created bullets", "count", len(bullets))

	return bullets, nil
}

// min returns the smaller of two ints.

// Prompt templates for bullet generation.

const bulletGenerationSystemPrompt = `You are an expert at distilling insights into concise, actionable strategies.

Extract concrete, specific strategies or domain knowledge from the input.

Each strategy should be:
- Actionable (not vague advice)
- Self-contained (understandable alone)
- Concise (under 200 chars preferred)
- Specific to the domain/task

Format: Output one strategy per line, numbered (1., 2., 3., etc.).`

const taskBulletPrompt = `Extract key strategies for solving this type of task:

%s

Output numbered strategies (1., 2., 3., etc.), one per line. Generate as many insights as are genuinely useful and actionable.`

const trajectoryBulletPrompt = `Analyze this execution trajectory and extract key lessons or strategies:

%s

Output numbered strategies (1., 2., 3., etc.), one per line. Generate as many insights as are genuinely useful and actionable.`

const feedbackBulletPrompt = `Based on this feedback, extract actionable improvement strategies:

%s

Output numbered strategies (1., 2., 3., etc.), one per line. Generate as many insights as are genuinely useful and actionable.`

const errorBulletPrompt = `Analyze this error and extract strategies to prevent it in the future:

%s

Output numbered strategies (1., 2., 3., etc.), one per line. Generate as many insights as are genuinely useful and actionable.`

// parseBulletCandidates extracts bullet content from LLM output.
func parseBulletCandidates(output string) []string {
	candidates := []string{}
	lines := strings.SplitSeq(output, "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to extract numbered items (1., 2., etc.)
		// Pattern: "1. Content" or "1) Content" or "- Content".
		content := line

		// Remove common prefixes.
		if len(line) > 2 {
			if line[0] >= '0' && line[0] <= '9' {
				// Starts with digit.
				if line[1] == '.' || line[1] == ')' {
					content = strings.TrimSpace(line[2:])
				}
			} else if strings.HasPrefix(line, "- ") {
				content = strings.TrimSpace(line[2:])
			} else if strings.HasPrefix(line, "* ") {
				content = strings.TrimSpace(line[2:])
			}
		}

		if content != "" && content != line {
			// Successfully extracted content.
			candidates = append(candidates, content)
		}
	}

	return candidates
}
