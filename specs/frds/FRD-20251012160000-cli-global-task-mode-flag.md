# FRD-20251012160000: CLI Global Task Mode Flag

**Status**: Draft
**Created**: 2025-10-12 16:00
**Author**: Spin Agent
**Related Roadmap**: [specs/task-modes/ROADMAP.md](../task-modes/ROADMAP.md) - P3.1
**Depends On**: Phase 1 (P1.1-P1.5) ✅, Phase 2 (P2.1-P2.3) ✅
**Complexity**: Low (30 minutes)
**Priority**: HIGH - Blocks P3.2-P3.4

## Overview

Add a global `--mode` flag to the Spin CLI that allows users to specify the task mode when launching the agent. This is the first user-facing interface for the task mode system implemented in Phase 1 and Phase 2.

## Problem Statement

The task mode system is fully implemented in the core agent and conversation layers, but there is no CLI interface for users to:
1. Specify initial task mode when starting Spin
2. Know what modes are available
3. Validate their mode selection

Users need a simple, intuitive flag to control the agent's behavior mode from the command line.

## Requirements

### Functional Requirements

**FR1**: Add `--mode` persistent flag to root command
- Long form: `--mode <mode-name>`
- Short form: `-m <mode-name>`
- Default value: `"regular"`
- Applies to all subcommands that create conversations

**FR2**: Flag validation
- Must validate mode name against list of valid modes
- Valid modes: `regular`, `review`, `compact`, `planning`
- Invalid mode should show clear error message with valid options

**FR3**: Help text
- Flag description must explain purpose
- Help text must list all valid modes with brief descriptions

**FR4**: Integration with conversation creation
- Mode flag value must be passed to conversation initialization
- Must work with both TUI and exec modes

### Non-Functional Requirements

**NFR1**: Backward Compatibility
- Default behavior unchanged (uses "regular" mode)
- Existing scripts/workflows continue to work

**NFR2**: User Experience
- Clear error messages for invalid modes
- Helpful flag description in `--help` output

**NFR3**: Testability
- Flag parsing must be unit testable
- Mode validation must be testable

## Design

### Architecture

```
┌─────────────┐
│   CLI Args  │
│  --mode foo │
└──────┬──────┘
       │
       v
┌──────────────┐     validate     ┌────────────────┐
│  root.go     ├─────────────────>│  Valid Modes   │
│  (cobra)     │                  │  [regular,     │
└──────┬───────┘                  │   review,      │
       │                          │   compact,     │
       │ set flagTaskMode         │   planning]    │
       v                          └────────────────┘
┌──────────────┐
│  Global Var  │
│ flagTaskMode │
└──────┬───────┘
       │
       │ read by TUI/exec
       v
┌──────────────┐
│  TUI/Exec    │
│  Commands    │
└──────┬───────┘
       │
       │ pass to NewConversationWithTask
       v
┌──────────────┐
│ Conversation │
│  (Phase 2)   │
└──────────────┘
```

### Implementation Details

#### 1. Add Global Variable

**File**: `cmd/spin/root.go`

```go
// Global flags
var (
	flagModel      string
	flagProvider   string
	flagSandbox    string
	flagWorkDir    string
	flagConfigFile string
	flagConfig     []string
	flagTaskMode   string  // NEW: Task mode flag
)
```

#### 2. Add Flag to Root Command

**File**: `cmd/spin/root.go`

```go
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spin",
		Short: "AI-powered coding assistant",
		Long: `Spin is an open-source AI coding assistant that works with multiple LLM providers.

It provides an interactive terminal UI, non-interactive execution mode,
and integrates with IDEs via JSON-RPC.

Compatible with: Ollama, LMStudio, OpenAI, Anthropic, and any OpenAI-compatible API.`,
		Version: version.ShortVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: launch TUI when no subcommand is provided
			return runTUI(cmd, args)
		},
		SilenceUsage: true,
	}

	// Set custom version template
	cmd.SetVersionTemplate(version.String() + "\n")

	// Global flags
	cmd.PersistentFlags().StringVar(&flagModel, "model", "", "Model to use (e.g., llama3.1, mixtral, gpt-4o)")
	cmd.PersistentFlags().StringVar(&flagProvider, "provider", "", "Provider (ollama, lmstudio, openai, anthropic)")
	cmd.PersistentFlags().StringVar(&flagSandbox, "sandbox", "", "Sandbox mode (read-only, workspace-write, full-access)")
	cmd.PersistentFlags().StringVar(&flagWorkDir, "cd", "", "Working directory")
	cmd.PersistentFlags().StringVar(&flagConfigFile, "config-file", "", "Path to configuration file")
	cmd.PersistentFlags().StringSliceVarP(&flagConfig, "config", "c", nil, "Config overrides (key=value)")

	// NEW: Add task mode flag
	cmd.PersistentFlags().StringVarP(
		&flagTaskMode,
		"mode",
		"m",
		"regular",
		"Task mode: regular (full-featured, 16K tokens), review (read-only, 12K tokens), compact (minimal, 4K tokens), planning (context-only, 4K tokens)",
	)

	// Add commands
	cmd.AddCommand(newTUICmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newMCPCmd())
	cmd.AddCommand(newDebugCmd())
	cmd.AddCommand(newApplyPatchCmd())

	return cmd
}
```

#### 3. Add Validation Helper

**File**: `cmd/spin/root.go`

```go
// validTaskModes lists all valid task mode names
var validTaskModes = map[string]bool{
	"regular":  true,
	"review":   true,
	"compact":  true,
	"planning": true,
}

// validateTaskMode validates the task mode flag value.
// Returns error if the mode is invalid.
func validateTaskMode(mode string) error {
	if mode == "" {
		return fmt.Errorf("task mode cannot be empty")
	}

	if !validTaskModes[mode] {
		return fmt.Errorf("invalid task mode: %s (valid modes: regular, review, compact, planning)", mode)
	}

	return nil
}
```

#### 4. Integrate with TUI Command

**File**: `cmd/spin/tui.go` (existing file)

Update the TUI initialization to use the mode flag:

```go
func runTUI(cmd *cobra.Command, args []string) error {
	// ... existing setup code ...

	// NEW: Validate task mode flag
	if err := validateTaskMode(flagTaskMode); err != nil {
		return fmt.Errorf("invalid mode flag: %w", err)
	}

	// ... create manager ...

	// NEW: Create conversation with specified task mode
	var conv *core.Conversation
	var err error
	if flagTaskMode != "" && flagTaskMode != "regular" {
		conv, err = mgr.NewConversationWithTask(ctx, workDir, flagTaskMode)
	} else {
		conv, err = mgr.NewConversation(ctx, workDir)
	}
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	// ... rest of TUI setup ...
}
```

#### 5. Integrate with Exec Command

**File**: `cmd/spin/exec.go` (existing file)

Update the exec command to use the mode flag:

```go
func runExec(cmd *cobra.Command, args []string) error {
	// ... existing setup code ...

	// NEW: Validate task mode flag
	if err := validateTaskMode(flagTaskMode); err != nil {
		return fmt.Errorf("invalid mode flag: %w", err)
	}

	// ... create manager ...

	// NEW: Create conversation with specified task mode
	var conv *core.Conversation
	var err error
	if flagTaskMode != "" && flagTaskMode != "regular" {
		conv, err = mgr.NewConversationWithTask(ctx, workDir, flagTaskMode)
	} else {
		conv, err = mgr.NewConversation(ctx, workDir)
	}
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	// ... rest of exec logic ...
}
```

### Edge Cases

**EC1**: Empty mode string
- Validation should catch and error
- Should never reach conversation creation

**EC2**: Case sensitivity
- Accept only lowercase: "regular", not "Regular" or "REGULAR"
- Error message should show valid lowercase options

**EC3**: Unknown mode
- Show clear error listing valid modes
- Exit with non-zero status code

**EC4**: Future mode additions
- Adding new modes only requires updating `validTaskModes` map
- No changes to flag registration needed

## Testing Strategy

### Unit Tests

**File**: `cmd/spin/root_test.go`

```go
func TestValidateTaskMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{
			name:    "valid regular mode",
			mode:    "regular",
			wantErr: false,
		},
		{
			name:    "valid review mode",
			mode:    "review",
			wantErr: false,
		},
		{
			name:    "valid compact mode",
			mode:    "compact",
			wantErr: false,
		},
		{
			name:    "valid planning mode",
			mode:    "planning",
			wantErr: false,
		},
		{
			name:    "invalid mode",
			mode:    "invalid",
			wantErr: true,
		},
		{
			name:    "empty mode",
			mode:    "",
			wantErr: true,
		},
		{
			name:    "uppercase mode",
			mode:    "REGULAR",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskMode(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTaskMode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRootCmd_ModeFlag(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantMode   string
		wantErr    bool
	}{
		{
			name:     "default mode",
			args:     []string{},
			wantMode: "regular",
			wantErr:  false,
		},
		{
			name:     "explicit regular mode",
			args:     []string{"--mode", "regular"},
			wantMode: "regular",
			wantErr:  false,
		},
		{
			name:     "review mode",
			args:     []string{"--mode", "review"},
			wantMode: "review",
			wantErr:  false,
		},
		{
			name:     "short flag",
			args:     []string{"-m", "compact"},
			wantMode: "compact",
			wantErr:  false,
		},
		{
			name:     "invalid mode",
			args:     []string{"--mode", "invalid"},
			wantMode: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flagTaskMode
			flagTaskMode = "regular"

			cmd := newRootCmd()
			cmd.SetArgs(tt.args)

			// Parse flags only (don't execute)
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// Validate mode if no parse error
			err = validateTaskMode(flagTaskMode)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTaskMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && flagTaskMode != tt.wantMode {
				t.Errorf("flagTaskMode = %v, want %v", flagTaskMode, tt.wantMode)
			}
		})
	}
}
```

### Integration Tests

**Manual Testing Checklist**:

```bash
# Test 1: Default mode (should use regular)
spin

# Test 2: Explicit regular mode
spin --mode regular

# Test 3: Review mode
spin --mode review

# Test 4: Short flag
spin -m compact

# Test 5: Invalid mode (should error)
spin --mode invalid
# Expected: Error: invalid task mode: invalid (valid modes: regular, review, compact, planning)

# Test 6: Help shows mode flag
spin --help | grep mode
# Expected: Shows --mode flag with description

# Test 7: Mode persists across subcommands
spin --mode review tui
spin -m compact exec "hello"
```

### E2E Tests

**File**: `e2e/test_cli_mode_flag.sh` (NEW)

```bash
#!/bin/bash
set -euo pipefail

# Test CLI mode flag functionality

echo "Testing CLI mode flag..."

# Test 1: Default mode
echo "Test 1: Default mode works"
spin --help > /dev/null || {
    echo "FAIL: Default mode failed"
    exit 1
}
echo "PASS"

# Test 2: Invalid mode fails
echo "Test 2: Invalid mode errors correctly"
if spin --mode invalid 2>&1 | grep -q "invalid task mode"; then
    echo "PASS"
else
    echo "FAIL: Expected error for invalid mode"
    exit 1
fi

# Test 3: Help text shows mode flag
echo "Test 3: Help text includes mode flag"
if spin --help | grep -q "\-\-mode"; then
    echo "PASS"
else
    echo "FAIL: Mode flag not in help text"
    exit 1
fi

# Test 4: Short flag works
echo "Test 4: Short flag -m works"
spin -m regular --help > /dev/null || {
    echo "FAIL: Short flag failed"
    exit 1
}
echo "PASS"

echo "All CLI mode flag tests passed!"
```

## Implementation Plan

### Step 1: Add Global Variable and Validation (5 min)
1. Add `flagTaskMode` variable to root.go
2. Add `validTaskModes` map
3. Add `validateTaskMode()` function

### Step 2: Register Flag (5 min)
1. Add `PersistentFlags().StringVarP()` call in `newRootCmd()`
2. Add descriptive help text

### Step 3: Write Unit Tests (10 min)
1. Test `validateTaskMode()` with all cases
2. Test flag parsing with cobra
3. Test default value

### Step 4: Integrate with TUI Command (5 min)
1. Add validation call in `runTUI()`
2. Use `NewConversationWithTask()` when mode specified

### Step 5: Integrate with Exec Command (5 min)
1. Add validation call in `runExec()`
2. Use `NewConversationWithTask()` when mode specified

### Step 6: Manual Testing (5 min)
1. Run through manual testing checklist
2. Verify help text
3. Test all valid modes
4. Test invalid modes

### Step 7: Lint and Race Detection (5 min)
1. Run `make lint`
2. Run `go test -race ./cmd/spin/...`
3. Fix any issues

## Acceptance Criteria

### CLI Usage

```bash
# ✅ Default behavior (regular mode)
$ spin
# Launches in regular mode

# ✅ Explicit mode
$ spin --mode review
# Launches in review mode

# ✅ Short flag
$ spin -m compact
# Launches in compact mode

# ✅ Invalid mode shows error
$ spin --mode invalid
Error: invalid task mode: invalid (valid modes: regular, review, compact, planning)

# ✅ Help shows flag
$ spin --help
Flags:
  ...
  -m, --mode string   Task mode: regular (full-featured, 16K tokens),
                      review (read-only, 12K tokens), compact (minimal, 4K tokens),
                      planning (context-only, 4K tokens) (default "regular")
```

### Test Coverage

- ✅ `validateTaskMode()` has 100% coverage
- ✅ All valid modes tested
- ✅ All error cases tested
- ✅ Flag parsing tested

### Quality Gates

- ✅ `make lint` passes (zero errors)
- ✅ `go test -race ./cmd/spin/...` passes
- ✅ All unit tests pass
- ✅ Test coverage ≥90% for new code

## Non-Goals

**Out of Scope for P3.1**:
- Runtime mode switching (P3.2)
- Mode info command (P3.3)
- Environment variable support (future)
- Config file mode setting (future)

## Risks & Mitigations

### Risk 1: Flag Validation Happens After Cobra Parses
**Impact**: Invalid mode might not be caught early enough
**Mitigation**: Add validation at start of command execution, before heavy operations
**Contingency**: Add cobra PreRunE validation hook if needed

### Risk 2: TUI/Exec Integration Not Tested
**Impact**: Mode might not actually be applied to conversation
**Mitigation**: Manual testing checklist includes verification of mode behavior
**Contingency**: Add integration test that verifies tools are filtered correctly

### Risk 3: Typos in Mode Names
**Impact**: User frustration with rejected modes
**Mitigation**: Clear error message lists all valid modes
**Contingency**: Add "did you mean" suggestions in future iteration

## Future Enhancements

**P3.2**: Runtime mode switching via `/mode` command
**P3.3**: `spin mode` subcommand for mode information
**P4.x**: Environment variable `SPIN_MODE` support
**P5.x**: Config file `default_mode` setting

## References

- [Task Modes ROADMAP](../task-modes/ROADMAP.md) - Overall integration plan
- [Task Modes Specification](../task-modes/specification.md) - Technical details
- [Phase 1 FRDs](.) - Core agent integration (P1.1-P1.5)
- [Phase 2 FRDs](.) - Conversation integration (P2.1-P2.3)
- [Cobra Documentation](https://github.com/spf13/cobra) - Flag handling

## Definition of Done

- [x] FRD reviewed and approved
- [ ] Global `flagTaskMode` variable added
- [ ] `--mode` flag registered with short form `-m`
- [ ] `validateTaskMode()` function implemented
- [ ] `validTaskModes` map defined
- [ ] TUI command integrated (uses mode flag)
- [ ] Exec command integrated (uses mode flag)
- [ ] Unit tests written (≥90% coverage)
- [ ] All tests pass
- [ ] `make lint` clean (zero errors)
- [ ] Race detector clean
- [ ] Manual testing checklist completed
- [ ] Help text verified
- [ ] Roadmap updated (P3.1 marked complete)
- [ ] Godoc complete on all public functions

---

**Estimated Effort**: 30 minutes implementation + 10 minutes testing = **40 minutes total**
**Blocked By**: None (Phase 1 & 2 complete)
**Blocks**: P3.2 (REPL mode switching), P3.3 (Mode info command), P3.4 (CLI integration tests)
