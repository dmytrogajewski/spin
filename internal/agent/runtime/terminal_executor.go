package runtime

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
	var env []EnvVar

	if opts != nil {
		// Check if opts has Env field (struct-based, e.g., ExecuteOptions).
		if envMap := extractEnvFromStruct(opts); envMap != nil {
			env = envMap
		} else if envMap, ok := opts.(map[string]any); ok {
			// Check if opts is a map with env vars.
			if envVars, exists := envMap["env"]; exists {
				if envSlice, ok := envVars.([]any); ok {
					env = make([]EnvVar, 0, len(envSlice))
					for _, ev := range envSlice {
						if evMap, ok := ev.(map[string]any); ok {
							name, _ := evMap["name"].(string)

							value, _ := evMap["value"].(string)
							if name != "" {
								env = append(env, EnvVar{Name: name, Value: value})
							}
						}
					}
				} else if envMap, ok := envVars.(map[string]any); ok {
					// Handle map[string]string format.
					env = make([]EnvVar, 0, len(envMap))
					for name, value := range envMap {
						if strValue, ok := value.(string); ok {
							env = append(env, EnvVar{Name: name, Value: strValue})
						}
					}
				}
			}
		}
	}

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

	// Handle map[string]string format (e.g., ExecuteOptions.Env).
	if envField.Kind() == reflect.Map {
		env := make([]EnvVar, 0, envField.Len())
		for _, key := range envField.MapKeys() {
			value := envField.MapIndex(key)
			if key.Kind() == reflect.String && value.Kind() == reflect.String {
				env = append(env, EnvVar{Name: key.String(), Value: value.String()})
			}
		}

		return env
	}

	// Handle []EnvVar format.
	if envField.Kind() == reflect.Slice {
		env := make([]EnvVar, 0, envField.Len())
		for i := range envField.Len() {
			elem := envField.Index(i)
			if elem.Kind() == reflect.Struct {
				nameField := elem.FieldByName("Name")

				valueField := elem.FieldByName("Value")
				if nameField.IsValid() && valueField.IsValid() {
					name := nameField.String()

					value := valueField.String()
					if name != "" {
						env = append(env, EnvVar{Name: name, Value: value})
					}
				}
			}
		}

		return env
	}

	return nil
}

// terminalExecutionResult implements tools.ExecutionResult for terminal output.
type terminalExecutionResult struct {
	stdout     string
	stderr     string
	exitCode   int
	truncated  bool
	terminalID string
}

func (r *terminalExecutionResult) GetStdout() string {
	return r.stdout
}

func (r *terminalExecutionResult) GetStderr() string {
	return r.stderr
}

func (r *terminalExecutionResult) GetExitCode() int {
	return r.exitCode
}

func (r *terminalExecutionResult) GetMetadata() map[string]any {
	return map[string]any{
		"terminal_id": r.terminalID,
	}
}
