package config

// MigrateV1ToV2 converts a v1 Config to v2 ConfigV2.
// This provides backward compatibility during the transition period.
func MigrateV1ToV2(v1 *Config) *ConfigV2 {
	v2 := &ConfigV2{
		Version: "2.0",
	}

	// Migrate LLM configuration
	v2.LLM = LLMConfigV2{
		Provider:    v1.Provider,
		Model:       v1.Model,
		Temperature: v1.Temperature,
		MaxTokens:   v1.MaxTokens,
		Timeout:     v1.LLMTimeout,
		BaseURL:     "", // v1 doesn't have this field
		APIKey:      "", // v1 doesn't have this field (uses env var)
	}

	// Migrate Agent configuration
	v2.Agent = AgentConfigV2{
		MaxTurns:        v1.MaxTurns,
		Timeout:         v1.Timeout,
		WorkDir:         v1.WorkDir,
		RequireApproval: v1.RequireApproval,
	}

	// Migrate ACE configuration
	v2.ACE = ACEConfigV2{
		Enabled:        v1.ACEEnabled,
		PlaybookPath:   v1.ACEPlaybookPath,
		TrajectoryPath: v1.ACETrajectoryPath,
		TopK:           v1.ACETopK,
		MinScore:       v1.ACEMinScore,
	}

	// Migrate Security configuration
	v2.Security = SecurityConfigV2{
		SandboxMode:     v1.SandboxMode,
		PolicyFile:      v1.PolicyFile,
		AllowedCommands: v1.AllowedCommands,
	}

	// Migrate Protocol configuration
	v2.Protocol = ProtocolConfigV2{
		EnableMCP:    v1.EnableMCP,
		MCPServers:   migrateMCPServers(v1.MCPServers),
		EnableGit:    v1.EnableGit,
		EnableShell:  v1.EnableShell,
		ShellTimeout: v1.ShellTimeout,
	}

	return v2
}

// migrateMCPServers converts v1 MCP server configs to v2.
func migrateMCPServers(v1Servers []MCPServerConfig) []MCPServerConfigV2 {
	v2Servers := make([]MCPServerConfigV2, len(v1Servers))
	for i, v1Server := range v1Servers {
		v2Servers[i] = MCPServerConfigV2{
			Name:    v1Server.Name,
			Command: v1Server.Command,
			Args:    v1Server.Args,
			Env:     v1Server.Env,
		}
	}
	return v2Servers
}

// LoadV1Compatible loads a config file in v1 or v2 format and returns ConfigV2.
// This provides automatic migration for v1 configs.
func (l *LoaderV2) LoadV1Compatible(path string) (*ConfigV2, error) {
	l.viper.SetConfigFile(path)

	if err := l.viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// Check if it's a v2 config (has version field)
	if l.viper.IsSet("version") {
		// It's already v2, load normally
		return l.unmarshalWithDefaults()
	}

	// It's a v1 config, unmarshal as v1 and migrate
	v1 := &Config{}
	if err := l.viper.Unmarshal(v1); err != nil {
		return nil, err
	}

	// Migrate to v2
	v2 := MigrateV1ToV2(v1)

	// Apply defaults for any unset fields
	// Since we migrated from v1, we need to manually check what was set
	// For simplicity, we'll apply defaults to any zero-value fields
	applyMigrationDefaults(v2)

	// Validate
	if err := v2.Validate(); err != nil {
		return nil, err
	}

	return v2, nil
}

// applyMigrationDefaults applies defaults to fields that weren't set in v1.
// This is simpler than the normal applyDefaults because we know it came from v1.
func applyMigrationDefaults(v2 *ConfigV2) {
	defaults := DefaultConfigV2()

	// Apply LLM defaults for zero values
	if v2.LLM.BaseURL == "" {
		v2.LLM.BaseURL = defaults.LLM.BaseURL
	}
	if v2.LLM.APIKey == "" {
		v2.LLM.APIKey = defaults.LLM.APIKey
	}

	// Apply Agent defaults for zero values
	if v2.Agent.MaxTurns == 0 {
		v2.Agent.MaxTurns = defaults.Agent.MaxTurns
	}
	if v2.Agent.Timeout == 0 {
		v2.Agent.Timeout = defaults.Agent.Timeout
	}
	if v2.Agent.WorkDir == "" {
		v2.Agent.WorkDir = defaults.Agent.WorkDir
	}

	// Apply ACE defaults if completely unset
	if !v2.ACE.Enabled && v2.ACE.PlaybookPath == "" && v2.ACE.TrajectoryPath == "" {
		// ACE wasn't configured in v1, use defaults
		v2.ACE = defaults.ACE
	} else if v2.ACE.Enabled {
		// ACE was enabled, apply field defaults
		if v2.ACE.PlaybookPath == "" {
			v2.ACE.PlaybookPath = defaults.ACE.PlaybookPath
		}
		if v2.ACE.TrajectoryPath == "" {
			v2.ACE.TrajectoryPath = defaults.ACE.TrajectoryPath
		}
		if v2.ACE.TopK == 0 {
			v2.ACE.TopK = defaults.ACE.TopK
		}
		if v2.ACE.MinScore == 0 {
			v2.ACE.MinScore = defaults.ACE.MinScore
		}
	}

	// Apply Security defaults
	if v2.Security.SandboxMode == "" {
		v2.Security.SandboxMode = defaults.Security.SandboxMode
	}

	// Apply Protocol defaults
	if !v2.Protocol.EnableMCP && !v2.Protocol.EnableGit && !v2.Protocol.EnableShell {
		// Protocol wasn't configured, use defaults
		if v2.Protocol.ShellTimeout == 0 {
			v2.Protocol = defaults.Protocol
		}
	}
	if v2.Protocol.EnableShell && v2.Protocol.ShellTimeout == 0 {
		v2.Protocol.ShellTimeout = defaults.Protocol.ShellTimeout
	}
}
