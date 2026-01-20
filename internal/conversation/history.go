package conversation

import (
	"github.com/dmytrogajewski/spin/internal/context/summarizer"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/history/compress"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

// createHistory creates a new history instance with event emitter, compression, and summarization configured.
func (b *Builder) createHistory() *history.History {
	h := history.NewHistoryWithDefaults()
	h.SetEventEmitter(b.emitter)

	// Set up compression with classifier and compressor
	classifier := compress.NewClassifierWithOptions(
		compress.WithVerboseThreshold(1000),
	)
	compressor := compress.NewHybridCompressor(classifier, compress.DefaultCompressorConfig())

	// Set up LLM-based summarization if provider is available
	if b.llm != nil {
		tok := &tokenizer.SimpleTokenizer{}
		llmSummarizer := summarizer.NewLLMSummarizer(
			b.llm,
			tok,
			summarizer.DefaultLLMSummarizerConfig(),
		)

		// Wrap with caching for efficiency
		cache := summarizer.NewCache(summarizer.DefaultCacheConfig())
		cachingSummarizer := summarizer.NewCachingSummarizer(llmSummarizer, cache)

		compressor = compressor.WithSummarizer(cachingSummarizer)
	}

	h.SetCompressor(compressor)
	h.SetCompressionConfig(history.DefaultCompressionConfig())

	_ = h.AddSystemMessage("You are a helpful AI coding assistant.")
	return h
}
