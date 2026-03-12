package executor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// TerminalExecutor implements tools.CommandExecutor using terminal protocol.
type TerminalExecutor struct {
	terminalClient TerminalClient
	sessionID      string
	workDir        string
}

// NewTerminalExecutor creates a new terminal executor.
func NewTerminalExecutor(terminalClient TerminalClient, sessionID, workDir string) tools.CommandExecutor {
	if terminalClient == nil {
		return nil
	}

	return &TerminalExecutor{
		terminalClient: terminalClient,
		sessionID:      sessionID,
		workDir:        workDir,
	}
}

// Execute implements tools.CommandExecutor using terminal protocol.
func (e *TerminalExecutor) Execute(ctx context.Context, cmd tools.CommandInfo, opts any) (tools.ExecutionResult, error) {
	// If executor has a specific session ID configured, use it.
	// Otherwise, expect the session ID to be already present in the context (e.g. from ACP agent).
	if e.sessionID != "" {
		ctx = ContextWithSessionID(ctx, e.sessionID)
	}

	// Get command details.
	program := cmd.GetProgram()
	args := cmd.GetArgs()

	workDir := cmd.GetWorkDir()
	if workDir == "" {
		workDir = e.workDir
	}

	// Extract env vars from opts if available.
	env := extractEnvVars(opts)

	// Use a default output limit (e.g. 1MB) to prevent 0-byte truncation.
	const defaultOutputLimit = 1024 * 1024

	// Create terminal and execute command.
	terminalID, err := e.terminalClient.Create(ctx, program, args, env, workDir, defaultOutputLimit)
	if err != nil {
		return nil, fmt.Errorf("create terminal: %w", err)
	}

	// NOTE: Terminal is NOT released here. It is released after the tool_call_update
	// notification is sent (which references the terminal). This ensures compliance with
	// ACP spec: "The terminal must be added before calling 'terminal/release'.".

	// Wait for command to complete.
	exitCode, signal, err := e.terminalClient.WaitForExit(ctx, terminalID)
	if err != nil {
		return nil, fmt.Errorf("wait for exit: %w", err)
	}

	// Get output.
	output, truncated, exitStatus, err := e.terminalClient.GetOutput(ctx, terminalID)
	if err != nil {
		return nil, fmt.Errorf("get output: %w", err)
	}

	// Determine final exit code.
	resultExitCode := exitCode
	if exitStatus != nil && exitStatus.ExitCode != nil {
		resultExitCode = *exitStatus.ExitCode
	} else if signal != nil {
		resultExitCode = -1
	}

	return &terminalExecutionResult{
		stdout:     output,
		stderr:     "",
		exitCode:   resultExitCode,
		truncated:  truncated,
		terminalID: terminalID,
	}, nil
}

// extractEnvVars extracts environment variables from opts, handling struct, map, and slice formats.
func extractEnvVars(opts any) []EnvVar {
	if opts == nil {
		return nil
	}

	// Try struct-based extraction first.
	if envMap := extractEnvFromStruct(opts); envMap != nil {
		return envMap
	}

	// Try map-based extraction.
	optsMap, ok := opts.(map[string]any)
	if !ok {
		return nil
	}

	return extractEnvFromMap(optsMap)
}

// extractEnvFromMap extracts env vars from a map[string]any options.
func extractEnvFromMap(optsMap map[string]any) []EnvVar {
	envVars, exists := optsMap["env"]
	if !exists {
		return nil
	}

	if envSlice, ok := envVars.([]any); ok {
		return extractEnvFromSlice(envSlice)
	}

	if envVarMap, ok := envVars.(map[string]any); ok {
		return extractEnvFromStringMap(envVarMap)
	}

	return nil
}

// extractEnvFromSlice extracts env vars from a []any slice.
func extractEnvFromSlice(envSlice []any) []EnvVar {
	env := make([]EnvVar, 0, len(envSlice))

	for _, ev := range envSlice {
		evMap, ok := ev.(map[string]any)
		if !ok {
			continue
		}

		name, _ := evMap["name"].(string)
		value, _ := evMap["value"].(string)

		if name != "" {
			env = append(env, EnvVar{Name: name, Value: value})
		}
	}

	return env
}

// extractEnvFromStringMap extracts env vars from a map[string]any where values are strings.
func extractEnvFromStringMap(envVarMap map[string]any) []EnvVar {
	env := make([]EnvVar, 0, len(envVarMap))

	for name, value := range envVarMap {
		if strValue, ok := value.(string); ok {
			env = append(env, EnvVar{Name: name, Value: strValue})
		}
	}

	return env
}

// extractEnvFromStruct extracts environment variables from a struct using reflection.
// Handles both map[string]string and []EnvVar formats.
func extractEnvFromStruct(opts any) []EnvVar {
	if opts == nil {
		return nil
	}

	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	envField := v.FieldByName("Env")
	if !envField.IsValid() {
		return nil
	}

	switch envField.Kind() {
	case reflect.Map:
		return extractEnvFromReflectMap(envField)
	case reflect.Slice:
		return extractEnvFromReflectSlice(envField)
	default:
		return nil
	}
}

// extractEnvFromReflectMap extracts env vars from a reflect.Value of kind Map.
func extractEnvFromReflectMap(envField reflect.Value) []EnvVar {
	env := make([]EnvVar, 0, envField.Len())

	for _, key := range envField.MapKeys() {
		value := envField.MapIndex(key)
		if key.Kind() == reflect.String && value.Kind() == reflect.String {
			env = append(env, EnvVar{Name: key.String(), Value: value.String()})
		}
	}

	return env
}

// extractEnvFromReflectSlice extracts env vars from a reflect.Value of kind Slice.
func extractEnvFromReflectSlice(envField reflect.Value) []EnvVar {
	env := make([]EnvVar, 0, envField.Len())

	for i := range envField.Len() {
		elem := envField.Index(i)
		if elem.Kind() != reflect.Struct {
			continue
		}

		nameField := elem.FieldByName("Name")
		valueField := elem.FieldByName("Value")

		if !nameField.IsValid() || !valueField.IsValid() {
			continue
		}

		name := nameField.String()
		if name != "" {
			env = append(env, EnvVar{Name: name, Value: valueField.String()})
		}
	}

	return env
}

// terminalExecutionResult implements tools.ExecutionResult for terminal output.
type terminalExecutionResult struct {
	stdout     string
	stderr     string
	exitCode   int
	truncated  bool
	terminalID string
}

// GetStdout implements the GetStdout operation.
func (r *terminalExecutionResult) GetStdout() string {
	return r.stdout
}

// GetStderr implements the GetStderr operation.
func (r *terminalExecutionResult) GetStderr() string {
	return r.stderr
}

// GetExitCode implements the GetExitCode operation.
func (r *terminalExecutionResult) GetExitCode() int {
	return r.exitCode
}

// GetMetadata implements the GetMetadata operation.
func (r *terminalExecutionResult) GetMetadata() map[string]any {
	return map[string]any{
		"terminal_id": r.terminalID,
	}
}
