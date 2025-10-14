package compress

import (
	"context"
	"fmt"
	"strings"
)

// LLMProvider is the minimal interface needed for summarization.
// In production, pass a real llm.Provider via LLMProviderAdapter.
type LLMProvider interface {
	Complete(ctx context.Context, req interface{}) (string, error)
}

// LLMSummarizer uses an LLM to summarize conversation history.
//
// Instead of simply removing messages, this compressor uses an LLM to
// generate concise summaries of older messages, preserving semantic
// information while reducing token count.
//
// Algorithm:
//  1. Separate critical messages (never summarize)
//  2. Keep recent messages verbatim (better context continuity)
//  3. Divide old non-critical messages into chunks
//  4. For each chunk, send to LLM for summarization
//  5. Replace chunk with single summary message
//  6. Combine: critical + summaries + recent
//  7. Fallback to hybrid compression if still over budget
//
// This provides higher semantic fidelity than simple selection-based
// compression, at the cost of LLM API calls.
type LLMSummarizer struct {
	llm          LLMProvider
	classifier   *MessageClassifier
	chunkSize    int     // Messages per chunk (default: 10)
	recentWindow int     // Recent messages to keep verbatim (default: 20)
	temperature  float64 // LLM temperature for summarization (default: 0.3)
	maxTokens    int     // Max tokens per summary (default: 200)
}

// LLMSummarizerConfig configures the LLM summarizer.
type LLMSummarizerConfig struct {
	ChunkSize    int
	RecentWindow int
	Temperature  float64
	MaxTokens    int
}

// DefaultLLMSummarizerConfig returns sensible defaults.
func DefaultLLMSummarizerConfig() LLMSummarizerConfig {
	return LLMSummarizerConfig{
		ChunkSize:    10,
		RecentWindow: 20,
		Temperature:  0.3, // Lower temperature for factual summarization
		MaxTokens:    200, // Compact summaries
	}
}

// NewLLMSummarizer creates a new LLM-based summarizer.
func NewLLMSummarizer(llm LLMProvider, config LLMSummarizerConfig) *LLMSummarizer {
	return &LLMSummarizer{
		llm:          llm,
		classifier:   &MessageClassifier{},
		chunkSize:    config.ChunkSize,
		recentWindow: config.RecentWindow,
		temperature:  config.Temperature,
		maxTokens:    config.MaxTokens,
	}
}

// NewDefaultLLMSummarizer creates a summarizer with default configuration.
func NewDefaultLLMSummarizer(llm LLMProvider) *LLMSummarizer {
	return NewLLMSummarizer(llm, DefaultLLMSummarizerConfig())
}

// Compress implements the Compressor interface using LLM-based summarization.
func (s *LLMSummarizer) Compress(
	ctx context.Context,
	messages []CompressibleMessage,
	targetTokens int,
	tokenizer Tokenizer,
) ([]CompressibleMessage, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Calculate current token count
	currentTokens := 0
	for _, msg := range messages {
		currentTokens += msg.Tokens
	}

	// If already under target, no compression needed
	if currentTokens <= targetTokens {
		return messages, nil
	}

	// Step 1: Separate critical messages (never summarize)
	critical := make([]CompressibleMessage, 0)
	nonCritical := make([]CompressibleMessage, 0)

	for _, msg := range messages {
		if s.classifier.Classify(msg) == ImportanceCritical {
			critical = append(critical, msg)
		} else {
			nonCritical = append(nonCritical, msg)
		}
	}

	// Step 2: Keep recent non-critical messages verbatim
	var recentMsgs []CompressibleMessage
	var summarizableMsgs []CompressibleMessage

	if len(nonCritical) > s.recentWindow {
		summarizableMsgs = nonCritical[:len(nonCritical)-s.recentWindow]
		recentMsgs = nonCritical[len(nonCritical)-s.recentWindow:]
	} else {
		recentMsgs = nonCritical
		summarizableMsgs = []CompressibleMessage{}
	}

	// Step 3: Summarize older non-critical messages in chunks
	summaries := make([]CompressibleMessage, 0)

	for i := 0; i < len(summarizableMsgs); i += s.chunkSize {
		end := i + s.chunkSize
		if end > len(summarizableMsgs) {
			end = len(summarizableMsgs)
		}

		chunk := summarizableMsgs[i:end]
		summary, err := s.summarizeChunk(ctx, chunk, tokenizer)
		if err != nil {
			// On error, fallback to keeping original messages
			summaries = append(summaries, chunk...)
			continue
		}

		summaries = append(summaries, summary)
	}

	// Step 4: Combine critical + summaries + recent
	result := make([]CompressibleMessage, 0, len(critical)+len(summaries)+len(recentMsgs))
	result = append(result, critical...)
	result = append(result, summaries...)
	result = append(result, recentMsgs...)

	// Step 5: If still over budget, fallback to hybrid compression
	resultTokens := 0
	for _, msg := range result {
		resultTokens += msg.Tokens
	}

	if resultTokens > targetTokens {
		// Fallback: use hybrid compressor
		hybrid := NewDefaultHybridCompressor()
		return hybrid.Compress(ctx, result, targetTokens, tokenizer)
	}

	return result, nil
}

// summarizeChunk summarizes a chunk of messages using the LLM.
func (s *LLMSummarizer) summarizeChunk(ctx context.Context, chunk []CompressibleMessage, tokenizer Tokenizer) (CompressibleMessage, error) {
	// Build prompt
	prompt := s.buildSummarizationPrompt(chunk)

	// Call LLM (using minimal interface to avoid import cycles)
	content, err := s.llm.Complete(ctx, prompt)
	if err != nil {
		return CompressibleMessage{}, fmt.Errorf("LLM summarization failed: %w", err)
	}

	// Calculate tokens for summary
	summaryTokens := tokenizer.Count(content) + 4 // Add message overhead

	// Create summary message
	summary := CompressibleMessage{
		ID:            fmt.Sprintf("summary-%d-%d", len(chunk), summaryTokens),
		Role:          RoleAssistant,
		Content:       "Summary: " + content,
		ToolCallCount: 0,
		Tokens:        summaryTokens,
	}

	return summary, nil
}

// buildSummarizationPrompt creates a prompt for summarizing a chunk of messages.
func (s *LLMSummarizer) buildSummarizationPrompt(chunk []CompressibleMessage) string {
	var b strings.Builder

	b.WriteString("Summarize the following conversation segment concisely, preserving key facts, decisions, and context.\n")
	b.WriteString("Be factual and avoid unnecessary details.\n\n")

	for i, msg := range chunk {
		b.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
		if i < len(chunk)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// Name returns the compressor strategy name.
func (s *LLMSummarizer) Name() string {
	return "llm-summary"
}
