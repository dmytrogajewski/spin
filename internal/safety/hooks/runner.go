// Package hooks provides lifecycle hook execution for the safety layer (Layer 5).
// Hook scripts run at defined lifecycle points and can block or audit operations.
package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Exit code indicating the hook wants to block the operation.
const exitCodeBlock = 2

// defaultTimeout is the default hook execution timeout.
const defaultTimeout = 5 * time.Second

// ErrHookFailed indicates a hook script exited with a non-zero, non-block code.
var ErrHookFailed = errors.New("hook failed")

// hookOutputJSON is the structure returned by blocking hooks on stdout.
type hookOutputJSON struct {
	Reason       string `json:"reason"`
	UpdatedInput string `json:"updated_input"`
}

// Config configures hook discovery and execution.
type Config struct {
	// GlobalDir is the global hooks directory (e.g., ~/.spin/hooks/).
	GlobalDir string
	// ProjectDir is the project-level hooks directory (e.g., .spin/hooks/).
	ProjectDir string
	// Timeout overrides the default hook execution timeout.
	// Zero uses defaultTimeout.
	Timeout time.Duration
	// Logger for hook execution events.
	Logger *slog.Logger
}

// Runner discovers and executes lifecycle hook scripts.
type Runner struct {
	globalDir        string
	projectDir       string
	timeout          time.Duration
	logger           *slog.Logger
	validScriptNames map[string]bool
}

// NewRunner creates a Runner from the given configuration.
func NewRunner(cfg Config) *Runner {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// Pre-build valid script name set for hook discovery validation.
	validNames := make(map[string]bool, eventCount)
	for _, evt := range AllEvents() {
		validNames[evt.ScriptName()] = true
	}

	return &Runner{
		globalDir:        cfg.GlobalDir,
		projectDir:       cfg.ProjectDir,
		timeout:          timeout,
		logger:           logger,
		validScriptNames: validNames,
	}
}

// Execute runs all hook scripts for the given event and context.
// For blocking events, returns a HookResult indicating whether the
// operation was blocked.
// For non-blocking events, hooks fire asynchronously and the method
// returns immediately with a zero-value HookResult.
func (r *Runner) Execute(
	ctx context.Context,
	event Event,
	evtCtx EventContext,
) HookResult {
	evtCtx.Event = event
	scripts := r.discoverScripts(event)

	if len(scripts) == 0 {
		return HookResult{}
	}

	if event.IsBlocking() {
		return r.executeBlocking(ctx, scripts, evtCtx)
	}

	r.executeAsync(ctx, scripts, evtCtx)

	return HookResult{}
}

// discoverScripts finds hook scripts for the event in global and project dirs.
// Project scripts run after global scripts (global first for org-wide policies).
func (r *Runner) discoverScripts(event Event) []string {
	name := event.ScriptName()

	var scripts []string

	for _, dir := range []string{r.globalDir, r.projectDir} {
		if dir == "" {
			continue
		}

		path := filepath.Join(dir, name)

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.Mode().IsRegular() {
			scripts = append(scripts, path)
		}
	}

	return scripts
}

// executeBlocking runs scripts sequentially. First script to return exit
// code 2 blocks the operation.
func (r *Runner) executeBlocking(
	ctx context.Context,
	scripts []string,
	evtCtx EventContext,
) HookResult {
	for _, script := range scripts {
		result, err := r.runScript(ctx, script, evtCtx)
		if err != nil {
			r.logger.WarnContext(ctx, "hook script error",
				slog.String("script", script),
				slog.String("error", err.Error()),
			)

			continue
		}

		if result.Blocked {
			return result
		}
	}

	return HookResult{}
}

// executeAsync fires all scripts in goroutines (non-blocking events).
func (r *Runner) executeAsync(
	ctx context.Context,
	scripts []string,
	evtCtx EventContext,
) {
	for _, script := range scripts {
		go func(s string) {
			_, err := r.runScript(ctx, s, evtCtx)
			if err != nil {
				r.logger.WarnContext(ctx, "async hook error",
					slog.String("script", s),
					slog.String("error", err.Error()),
				)
			}
		}(script)
	}
}

// runScript executes a single hook script with timeout.
func (r *Runner) runScript(
	ctx context.Context,
	scriptPath string,
	evtCtx EventContext,
) (HookResult, error) {
	inputJSON, err := json.Marshal(evtCtx)
	if err != nil {
		return HookResult{}, fmt.Errorf("marshal context: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "/bin/sh", scriptPath)
	cmd.Stdin = bytes.NewReader(inputJSON)
	cmd.WaitDelay = time.Second

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if execCtx.Err() != nil && errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		r.logger.WarnContext(ctx, "hook timed out, treating as non-blocking",
			slog.String("script", scriptPath),
			slog.Duration("timeout", r.timeout),
		)

		return HookResult{}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == exitCodeBlock {
			return r.parseBlockResult(stdout.String()), nil
		}

		return HookResult{}, fmt.Errorf(
			"%w: exit code %d: %s",
			ErrHookFailed,
			exitErr.ExitCode(),
			strings.TrimSpace(stderr.String()),
		)
	}

	if err != nil {
		return HookResult{}, fmt.Errorf("hook exec: %w", err)
	}

	return HookResult{}, nil
}

// parseBlockResult extracts block reason and optional updated input from
// hook stdout. Supports both plain text and JSON output.
func (r *Runner) parseBlockResult(output string) HookResult {
	trimmed := strings.TrimSpace(output)

	var parsed hookOutputJSON
	if json.Unmarshal([]byte(trimmed), &parsed) == nil && parsed.Reason != "" {
		return HookResult{
			Blocked:      true,
			Reason:       parsed.Reason,
			UpdatedInput: parsed.UpdatedInput,
		}
	}

	return HookResult{
		Blocked: true,
		Reason:  trimmed,
	}
}
