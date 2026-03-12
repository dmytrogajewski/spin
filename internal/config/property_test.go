package config

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

// TestV2_RoundTrip_YAML tests that V2 can be marshaled and unmarshaled without data loss.
// This is a property-based test that generates random valid configs.
func TestV2_RoundTrip_YAML(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		cfg := genValidV2(t)

		yamlBytes, err := yaml.Marshal(cfg)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		var cfg2 V2
		if err = yaml.Unmarshal(yamlBytes, &cfg2); err != nil {
			t.Fatalf("failed to unmarshal config: %v", err)
		}

		assertConfigRoundTrip(t, cfg, &cfg2)
	})
}

// assertConfigRoundTrip verifies key fields match between two configs.
func assertConfigRoundTrip(t *rapid.T, cfg, cfg2 *V2) {
	t.Helper()

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Version", cfg2.Version, cfg.Version},
		{"LLM.Provider", cfg2.LLM.Provider, cfg.LLM.Provider},
		{"LLM.Model", cfg2.LLM.Model, cfg.LLM.Model},
		{"LLM.Temperature", cfg2.LLM.Temperature, cfg.LLM.Temperature},
		{"Agent.MaxTurns", cfg2.Agent.MaxTurns, cfg.Agent.MaxTurns},
		{"ACE.Enabled", cfg2.ACE.Enabled, cfg.ACE.Enabled},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Fatalf("%s mismatch: %v != %v", c.name, c.got, c.want)
		}
	}
}

// TestV2_Validation_GeneratedConfigs tests that generated configs pass validation.
// This ensures our validation logic is consistent with our data model.
func TestV2_Validation_GeneratedConfigs(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		cfg := genValidV2(t)

		err := cfg.Validate()
		if err != nil {
			t.Fatalf("generated valid config failed validation: %v\nConfig: %+v", err, cfg)
		}
	})
}

// genValidV2 generates a random valid V2 using rapid.
func genValidV2(t *rapid.T) *V2 {
	return &V2{
		Version: "2.0",
		LLM:     genValidLLMConfig(t),
		Agent:   genValidAgentConfig(t),
		ACE:     genValidACEConfig(t),
		Security: SecurityV2{
			SandboxMode:     rapid.SampledFrom([]string{"none", "docker", "firejail"}).Draw(t, "sandbox_mode"),
			PolicyFile:      rapid.StringMatching(`^[-a-zA-Z0-9_./]*$`).Draw(t, "policy_file"),
			AllowedCommands: rapid.SliceOfN(rapid.StringMatching(`^[a-z]+$`), 0, 10).Draw(t, "allowed_commands"),
		},
		Protocol: genValidProtocolConfig(t),
	}
}

// genValidLLMConfig generates a random valid LLMV2.
func genValidLLMConfig(t *rapid.T) LLMV2 {
	timeoutSec := rapid.Int64Range(1, 3600).Draw(t, "llm_timeout_sec")

	return LLMV2{
		Provider:    rapid.SampledFrom([]string{"ollama", "openai", "anthropic"}).Draw(t, "llm_provider"),
		Model:       rapid.StringMatching(`^[a-z0-9.-]+$`).Draw(t, "llm_model"),
		Temperature: rapid.Float64Range(0, 2).Draw(t, "llm_temperature"),
		MaxTokens:   rapid.IntRange(1, 100000).Draw(t, "llm_max_tokens"),
		Timeout:     time.Duration(timeoutSec) * time.Second,
		BaseURL:     rapid.StringMatching(`^(https?://)?[-a-zA-Z0-9.:]*$`).Draw(t, "llm_base_url"),
		APIKey:      rapid.StringMatching(`^[a-zA-Z0-9_-]*$`).Draw(t, "llm_api_key"),
	}
}

// genValidAgentConfig generates a random valid AgentV2.
func genValidAgentConfig(t *rapid.T) AgentV2 {
	timeoutSec := rapid.Int64Range(1, 7200).Draw(t, "agent_timeout_sec")

	return AgentV2{
		MaxTurns:        rapid.IntRange(1, 1000).Draw(t, "agent_max_turns"),
		Timeout:         time.Duration(timeoutSec) * time.Second,
		WorkDir:         rapid.StringMatching(`^[/.][-a-zA-Z0-9_./]*$`).Draw(t, "agent_work_dir"),
		RequireApproval: rapid.Bool().Draw(t, "agent_require_approval"),
	}
}

// genValidACEConfig generates a random valid ACEV2.
func genValidACEConfig(t *rapid.T) ACEV2 {
	enabled := rapid.Bool().Draw(t, "ace_enabled")

	if !enabled {
		return ACEV2{
			Enabled: false,
		}
	}

	return ACEV2{
		Enabled:        true,
		PlaybookPath:   rapid.StringMatching(`^[/~][-a-zA-Z0-9_./]+\.json$`).Draw(t, "ace_playbook_path"),
		TrajectoryPath: rapid.StringMatching(`^[/~][-a-zA-Z0-9_./]+/?$`).Draw(t, "ace_trajectory_path"),
		TopK:           rapid.IntRange(1, 100).Draw(t, "ace_top_k"),
		MinScore:       rapid.Float64Range(0, 1).Draw(t, "ace_min_score"),
	}
}

// genValidProtocolConfig generates a random valid ProtocolV2.
func genValidProtocolConfig(t *rapid.T) ProtocolV2 {
	enableShell := rapid.Bool().Draw(t, "protocol_enable_shell")

	var shellTimeout time.Duration

	if enableShell {
		timeoutSec := rapid.Int64Range(1, 3600).Draw(t, "protocol_shell_timeout_sec")
		shellTimeout = time.Duration(timeoutSec) * time.Second
	} else {
		shellTimeout = 0
	}

	return ProtocolV2{
		EnableMCP:    rapid.Bool().Draw(t, "protocol_enable_mcp"),
		MCPServers:   []MCPServerConfigV2{}, // Keep simple for now.
		EnableGit:    rapid.Bool().Draw(t, "protocol_enable_git"),
		EnableShell:  enableShell,
		ShellTimeout: shellTimeout,
	}
}
