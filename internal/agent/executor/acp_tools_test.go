package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	errInvalidPath  = errors.New("invalid path")
	errInvalidPath2 = errors.New("invalid path")
)

// mockFilesystemClient implements FilesystemClient for testing.
type mockFilesystemClient struct {
	readPath    string
	writePath   string
	writeErr    error
	readContent string
	readErr     error
}

func (m *mockFilesystemClient) ReadTextFile(_ context.Context, path string, _, _ *int) (string, error) {
	m.readPath = path
	if m.readErr != nil {
		return "", m.readErr
	}

	return m.readContent, nil
}

func (m *mockFilesystemClient) WriteTextFile(_ context.Context, path, _ string) error {
	m.writePath = path

	return m.writeErr
}

type pathResolutionCase struct {
	name          string
	workDir       string
	inputPath     string
	expectedPath  string
	expectError   bool
	errorContains string
}

func pathResolutionTestCases() []pathResolutionCase {
	return []pathResolutionCase{
		{
			name: "relative path is resolved to workDir", workDir: "/home/user/workspace",
			inputPath: "src/main.py", expectedPath: "/home/user/workspace/src/main.py",
		},
		{
			name: "absolute path within workDir is allowed", workDir: "/home/user/workspace",
			inputPath: "/home/user/workspace/src/main.py", expectedPath: "/home/user/workspace/src/main.py",
		},
		{
			name: "absolute path outside workDir is rejected", workDir: "/home/user/workspace",
			inputPath: "/tmp/file.py", expectError: true, errorContains: "outside the allowed workspace",
		},
		{
			name: "path traversal with .. is rejected", workDir: "/home/user/workspace",
			inputPath: "../../../etc/passwd", expectError: true, errorContains: "outside the allowed workspace",
		},
		{
			name:    "absolute path with similar prefix but different dir is rejected",
			workDir: "/home/user/workspace", inputPath: "/home/user/workspace2/file.py",
			expectError: true, errorContains: "outside the allowed workspace",
		},
		{
			name: "path with . components is cleaned", workDir: "/home/user/workspace",
			inputPath: "./src/../src/main.py", expectedPath: "/home/user/workspace/src/main.py",
		},
		{
			name: "simple filename is resolved to workDir", workDir: "/home/user/workspace",
			inputPath: "file.txt", expectedPath: "/home/user/workspace/file.txt",
		},
		{
			name: "nested relative path is resolved", workDir: "/home/user/workspace",
			inputPath: "src/components/Button.tsx", expectedPath: "/home/user/workspace/src/components/Button.tsx",
		},
	}
}

func TestACPWriteFileTool_PathResolution(t *testing.T) {
	t.Parallel()

	for _, tt := range pathResolutionTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockFS := &mockFilesystemClient{}
			runtime := &ACPRuntime{workDir: tt.workDir, filesystemClient: mockFS}
			tool := NewACPWriteFileTool(runtime)

			params, err := tools.FromMap(map[string]any{"path": tt.inputPath, "content": "test content"})
			if err != nil {
				t.Fatalf("failed to create params: %v", err)
			}

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Fatalf("unexpected error from Execute: %v", err)
			}

			assertToolResult(t, result, tt.expectError, tt.errorContains, mockFS.writePath, tt.expectedPath)
		})
	}
}

func TestACPReadFileTool_PathResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		workDir       string
		inputPath     string
		expectedPath  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "relative path is resolved to workDir",
			workDir:      "/home/user/workspace",
			inputPath:    "src/main.py",
			expectedPath: "/home/user/workspace/src/main.py",
			expectError:  false,
		},
		{
			name:         "absolute path within workDir is allowed",
			workDir:      "/home/user/workspace",
			inputPath:    "/home/user/workspace/src/main.py",
			expectedPath: "/home/user/workspace/src/main.py",
			expectError:  false,
		},
		{
			name:          "absolute path outside workDir is rejected",
			workDir:       "/home/user/workspace",
			inputPath:     "/etc/passwd",
			expectError:   true,
			errorContains: "outside the allowed workspace",
		},
		{
			name:          "path traversal with .. is rejected",
			workDir:       "/home/user/workspace",
			inputPath:     "../../etc/passwd",
			expectError:   true,
			errorContains: "outside the allowed workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockFS := &mockFilesystemClient{
				readContent: "file content",
			}
			runtime := &ACPRuntime{
				workDir:          tt.workDir,
				filesystemClient: mockFS,
			}
			tool := NewACPReadFileTool(runtime)

			params, err := tools.FromMap(map[string]any{
				"path": tt.inputPath,
			})
			if err != nil {
				t.Fatalf("failed to create params: %v", err)
			}

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Fatalf("unexpected error from Execute: %v", err)
			}

			assertToolResult(t, result, tt.expectError, tt.errorContains, mockFS.readPath, tt.expectedPath)
		})
	}
}

// assertToolResult validates a tool result based on expected error or success conditions.
func assertToolResult(t *testing.T, result tools.ToolResult, expectError bool, errorContains, actualPath, expectedPath string) {
	t.Helper()

	if expectError {
		assertToolError(t, result, errorContains)

		return
	}

	if !result.Success {
		t.Errorf("expected success but got error: %s", result.Error)
	}

	if actualPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, actualPath)
	}
}

// assertToolError validates that a tool result is an error with expected content.
func assertToolError(t *testing.T, result tools.ToolResult, errorContains string) {
	t.Helper()

	if result.Success {
		t.Errorf("expected failure but got success")
	}

	if errorContains != "" && !containsString(result.Error, errorContains) {
		t.Errorf("expected error to contain %q, got %q", errorContains, result.Error)
	}
}

func TestACPWriteFileTool_InvalidPathErrorMessage(t *testing.T) {
	t.Parallel()
	// Test that when the client returns "invalid path" error, we provide a helpful message.
	mockFS := &mockFilesystemClient{
		writeErr: errInvalidPath,
	}
	runtime := &ACPRuntime{
		workDir:          "/home/user/workspace",
		filesystemClient: mockFS,
	}
	tool := NewACPWriteFileTool(runtime)

	params, err := tools.FromMap(map[string]any{
		"path":    "test.txt",
		"content": "test content",
	})
	if err != nil {
		t.Fatalf("failed to create params: %v", err)
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error from Execute: %v", err)
	}

	if result.Success {
		t.Error("expected failure but got success")
	}

	if !containsString(result.Error, "outside the allowed workspace") {
		t.Errorf("expected helpful error message, got: %s", result.Error)
	}

	if !containsString(result.Error, "/home/user/workspace") {
		t.Errorf("expected error to mention workspace directory, got: %s", result.Error)
	}
}

func TestACPReadFileTool_InvalidPathErrorMessage(t *testing.T) {
	t.Parallel()
	// Test that when the client returns "invalid path" error, we provide a helpful message.
	mockFS := &mockFilesystemClient{
		readErr: errInvalidPath2,
	}
	runtime := &ACPRuntime{
		workDir:          "/home/user/workspace",
		filesystemClient: mockFS,
	}
	tool := NewACPReadFileTool(runtime)

	params, err := tools.FromMap(map[string]any{
		"path": "test.txt",
	})
	if err != nil {
		t.Fatalf("failed to create params: %v", err)
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error from Execute: %v", err)
	}

	if result.Success {
		t.Error("expected failure but got success")
	}

	if !containsString(result.Error, "outside the allowed workspace") {
		t.Errorf("expected helpful error message, got: %s", result.Error)
	}
}

func TestIsPathWithinWorkspace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		workDir  string
		expected bool
	}{
		{
			name:     "exact match",
			path:     "/home/user/workspace",
			workDir:  "/home/user/workspace",
			expected: true,
		},
		{
			name:     "file within workspace",
			path:     "/home/user/workspace/file.txt",
			workDir:  "/home/user/workspace",
			expected: true,
		},
		{
			name:     "nested file within workspace",
			path:     "/home/user/workspace/src/main.py",
			workDir:  "/home/user/workspace",
			expected: true,
		},
		{
			name:     "path outside workspace",
			path:     "/tmp/file.txt",
			workDir:  "/home/user/workspace",
			expected: false,
		},
		{
			name:     "similar prefix but different directory",
			path:     "/home/user/workspace2/file.txt",
			workDir:  "/home/user/workspace",
			expected: false,
		},
		{
			name:     "parent directory",
			path:     "/home/user",
			workDir:  "/home/user/workspace",
			expected: false,
		},
		{
			name:     "path with trailing separator",
			path:     "/home/user/workspace/src/",
			workDir:  "/home/user/workspace",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isPathWithinWorkspace(tt.path, tt.workDir)
			if result != tt.expected {
				t.Errorf("isPathWithinWorkspace(%q, %q) = %v, want %v", tt.path, tt.workDir, result, tt.expected)
			}
		})
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// TestACPWriteFileTool_ContextWorkDir tests that workDir from context takes precedence over runtime.workDir.
func TestACPWriteFileTool_ContextWorkDir(t *testing.T) {
	t.Parallel()

	mockFS := &mockFilesystemClient{}
	runtime := &ACPRuntime{
		workDir:          "/runtime/workspace", // This should be overridden by context.
		filesystemClient: mockFS,
	}
	tool := NewACPWriteFileTool(runtime)

	// Create context with different workDir.
	ctx := ContextWithWorkDir(context.Background(), "/session/workspace")

	params, err := tools.FromMap(map[string]any{
		"path":    "test.py",
		"content": "test content",
	})
	if err != nil {
		t.Fatalf("failed to create params: %v", err)
	}

	result, execErr := tool.Execute(ctx, params)
	if execErr != nil {
		t.Fatalf("unexpected error from Execute: %v", execErr)
	}

	if !result.Success {
		t.Errorf("expected success but got error: %s", result.Error)
	}

	// Verify that the path was resolved using context workDir, not runtime workDir.
	expectedPath := "/session/workspace/test.py"
	if mockFS.writePath != expectedPath {
		t.Errorf("expected path %q (from context workDir), got %q", expectedPath, mockFS.writePath)
	}
}

// TestACPReadFileTool_ContextWorkDir tests that workDir from context takes precedence over runtime.workDir.
func TestACPReadFileTool_ContextWorkDir(t *testing.T) {
	t.Parallel()

	mockFS := &mockFilesystemClient{
		readContent: "test content",
	}
	runtime := &ACPRuntime{
		workDir:          "/runtime/workspace", // This should be overridden by context.
		filesystemClient: mockFS,
	}
	tool := NewACPReadFileTool(runtime)

	// Create context with different workDir.
	ctx := ContextWithWorkDir(context.Background(), "/session/workspace")

	params, err := tools.FromMap(map[string]any{
		"path": "test.py",
	})
	if err != nil {
		t.Fatalf("failed to create params: %v", err)
	}

	result, execErr := tool.Execute(ctx, params)
	if execErr != nil {
		t.Fatalf("unexpected error from Execute: %v", execErr)
	}

	if !result.Success {
		t.Errorf("expected success but got error: %s", result.Error)
	}

	// Verify that the path was resolved using context workDir, not runtime workDir.
	expectedPath := "/session/workspace/test.py"
	if mockFS.readPath != expectedPath {
		t.Errorf("expected path %q (from context workDir), got %q", expectedPath, mockFS.readPath)
	}
}

// TestACPWriteFileTool_FallbackToRuntimeWorkDir tests that runtime.workDir is used when context has no workDir.
func TestACPWriteFileTool_FallbackToRuntimeWorkDir(t *testing.T) {
	t.Parallel()

	mockFS := &mockFilesystemClient{}
	runtime := &ACPRuntime{
		workDir:          "/runtime/workspace",
		filesystemClient: mockFS,
	}
	tool := NewACPWriteFileTool(runtime)

	// Use context without workDir.
	ctx := context.Background()

	params, err := tools.FromMap(map[string]any{
		"path":    "test.py",
		"content": "test content",
	})
	if err != nil {
		t.Fatalf("failed to create params: %v", err)
	}

	result, execErr := tool.Execute(ctx, params)
	if execErr != nil {
		t.Fatalf("unexpected error from Execute: %v", execErr)
	}

	if !result.Success {
		t.Errorf("expected success but got error: %s", result.Error)
	}

	// Verify that the path was resolved using runtime workDir.
	expectedPath := "/runtime/workspace/test.py"
	if mockFS.writePath != expectedPath {
		t.Errorf("expected path %q (from runtime workDir), got %q", expectedPath, mockFS.writePath)
	}
}
