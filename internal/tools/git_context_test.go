package tools

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestGitContextTool_NotARepository(t *testing.T) {
	// Use os.MkdirTemp with empty string to use system temp dir,
	// ensuring the directory is outside any parent git repository
	// (since GOTMPDIR may be set to a directory inside the project).
	tmpDir, err := os.MkdirTemp("", "spin-test-nonrepo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	tool := NewGitContextTool(tmpDir)

	params, _ := FromMap(map[string]any{})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success (graceful handling), got error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "Not a Git repository") {
		t.Errorf("expected message about not being a git repo, got: %s", result.Output)
	}
}

func TestGitContextTool_ValidRepository(t *testing.T) {
	// Use the current repository (spin project itself).
	tool := NewGitContextTool(".")

	params, _ := FromMap(map[string]any{})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Should contain git context information.
	expectedStrings := []string{
		"Git Repository Context:",
		"Branch:",
		"Commit:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result.Output, expected) {
			t.Errorf("expected output to contain %q, got: %s", expected, result.Output)
		}
	}
}

func TestGitContextTool_Schema(t *testing.T) {
	tool := NewGitContextTool("/tmp")
	schema := tool.Schema()

	if schema.Function.Name != "git_context" {
		t.Errorf("expected name 'git_context', got: %s", schema.Function.Name)
	}

	// Tool should have no required parameters.
	if len(schema.Function.Parameters.Required) != 0 {
		t.Errorf("expected no required parameters, got: %d", len(schema.Function.Parameters.Required))
	}

	// Should have optional parameters.
	props := schema.Function.Parameters.Properties
	if _, exists := props["workspace_root"]; !exists {
		t.Errorf("expected 'workspace_root' parameter to be defined")
	}

	if _, exists := props["include_diff"]; !exists {
		t.Errorf("expected 'include_diff' parameter to be defined")
	}
}

func TestGitContextTool_ErrorCases(t *testing.T) {
	tool := NewGitContextTool("/tmp/test")

	// GitContextTool has only optional parameters, so it doesn't fail on invalid params
	// Testing that it handles defaults properly.
	params, _ := FromMap(map[string]any{})

	_, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Git context tool might succeed or fail depending on git availability,
	// so we just check it doesn't panic.
}
