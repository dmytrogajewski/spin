package tools

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/filesearch"
	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/patchapply"
)

// ReadFileTool implements file reading functionality.
type ReadFileTool struct{}

// NewReadFileTool creates a new read file tool.
func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (t *ReadFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"path": {
						Type:        "string",
						Description: "The path to the file to read",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  string(content),
	}, nil
}

// WriteFileTool implements file writing functionality.
type WriteFileTool struct{}

// NewWriteFileTool creates a new write file tool.
func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

func (t *WriteFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"path": {
						Type:        "string",
						Description: "The path to the file to write",
					},
					"content": {
						Type:        "string",
						Description: "The content to write to the file",
					},
				},
				Required: []string{"path", "content"},
			},
		},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	content, ok := params["content"].(string)
	if !ok {
		return ToolResult{
			Success: false,
			Error:   "content parameter must be a string",
		}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
	}, nil
}

// ListDirectoryTool implements directory listing functionality.
type ListDirectoryTool struct{}

// NewListDirectoryTool creates a new list directory tool.
func NewListDirectoryTool() *ListDirectoryTool {
	return &ListDirectoryTool{}
}

func (t *ListDirectoryTool) Name() string {
	return "list_directory"
}

func (t *ListDirectoryTool) Description() string {
	return "List the contents of a directory"
}

func (t *ListDirectoryTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"path": {
						Type:        "string",
						Description: "The path to the directory to list",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

func (t *ListDirectoryTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read directory: %v", err),
		}, nil
	}

	var output strings.Builder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		typeStr := "file"
		if entry.IsDir() {
			typeStr = "dir"
		}

		fmt.Fprintf(&output, "%s\t%s\t%d bytes\n", entry.Name(), typeStr, info.Size())
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

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
					"workdir": {
						Type:        "string",
						Description: "The working directory for the command (optional)",
					},
				},
				Required: []string{"command"},
			},
		},
	}
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
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

	executeMethod, err := t.getExecuteMethod()
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}, nil
	}

	cmdValue, err := t.createCommand(executeMethod.Type(), parts, params)
	if err != nil {
		return ToolResult{Success: false, Error: err.Error()}, nil
	}

	return t.executeCommand(ctx, executeMethod, cmdValue)
}

// validateExecutor validates that the executor is configured.
func (t *ExecuteCommandTool) validateExecutor() error {
	if t.executor == nil {
		return fmt.Errorf("executor not configured")
	}
	return nil
}

// extractCommand extracts and validates the command parameter.
func (t *ExecuteCommandTool) extractCommand(params map[string]interface{}) (string, error) {
	cmdStr, ok := params["command"].(string)
	if !ok || cmdStr == "" {
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
func (t *ExecuteCommandTool) createCommand(methodType reflect.Type, parts []string, params map[string]interface{}) (reflect.Value, error) {
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
func (t *ExecuteCommandTool) createDynamicCommand(parts []string, params map[string]interface{}) reflect.Value {
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

	if workDir, ok := params["workdir"].(string); ok && workDir != "" {
		cmd.WorkDir = workDir
	}

	return reflect.ValueOf(cmd)
}

// createTypedCommand creates a typed command for pointer parameters.
func (t *ExecuteCommandTool) createTypedCommand(cmdType reflect.Type, parts []string, params map[string]interface{}) (reflect.Value, error) {
	cmdStructType := cmdType.Elem()
	cmdValue := reflect.New(cmdStructType)
	cmdElem := cmdValue.Elem()

	t.setCommandFields(cmdElem, parts, params)

	return cmdValue, nil
}

// setCommandFields sets the fields of a typed command.
func (t *ExecuteCommandTool) setCommandFields(cmdElem reflect.Value, parts []string, params map[string]interface{}) {
	t.setStringField(cmdElem, "Program", parts[0])
	t.setStringSliceField(cmdElem, "Args", parts[1:])
	t.setStringField(cmdElem, "Raw", strings.Join(parts, " "))

	if workDir, ok := params["workdir"].(string); ok && workDir != "" {
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

// GetContextTool implements environment context retrieval.
type GetContextTool struct {
	context interface{} // agent.Environment - using interface{} to avoid circular import
}

// NewGetContextTool creates a new get context tool.
func NewGetContextTool(context interface{}) *GetContextTool {
	return &GetContextTool{
		context: context,
	}
}

func (t *GetContextTool) Name() string {
	return "get_context"
}

func (t *GetContextTool) Description() string {
	return "Get environment context information"
}

func (t *GetContextTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type:       "object",
				Properties: map[string]PropertyDefinition{},
				Required:   []string{},
			},
		},
	}
}

func (t *GetContextTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	if t.context == nil {
		return ToolResult{
			Success: false,
			Error:   "context not available",
		}, nil
	}

	// Use reflection to call String() method to avoid circular import
	// The context is agent.Environment which implements String() string
	val := reflect.ValueOf(t.context)

	// Check if the context has a String() method
	stringMethod := val.MethodByName("String")
	if !stringMethod.IsValid() {
		return ToolResult{
			Success: false,
			Error:   "context does not implement String() method",
		}, nil
	}

	// Call String() method
	results := stringMethod.Call(nil)
	if len(results) != 1 {
		return ToolResult{
			Success: false,
			Error:   "invalid String() method signature",
		}, nil
	}

	// Extract string result
	output, ok := results[0].Interface().(string)
	if !ok {
		return ToolResult{
			Success: false,
			Error:   "String() method did not return a string",
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  output,
	}, nil
}

// ApplyPatchTool implements structured patch application functionality.
type ApplyPatchTool struct {
	workspaceRoot string
}

// NewApplyPatchTool creates a new apply patch tool.
func NewApplyPatchTool(workspaceRoot string) *ApplyPatchTool {
	return &ApplyPatchTool{
		workspaceRoot: workspaceRoot,
	}
}

func (t *ApplyPatchTool) Name() string {
	return "apply_patch"
}

func (t *ApplyPatchTool) Description() string {
	return "Apply a structured patch to modify files in the workspace"
}

func (t *ApplyPatchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"patch_text": {
						Type:        "string",
						Description: "The patch text in Spin's patch format (*** Begin Patch...*** End Patch)",
					},
					"workspace_root": {
						Type:        "string",
						Description: "The workspace root directory (optional, defaults to tool's workspace)",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true, validate without applying changes",
					},
					"force": {
						Type:        "boolean",
						Description: "If true, allow overwriting existing files on Add operations",
					},
				},
				Required: []string{"patch_text"},
			},
		},
	}
}

func (t *ApplyPatchTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	// Extract patch_text parameter
	patchText, ok := params["patch_text"].(string)
	if !ok || patchText == "" {
		return ToolResult{
			Success: false,
			Error:   "patch_text parameter must be a non-empty string",
		}, nil
	}

	// Extract workspace_root parameter (optional)
	workspaceRoot := t.workspaceRoot
	if customRoot, ok := params["workspace_root"].(string); ok && customRoot != "" {
		workspaceRoot = customRoot
	}

	// Extract dry_run parameter (optional)
	dryRun := false
	if dryRunVal, ok := params["dry_run"].(bool); ok {
		dryRun = dryRunVal
	}

	// Extract force parameter (optional)
	force := false
	if forceVal, ok := params["force"].(bool); ok {
		force = forceVal
	}

	// Parse the patch
	parser := patchapply.NewParser(patchText)
	patch, err := parser.Parse()
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("parse error: %v", err),
		}, nil
	}

	// Create applier
	applier, err := patchapply.NewApplier(workspaceRoot)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create applier: %v", err),
		}, nil
	}

	// Configure applier
	applier.SetDryRun(dryRun)
	applier.SetForceOverwrite(force)

	// Apply the patch
	result, err := applier.Apply(patch)
	if err != nil {
		// Extract error message
		errMsg := err.Error()
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to apply patch: %v", errMsg),
		}, nil
	}

	// Format output
	var output strings.Builder
	if dryRun {
		output.WriteString("Dry run completed successfully. No files were modified.\n\n")
	} else {
		output.WriteString("Patch applied successfully.\n\n")
	}

	if len(result.FilesCreated) > 0 {
		output.WriteString(fmt.Sprintf("Created %d file(s):\n", len(result.FilesCreated)))
		for _, file := range result.FilesCreated {
			output.WriteString(fmt.Sprintf("  + %s\n", file))
		}
	}

	if len(result.FilesDeleted) > 0 {
		output.WriteString(fmt.Sprintf("Deleted %d file(s):\n", len(result.FilesDeleted)))
		for _, file := range result.FilesDeleted {
			output.WriteString(fmt.Sprintf("  - %s\n", file))
		}
	}

	if len(result.FilesUpdated) > 0 {
		output.WriteString(fmt.Sprintf("Updated %d file(s):\n", len(result.FilesUpdated)))
		for _, file := range result.FilesUpdated {
			output.WriteString(fmt.Sprintf("  ~ %s\n", file))
		}
	}

	if len(result.FilesMoved) > 0 {
		output.WriteString(fmt.Sprintf("Moved %d file(s):\n", len(result.FilesMoved)))
		for oldPath, newPath := range result.FilesMoved {
			output.WriteString(fmt.Sprintf("  %s → %s\n", oldPath, newPath))
		}
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// FileSearchTool implements file search functionality with fuzzy matching.
type FileSearchTool struct {
	workspaceRoot string
	searcher      *filesearch.Searcher
	mu            sync.RWMutex
}

// NewFileSearchTool creates a new file search tool.
func NewFileSearchTool(workspaceRoot string) *FileSearchTool {
	return &FileSearchTool{
		workspaceRoot: workspaceRoot,
	}
}

func (t *FileSearchTool) Name() string {
	return "file_search"
}

func (t *FileSearchTool) Description() string {
	return "Search for files in the workspace using fuzzy matching with .gitignore support"
}

func (t *FileSearchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"query": {
						Type:        "string",
						Description: "The search query (fuzzy matched against file paths)",
					},
					"workspace_root": {
						Type:        "string",
						Description: "The workspace root directory (optional, defaults to tool's workspace)",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of results to return (default: 10)",
					},
				},
				Required: []string{"query"},
			},
		},
	}
}

func (t *FileSearchTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	// Extract query parameter
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return ToolResult{
			Success: false,
			Error:   "query parameter must be a non-empty string",
		}, nil
	}

	// Extract workspace_root parameter (optional)
	workspaceRoot := t.workspaceRoot
	if customRoot, ok := params["workspace_root"].(string); ok && customRoot != "" {
		workspaceRoot = customRoot
	}

	// Extract limit parameter (optional, default 10)
	limit := 10
	if limitVal, ok := params["limit"].(float64); ok {
		limit = int(limitVal)
	} else if limitVal, ok := params["limit"].(int); ok {
		limit = limitVal
	}

	// Get or create searcher for this workspace
	searcher, err := t.getOrCreateSearcher(workspaceRoot)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create searcher: %v", err),
		}, nil
	}

	// Index if not already indexed
	if !searcher.IsIndexed() {
		if err := searcher.IndexAsync(ctx); err != nil {
			return ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to index workspace: %v", err),
			}, nil
		}
	}

	// Search
	matches := searcher.Search(query, limit)

	// Format output
	var output strings.Builder
	if len(matches) == 0 {
		output.WriteString(fmt.Sprintf("No files found matching '%s'\n", query))
	} else {
		output.WriteString(fmt.Sprintf("Found %d file(s) matching '%s':\n\n", len(matches), query))
		for i, match := range matches {
			output.WriteString(fmt.Sprintf("%d. %s (score: %d)\n", i+1, match.Path, match.Score))
		}
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// getOrCreateSearcher returns the searcher for the given workspace, creating it if needed.
func (t *FileSearchTool) getOrCreateSearcher(workspaceRoot string) (*filesearch.Searcher, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If searcher exists and matches workspace, return it
	if t.searcher != nil && t.workspaceRoot == workspaceRoot {
		return t.searcher, nil
	}

	// Create new searcher
	searcher, err := filesearch.NewSearcher(workspaceRoot)
	if err != nil {
		return nil, err
	}

	// Update state
	t.searcher = searcher
	t.workspaceRoot = workspaceRoot

	return searcher, nil
}

// GitContextTool implements Git repository context retrieval.
type GitContextTool struct {
	workspaceRoot string
}

// NewGitContextTool creates a new git context tool.
func NewGitContextTool(workspaceRoot string) *GitContextTool {
	return &GitContextTool{
		workspaceRoot: workspaceRoot,
	}
}

func (t *GitContextTool) Name() string {
	return "git_context"
}

func (t *GitContextTool) Description() string {
	return "Get Git repository context including branch, status, and modifications"
}

func (t *GitContextTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"workspace_root": {
						Type:        "string",
						Description: "The workspace root directory (optional, defaults to tool's workspace)",
					},
					"include_diff": {
						Type:        "boolean",
						Description: "If true, include diff summary (default: false)",
					},
				},
				Required: []string{},
			},
		},
	}
}

func (t *GitContextTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	// Get workspace root
	workspaceRoot := t.workspaceRoot
	if root, ok := params["workspace_root"].(string); ok && root != "" {
		workspaceRoot = root
	}

	// Discover git repository
	repo, err := git.Discover(ctx, workspaceRoot)
	if err != nil {
		// Gracefully handle non-git directories
		return ToolResult{
			Success: true,
			Output:  fmt.Sprintf("Not a Git repository: %v\n", err),
		}, nil
	}

	var output strings.Builder
	output.WriteString("Git Repository Context:\n")
	output.WriteString("======================\n\n")

	// Get status (includes branch info)
	status, err := repo.Status(ctx)
	if err != nil {
		return ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to get git status: %v\n", err),
		}, nil
	}

	// Branch info
	output.WriteString(fmt.Sprintf("Branch: %s\n", status.Branch))
	if status.RemoteBranch != "" {
		output.WriteString(fmt.Sprintf("Remote: %s\n", status.RemoteBranch))
		output.WriteString(fmt.Sprintf("Ahead: %d, Behind: %d\n", status.Ahead, status.Behind))
	}
	if status.Detached {
		output.WriteString("(detached HEAD)\n")
	}

	// Commit hash
	if status.Hash != "" {
		hashLen := len(status.Hash)
		if hashLen > 8 {
			hashLen = 8
		}
		output.WriteString(fmt.Sprintf("Commit: %s\n", status.Hash[:hashLen]))
	}

	// File status
	output.WriteString(fmt.Sprintf("\nModified files: %d\n", len(status.ModifiedFiles)))
	output.WriteString(fmt.Sprintf("Untracked files: %d\n", len(status.UntrackedFiles)))

	if len(status.ModifiedFiles) > 0 {
		output.WriteString("\nModified:\n")
		for _, file := range status.ModifiedFiles {
			output.WriteString(fmt.Sprintf("  - %s (%s)\n", file.Path, file.Worktree))
		}
	}

	if len(status.UntrackedFiles) > 0 && len(status.UntrackedFiles) < 20 {
		output.WriteString("\nUntracked:\n")
		for _, file := range status.UntrackedFiles {
			output.WriteString(fmt.Sprintf("  - %s\n", file))
		}
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}
