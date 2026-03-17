// Package summarizer provides progressive context summarization capabilities.
//
// This package implements intelligent summarization for conversation history
// and content, enabling longer effective conversations within token limits.
// Instead of simply removing content when approaching token limits, summarizer
// condenses older messages while preserving essential information.
//
// # Core Components
//
// The package provides several components:
//
//   - Summarizer: Core interface for content summarization
//   - LLMSummarizer: LLM-based implementation using any llm.Provider
//   - Cache: LRU cache with TTL for efficient summary retrieval
//   - CachingSummarizer: Decorator that adds caching to any Summarizer
//
// # Usage
//
// Basic usage with LLM summarizer:
//
//	summarizer := summarizer.NewLLMSummarizer(
//	    provider,
//	    tokenizer,
//	    summarizer.DefaultLLMSummarizerConfig(),
//	)
//
//	result, err := summarizer.Summarize(ctx, content, summarizer.Options{
//	    MaxTokens:   500,
//	    TargetRatio: 0.3,
//	    Style:       summarizer.StyleNarrative,
//	})
//
// With caching:
//
//	cache := summarizer.NewCache(summarizer.DefaultCacheConfig())
//	cached := summarizer.NewCachingSummarizer(summarizer, cache)
//
// # Summary Styles
//
// The package supports multiple summary styles:
//
//   - StyleBrief: Minimal, key points only
//   - StyleDetailed: More context preserved
//   - StyleBullet: Bullet point format
//   - StyleNarrative: Flowing narrative (default)
//
// # Thread Safety
//
// All implementations in this package are safe for concurrent use.
package summarizer
