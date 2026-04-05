package llmutil

// Context window size constants for known LLM models.
const (
	contextWindow4K   = 4096
	contextWindow8K   = 8192
	contextWindow16K  = 16385
	contextWindow32K  = 32768
	contextWindow64K  = 64000
	contextWindow65K  = 65536
	contextWindow128K = 128000
	contextWindow131K = 131072
	contextWindow200K = 200000
	contextWindow1M   = 1000000
)

// knownContextWindows maps model names to their context window sizes.
var knownContextWindows = map[string]int{
	// OpenAI GPT-4 models.
	"gpt-4":                  contextWindow8K,
	"gpt-4-32k":              contextWindow32K,
	"gpt-4-turbo":            contextWindow128K,
	"gpt-4-turbo-preview":    contextWindow128K,
	"gpt-4-0125-preview":     contextWindow128K,
	"gpt-4-1106-preview":     contextWindow128K,
	"gpt-4o":                 contextWindow128K,
	"gpt-4o-mini":            contextWindow128K,
	"gpt-4o-2024-05-13":      contextWindow128K,
	"gpt-4o-2024-08-06":      contextWindow128K,
	"gpt-4o-2024-11-20":      contextWindow128K,
	"gpt-4o-mini-2024-07-18": contextWindow128K,
	"gpt-4.1":                contextWindow1M,
	"gpt-4.1-mini":           contextWindow1M,
	"gpt-4.1-nano":           contextWindow1M,
	"o1":                     contextWindow200K,
	"o1-mini":                contextWindow128K,
	"o1-preview":             contextWindow128K,
	"o3":                     contextWindow200K,
	"o3-mini":                contextWindow200K,
	"o4-mini":                contextWindow200K,

	// OpenAI GPT-3.5 models.
	"gpt-3.5-turbo":          contextWindow16K,
	"gpt-3.5-turbo-16k":      contextWindow16K,
	"gpt-3.5-turbo-0125":     contextWindow16K,
	"gpt-3.5-turbo-1106":     contextWindow16K,
	"gpt-3.5-turbo-instruct": contextWindow4K,

	// Anthropic Claude models (for OpenAI-compatible endpoints).
	"claude-3-opus":            contextWindow200K,
	"claude-3-opus-20240229":   contextWindow200K,
	"claude-3-sonnet":          contextWindow200K,
	"claude-3-sonnet-20240229": contextWindow200K,
	"claude-3-haiku":           contextWindow200K,
	"claude-3-haiku-20240307":  contextWindow200K,
	"claude-3.5-sonnet":        contextWindow200K,
	"claude-3-5-sonnet":        contextWindow200K,
	"claude-3.5-haiku":         contextWindow200K,
	"claude-3-5-haiku":         contextWindow200K,
	"claude-sonnet-4":          contextWindow200K,
	"claude-opus-4":            contextWindow200K,

	// DeepSeek models.
	"deepseek-chat":     contextWindow64K,
	"deepseek-coder":    contextWindow64K,
	"deepseek-r1":       contextWindow64K,
	"deepseek-v3":       contextWindow64K,
	"deepseek-v2":       contextWindow128K,
	"deepseek-v2.5":     contextWindow128K,
	"deepseek-reasoner": contextWindow64K,

	// Google Gemini models (for OpenAI-compatible endpoints).
	"gemini-pro":       contextWindow32K,
	"gemini-1.5-pro":   contextWindow1M,
	"gemini-1.5-flash": contextWindow1M,
	"gemini-2.0-flash": contextWindow1M,
	"gemini-2.0-pro":   contextWindow1M,
	"gemini-2.5-pro":   contextWindow1M,
	"gemini-2.5-flash": contextWindow1M,

	// Mistral models.
	"mistral-tiny":       contextWindow32K,
	"mistral-small":      contextWindow32K,
	"mistral-medium":     contextWindow32K,
	"mistral-large":      contextWindow128K,
	"mistral-nemo":       contextWindow128K,
	"codestral":          contextWindow32K,
	"codestral-latest":   contextWindow32K,
	"open-mistral-7b":    contextWindow32K,
	"open-mixtral-8x7b":  contextWindow32K,
	"open-mixtral-8x22b": contextWindow65K,

	// Groq-hosted models.
	"llama3-8b-8192":     contextWindow8K,
	"llama3-70b-8192":    contextWindow8K,
	"llama-3.1-8b":       contextWindow131K,
	"llama-3.1-70b":      contextWindow131K,
	"llama-3.1-405b":     contextWindow131K,
	"llama-3.2-1b":       contextWindow131K,
	"llama-3.2-3b":       contextWindow131K,
	"llama-3.2-11b":      contextWindow131K,
	"llama-3.2-90b":      contextWindow131K,
	"llama-3.3-70b":      contextWindow131K,
	"mixtral-8x7b-32768": contextWindow32K,
	"gemma-7b-it":        contextWindow8K,
	"gemma2-9b-it":       contextWindow8K,

	// Qwen models.
	"qwen-turbo":    contextWindow8K,
	"qwen-plus":     contextWindow32K,
	"qwen-max":      contextWindow32K,
	"qwen2-72b":     contextWindow131K,
	"qwen2.5-72b":   contextWindow131K,
	"qwen2.5-coder": contextWindow131K,
}

// ModelContextWindow returns the context window size for a known model.
// Returns 0 if the model is not recognized (callers should use a sensible default).
func ModelContextWindow(model string) int {
	if ctxLen, ok := knownContextWindows[model]; ok {
		return ctxLen
	}

	return 0
}
