// Package compress provides context compression for conversation history.
//
// The compression system offers multiple strategies for reducing conversation
// history to fit within token budgets while preserving critical information.
//
// Compression Strategies:
//
//   - CompositeCompressor: Primary + fallback (recommended for production)
//   - LLMSummarizer: Uses LLM to summarize old messages (best semantic preservation)
//   - HybridCompressor: Importance-weighted selection (fast, no LLM required)
//
// Key Components:
//   - Compressor: Interface for compression strategies
//   - MessageClassifier: Assigns importance to messages (4 levels)
//   - HybridCompressor: Greedy selection implementation
//   - LLMSummarizer: LLM-based summarization
//   - CompositeCompressor: Chains strategies with fallback
//
// Importance Levels:
//   - Critical: User messages, tool results, errors (100% retention)
//   - High: Code changes, decisions
//   - Medium: Regular assistant responses
//   - Low: Verbose reasoning, "thinking" content
//
// Recommended Usage (Composite with LLM + Hybrid):
//
//	// At history package level
//	history := history.NewHistoryWithLLMSummarization(16384, tokenizer, llmProvider, nil)
//
// Basic Usage (Hybrid only):
//
//	classifier := &MessageClassifier{}
//	compressor := &HybridCompressor{
//	    classifier: classifier,
//	    config: CompressorConfig{PreserveCritical: true},
//	}
//	compressed, _ := compressor.Compress(ctx, messages, 8000, tokenizer)
package compress
