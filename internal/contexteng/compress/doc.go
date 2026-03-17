// Package compress provides context compression for conversation history.
//
// The compression system uses importance-weighted message selection to
// preserve critical information while staying within token budgets.
// It optionally integrates with the summarizer package to create
// semantic summaries of removed content.
//
// Key Components:
//   - Compressor: Interface for compression strategies
//   - MessageClassifier: Assigns importance levels to messages
//   - HybridCompressor: Greedy selection with optional LLM summarization
//
// Importance Levels:
//   - Critical: User messages, tool results, errors (100% retention)
//   - High: Code changes, decisions
//   - Medium: Regular assistant responses
//   - Low: Verbose reasoning, "thinking" content
//
// Example:
//
//	classifier := compress.NewClassifier()
//	compressor := compress.NewHybridCompressor(classifier, compress.CompressorConfig{
//	    PreserveCritical: true,
//	    MinRetention:     0.3,
//	})
//	compressed, err := compressor.Compress(ctx, messages, 8000, tokenizer)
package compress
