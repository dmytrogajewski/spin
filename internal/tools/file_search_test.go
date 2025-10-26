package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSearchTool_BasicSearch(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{
		"test.txt",
		"test_helper.go",
		"main.go",
		"config.toml",
		"testdata/file.txt",
	}

	for _, file := range files {
		dir := filepath.Dir(filepath.Join(tmpDir, file))
		if dir != tmpDir {
			os.MkdirAll(dir, 0755)
		}
		os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644)
	}

	tool := NewFileSearchTool(tmpDir)

	params, _ := FromMap(map[string]interface{}{
		"query": "test",
		"limit": 5,
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Should find files with "test" in the name
	if !strings.Contains(result.Output, "test") {
		t.Errorf("expected output to contain 'test', got: %s", result.Output)
	}
}

func TestFileSearchTool_MissingQuery(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileSearchTool(tmpDir)

	params, _ := FromMap(map[string]interface{}{})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Errorf("expected failure for missing query")
	}

	if !strings.Contains(result.Error, "query parameter must be a non-empty string") {
		t.Errorf("expected error about missing query, got: %s", result.Error)
	}
}

func TestFileSearchTool_NoResults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one file that won't match
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)

	tool := NewFileSearchTool(tmpDir)

	params, _ := FromMap(map[string]interface{}{
		"query": "nonexistent_unique_string_xyz",
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "No files found") {
		t.Errorf("expected 'No files found' message, got: %s", result.Output)
	}
}

func TestFileSearchTool_CustomWorkspace(t *testing.T) {
	workspace1 := t.TempDir()
	workspace2 := t.TempDir()

	// Create files in workspace2
	os.WriteFile(filepath.Join(workspace2, "search_me.txt"), []byte("content"), 0644)

	// Tool with workspace1 as default
	tool := NewFileSearchTool(workspace1)

	// Search in workspace2
	params, _ := FromMap(map[string]interface{}{
		"query":          "search",
		"workspace_root": workspace2,
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "search_me.txt") {
		t.Errorf("expected to find file from workspace2, got: %s", result.Output)
	}
}

func TestFileSearchTool_Schema(t *testing.T) {
	tool := NewFileSearchTool("/tmp")
	schema := tool.Schema()

	if schema.Function.Name != "file_search" {
		t.Errorf("expected name 'file_search', got: %s", schema.Function.Name)
	}

	// Verify required parameter
	required := schema.Function.Parameters.Required
	if len(required) != 1 || required[0] != "query" {
		t.Errorf("expected required parameter 'query', got: %v", required)
	}
}

func TestFileSearchTool_ErrorCases(t *testing.T) {
	tool := NewFileSearchTool("/tmp/test")

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name:   "missing query",
			params: map[string]interface{}{},
		},
		{
			name:   "invalid query type",
			params: map[string]interface{}{"query": 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := FromMap(tt.params)
			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result.Success {
				t.Error("expected failure result")
			}
		})
	}
}
