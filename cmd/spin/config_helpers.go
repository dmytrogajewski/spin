package main

import (
	"github.com/dmytrogajewski/spin/internal/config"
)

// loadConfigForMode loads configuration, using defaults if not found.
// This ensures all modes (TUI/EXEC/ACP) use the same config loading logic.
//
// Parameters:
//   - configFile: Optional path to config file. If empty, uses flagConfigFile or loads from default locations.
//   - maxTurns: Optional max turns override. If 0, uses config file or default.
//   - workDir: Working directory for the agent.
//
// Returns:
//   - *config.ConfigV2: Loaded configuration with defaults applied
//   - error: Error if config file is specified but cannot be loaded
func loadConfigForMode(configFile string, maxTurns int, workDir string) (*config.ConfigV2, error) {
	loader := config.NewLoaderV2()

	// Use provided configFile, or fall back to flagConfigFile, or default locations
	fileToLoad := configFile
	if fileToLoad == "" {
		fileToLoad = flagConfigFile
	}

	// Load from explicit config file if provided
	if fileToLoad != "" {
		if _, err := loader.LoadFromFile(fileToLoad); err != nil {
			return nil, err
		}
	} else {
		// Try to load from default locations (ignore error if not found)
		_, _ = loader.Load()
	}

	// Start with defaults
	cfg := config.DefaultConfigV2()
	cfg.Agent.WorkDir = workDir

	// Layer 1: Load from config file
	var fileCfg config.ConfigV2
	if err := loader.Unmarshal(&fileCfg); err == nil {
		applyFileConfig(cfg, &fileCfg)
	}

	// Layer 2: Override with CLI flags
	applyCLIFlags(cfg, maxTurns)

	return cfg, nil
}

// applyFileConfig applies configuration from file to the main config.
// This is extracted from tui.go to be shared across modes.
func applyFileConfig(cfg *config.ConfigV2, fileCfg *config.ConfigV2) {
	if fileCfg.LLM.Provider != "" {
		cfg.LLM.Provider = fileCfg.LLM.Provider
	}
	if fileCfg.LLM.Model != "" {
		cfg.LLM.Model = fileCfg.LLM.Model
	}
	if fileCfg.Agent.MaxTurns > 0 {
		cfg.Agent.MaxTurns = fileCfg.Agent.MaxTurns
	}
	if fileCfg.Agent.Timeout > 0 {
		cfg.Agent.Timeout = fileCfg.Agent.Timeout
	}
	if fileCfg.LLM.MaxTokens > 0 {
		cfg.LLM.MaxTokens = fileCfg.LLM.MaxTokens
	}
}

// applyCLIFlags applies CLI flags to the configuration.
// This is extracted from tui.go to be shared across modes.
func applyCLIFlags(cfg *config.ConfigV2, maxTurns int) {
	if maxTurns > 0 {
		cfg.Agent.MaxTurns = maxTurns
	}
	if flagProvider != "" {
		cfg.LLM.Provider = flagProvider
	}
	if flagModel != "" {
		cfg.LLM.Model = flagModel
	}
}

// applyDebugFlag applies the debug flag to configuration.
// This is extracted from exec.go to be shared across modes.
func applyDebugFlag(cfg *config.ConfigV2, debug bool) {
	if debug {
		cfg.Agent.Debug = true
		cfg.Agent.LogLevel = "debug"
		// Don't suppress INFO logs when debug is enabled
		// This allows debug logging to work properly
	}
}

