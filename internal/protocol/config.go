package protocol

// Configuration types

// SandboxMode defines file access restrictions
type SandboxMode string

const (
	SandboxModeReadOnly         SandboxMode = "read_only"
	SandboxModeWorkspaceWrite   SandboxMode = "workspace_write"
	SandboxModeDangerFullAccess SandboxMode = "danger_full_access"
)

// ShellEnvironmentPolicy controls environment variable exposure
type ShellEnvironmentPolicy struct {
	IncludeOnly []string `json:"include_only,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
}

// ModelProviderConfig defines LLM provider configuration
type ModelProviderConfig struct {
	Name        string            `json:"name"`
	BaseURL     string            `json:"base_url"`
	APIKey      string            `json:"api_key,omitempty"`
	WireAPI     WireAPI           `json:"wire_api"`
	QueryParams map[string]string `json:"query_params,omitempty"`
}

// WireAPI specifies the API endpoint format
type WireAPI string

const (
	WireAPIChat      WireAPI = "chat"      // /v1/chat/completions (OpenAI-compatible)
	WireAPIResponses WireAPI = "responses" // /v1/responses (alternative format)
)

// Example configurations for different providers
var (
	// OllamaConfig is the default configuration for Ollama
	OllamaConfig = ModelProviderConfig{
		Name:    "ollama",
		BaseURL: "http://localhost:11434",
		WireAPI: WireAPIChat,
	}

	// LMStudioConfig is the default configuration for LM Studio
	LMStudioConfig = ModelProviderConfig{
		Name:    "lmstudio",
		BaseURL: "http://localhost:1234/v1",
		WireAPI: WireAPIChat,
	}

	// OpenAIConfig is the default configuration for OpenAI (for reference)
	OpenAIConfig = ModelProviderConfig{
		Name:    "openai",
		BaseURL: "https://api.openai.com",
		WireAPI: WireAPIChat,
	}
)
