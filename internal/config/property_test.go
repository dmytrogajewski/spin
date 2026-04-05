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

	rapid.Check(t, func(tcase *rapid.T) {
		cfg := genValidV2(tcase)

		yamlBytes, err := yaml.Marshal(cfg)
		if err != nil {
			tcase.Fatalf("failed to marshal config: %v", err)
		}

		var cfg2 V2
		if err = yaml.Unmarshal(yamlBytes, &cfg2); err != nil {
			tcase.Fatalf("failed to unmarshal config: %v", err)
		}

		assertConfigRoundTrip(tcase, cfg, &cfg2)
	})
}

// assertConfigRoundTrip verifies key fields match between two configs.
func assertConfigRoundTrip(tcase *rapid.T, cfg, cfg2 *V2) {
	tcase.Helper()

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

	for _, check := range checks {
		if check.got != check.want {
			tcase.Fatalf("%s mismatch: %v != %v", check.name, check.got, check.want)
		}
	}
}

// TestV2_Validation_GeneratedConfigs tests that generated configs pass validation.
// This ensures our validation logic is consistent with our data model.
func TestV2_Validation_GeneratedConfigs(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(tcase *rapid.T) {
		cfg := genValidV2(tcase)

		err := cfg.Validate()
		if err != nil {
			tcase.Fatalf("generated valid config failed validation: %v\nConfig: %+v", err, cfg)
		}
	})
}

// genValidV2 generates a random valid V2 using rapid.
func genValidV2(tcase *rapid.T) *V2 {
	return &V2{
		Version: "2.0",
		LLM:     genValidLLMConfig(tcase),
		Agent:   genValidAgentConfig(tcase),
		ACE:     genValidACEConfig(tcase),
		Security: SecurityV2{
			SandboxMode:     rapid.SampledFrom([]string{"none", "docker", "firejail"}).Draw(tcase, "sandbox_mode"),
			PolicyFile:      rapid.StringMatching(`^[-a-zA-Z0-9_./]*$`).Draw(tcase, "policy_file"),
			AllowedCommands: rapid.SliceOfN(rapid.StringMatching(`^[a-z]+$`), 0, 10).Draw(tcase, "allowed_commands"),
		},
		Protocol: genValidProtocolConfig(tcase),
	}
}

// genValidLLMConfig generates a random valid LLMV2.
func genValidLLMConfig(tcase *rapid.T) LLMV2 {
	timeoutSec := rapid.Int64Range(1, 3600).Draw(tcase, "llm_timeout_sec")

	return LLMV2{
		Provider:    rapid.SampledFrom([]string{"ollama", "openai", "anthropic"}).Draw(tcase, "llm_provider"),
		Model:       rapid.StringMatching(`^[a-z0-9.-]+$`).Draw(tcase, "llm_model"),
		Temperature: rapid.Float64Range(0, 2).Draw(tcase, "llm_temperature"),
		MaxTokens:   rapid.IntRange(1, 100000).Draw(tcase, "llm_max_tokens"),
		Timeout:     time.Duration(timeoutSec) * time.Second,
		BaseURL:     rapid.StringMatching(`^(https?://)?[-a-zA-Z0-9.:]*$`).Draw(tcase, "llm_base_url"),
		APIKey:      rapid.StringMatching(`^[a-zA-Z0-9_-]*$`).Draw(tcase, "llm_api_key"),
	}
}

// genValidAgentConfig generates a random valid AgentV2.
func genValidAgentConfig(tcase *rapid.T) AgentV2 {
	timeoutSec := rapid.Int64Range(1, 7200).Draw(tcase, "agent_timeout_sec")

	return AgentV2{
		MaxTurns:        rapid.IntRange(1, 1000).Draw(tcase, "agent_max_turns"),
		Timeout:         time.Duration(timeoutSec) * time.Second,
		WorkDir:         rapid.StringMatching(`^[/.][-a-zA-Z0-9_./]*$`).Draw(tcase, "agent_work_dir"),
		RequireApproval: rapid.Bool().Draw(tcase, "agent_require_approval"),
	}
}

// genValidACEConfig generates a random valid ACEV2.
func genValidACEConfig(tcase *rapid.T) ACEV2 {
	enabled := rapid.Bool().Draw(tcase, "ace_enabled")

	if !enabled {
		return ACEV2{
			Enabled: false,
		}
	}

	return ACEV2{
		Enabled:        true,
		PlaybookPath:   rapid.StringMatching(`^[/~][-a-zA-Z0-9_./]+\.json$`).Draw(tcase, "ace_playbook_path"),
		TrajectoryPath: rapid.StringMatching(`^[/~][-a-zA-Z0-9_./]+/?$`).Draw(tcase, "ace_trajectory_path"),
		TopK:           rapid.IntRange(1, 100).Draw(tcase, "ace_top_k"),
		MinScore:       rapid.Float64Range(0, 1).Draw(tcase, "ace_min_score"),
	}
}

// genValidProtocolConfig generates a random valid ProtocolV2.
func genValidProtocolConfig(tcase *rapid.T) ProtocolV2 {
	enableShell := rapid.Bool().Draw(tcase, "protocol_enable_shell")

	var shellTimeout time.Duration

	if enableShell {
		timeoutSec := rapid.Int64Range(1, 3600).Draw(tcase, "protocol_shell_timeout_sec")
		shellTimeout = time.Duration(timeoutSec) * time.Second
	} else {
		shellTimeout = 0
	}

	return ProtocolV2{
		EnableMCP:    rapid.Bool().Draw(tcase, "protocol_enable_mcp"),
		MCPServers:   []MCPServerConfigV2{}, // Keep simple for now.
		EnableGit:    rapid.Bool().Draw(tcase, "protocol_enable_git"),
		EnableShell:  enableShell,
		ShellTimeout: shellTimeout,
	}
}
