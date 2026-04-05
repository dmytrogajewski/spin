package tools

import (
	"context"
	"strings"
	"testing"
)

// Mock type that implements String() for testing.
type mockEnvironment struct {
	data string
}

func (m *mockEnvironment) String() string {
	return m.data
}

func TestGetContextTool_Success(t *testing.T) {
	t.Parallel(
	// Create a valid mock context with String() method.
	)

	env := &mockEnvironment{
		data: `Environment Context:
- OS: linux (amd64)
- Working Directory: /test/project
- Project Type: go
- Languages: Go`,
	}

	tool := NewGetContextTool(env)
	params, _ := FromMap(map[string]any{})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify output contains expected sections.
	expectedStrings := []string{
		"Environment Context:",
		"linux",
		"amd64",
		"/test/project",
		"go",
		"Go",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result.Output, expected) {
			t.Errorf("expected output to contain %q, got: %q", expected, result.Output)
		}
	}
}

func TestGetContextTool_NilContext(t *testing.T) {
	t.Parallel()

	tool := NewGetContextTool(nil)
	params, _ := FromMap(map[string]any{})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Errorf("expected failure for nil context")
	}

	if result.Error != "context not available" {
		t.Errorf("expected error message 'context not available', got: %s", result.Error)
	}
}

// TestGetContextTool_InvalidType is no longer needed: the NewGetContextTool
// parameter is now fmt.Stringer, so non-conforming types are caught at compile time.

func TestGetContextTool_WithGitInfo(t *testing.T) {
	t.Parallel(
	// Create mock environment with Git information.
	)

	env := &mockEnvironment{
		data: `Environment Context:
- OS: darwin (arm64)
- Working Directory: /Users/test/project
- Project Type: go
- Languages: Go
- Git Branch: master (dirty)`,
	}

	tool := NewGetContextTool(env)
	params, _ := FromMap(map[string]any{})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify Git information is in output.
	if !strings.Contains(result.Output, "master") {
		t.Errorf("expected output to contain Git branch 'master', got: %q", result.Output)
	}

	if !strings.Contains(result.Output, "dirty") {
		t.Errorf("expected output to contain 'dirty' status, got: %q", result.Output)
	}
}

func TestGetContextTool_OutputFormat(t *testing.T) {
	t.Parallel(
	// Verify the fmt.Stringer interface is called correctly.
	)

	env := &mockEnvironment{
		data: `Environment Context:
- OS: linux (amd64)
- Kernel: 6.16.8
- Shell: /bin/bash
- Working Directory: /home/user/project
- Project Type: go
- Languages: Go, Python

Project Structure: 2 files
- main.go (Go, 100 lines)
- test.py (Python, 50 lines)`,
	}

	tool := NewGetContextTool(env)
	params, _ := FromMap(map[string]any{})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify output format matches Environment.String() structure.
	expectedSections := []string{
		"Environment Context:",
		"- OS:",
		"- Kernel:",
		"- Shell:",
		"- Working Directory:",
		"- Project Type:",
		"- Languages:",
		"Project Structure:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(result.Output, section) {
			t.Errorf("expected output to contain section %q, got: %q", section, result.Output)
		}
	}

	// Verify specific values.
	if !strings.Contains(result.Output, "6.16.8") {
		t.Errorf("expected kernel version in output")
	}

	if !strings.Contains(result.Output, "/bin/bash") {
		t.Errorf("expected shell in output")
	}

	if !strings.Contains(result.Output, "Go, Python") {
		t.Errorf("expected languages list in output")
	}
}

func TestGetContextTool_Schema(t *testing.T) {
	t.Parallel()

	tool := NewGetContextTool(nil)
	schema := tool.Schema()

	if schema.Function.Name != "get_context" {
		t.Errorf("expected name 'get_context', got: %s", schema.Function.Name)
	}

	if schema.Function.Description == "" {
		t.Errorf("expected non-empty description")
	}

	// Tool should have no required parameters.
	if len(schema.Function.Parameters.Required) != 0 {
		t.Errorf("expected no required parameters, got: %d", len(schema.Function.Parameters.Required))
	}
}

func TestGetContextTool_ErrorCases(t *testing.T) {
	t.Parallel()

	tool := NewGetContextTool(nil)

	// GetContextTool has only optional parameters, so it doesn't fail on invalid params
	// Testing that it handles defaults properly.
	params, _ := FromMap(map[string]any{"query": 123})

	_, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// get_context tool might succeed or fail depending on context availability,
	// so we just check it doesn't panic.
}
