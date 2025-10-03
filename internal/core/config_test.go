package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultConfig verifies default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxTurns != 50 {
		t.Errorf("MaxTurns = %d, want 50", cfg.MaxTurns)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", cfg.Timeout)
	}
	if cfg.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d, want 8192", cfg.MaxTokens)
	}
	if cfg.StreamBuffer != 100 {
		t.Errorf("StreamBuffer = %d, want 100", cfg.StreamBuffer)
	}
	if cfg.HistoryLimit != 1000 {
		t.Errorf("HistoryLimit = %d, want 1000", cfg.HistoryLimit)
	}
	if cfg.SandboxMode != "workspace-only" {
		t.Errorf("SandboxMode = %s, want workspace-only", cfg.SandboxMode)
	}
	if !cfg.EnableGit {
		t.Error("EnableGit should be true by default")
	}
	if !cfg.EnableShell {
		t.Error("EnableShell should be true by default")
	}
}

// TestConfig_Load_ValidYAML tests loading valid YAML configuration
func TestConfig_Load_ValidYAML(t *testing.T) {
	path := filepath.Join("testdata", "valid_config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Provider != "ollama" {
		t.Errorf("Provider = %s, want ollama", cfg.Provider)
	}
	if cfg.Model != "codellama:13b" {
		t.Errorf("Model = %s, want codellama:13b", cfg.Model)
	}
	if cfg.MaxTurns != 50 {
		t.Errorf("MaxTurns = %d, want 50", cfg.MaxTurns)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", cfg.Timeout)
	}
	if cfg.SandboxMode != "workspace-only" {
		t.Errorf("SandboxMode = %s, want workspace-only", cfg.SandboxMode)
	}
	if len(cfg.AllowedCommands) != 3 {
		t.Errorf("len(AllowedCommands) = %d, want 3", len(cfg.AllowedCommands))
	}
	if !cfg.EnableMCP {
		t.Error("EnableMCP should be true")
	}
	if len(cfg.MCPServers) != 2 {
		t.Errorf("len(MCPServers) = %d, want 2", len(cfg.MCPServers))
	}
}

// TestConfig_Load_MinimalYAML tests loading minimal configuration
func TestConfig_Load_MinimalYAML(t *testing.T) {
	path := filepath.Join("testdata", "minimal_config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Provider != "ollama" {
		t.Errorf("Provider = %s, want ollama", cfg.Provider)
	}
	if cfg.Model != "codellama:13b" {
		t.Errorf("Model = %s, want codellama:13b", cfg.Model)
	}
	if cfg.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %s, want /tmp", cfg.WorkDir)
	}
}

// TestConfig_Load_InvalidYAML tests handling invalid YAML syntax
func TestConfig_Load_InvalidYAML(t *testing.T) {
	path := filepath.Join("testdata", "invalid_syntax.yaml")
	_, err := Load(path)
	if err == nil {
		t.Error("Load() should return error for invalid YAML")
	}
}

// TestConfig_Load_MissingFile tests handling missing configuration file
func TestConfig_Load_MissingFile(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("Load() should return error for missing file")
	}
}

// TestConfig_Validate_Valid tests validation of valid configuration
func TestConfig_Validate_Valid(t *testing.T) {
	cfg := &Config{
		Provider:    "ollama",
		Model:       "codellama:13b",
		WorkDir:     "/tmp",
		MaxTurns:    50,
		Timeout:     5 * time.Minute,
		SandboxMode: "workspace-only",
		MaxTokens:   8192,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestConfig_Validate_MissingProvider tests validation fails for missing provider
func TestConfig_Validate_MissingProvider(t *testing.T) {
	cfg := &Config{
		Model:       "codellama:13b",
		WorkDir:     "/tmp",
		MaxTurns:    50,
		Timeout:     5 * time.Minute,
		SandboxMode: "workspace-only",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing provider")
	}
}

// TestConfig_Validate_MissingModel tests validation fails for missing model
func TestConfig_Validate_MissingModel(t *testing.T) {
	cfg := &Config{
		Provider:    "ollama",
		WorkDir:     "/tmp",
		MaxTurns:    50,
		Timeout:     5 * time.Minute,
		SandboxMode: "workspace-only",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing model")
	}
}

// TestConfig_Validate_InvalidMaxTurns tests validation fails for invalid MaxTurns
func TestConfig_Validate_InvalidMaxTurns(t *testing.T) {
	cfg := &Config{
		Provider:    "ollama",
		Model:       "codellama:13b",
		WorkDir:     "/tmp",
		MaxTurns:    -1,
		Timeout:     5 * time.Minute,
		SandboxMode: "workspace-only",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for negative MaxTurns")
	}
}

// TestConfig_Validate_InvalidTimeout tests validation fails for invalid Timeout
func TestConfig_Validate_InvalidTimeout(t *testing.T) {
	cfg := &Config{
		Provider:    "ollama",
		Model:       "codellama:13b",
		WorkDir:     "/tmp",
		MaxTurns:    50,
		Timeout:     -5 * time.Second,
		SandboxMode: "workspace-only",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for negative Timeout")
	}
}

// TestConfig_Validate_InvalidSandboxMode tests validation fails for invalid sandbox mode
func TestConfig_Validate_InvalidSandboxMode(t *testing.T) {
	cfg := &Config{
		Provider:    "ollama",
		Model:       "codellama:13b",
		WorkDir:     "/tmp",
		MaxTurns:    50,
		Timeout:     5 * time.Minute,
		SandboxMode: "invalid-mode",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for invalid SandboxMode")
	}
}

// TestConfig_Merge_BasicFields tests merging basic configuration fields
func TestConfig_Merge_BasicFields(t *testing.T) {
	base := &Config{
		Provider: "ollama",
		Model:    "codellama:7b",
		MaxTurns: 30,
	}

	override := &Config{
		Model:    "codellama:13b",
		MaxTurns: 50,
	}

	merged := base.Merge(override)

	if merged.Provider != "ollama" {
		t.Errorf("Provider = %s, want ollama", merged.Provider)
	}
	if merged.Model != "codellama:13b" {
		t.Errorf("Model = %s, want codellama:13b", merged.Model)
	}
	if merged.MaxTurns != 50 {
		t.Errorf("MaxTurns = %d, want 50", merged.MaxTurns)
	}
}

// TestConfig_Merge_Slices tests merging slice fields
func TestConfig_Merge_Slices(t *testing.T) {
	base := &Config{
		AllowedCommands: []string{"git", "make"},
	}

	override := &Config{
		AllowedCommands: []string{"go"},
	}

	merged := base.Merge(override)

	if len(merged.AllowedCommands) != 3 {
		t.Errorf("len(AllowedCommands) = %d, want 3", len(merged.AllowedCommands))
	}
	expected := []string{"git", "make", "go"}
	for i, cmd := range expected {
		if merged.AllowedCommands[i] != cmd {
			t.Errorf("AllowedCommands[%d] = %s, want %s", i, merged.AllowedCommands[i], cmd)
		}
	}
}

// TestConfig_Merge_Maps tests merging map fields
func TestConfig_Merge_Maps(t *testing.T) {
	base := &Config{
		ProviderConfig: map[string]interface{}{
			"temperature": 0.5,
			"top_p":       0.9,
		},
	}

	override := &Config{
		ProviderConfig: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2048,
		},
	}

	merged := base.Merge(override)

	if len(merged.ProviderConfig) != 3 {
		t.Errorf("len(ProviderConfig) = %d, want 3", len(merged.ProviderConfig))
	}
	if merged.ProviderConfig["temperature"] != 0.7 {
		t.Errorf("temperature = %v, want 0.7", merged.ProviderConfig["temperature"])
	}
	if merged.ProviderConfig["top_p"] != 0.9 {
		t.Errorf("top_p = %v, want 0.9", merged.ProviderConfig["top_p"])
	}
	if merged.ProviderConfig["max_tokens"] != 2048 {
		t.Errorf("max_tokens = %v, want 2048", merged.ProviderConfig["max_tokens"])
	}
}

// TestConfig_Merge_ZeroValues tests that zero values don't override non-zero
func TestConfig_Merge_ZeroValues(t *testing.T) {
	base := &Config{
		MaxTurns: 50,
		Timeout:  5 * time.Minute,
	}

	override := &Config{
		MaxTurns: 0, // Zero value should not override
	}

	merged := base.Merge(override)

	if merged.MaxTurns != 50 {
		t.Errorf("MaxTurns = %d, want 50 (should not be overridden by zero)", merged.MaxTurns)
	}
}

// TestLoadFromEnv tests loading configuration from environment variables
func TestLoadFromEnv(t *testing.T) {
	// Set test environment variables
	os.Setenv("SPIN_PROVIDER", "test-provider")
	os.Setenv("SPIN_MODEL", "test-model")
	os.Setenv("SPIN_MAX_TURNS", "42")
	os.Setenv("SPIN_TIMEOUT", "10m")
	defer func() {
		os.Unsetenv("SPIN_PROVIDER")
		os.Unsetenv("SPIN_MODEL")
		os.Unsetenv("SPIN_MAX_TURNS")
		os.Unsetenv("SPIN_TIMEOUT")
	}()

	cfg := loadFromEnv()

	if cfg.Provider != "test-provider" {
		t.Errorf("Provider = %s, want test-provider", cfg.Provider)
	}
	if cfg.Model != "test-model" {
		t.Errorf("Model = %s, want test-model", cfg.Model)
	}
	if cfg.MaxTurns != 42 {
		t.Errorf("MaxTurns = %d, want 42", cfg.MaxTurns)
	}
	if cfg.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want 10m", cfg.Timeout)
	}
}

// TestLoadConfig_FileOnly tests loading configuration from file only
func TestLoadConfig_FileOnly(t *testing.T) {
	path := filepath.Join("testdata", "minimal_config.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Provider != "ollama" {
		t.Errorf("Provider = %s, want ollama", cfg.Provider)
	}
	if cfg.Model != "codellama:13b" {
		t.Errorf("Model = %s, want codellama:13b", cfg.Model)
	}
}

// TestLoadConfig_EnvOverride tests environment variables override file config
func TestLoadConfig_EnvOverride(t *testing.T) {
	path := filepath.Join("testdata", "minimal_config.yaml")

	// Set environment variable to override
	os.Setenv("SPIN_MODEL", "overridden-model")
	defer os.Unsetenv("SPIN_MODEL")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Model != "overridden-model" {
		t.Errorf("Model = %s, want overridden-model (env should override file)", cfg.Model)
	}
	if cfg.Provider != "ollama" {
		t.Errorf("Provider = %s, want ollama (from file)", cfg.Provider)
	}
}

// TestMCPServerConfig_Parsing tests parsing MCP server configuration
func TestMCPServerConfig_Parsing(t *testing.T) {
	path := filepath.Join("testdata", "valid_config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.MCPServers) != 2 {
		t.Fatalf("len(MCPServers) = %d, want 2", len(cfg.MCPServers))
	}

	// Check first server
	server1 := cfg.MCPServers[0]
	if server1.Name != "filesystem" {
		t.Errorf("MCPServers[0].Name = %s, want filesystem", server1.Name)
	}
	if server1.Command != "mcp-server-filesystem" {
		t.Errorf("MCPServers[0].Command = %s, want mcp-server-filesystem", server1.Command)
	}
	if len(server1.Args) != 1 || server1.Args[0] != "/workspace" {
		t.Errorf("MCPServers[0].Args = %v, want [/workspace]", server1.Args)
	}

	// Check second server
	server2 := cfg.MCPServers[1]
	if server2.Name != "github" {
		t.Errorf("MCPServers[1].Name = %s, want github", server2.Name)
	}
	if server2.Env["GITHUB_TOKEN"] != "test-token" {
		t.Errorf("MCPServers[1].Env[GITHUB_TOKEN] = %s, want test-token", server2.Env["GITHUB_TOKEN"])
	}
}

// TestConfig_Validate_InvalidValues tests loading file with invalid values
func TestConfig_Validate_InvalidValues(t *testing.T) {
	path := filepath.Join("testdata", "invalid_values.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Validation should fail
	err = cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for invalid values")
	}
}

// TestConfig_ProviderConfig tests provider-specific configuration
func TestConfig_ProviderConfig(t *testing.T) {
	path := filepath.Join("testdata", "valid_config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ProviderConfig == nil {
		t.Fatal("ProviderConfig should not be nil")
	}
	if cfg.ProviderConfig["temperature"] != 0.7 {
		t.Errorf("temperature = %v, want 0.7", cfg.ProviderConfig["temperature"])
	}
	if cfg.ProviderConfig["top_p"] != 0.9 {
		t.Errorf("top_p = %v, want 0.9", cfg.ProviderConfig["top_p"])
	}
}
