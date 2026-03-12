package reflector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// ErrNotDetailedFormat is returned when the reflector response cannot be parsed.
var ErrNotDetailedFormat = errors.New("not detailed format")

const (
	defaultReflectorMaxTokens = 4096
	reflectorTemperature      = 0.3
)

// Reflector analyzes trajectories to extract insights.
type Reflector interface {
	// Reflect analyzes trajectories and extracts insights.
	Reflect(ctx context.Context, req ReflectionRequest) (*ReflectionResponse, error)

	// RefineInsights improves insights through multiple iterations.
	RefineInsights(ctx context.Context, insights []*Insight, iterations int) ([]*Insight, error)
}

// reflector implements Reflector interface.
type reflector struct {
	llm           llm.Provider
	promptBuilder *PromptBuilder
	validator     *InsightValidator
	logger        *slog.Logger
	maxTokens     int
}

// Option configures a Reflector.
type Option func(*reflector)

// WithMaxTokens sets the maximum tokens for LLM calls.
func WithMaxTokens(maxTokens int) Option {
	return func(r *reflector) {
		r.maxTokens = maxTokens
	}
}

// NewReflector creates a new reflector.
func NewReflector(llmProvider llm.Provider, opts ...Option) Reflector {
	r := &reflector{
		llm:           llmProvider,
		promptBuilder: NewPromptBuilder(),
		validator:     NewInsightValidator(),
		logger:        slog.Default(),
		maxTokens:     defaultReflectorMaxTokens, // Default max tokens for LLM calls.
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Reflect analyzes trajectories and extracts insights.
func (r *reflector) Reflect(ctx context.Context, req ReflectionRequest) (*ReflectionResponse, error) {
	startTime := time.Now()

	r.logger.DebugContext(ctx, "Reflector starting analysis", "num_trajectories", len(req.Trajectories))

	// Handle empty trajectory list.
	if len(req.Trajectories) == 0 {
		r.logger.DebugContext(ctx, "No trajectories to reflect on")

		return &ReflectionResponse{
			Insights:    []*Insight{},
			Iterations:  0,
			TotalTokens: 0,
			Duration:    time.Since(startTime),
		}, nil
	}

	// Build prompt based on number of trajectories.
	var (
		prompt   string
		sourceID string
	)

	if len(req.Trajectories) == 1 {
		// Single trajectory.
		traj := req.Trajectories[0]
		r.logger.DebugContext(ctx, "Building prompt for single trajectory",
			"id", traj.ID,
			"query", traj.Query,
			"steps", len(traj.Steps),
			"success", traj.Success)
		prompt = r.promptBuilder.BuildSingleTrajectory(traj)
		sourceID = traj.ID
	} else {
		// Multiple trajectories - batch analysis.
		r.logger.DebugContext(ctx, "Building prompt for batch trajectories", "count", len(req.Trajectories))
		prompt = r.promptBuilder.BuildBatchTrajectory(req.Trajectories)
		sourceID = "batch"
	}

	r.logger.DebugContext(ctx, "Reflector prompt generated", "length", len(prompt))
	r.logger.DebugContext(ctx, "Reflector prompt content", "prompt", prompt)

	// Call LLM.
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		Temperature: openai.F(reflectorTemperature),
	}

	// Set MaxTokens if configured.
	if r.maxTokens > 0 {
		params.MaxTokens = openai.F(int64(r.maxTokens))
	}

	r.logger.DebugContext(ctx, "Calling LLM for reflection", "temperature", reflectorTemperature, "max_tokens", r.maxTokens)

	completion, err := r.llm.Complete(ctx, params)
	if err != nil {
		r.logger.WarnContext(ctx, "LLM call failed during reflection", "error", err)

		return nil, err
	}

	// Parse JSON response into insights.
	responseText := completion.Choices[0].Message.Content

	r.logger.DebugContext(ctx, "LLM response received",
		"length", len(responseText),
		"tokens", completion.Usage.TotalTokens)
	r.logger.DebugContext(ctx, "LLM response content", "response", responseText)

	// Clean response text to extract JSON from markdown code blocks.
	responseText = cleanJSONResponse(responseText)

	insights, err := r.parseReflectionResponse(ctx, responseText, sourceID)
	if err != nil {
		return nil, err
	}

	return &ReflectionResponse{
		Insights:    insights,
		Iterations:  1,
		TotalTokens: int(completion.Usage.TotalTokens),
		Duration:    time.Since(startTime),
	}, nil
}

// parseReflectionResponse parses the LLM response into insights, trying detailed format first.
func (r *reflector) parseReflectionResponse(ctx context.Context, responseText, sourceID string) ([]*Insight, error) {
	insights, err := r.parseDetailedFormat(ctx, responseText, sourceID)
	if err == nil {
		return insights, nil
	}

	return r.parseSimplifiedFormat(ctx, responseText, sourceID)
}

// parseDetailedFormat parses response as a single detailed reflection object.
func (r *reflector) parseDetailedFormat(ctx context.Context, responseText, sourceID string) ([]*Insight, error) {
	var resp struct {
		Reasoning           string  `json:"reasoning"`
		ErrorIdentification string  `json:"error_identification"`
		RootCauseAnalysis   string  `json:"root_cause_analysis"`
		CorrectApproach     string  `json:"correct_approach"`
		KeyInsight          string  `json:"key_insight"`
		Category            string  `json:"category"`
		Confidence          float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(responseText), &resp); err != nil || resp.KeyInsight == "" {
		return nil, ErrNotDetailedFormat
	}

	insight := NewInsight(resp.KeyInsight, InsightCategory(resp.Category))
	insight.Source = sourceID
	insight.Confidence = resp.Confidence
	insight.Evidence = buildEvidence(resp.ErrorIdentification, resp.RootCauseAnalysis, resp.CorrectApproach)
	insight.Iteration = 0
	insight.CreatedAt = time.Now()

	if err := r.validator.Validate(insight); err != nil {
		r.logger.DebugContext(ctx, "Insight validation failed", "error", err, "content", insight.Content)

		return []*Insight{}, nil
	}

	return []*Insight{insight}, nil
}

// buildEvidence collects non-empty evidence strings.
func buildEvidence(errorID, rootCause, correctApproach string) []string {
	evidence := make([]string, 0, 3)
	if errorID != "" && errorID != "N/A" {
		evidence = append(evidence, errorID)
	}

	if rootCause != "" {
		evidence = append(evidence, rootCause)
	}

	if correctApproach != "" {
		evidence = append(evidence, correctApproach)
	}

	return evidence
}

// parseSimplifiedFormat parses response as an array of simplified insights.
func (r *reflector) parseSimplifiedFormat(ctx context.Context, responseText, sourceID string) ([]*Insight, error) {
	type simplifiedInsight struct {
		Content    string   `json:"content"`
		Evidence   []string `json:"evidence"`
		Confidence float64  `json:"confidence"`
		Category   string   `json:"category"`
	}

	var simpleInsights []simplifiedInsight
	if err := json.Unmarshal([]byte(responseText), &simpleInsights); err != nil {
		r.logger.WarnContext(ctx, "Failed to parse reflection response in both formats", "error", err, "response", responseText)

		return nil, fmt.Errorf("failed to parse reflection response: %w", err)
	}

	insights := make([]*Insight, 0, len(simpleInsights))
	for _, simple := range simpleInsights {
		insight := NewInsight(simple.Content, InsightCategory(simple.Category))
		insight.Source = sourceID
		insight.Confidence = simple.Confidence
		insight.Evidence = simple.Evidence
		insight.Iteration = 0
		insight.CreatedAt = time.Now()

		if err := r.validator.Validate(insight); err != nil {
			r.logger.DebugContext(ctx, "Insight validation failed", "error", err, "content", insight.Content)

			continue
		}

		insights = append(insights, insight)
	}

	return insights, nil
}

// RefineInsights improves insights through multiple iterations.
func (r *reflector) RefineInsights(ctx context.Context, insights []*Insight, iterations int) ([]*Insight, error) {
	// Handle empty input.
	if len(insights) == 0 {
		return []*Insight{}, nil
	}

	// Start with current insights.
	current := make([]*Insight, len(insights))
	copy(current, insights)

	// Perform refinement iterations.
	for i := range iterations {
		refined, err := r.refineOnce(ctx, current, i+1)
		if err != nil {
			return nil, err
		}

		current = refined
	}

	return current, nil
}

// refineOnce performs one refinement iteration on insights.
func (r *reflector) refineOnce(ctx context.Context, insights []*Insight, iteration int) ([]*Insight, error) {
	// Build refinement prompt.
	prompt := r.promptBuilder.BuildRefinementPrompt(insights)

	// Call LLM.
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		Temperature: openai.F(reflectorTemperature),
	}

	// Set MaxTokens if configured.
	if r.maxTokens > 0 {
		params.MaxTokens = openai.F(int64(r.maxTokens))
	}

	completion, err := r.llm.Complete(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse JSON response.
	responseText := completion.Choices[0].Message.Content

	// Clean response text to extract JSON from markdown code blocks.
	responseText = cleanJSONResponse(responseText)

	// Parse refinement response as array of insights.
	type refinedInsightResponse struct {
		Content    string   `json:"content"`
		Evidence   []string `json:"evidence"`
		Confidence float64  `json:"confidence"`
		Category   string   `json:"category"`
	}

	var rawInsights []refinedInsightResponse

	err = json.Unmarshal([]byte(responseText), &rawInsights)
	if err != nil {
		r.logger.WarnContext(ctx, "Failed to parse refinement response", "error", err)

		return nil, fmt.Errorf("failed to parse refinement response: %w", err)
	}

	// Convert to Insight structs with updated iteration.
	refined := make([]*Insight, 0, len(rawInsights))
	for i, raw := range rawInsights {
		// Use source from original insight if available.
		source := ""
		if i < len(insights) {
			source = insights[i].Source
		}

		// Create insight using constructor.
		insight := NewInsight(raw.Content, InsightCategory(raw.Category))
		insight.Source = source
		insight.Confidence = raw.Confidence
		insight.Evidence = raw.Evidence
		insight.Iteration = iteration
		insight.CreatedAt = time.Now()
		refined = append(refined, insight)
	}

	return refined, nil
}

// cleanJSONResponse extracts JSON content from markdown code blocks.
// LLMs often wrap JSON responses in ```json ... ``` blocks, which need to be removed.
func cleanJSONResponse(response string) string {
	// Trim whitespace.
	response = strings.TrimSpace(response)

	// Check for markdown code block with ```json or just ```.
	if after, ok := strings.CutPrefix(response, "```json"); ok {
		// Remove ```json prefix and ``` suffix.
		response = after
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if afterPlain, okPlain := strings.CutPrefix(response, "```"); okPlain {
		// Remove ``` prefix and ``` suffix.
		response = afterPlain
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	return response
}
