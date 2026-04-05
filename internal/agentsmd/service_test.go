package agentsmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	svc := NewService(cfg, "/tmp", "")

	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	if svc.config != cfg {
		t.Error("NewService() config not set correctly")
	}

	if svc.workDir != "/tmp" {
		t.Errorf("NewService() workDir = %v, want /tmp", svc.workDir)
	}
}

func TestNewService_NilConfig(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, "/tmp", "")

	if svc == nil {
		t.Fatal("NewService() returned nil with nil config")
	}

	if svc.config == nil {
		t.Error("NewService() should use default config when nil is passed")
	}
}

func TestService_Load_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	agentsPath := filepath.Join(tempDir, FileName)

	content := "# Test Project Instructions\n\nThis is a test."

	err := os.WriteFile(agentsPath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &Config{
		Enabled: true,
		MaxSize: 100 * 1024,
	}
	svc := NewService(cfg, tempDir, "")

	ctx := context.Background()

	err = svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !svc.IsLoaded() {
		t.Error("IsLoaded() = false after successful Load()")
	}

	if svc.Content() != content {
		t.Errorf("Content() = %v, want %v", svc.Content(), content)
	}

	if svc.Path() != agentsPath {
		t.Errorf("Path() = %v, want %v", svc.Path(), agentsPath)
	}
}

func TestService_Load_Disabled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	agentsPath := filepath.Join(tempDir, FileName)

	err := os.WriteFile(agentsPath, []byte("content"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &Config{
		Enabled: false,
	}
	svc := NewService(cfg, tempDir, "")

	ctx := context.Background()

	err = svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil when disabled", err)
	}

	if svc.IsLoaded() {
		t.Error("IsLoaded() = true when service is disabled")
	}

	if svc.Content() != "" {
		t.Error("Content() should be empty when disabled")
	}
}

func TestService_Load_CustomPath(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	customPath := filepath.Join(tempDir, "custom-agents.md")

	content := "# Custom Instructions"

	err := os.WriteFile(customPath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &Config{
		Enabled: true,
		Path:    customPath,
		MaxSize: 100 * 1024,
	}
	svc := NewService(cfg, "/nonexistent", "") // workDir doesn't matter with custom path.

	ctx := context.Background()

	err = svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if svc.Path() != customPath {
		t.Errorf("Path() = %v, want %v", svc.Path(), customPath)
	}

	if svc.Content() != content {
		t.Errorf("Content() = %v, want %v", svc.Content(), content)
	}
}

func TestService_Load_CustomPath_NotFound(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enabled: true,
		Path:    "/nonexistent/path/AGENTS.md",
	}
	svc := NewService(cfg, "/tmp", "")

	ctx := context.Background()

	err := svc.Load(ctx)
	if err == nil {
		t.Error("Load() error = nil, want error for missing custom path")
	}
}

func TestService_Load_NotFound(t *testing.T) {
	t.Parallel()

	// Create an isolated directory structure to prevent finding
	// real AGENTS.md files when walking up the directory tree.
	tempDir := t.TempDir()

	isolatedDir := filepath.Join(tempDir, "isolated", "deep", "path")

	err := os.MkdirAll(isolatedDir, 0o750)
	if err != nil {
		t.Fatalf("failed to create isolated dir: %v", err)
	}

	cfg := &Config{
		Enabled: true,
		MaxSize: 100 * 1024,
	}
	// Pass tempDir as gitRoot to limit the walk-up scope.
	svc := NewService(cfg, isolatedDir, tempDir)

	ctx := context.Background()

	err = svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil when file not found", err)
	}

	if svc.IsLoaded() {
		t.Error("IsLoaded() = true when file was not found")
	}
}

func TestService_Load_SizeLimit(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	agentsPath := filepath.Join(tempDir, FileName)

	// Create content larger than limit.
	largeContent := strings.Repeat("A", 1000)

	err := os.WriteFile(agentsPath, []byte(largeContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &Config{
		Enabled: true,
		MaxSize: 100, // Only 100 bytes.
	}
	svc := NewService(cfg, tempDir, "")

	ctx := context.Background()

	err = svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !svc.IsLoaded() {
		t.Error("IsLoaded() = false after Load()")
	}

	// Content should be truncated.
	content := svc.Content()
	if len(content) > 150 { // 100 bytes + truncation notice.
		t.Errorf("Content length = %d, expected truncated content", len(content))
	}

	if !strings.Contains(content, "[Content truncated") {
		t.Error("Truncated content should contain truncation notice")
	}
}

func TestService_Refresh(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	agentsPath := filepath.Join(tempDir, FileName)

	// Initial content.
	err := os.WriteFile(agentsPath, []byte("# Version 1"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &Config{
		Enabled: true,
		MaxSize: 100 * 1024,
	}
	svc := NewService(cfg, tempDir, "")

	ctx := context.Background()

	err = svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if svc.Content() != "# Version 1" {
		t.Errorf("Initial Content() = %v, want '# Version 1'", svc.Content())
	}

	// Update file.
	err = os.WriteFile(agentsPath, []byte("# Version 2"), 0o600)
	if err != nil {
		t.Fatalf("failed to update test file: %v", err)
	}

	// Refresh should reload.
	err = svc.Refresh(ctx)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if svc.Content() != "# Version 2" {
		t.Errorf("After Refresh() Content() = %v, want '# Version 2'", svc.Content())
	}
}

func TestService_ThreadSafety(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	agentsPath := filepath.Join(tempDir, FileName)

	err := os.WriteFile(agentsPath, []byte("# Concurrent Test"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &Config{
		Enabled: true,
		MaxSize: 100 * 1024,
	}
	svc := NewService(cfg, tempDir, "")

	ctx := context.Background()

	err = svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Concurrent reads should be safe.
	done := make(chan bool)

	for range 10 {
		go func() {
			_ = svc.Content()
			_ = svc.IsLoaded()
			_ = svc.Path()

			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}
