package config

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

// TestConfigV2_RoundTrip_YAML tests that ConfigV2 can be marshaled and unmarshaled without data loss.
// This is a property-based test that generates random valid configs.
func TestConfigV2_RoundTrip_YAML(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid ConfigV2.
		cfg := genValidConfigV2(t)

		// Marshal to YAML.
		yamlBytes, err := yaml.Marshal(cfg)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		// Unmarshal back.
		var cfg2 ConfigV2

		err = yaml.Unmarshal(yamlBytes, &cfg2)
		if err != nil {
			t.Fatalf("failed to unmarshal config: %v", err)
		}

		// Verify fields match.
		if cfg.Version != cfg2.Version {
			t.Fatalf("Version mismatch: %v != %v", cfg.Version, cfg2.Version)
		}

		if cfg.LLM.Provider != cfg2.LLM.Provider {
			t.Fatalf("LLM.Provider mismatch: %v != %v", cfg.LLM.Provider, cfg2.LLM.Provider)
		}

		if cfg.LLM.Model != cfg2.LLM.Model {
			t.Fatalf("LLM.Model mismatch: %v != %v", cfg.LLM.Model, cfg2.LLM.Model)
		}

		if cfg.LLM.Temperature != cfg2.LLM.Temperature {
			t.Fatalf("LLM.Temperature mismatch: %v != %v", cfg.LLM.Temperature, cfg2.LLM.Temperature)
		}

		if cfg.Agent.MaxTurns != cfg2.Agent.MaxTurns {
			t.Fatalf("Agent.MaxTurns mismatch: %v != %v", cfg.Agent.MaxTurns, cfg2.Agent.MaxTurns)
		}

		if cfg.ACE.Enabled != cfg2.ACE.Enabled {
			t.Fatalf("ACE.Enabled mismatch: %v != %v", cfg.ACE.Enabled, cfg2.ACE.Enabled)
		}
	})
}

// TestConfigV2_Validation_GeneratedConfigs tests that generated configs pass validation.
// This ensures our validation logic is consistent with our data model.
func TestConfigV2_Validation_GeneratedConfigs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := genValidConfigV2(t)

		err := cfg.Validate()
		if err != nil {
			t.Fatalf("generated valid config failed validation: %v\nConfig: %+v", err, cfg)
		}
	})
}

// genValidConfigV2 generates a random valid ConfigV2 using rapid.
func genValidConfigV2(t *rapid.T) *ConfigV2 {
	return &ConfigV2{
		Version: "2.0",
		LLM:     genValidLLMConfig(t),
		Agent:   genValidAgentConfig(t),
		ACE:     genValidACEConfig(t),
		Security: SecurityConfigV2{
			SandboxMode:     rapid.SampledFrom([]string{"none", "docker", "firejail"}).Draw(t, "sandbox_mode"),
			PolicyFile:      rapid.StringMatching(`^[-a-zA-Z0-9_./]*$`).Draw(t, "policy_file"),
			AllowedCommands: rapid.SliceOfN(rapid.StringMatching(`^[a-z]+$`), 0, 10).Draw(t, "allowed_commands"),
		},
		Protocol: genValidProtocolConfig(t),
	}
}

// genValidLLMConfig generates a random valid LLMConfigV2.
func genValidLLMConfig(t *rapid.T) LLMConfigV2 {
	timeoutSec := rapid.Int64Range(1, 3600).Draw(t, "llm_timeout_sec")

	return LLMConfigV2{
		Provider:    rapid.SampledFrom([]string{"ollama", "openai", "anthropic"}).Draw(t, "llm_provider"),
		Model:       rapid.StringMatching(`^[a-z0-9.-]+$`).Draw(t, "llm_model"),
		Temperature: rapid.Float64Range(0, 2).Draw(t, "llm_temperature"),
		MaxTokens:   rapid.IntRange(1, 100000).Draw(t, "llm_max_tokens"),
		Timeout:     time.Duration(timeoutSec) * time.Second,
		BaseURL:     rapid.StringMatching(`^(https?://)?[-a-zA-Z0-9.:]*$`).Draw(t, "llm_base_url"),
		APIKey:      rapid.StringMatching(`^[a-zA-Z0-9_-]*$`).Draw(t, "llm_api_key"),
	}
}

// genValidAgentConfig generates a random valid AgentConfigV2.
func genValidAgentConfig(t *rapid.T) AgentConfigV2 {
	timeoutSec := rapid.Int64Range(1, 7200).Draw(t, "agent_timeout_sec")

	return AgentConfigV2{
		MaxTurns:        rapid.IntRange(1, 1000).Draw(t, "agent_max_turns"),
		Timeout:         time.Duration(timeoutSec) * time.Second,
		WorkDir:         rapid.StringMatching(`^[/.][-a-zA-Z0-9_./]*$`).Draw(t, "agent_work_dir"),
		RequireApproval: rapid.Bool().Draw(t, "agent_require_approval"),
	}
}

// genValidACEConfig generates a random valid ACEConfigV2.
func genValidACEConfig(t *rapid.T) ACEConfigV2 {
	enabled := rapid.Bool().Draw(t, "ace_enabled")

	if !enabled {
		return ACEConfigV2{
			Enabled: false,
		}
	}

	return ACEConfigV2{
		Enabled:        true,
		PlaybookPath:   rapid.StringMatching(`^[/~][-a-zA-Z0-9_./]+\.json$`).Draw(t, "ace_playbook_path"),
		TrajectoryPath: rapid.StringMatching(`^[/~][-a-zA-Z0-9_./]+/?$`).Draw(t, "ace_trajectory_path"),
		TopK:           rapid.IntRange(1, 100).Draw(t, "ace_top_k"),
		MinScore:       rapid.Float64Range(0, 1).Draw(t, "ace_min_score"),
	}
}

// genValidProtocolConfig generates a random valid ProtocolConfigV2.
func genValidProtocolConfig(t *rapid.T) ProtocolConfigV2 {
	enableShell := rapid.Bool().Draw(t, "protocol_enable_shell")

	var shellTimeout time.Duration

	if enableShell {
		timeoutSec := rapid.Int64Range(1, 3600).Draw(t, "protocol_shell_timeout_sec")
		shellTimeout = time.Duration(timeoutSec) * time.Second
	} else {
		shellTimeout = 0
	}

	return ProtocolConfigV2{
		EnableMCP:    rapid.Bool().Draw(t, "protocol_enable_mcp"),
		MCPServers:   []MCPServerConfigV2{}, // Keep simple for now.
		EnableGit:    rapid.Bool().Draw(t, "protocol_enable_git"),
		EnableShell:  enableShell,
		ShellTimeout: shellTimeout,
	}
}
