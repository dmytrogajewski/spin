package reflector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/openai/openai-go"
)

// Reflector analyzes trajectories to extract insights.
type Reflector interface {
	// Reflect analyzes trajectories and extracts insights
	Reflect(ctx context.Context, req ReflectionRequest) (*ReflectionResponse, error)

	// RefineInsights improves insights through multiple iterations
	RefineInsights(ctx context.Context, insights []*Insight, iterations int) ([]*Insight, error)
}

// reflector implements Reflector interface.
type reflector struct {
	llm           llm.Provider
	promptBuilder *PromptBuilder
	validator     *InsightValidator
}

// NewReflector creates a new reflector.
func NewReflector(llmProvider llm.Provider) Reflector {
	return &reflector{
		llm:           llmProvider,
		promptBuilder: NewPromptBuilder(),
		validator:     NewInsightValidator(),
	}
}

// Reflect analyzes trajectories and extracts insights.
func (r *reflector) Reflect(ctx context.Context, req ReflectionRequest) (*ReflectionResponse, error) {
	startTime := time.Now()

	slog.Debug("Reflector starting analysis", "num_trajectories", len(req.Trajectories))

	// Handle empty trajectory list
	if len(req.Trajectories) == 0 {
		slog.Debug("No trajectories to reflect on")
		return &ReflectionResponse{
			Insights:    []*Insight{},
			Iterations:  0,
			TotalTokens: 0,
			Duration:    time.Since(startTime),
		}, nil
	}

	// Build prompt based on number of trajectories
	var prompt string
	var sourceID string

	if len(req.Trajectories) == 1 {
		// Single trajectory
		traj := req.Trajectories[0]
		slog.Debug("Building prompt for single trajectory",
			"id", traj.ID,
			"query", traj.Query,
			"steps", len(traj.Steps),
			"success", traj.Success)
		prompt = r.promptBuilder.BuildSingleTrajectory(traj)
		sourceID = traj.ID
	} else {
		// Multiple trajectories - batch analysis
		slog.Debug("Building prompt for batch trajectories", "count", len(req.Trajectories))
		prompt = r.promptBuilder.BuildBatchTrajectory(req.Trajectories)
		sourceID = "batch"
	}

	slog.Debug("Reflector prompt generated", "length", len(prompt))
	slog.Debug("Reflector prompt content", "prompt", prompt)

	// Call LLM
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		Temperature: openai.F(0.3),
	}

	slog.Debug("Calling LLM for reflection", "temperature", 0.3)

	completion, err := r.llm.Complete(ctx, params)
	if err != nil {
		slog.Warn("LLM call failed during reflection", "error", err)
		return nil, err
	}

	// Parse JSON response into insights
	responseText := completion.Choices[0].Message.Content

	slog.Debug("LLM response received",
		"length", len(responseText),
		"tokens", completion.Usage.TotalTokens)
	slog.Debug("LLM response content", "response", responseText)

	// Clean response text to extract JSON from markdown code blocks
	responseText = cleanJSONResponse(responseText)

	// Try to parse as detailed reflection response (object) first
	var reflectionResp struct {
		Reasoning           string  `json:"reasoning"`
		ErrorIdentification string  `json:"error_identification"`
		RootCauseAnalysis   string  `json:"root_cause_analysis"`
		CorrectApproach     string  `json:"correct_approach"`
		KeyInsight          string  `json:"key_insight"`
		Category            string  `json:"category"`
		Confidence          float64 `json:"confidence"`
	}

	insights := make([]*Insight, 0, 1)

	if err := json.Unmarshal([]byte(responseText), &reflectionResp); err == nil && reflectionResp.KeyInsight != "" {
		// Successfully parsed as detailed object format
		insight := NewInsight(reflectionResp.KeyInsight, InsightCategory(reflectionResp.Category))
		insight.Source = sourceID
		insight.Confidence = reflectionResp.Confidence

		// Build evidence from the analysis
		evidence := make([]string, 0, 3)
		if reflectionResp.ErrorIdentification != "" && reflectionResp.ErrorIdentification != "N/A" {
			evidence = append(evidence, reflectionResp.ErrorIdentification)
		}
		if reflectionResp.RootCauseAnalysis != "" {
			evidence = append(evidence, reflectionResp.RootCauseAnalysis)
		}
		if reflectionResp.CorrectApproach != "" {
			evidence = append(evidence, reflectionResp.CorrectApproach)
		}
		insight.Evidence = evidence
		insight.Iteration = 0
		insight.CreatedAt = time.Now()

		// Validate insight before adding
		if err := r.validator.Validate(insight); err == nil {
			insights = append(insights, insight)
		} else {
			slog.Debug("Insight validation failed", "error", err, "content", insight.Content)
		}
	} else {
		// Try parsing as simplified array format (for compatibility with tests/simple responses)
		type simplifiedInsight struct {
			Content    string   `json:"content"`
			Evidence   []string `json:"evidence"`
			Confidence float64  `json:"confidence"`
			Category   string   `json:"category"`
		}

		var simpleInsights []simplifiedInsight
		if err := json.Unmarshal([]byte(responseText), &simpleInsights); err != nil {
			slog.Warn("Failed to parse reflection response in both formats", "error", err, "response", responseText)
			return nil, fmt.Errorf("failed to parse reflection response: %w", err)
		}

		// Convert simplified insights
		for _, simple := range simpleInsights {
			insight := NewInsight(simple.Content, InsightCategory(simple.Category))
			insight.Source = sourceID
			insight.Confidence = simple.Confidence
			insight.Evidence = simple.Evidence
			insight.Iteration = 0
			insight.CreatedAt = time.Now()

			// Validate insight before adding
			if err := r.validator.Validate(insight); err == nil {
				insights = append(insights, insight)
			} else {
				slog.Debug("Insight validation failed", "error", err, "content", insight.Content)
			}
		}
	}

	return &ReflectionResponse{
		Insights:    insights,
		Iterations:  1,
		TotalTokens: int(completion.Usage.TotalTokens),
		Duration:    time.Since(startTime),
	}, nil
}

// RefineInsights improves insights through multiple iterations.
func (r *reflector) RefineInsights(ctx context.Context, insights []*Insight, iterations int) ([]*Insight, error) {
	// Handle empty input
	if len(insights) == 0 {
		return []*Insight{}, nil
	}

	// Start with current insights
	current := make([]*Insight, len(insights))
	copy(current, insights)

	// Perform refinement iterations
	for i := 0; i < iterations; i++ {
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
	// Build refinement prompt
	prompt := r.promptBuilder.BuildRefinementPrompt(insights)

	// Call LLM
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		Temperature: openai.F(0.3),
	}

	completion, err := r.llm.Complete(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	responseText := completion.Choices[0].Message.Content

	// Clean response text to extract JSON from markdown code blocks
	responseText = cleanJSONResponse(responseText)

	// Parse refinement response as array of insights
	type refinedInsightResponse struct {
		Content    string   `json:"content"`
		Evidence   []string `json:"evidence"`
		Confidence float64  `json:"confidence"`
		Category   string   `json:"category"`
	}

	var rawInsights []refinedInsightResponse
	if err := json.Unmarshal([]byte(responseText), &rawInsights); err != nil {
		slog.Warn("Failed to parse refinement response", "error", err)
		return nil, fmt.Errorf("failed to parse refinement response: %w", err)
	}

	// Convert to Insight structs with updated iteration
	refined := make([]*Insight, 0, len(rawInsights))
	for i, raw := range rawInsights {
		// Use source from original insight if available
		source := ""
		if i < len(insights) {
			source = insights[i].Source
		}

		// Create insight using constructor
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
	// Trim whitespace
	response = strings.TrimSpace(response)

	// Check for markdown code block with ```json or just ```
	if strings.HasPrefix(response, "```json") {
		// Remove ```json prefix and ``` suffix
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		// Remove ``` prefix and ``` suffix
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	return response
}
