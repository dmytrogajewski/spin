package conversation

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/context/summarizer"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/history/compress"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

const (
	verboseThresholdTokens = 1000
	historyContextRatio    = 0.75
)

// createHistory creates a new history instance with event emitter, compression, and summarization configured.
func (b *Builder) createHistory(ctx context.Context) *history.History {
	// Use a reasonable default based on the LLM's max tokens.
	maxTokens := b.getHistoryMaxTokens()

	h := history.NewHistory(maxTokens, &tokenizer.SimpleTokenizer{})
	h.SetEventEmitter(b.emitter)

	// Set up compression with classifier and compressor.
	classifier := compress.NewClassifierWithOptions(
		compress.WithVerboseThreshold(verboseThresholdTokens),
	)
	compressor := compress.NewHybridCompressor(classifier, compress.DefaultCompressorConfig())

	// Set up LLM-based summarization if provider is available.
	if b.llm != nil {
		tok := &tokenizer.SimpleTokenizer{}
		llmSummarizer := summarizer.NewLLMSummarizer(
			b.llm,
			tok,
			summarizer.DefaultLLMSummarizerConfig(),
		)

		// Wrap with caching for efficiency.
		cache := summarizer.NewCache(summarizer.DefaultCacheConfig())
		cachingSummarizer := summarizer.NewCachingSummarizer(llmSummarizer, cache)

		compressor = compressor.WithSummarizer(cachingSummarizer)
	}

	h.SetCompressor(compressor)
	h.SetCompressionConfig(history.DefaultCompressionConfig())

	_ = h.AddSystemMessage(ctx, "You are a helpful AI coding assistant.")

	return h
}

// getHistoryMaxTokens determines appropriate max tokens for history based on LLM context window.
// Priority order:
//  1. Config context_window override (if set - for custom/fine-tuned models)
//  2. Provider's auto-detected context window (from Capabilities)
//  3. Default of 8192 tokens
//
// Note: LLM.MaxTokens is intentionally NOT used here - it's for generation limit,
// not context window. Providers should report context window via Capabilities().
func (b *Builder) getHistoryMaxTokens() int {
	const (
		defaultTokens = 8192
		minTokens     = 2048
	)

	var contextWindow int

	// Priority 1: Config override for custom/fine-tuned models.
	if b.cfg != nil && b.cfg.LLM.ContextWindow > 0 {
		contextWindow = b.cfg.LLM.ContextWindow
	}

	// Priority 2: Provider's auto-detected context window (primary mechanism).
	if contextWindow == 0 && b.llm != nil {
		caps := b.llm.Capabilities()
		if caps.ContextWindow > 0 {
			contextWindow = caps.ContextWindow
		}
	}

	// Priority 3: Default.
	if contextWindow == 0 {
		return defaultTokens
	}

	// Use 75% of context window for history (leave room for responses).
	historyTokens := int(float64(contextWindow) * historyContextRatio)

	if historyTokens < minTokens {
		historyTokens = defaultTokens
	}

	return historyTokens
}
