package appserver

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchFiles(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	files := []string{
		"main.go",
		"handler.go",
		"server.go",
		"test_file.txt",
		"src/utils.go",
		"src/helper.go",
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	tests := []struct {
		name     string
		query    string
		limit    int
		minCount int
	}{
		{
			name:     "Exact match",
			query:    "main.go",
			limit:    10,
			minCount: 1,
		},
		{
			name:     "Partial match",
			query:    "go",
			limit:    10,
			minCount: 4,
		},
		{
			name:     "Substring match",
			query:    "handler",
			limit:    10,
			minCount: 1,
		},
		{
			name:     "With limit",
			query:    "go",
			limit:    2,
			minCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := SearchFiles(tmpDir, tt.query, tt.limit)
			if err != nil {
				t.Fatalf("SearchFiles failed: %v", err)
			}

			if len(matches) < tt.minCount {
				t.Errorf("Expected at least %d matches, got %d", tt.minCount, len(matches))
			}

			if len(matches) > tt.limit && tt.limit > 0 {
				t.Errorf("Expected at most %d matches, got %d", tt.limit, len(matches))
			}

			// Verify all matches have scores
			for _, match := range matches {
				if match.Score <= 0 {
					t.Errorf("Match %s has invalid score: %f", match.Path, match.Score)
				}
			}
		})
	}
}

func TestSearchFiles_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that won't match
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)

	matches, err := SearchFiles(tmpDir, "nonexistent", 10)
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}

	if len(matches) > 0 {
		t.Errorf("Expected no matches, got %d", len(matches))
	}
}

func TestSearchFiles_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden file
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("content"), 0644)

	// Create hidden directory with file
	os.MkdirAll(filepath.Join(tmpDir, ".hidden_dir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".hidden_dir", "file.txt"), []byte("content"), 0644)

	// Create visible file
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("content"), 0644)

	matches, err := SearchFiles(tmpDir, "txt", 10)
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}

	// Should only match visible.txt
	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}

	if len(matches) > 0 && matches[0].Path != "visible.txt" {
		t.Errorf("Expected visible.txt, got %s", matches[0].Path)
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		target   string
		query    string
		expected bool
	}{
		{"hello.go", "hello", true},
		{"hello.go", "go", true},
		{"hello.go", "hlo", true}, // subsequence
		{"hello.go", "xyz", false},
		{"main.go", "main.go", true},
		{"handler.go", "hdl", true}, // subsequence
	}

	for _, tt := range tests {
		t.Run(tt.target+":"+tt.query, func(t *testing.T) {
			result := fuzzyMatch(tt.target, tt.query)
			if result != tt.expected {
				t.Errorf("fuzzyMatch(%s, %s) = %v, expected %v",
					tt.target, tt.query, result, tt.expected)
			}
		})
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		query    string
		minScore float64
	}{
		{
			name:     "Exact match",
			target:   "test.go",
			query:    "test.go",
			minScore: 1.0,
		},
		{
			name:     "Contains at start",
			target:   "testfile.go",
			query:    "test",
			minScore: 0.9,
		},
		{
			name:     "Contains in middle",
			target:   "my_test_file.go",
			query:    "test",
			minScore: 0.8,
		},
		{
			name:     "Subsequence",
			target:   "handler.go",
			query:    "hdl",
			minScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateScore(tt.target, tt.query)
			if score < tt.minScore {
				t.Errorf("Expected score >= %f, got %f", tt.minScore, score)
			}
		})
	}
}

func TestSearchFiles_DefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many files (more than default limit)
	for i := 0; i < 100; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.go", i))
		os.WriteFile(filename, []byte("content"), 0644)
	}

	// Search with limit 0 (should use default 50)
	matches, err := SearchFiles(tmpDir, "go", 0)
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}

	// Should be limited to 50 even though there are 100 files
	if len(matches) != 50 {
		t.Errorf("Expected 50 matches (default limit), got %d", len(matches))
	}
}
