package summarizer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/pkg/tokenizer"
)

const (
	defaultSummarizerTimeout     = 10 * time.Second
	defaultSummarizerMaxTokens   = 500
	defaultSummarizerTargetRatio = 0.3
	percentConversion            = 100
)

// LLMSummarizerConfig configures the LLM summarizer.
type LLMSummarizerConfig struct {
	// Model is the LLM model to use for summarization.
	Model string

	// Timeout is the maximum time for a summarization request.
	Timeout time.Duration

	// DefaultMaxTokens is the default target token count.
	DefaultMaxTokens int

	// DefaultTargetRatio is the default compression ratio.
	DefaultTargetRatio float64

	// DefaultStyle is the default summary style.
	DefaultStyle SummaryStyle
}

// DefaultLLMSummarizerConfig returns sensible default configuration.
func DefaultLLMSummarizerConfig() LLMSummarizerConfig {
	return LLMSummarizerConfig{
		Model:              "gpt-4o-mini",
		Timeout:            defaultSummarizerTimeout,
		DefaultMaxTokens:   defaultSummarizerMaxTokens,
		DefaultTargetRatio: defaultSummarizerTargetRatio,
		DefaultStyle:       StyleNarrative,
	}
}

// LLMSummarizer implements Summarizer using an LLM provider.
type LLMSummarizer struct {
	provider  llm.Provider
	tokenizer tokenizer.Tokenizer
	config    LLMSummarizerConfig
}

// NewLLMSummarizer creates a new LLM-based summarizer.
func NewLLMSummarizer(provider llm.Provider, tok tokenizer.Tokenizer, config LLMSummarizerConfig) *LLMSummarizer {
	if tok == nil {
		tok = &tokenizer.SimpleTokenizer{}
	}

	return &LLMSummarizer{
		provider:  provider,
		tokenizer: tok,
		config:    config,
	}
}

// Summarize implements Summarizer.Summarize.
func (s *LLMSummarizer) Summarize(ctx context.Context, content string, opts Options) (*Result, error) {
	if content == "" {
		return &Result{
			Summary:          "",
			OriginalTokens:   0,
			SummaryTokens:    0,
			CompressionRatio: 1.0,
		}, nil
	}

	// Apply defaults.
	opts = s.applyDefaults(opts)

	// Count original tokens.
	originalTokens := s.tokenizer.Count(content)

	// Build prompt.
	prompt := s.buildPrompt(content, opts)

	// Apply timeout.
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Call LLM.
	params := openai.ChatCompletionNewParams{
		Model: s.config.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		MaxTokens: openai.Int(int64(opts.MaxTokens)),
	}

	completion, err := s.provider.Complete(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	// Extract summary from response.
	summary := ""
	if len(completion.Choices) > 0 {
		summary = completion.Choices[0].Message.Content
	}

	summaryTokens := s.tokenizer.Count(summary)

	compressionRatio := 0.0
	if originalTokens > 0 {
		compressionRatio = float64(summaryTokens) / float64(originalTokens)
	}

	return &Result{
		Summary:          summary,
		OriginalTokens:   originalTokens,
		SummaryTokens:    summaryTokens,
		CompressionRatio: compressionRatio,
		PreservedItems:   opts.PreserveList,
	}, nil
}

func (s *LLMSummarizer) applyDefaults(opts Options) Options {
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = s.config.DefaultMaxTokens
	}

	if opts.TargetRatio <= 0 {
		opts.TargetRatio = s.config.DefaultTargetRatio
	}

	if opts.Style == "" {
		opts.Style = s.config.DefaultStyle
	}

	return opts
}

func (s *LLMSummarizer) buildPrompt(content string, opts Options) string {
	styleGuide := s.getStyleGuide(opts.Style)

	preserveGuide := ""
	if len(opts.PreserveList) > 0 {
		preserveGuide = fmt.Sprintf("\nPreserve these items verbatim: %s", strings.Join(opts.PreserveList, ", "))
	}

	return fmt.Sprintf(`Summarize the following content concisely while preserving all essential information.

Target: approximately %d tokens (%d%% of original)
Style: %s
%s%s

Content:
---
%s
---

Summary:`,
		opts.MaxTokens,
		int(opts.TargetRatio*percentConversion),
		opts.Style,
		styleGuide,
		preserveGuide,
		content)
}

func (s *LLMSummarizer) getStyleGuide(style SummaryStyle) string {
	switch style {
	case StyleBrief:
		return "\nUse minimal words. Include only critical points."
	case StyleDetailed:
		return "\nPreserve important context and nuance."
	case StyleBullet:
		return "\nFormat as bullet points. One point per key item."
	case StyleNarrative:
		return "\nWrite as flowing prose. Maintain logical flow."
	default:
		return ""
	}
}

// SummarizeMessages implements Summarizer.SummarizeMessages.
func (s *LLMSummarizer) SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error) {
	if len(messages) == 0 {
		return &MessageResult{
			Summary: message.Message{
				Role:    message.RoleAssistant,
				Content: "",
			},
			OriginalCount:   0,
			SummarizedRange: [2]int{0, 0},
		}, nil
	}

	// Apply defaults.
	opts = s.applyDefaults(opts)

	// Format messages for summarization.
	formatted := s.formatMessages(messages)

	// Build specialized prompt for messages.
	prompt := s.buildMessagePrompt(formatted, opts, len(messages))

	// Apply timeout.
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Call LLM.
	params := openai.ChatCompletionNewParams{
		Model: s.config.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		MaxTokens: openai.Int(int64(opts.MaxTokens)),
	}

	completion, err := s.provider.Complete(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("message summarization failed: %w", err)
	}

	// Extract summary from response.
	summaryContent := ""
	if len(completion.Choices) > 0 {
		summaryContent = completion.Choices[0].Message.Content
	}

	return &MessageResult{
		Summary: message.Message{
			Role:    message.RoleAssistant,
			Content: fmt.Sprintf("[Summary of previous %d messages]\n%s", len(messages), summaryContent),
		},
		OriginalCount:   len(messages),
		SummarizedRange: [2]int{0, len(messages) - 1},
	}, nil
}

func (s *LLMSummarizer) formatMessages(messages []message.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		fmt.Fprintf(&sb, "[%s]: %s\n", msg.Role, msg.Content)
	}

	return sb.String()
}

func (s *LLMSummarizer) buildMessagePrompt(formatted string, opts Options, count int) string {
	return fmt.Sprintf(`Summarize this conversation segment while preserving:
1. All decisions made
2. Actions taken and their outcomes
3. Current state/context
4. Unresolved questions or tasks

Target: approximately %d tokens
Format: Single coherent narrative

Conversation (%d messages):
---
%s
---

Summary (as a single message capturing the above):`,
		opts.MaxTokens,
		count,
		formatted)
}
