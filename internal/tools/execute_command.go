package tools

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ExecuteCommandTool implements command execution functionality.
// It requires an Executor and Validator from the core package.
type ExecuteCommandTool struct {
	executor  interface{} // agent.Executor - using interface{} to avoid circular import
	validator interface{} // security.Validator - using interface{} to avoid circular import
}

// NewExecuteCommandTool creates a new execute command tool.
func NewExecuteCommandTool(executor, validator interface{}) *ExecuteCommandTool {
	return &ExecuteCommandTool{
		executor:  executor,
		validator: validator,
	}
}

func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

func (t *ExecuteCommandTool) Description() string {
	return "Execute a shell command"
}

func (t *ExecuteCommandTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"command": {
						Type:        "string",
						Description: "The command to execute",
					},
					"working_directory": {
						Type:        "string",
						Description: "The working directory for the command (optional)",
					},
					"timeout": {
						Type:        "number",
						Description: "Timeout in seconds for command execution (optional, defaults to 30s)",
					},
				},
				Required: []string{"command"},
			},
		},
	}
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := t.validateExecutor(); err != nil {
		return ToolResult{Success: false, Error: err.Error()}, nil
	}

	cmdStr, err := t.extractCommand(params)
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}, nil
	}

	parts := t.parseCommand(cmdStr)
	if len(parts) == 0 {
		return ToolResult{Success: false, Error: "command cannot be empty"}, nil
	}

	// Parse timeout parameter (optional, defaults to 30s)
	timeout := time.Duration(params.GetFloat64Or("timeout", 30.0)) * time.Second

	// Create a context with the specified timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	executeMethod, err := t.getExecuteMethod()
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}, nil
	}

	cmdValue, err := t.createCommand(executeMethod.Type(), parts, params)
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}, nil
	}

	return t.executeCommand(cmdCtx, executeMethod, cmdValue)
}

// validateExecutor validates that the executor is configured.
func (t *ExecuteCommandTool) validateExecutor() error {
	if t.executor == nil {
		return fmt.Errorf("executor not configured")
	}
	return nil
}

// extractCommand extracts and validates the command parameter.
func (t *ExecuteCommandTool) extractCommand(params ToolParameters) (string, error) {
	cmdStr, err := params.GetString("command")
	if err != nil || cmdStr == "" {
		return "", fmt.Errorf("command parameter must be a non-empty string")
	}
	return cmdStr, nil
}

// parseCommand parses the command string into parts.
func (t *ExecuteCommandTool) parseCommand(cmdStr string) []string {
	return strings.Fields(cmdStr)
}

// getExecuteMethod gets the Execute method using reflection.
func (t *ExecuteCommandTool) getExecuteMethod() (reflect.Value, error) {
	executorVal := reflect.ValueOf(t.executor)
	executeMethod := executorVal.MethodByName("Execute")
	if !executeMethod.IsValid() {
		return reflect.Value{}, fmt.Errorf("executor does not have Execute method")
	}

	methodType := executeMethod.Type()
	if methodType.NumIn() < 3 {
		return reflect.Value{}, fmt.Errorf("invalid Execute method signature")
	}

	return executeMethod, nil
}

// createCommand creates a command value based on the method signature.
func (t *ExecuteCommandTool) createCommand(methodType reflect.Type, parts []string, params ToolParameters) (reflect.Value, error) {
	cmdType := methodType.In(1)

	switch cmdType.Kind() {
	case reflect.Interface:
		return t.createDynamicCommand(parts, params), nil
	case reflect.Ptr:
		return t.createTypedCommand(cmdType, parts, params)
	default:
		return reflect.Value{}, fmt.Errorf("unexpected Execute command parameter type")
	}
}

// createDynamicCommand creates a dynamic command for interface{} parameters.
func (t *ExecuteCommandTool) createDynamicCommand(parts []string, params ToolParameters) reflect.Value {
	type dynamicCommand struct {
		Program string
		Args    []string
		WorkDir string
		Raw     string
	}

	cmd := &dynamicCommand{
		Program: parts[0],
		Args:    parts[1:],
		Raw:     strings.Join(parts, " "),
	}

	if workDir, err := params.GetString("working_directory"); err == nil && workDir != "" {
		cmd.WorkDir = workDir
	}

	return reflect.ValueOf(cmd)
}

// createTypedCommand creates a typed command for pointer parameters.
func (t *ExecuteCommandTool) createTypedCommand(cmdType reflect.Type, parts []string, params ToolParameters) (reflect.Value, error) {
	cmdStructType := cmdType.Elem()
	cmdValue := reflect.New(cmdStructType)
	cmdElem := cmdValue.Elem()

	t.setCommandFields(cmdElem, parts, params)

	return cmdValue, nil
}

// setCommandFields sets the fields of a typed command.
func (t *ExecuteCommandTool) setCommandFields(cmdElem reflect.Value, parts []string, params ToolParameters) {
	t.setStringField(cmdElem, "Program", parts[0])
	t.setStringSliceField(cmdElem, "Args", parts[1:])
	t.setStringField(cmdElem, "Raw", strings.Join(parts, " "))

	if workDir, err := params.GetString("working_directory"); err == nil && workDir != "" {
		t.setStringField(cmdElem, "WorkDir", workDir)
	}
}

// setStringField sets a string field if it exists and is settable.
func (t *ExecuteCommandTool) setStringField(cmdElem reflect.Value, fieldName, value string) {
	if field := cmdElem.FieldByName(fieldName); field.IsValid() && field.CanSet() {
		field.SetString(value)
	}
}

// setStringSliceField sets a string slice field if it exists and is settable.
func (t *ExecuteCommandTool) setStringSliceField(cmdElem reflect.Value, fieldName string, values []string) {
	if field := cmdElem.FieldByName(fieldName); field.IsValid() && field.CanSet() {
		argsSlice := reflect.MakeSlice(reflect.TypeOf([]string{}), len(values), len(values))
		for i, val := range values {
			argsSlice.Index(i).SetString(val)
		}
		field.Set(argsSlice)
	}
}

// executeCommand executes the command and returns the result.
func (t *ExecuteCommandTool) executeCommand(ctx context.Context, executeMethod reflect.Value, cmdValue reflect.Value) (ToolResult, error) {
	methodType := executeMethod.Type()
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		cmdValue,
		reflect.Zero(methodType.In(2)), // nil for ExecuteOptions
	}

	results := executeMethod.Call(args)
	if len(results) != 2 {
		return ToolResult{Success: false, Error: "unexpected Execute return values"}, nil
	}

	if !results[1].IsNil() {
		return t.handleExecuteError(results[0], results[1])
	}

	return t.extractExecuteResult(results[0])
}

// handleExecuteError handles execution errors.
func (t *ExecuteCommandTool) handleExecuteError(resultVal, errVal reflect.Value) (ToolResult, error) {
	stderr := t.extractStderr(resultVal)
	return ToolResult{
		Success: false,
		Output:  stderr,
		Error:   errVal.Interface().(error).Error(),
	}, nil
}

// extractStderr extracts stderr from the result value.
func (t *ExecuteCommandTool) extractStderr(resultVal reflect.Value) string {
	if resultVal.IsNil() {
		return ""
	}

	resultElem := t.dereferenceValue(resultVal)
	if stderrField := resultElem.FieldByName("Stderr"); stderrField.IsValid() {
		return stderrField.String()
	}
	return ""
}

// extractExecuteResult extracts the execution result.
func (t *ExecuteCommandTool) extractExecuteResult(resultVal reflect.Value) (ToolResult, error) {
	if resultVal.Kind() == reflect.Ptr && resultVal.IsNil() {
		return ToolResult{Success: false, Error: "nil result from Execute"}, nil
	}

	resultElem := t.dereferenceValue(resultVal)
	stdout := t.extractField(resultElem, "Stdout")
	stderr := t.extractField(resultElem, "Stderr")
	exitCode := t.extractIntField(resultElem, "ExitCode")

	output := t.combineOutput(stdout, stderr)

	return ToolResult{
		Success: exitCode == 0,
		Output:  output,
	}, nil
}

// dereferenceValue dereferences a reflect value if it's a pointer or interface.
func (t *ExecuteCommandTool) dereferenceValue(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	return value
}

// extractField extracts a string field from a struct.
func (t *ExecuteCommandTool) extractField(resultElem reflect.Value, fieldName string) string {
	if field := resultElem.FieldByName(fieldName); field.IsValid() {
		return field.String()
	}
	return ""
}

// extractIntField extracts an int field from a struct.
func (t *ExecuteCommandTool) extractIntField(resultElem reflect.Value, fieldName string) int {
	if field := resultElem.FieldByName(fieldName); field.IsValid() {
		return int(field.Int())
	}
	return 0
}

// combineOutput combines stdout and stderr into a single output string.
func (t *ExecuteCommandTool) combineOutput(stdout, stderr string) string {
	if stderr == "" {
		return stdout
	}
	if stdout == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}
