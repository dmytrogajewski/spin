# FRD-20251030-009: TUI ACE Integration

**Feature:** ACE Feature 9 (Partial) - TUI Integration  
**Status:** Draft  
**Created:** 2025-10-30  
**Author:** Spin Development Team  
**Depends On:** Feature 1-8 (ACE Foundation)

---

## Executive Summary

Integrate ACE (Agentic Context Engineering) into the Spin TUI command, making playbooks available for all agent interactions. This enables the agent to learn from coding patterns and improve over time when using `spin tui`.

**Key Innovation:** Enable ACE by default with sensible defaults, requiring zero configuration for users to benefit from context learning.

**Primary Goal:** Make ACE functionality accessible via `spin tui --model qwen2.5-coder:32b --provider ollama` with automatic playbook management.

---

## Background

### Problem Statement

ACE is fully implemented and integrated at the Agent level, but:
1. **No TUI Wiring**: ACE service not instantiated in TUI command
2. **No Config Loading**: Agent.ACE config section not loaded
3. **Manual Setup Required**: Users cannot access ACE without code changes

**Solution:** Wire ACE into TUI with enabled-by-default configuration.

---

## Goals

### Functional Goals

1. **Auto-Enable ACE**: Enable ACE by default in TUI
2. **Config Integration**: Load ACE config from `~/.spin/config.yaml`
3. **Service Creation**: Instantiate ACEService during agent setup
4. **Playbook Persistence**: Automatically save/load playbooks
5. **CLI Flags**: Add `--ace-enabled` flag to override config

### Non-Functional Goals

1. **Zero Config**: Work out-of-box with defaults
2. **Backward Compatible**: Don't break existing TUI usage
3. **Performance**: < 100ms overhead for ACE initialization

---

## Technical Design

### Changes Required

#### 1. Load ACE Config in TUI Command

**File:** `cmd/spin/tui.go`

```go
// After loading agent config, ensure ACE is configured
if agentCfg.ACE.Enabled {
    // Ensure default paths are set
    if agentCfg.ACE.PlaybookPath == "" {
        agentCfg.ACE.PlaybookPath = "~/.spin/ace/playbooks/default.json"
    }
    if agentCfg.ACE.TrajectoryPath == "" {
        agentCfg.ACE.TrajectoryPath = "~/.spin/ace/trajectories/"
    }
}
```

#### 2. Create ACE Service

**File:** `cmd/spin/tui.go` (in runTUI function)

```go
// Create ACE service if enabled
var aceService *agent.ACEService
if agentCfg.ACE.Enabled {
    var err error
    aceService, err = agent.NewACEService(&agentCfg.ACE, workDir)
    if err != nil {
        return fmt.Errorf("failed to create ACE service: %w", err)
    }
    defer aceService.SavePlaybook() // Save on exit
}
```

#### 3. Pass ACE Service to Agent

**File:** `cmd/spin/tui.go`

```go
// Build agent options
agentOpts := []agent.AgentOption{
    agent.WithEventEmitter(emitter),
}

// Add ACE service if enabled
if aceService != nil {
    agentOpts = append(agentOpts, agent.WithACEService(aceService))
}

// Create agent
agentInstance, err := agent.NewAgent(
    ctx,
    &agentCfg,
    llmProvider,
    toolRegistry,
    historyMgr,
    agentOpts...,
)
```

#### 4. Add CLI Flag

**File:** `cmd/spin/tui.go`

```go
tuiCmd.Flags().Bool("ace-enabled", true, "Enable ACE (Agentic Context Engineering)")
```

#### 5. Update Default Config

**File:** `internal/agent/config.go`

```go
// In DefaultConfig() function
ACE: ACEConfig{
    Enabled:        true,  // <- Enable by default
    PlaybookPath:   "~/.spin/ace/playbooks/default.json",
    TrajectoryPath: "~/.spin/ace/trajectories/",
    Retrieval: ACERetrievalConfig{
        TopK:     5,
        MinScore: 0.3,
    },
    ItemizedLearning: ACEItemizedLearningConfig{
        Enabled:       true,
        ParseFeedback: true,
        UpdateAsync:   false,
    },
    Generation: ACEGenerationConfig{
        Enabled:     false, // Phase 3+
        AutoReflect: false,
    },
},
```

---

## Implementation Steps

### Step 1: Enable ACE by Default in Config

```go
// internal/agent/config.go
func DefaultConfig() Config {
    return Config{
        // ... existing config
        ACE: ACEConfig{
            Enabled: true, // CHANGE: was false, now true
            // ... rest of config
        },
    }
}
```

### Step 2: Wire ACE in TUI Command

```go
// cmd/spin/tui.go

// After creating agent config, create ACE service
var aceService *agent.ACEService
if agentCfg.ACE.Enabled {
    aceService, err = agent.NewACEService(&agentCfg.ACE, workDir)
    if err != nil {
        return fmt.Errorf("failed to create ACE service: %w", err)
    }
}

// Add to agent options
if aceService != nil {
    agentOpts = append(agentOpts, agent.WithACEService(aceService))
}

// Save playbook on exit
defer func() {
    if aceService != nil {
        aceService.SavePlaybook()
    }
}()
```

---

## Testing Strategy

### Manual Testing

```bash
# Test 1: Basic usage with ACE enabled
spin tui --model qwen2.5-coder:32b --provider ollama

# Test 2: Disable ACE via flag
spin tui --model qwen2.5-coder:32b --provider ollama --ace-enabled=false

# Test 3: Verify playbook creation
ls ~/.spin/ace/playbooks/default.json

# Test 4: Multi-turn learning
# - Ask agent to write code
# - Check if bullets are created
# - Ask similar question
# - Verify bullet retrieval works
```

### Integration Tests

Not required for this minimal integration - existing agent tests cover ACE functionality.

---

## Acceptance Criteria

1. ✅ ACE enabled by default in agent config
2. ✅ ACE service created during TUI startup
3. ✅ ACE service passed to agent
4. ✅ Playbook automatically saved on TUI exit
5. ✅ Works with Ollama: `spin tui --model qwen2.5-coder:32b --provider ollama`
6. ✅ No errors or warnings in debug mode
7. ✅ Backward compatible (no breaking changes)

---

## Risks and Mitigations

### Risk: Playbook File Corruption

**Mitigation**: ACE playbook save has atomic write with validation

### Risk: Performance Overhead

**Mitigation**: ACE service creation is < 10ms, negligible impact

### Risk: User Confusion

**Mitigation**: ACE runs silently by default, no UI changes needed

---

## Future Enhancements

1. **TUI Feedback**: Show "Learning enabled" indicator
2. **Playbook Stats**: Display bullet count in status bar
3. **CLI Commands**: `spin ace list`, `spin ace stats`, `spin ace reset`
4. **Multi-Playbook**: Support multiple playbooks per project

---

**Status**: Ready for Implementation  
**Estimated Effort**: 30 minutes (minimal changes)
