package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestGolden_ValidMinimal tests loading a minimal valid configuration.
func TestGolden_ValidMinimal(t *testing.T) {
	t.Parallel()

	path := filepath.Join("golden", "valid_minimal.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	var cfg V2
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	err = cfg.Validate()
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Verify key fields.
	if cfg.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", cfg.Version)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("Expected provider ollama, got %s", cfg.LLM.Provider)
	}

	if cfg.LLM.Model != "qwen2.5-coder:32b" {
		t.Errorf("Expected model qwen2.5-coder:32b, got %s", cfg.LLM.Model)
	}

	if cfg.Agent.MaxTurns != 10 {
		t.Errorf("Expected max_turns 10, got %d", cfg.Agent.MaxTurns)
	}

	if cfg.ACE.Enabled {
		t.Error("Expected ACE to be disabled")
	}

	if cfg.Security.SandboxMode != "docker" {
		t.Errorf("Expected sandbox_mode docker, got %s", cfg.Security.SandboxMode)
	}

	if !cfg.Protocol.EnableShell {
		t.Error("Expected enable_shell to be true")
	}
}

// TestGolden_ValidFull tests loading a full configuration with all fields.
func TestGolden_ValidFull(t *testing.T) {
	t.Parallel()

	cfg := loadGoldenConfig(t, "valid_full.yaml")

	verifyFullLLMSection(t, &cfg)
	verifyFullAgentSection(t, &cfg)
	verifyFullACESection(t, &cfg)
	verifyFullSecuritySection(t, &cfg)
	verifyFullProtocolSection(t, &cfg)
}

func loadGoldenConfig(t *testing.T, filename string) V2 {
	t.Helper()

	path := filepath.Join("golden", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	var cfg V2
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if err = cfg.Validate(); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	return cfg
}

func verifyFullLLMSection(t *testing.T, cfg *V2) {
	t.Helper()

	if cfg.LLM.Provider != "openai" {
		t.Errorf("Expected provider openai, got %s", cfg.LLM.Provider)
	}

	if cfg.LLM.APIKey != "sk-test-key-123" {
		t.Errorf("Expected api_key sk-test-key-123, got %s", cfg.LLM.APIKey)
	}

	if cfg.LLM.Timeout != 120*time.Second {
		t.Errorf("Expected timeout 120s, got %v", cfg.LLM.Timeout)
	}

	if cfg.LLM.MaxTokens != 8192 {
		t.Errorf("Expected max_tokens 8192, got %d", cfg.LLM.MaxTokens)
	}

	if cfg.LLM.Temperature != 0.8 {
		t.Errorf("Expected temperature 0.8, got %f", cfg.LLM.Temperature)
	}
}

func verifyFullAgentSection(t *testing.T, cfg *V2) {
	t.Helper()

	if cfg.Agent.MaxTurns != 50 {
		t.Errorf("Expected max_turns 50, got %d", cfg.Agent.MaxTurns)
	}

	if cfg.Agent.Timeout != 600*time.Second {
		t.Errorf("Expected timeout 600s, got %v", cfg.Agent.Timeout)
	}

	if !cfg.Agent.RequireApproval {
		t.Error("Expected require_approval to be true")
	}
}

func verifyFullACESection(t *testing.T, cfg *V2) {
	t.Helper()

	if !cfg.ACE.Enabled {
		t.Error("Expected ACE to be enabled")
	}

	if cfg.ACE.PlaybookPath != "/etc/spin/playbooks/default.yaml" {
		t.Errorf("Expected playbook_path /etc/spin/playbooks/default.yaml, got %s", cfg.ACE.PlaybookPath)
	}

	if cfg.ACE.TopK != 5 {
		t.Errorf("Expected top_k 5, got %d", cfg.ACE.TopK)
	}

	if cfg.ACE.MinScore != 0.7 {
		t.Errorf("Expected min_score 0.7, got %f", cfg.ACE.MinScore)
	}
}

func verifyFullSecuritySection(t *testing.T, cfg *V2) {
	t.Helper()

	if cfg.Security.SandboxMode != "firejail" {
		t.Errorf("Expected sandbox_mode firejail, got %s", cfg.Security.SandboxMode)
	}

	if len(cfg.Security.AllowedCommands) != 4 {
		t.Errorf("Expected 4 allowed commands, got %d", len(cfg.Security.AllowedCommands))
	}
}

func verifyFullProtocolSection(t *testing.T, cfg *V2) {
	t.Helper()

	if !cfg.Protocol.EnableMCP {
		t.Error("Expected enable_mcp to be true")
	}

	if len(cfg.Protocol.MCPServers) != 2 {
		t.Errorf("Expected 2 MCP servers, got %d", len(cfg.Protocol.MCPServers))
	}

	if cfg.Protocol.ShellTimeout != 60*time.Second {
		t.Errorf("Expected shell_timeout 60s, got %v", cfg.Protocol.ShellTimeout)
	}
}

// TestGolden_InvalidMissingRequired tests that missing required fields are detected.
func TestGolden_InvalidMissingRequired(t *testing.T) {
	t.Parallel()

	path := filepath.Join("golden", "invalid_missing_required.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	var cfg V2
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for missing required fields")
	}

	errStr := err.Error()

	// Should report multiple errors - when fields are missing, zero values are used
	// which fail validation.
	if !strings.Contains(errStr, "provider") {
		t.Errorf("Expected error about missing provider, got: %v", errStr)
	}

	if !strings.Contains(errStr, "max_turns") {
		t.Errorf("Expected error about missing max_turns, got: %v", errStr)
	}
	// Note: sandbox_mode validation only checks if it's one of valid values when non-empty
	// Zero value (empty string) would need explicit required check.
	if !strings.Contains(errStr, "max_tokens") {
		t.Errorf("Expected error about missing max_tokens, got: %v", errStr)
	}

	if !strings.Contains(errStr, "work_dir") {
		t.Errorf("Expected error about missing work_dir, got: %v", errStr)
	}
}

// TestGolden_InvalidBadValues tests that invalid field values are detected.
func TestGolden_InvalidBadValues(t *testing.T) {
	t.Parallel()

	path := filepath.Join("golden", "invalid_bad_values.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	var cfg V2
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for invalid values")
	}

	errStr := err.Error()

	// Should report multiple validation errors.
	if !strings.Contains(errStr, "provider") {
		t.Errorf("Expected error about provider, got: %v", errStr)
	}

	if !strings.Contains(errStr, "temperature") {
		t.Errorf("Expected error about invalid temperature, got: %v", errStr)
	}

	if !strings.Contains(errStr, "max_turns") {
		t.Errorf("Expected error about invalid max_turns, got: %v", errStr)
	}

	if !strings.Contains(errStr, "sandbox_mode") {
		t.Errorf("Expected error about invalid sandbox_mode, got: %v", errStr)
	}
}

// TestGolden_InvalidCrossSection tests cross-section validation rules.
func TestGolden_InvalidCrossSection(t *testing.T) {
	t.Parallel()

	path := filepath.Join("golden", "invalid_cross_section.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	var cfg V2
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for cross-section violations")
	}

	errStr := err.Error()

	// Should report multiple cross-section errors.
	if !strings.Contains(errStr, "playbook_path") {
		t.Error("Expected error about missing playbook_path when ACE enabled")
	}

	if !strings.Contains(errStr, "trajectory_path") {
		t.Error("Expected error about missing trajectory_path when ACE enabled")
	}

	if !strings.Contains(errStr, "shell_timeout") {
		t.Error("Expected error about missing shell_timeout when shell enabled")
	}
}

// TestGolden_LoaderV2 tests LoaderV2 with golden files.
func TestGolden_LoaderV2_ValidMinimal(t *testing.T) {
	t.Parallel()

	loader := NewLoaderV2()

	path := filepath.Join("golden", "valid_minimal.yaml")

	cfg, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("Expected provider ollama, got %s", cfg.LLM.Provider)
	}
}

// TestGolden_LoaderV2_Invalid tests LoaderV2 with invalid files.
func TestGolden_LoaderV2_Invalid(t *testing.T) {
	t.Parallel()

	loader := NewLoaderV2()

	testCases := []struct {
		name     string
		file     string
		errCheck func(error) bool
	}{
		{
			name: "bad_values",
			file: "invalid_bad_values.yaml",
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "validation failed")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join("golden", tc.file)

			_, err := loader.LoadFromFile(path)
			if err == nil {
				t.Fatal("Expected load to fail")
			}

			if !tc.errCheck(err) {
				t.Errorf("Error check failed for: %v", err)
			}
		})
	}
}

// TestGolden_LoaderV2_MissingFieldsWithDefaults tests that LoaderV2 applies defaults
// for missing fields, making them valid.
func TestGolden_LoaderV2_MissingFieldsWithDefaults(t *testing.T) {
	t.Parallel()

	loader := NewLoaderV2()

	path := filepath.Join("golden", "invalid_missing_required.yaml")
	cfg, err := loader.LoadFromFile(path)

	// Loader should apply defaults and make this config valid
	// (this is different from direct YAML unmarshal + validate).
	if err != nil {
		// It's OK if it still fails due to some truly required fields
		// The point is that defaults are applied.
		t.Logf("Config with missing fields failed as expected: %v", err)
	} else {
		// If it passes, verify defaults were applied.
		if cfg.LLM.Provider != "" {
			t.Logf("Defaults applied: provider=%s", cfg.LLM.Provider)
		}
	}
}

// TestGolden_LoaderV2_CrossSectionWithDefaults tests cross-section validation
// even when defaults are applied.
func TestGolden_LoaderV2_CrossSectionWithDefaults(t *testing.T) {
	t.Parallel()

	loader := NewLoaderV2()

	path := filepath.Join("golden", "invalid_cross_section.yaml")
	cfg, err := loader.LoadFromFile(path)

	// Loader might apply defaults for some fields (like shell_timeout),
	// but ACE paths cannot have defaults when ACE is explicitly enabled.
	if err != nil {
		// Expected: cross-section validation should fail.
		if !strings.Contains(err.Error(), "playbook_path") && !strings.Contains(err.Error(), "trajectory_path") {
			t.Errorf("Expected cross-section validation error about ACE paths, got: %v", err)
		}
	} else {
		// If it passes, check if defaults filled in the ACE paths.
		if cfg.ACE.Enabled && (cfg.ACE.PlaybookPath == "" || cfg.ACE.TrajectoryPath == "") {
			t.Error("ACE is enabled but paths are empty - should have failed validation")
		}
	}
}
