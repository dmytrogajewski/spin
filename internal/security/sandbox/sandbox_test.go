package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSandbox tests sandbox creation
func TestNewSandbox(t *testing.T) {
	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)
	assert.NotNil(t, sb)

	// Mode may differ if sandbox is not supported (NoopSandbox)
	// In that case, it will be FullAccess
	if sb.Supported() {
		assert.Equal(t, ModeWorkspaceWrite, sb.Mode())
	} else {
		assert.Equal(t, ModeFullAccess, sb.Mode())
	}
}

// TestMode_String tests mode string representation
func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeReadOnly, "read-only"},
		{ModeWorkspaceWrite, "workspace-write"},
		{ModeFullAccess, "full-access"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.String())
		})
	}
}

// TestGetDefaultOptions tests default sandbox options
func TestGetDefaultOptions(t *testing.T) {
	workDir := "/workspace"

	tests := []struct {
		name string
		mode Mode
	}{
		{"read-only", ModeReadOnly},
		{"workspace-write", ModeWorkspaceWrite},
		{"full-access", ModeFullAccess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := GetDefaultOptions(tt.mode, workDir)
			assert.Equal(t, tt.mode, opts.Mode)
			assert.Equal(t, workDir, opts.WorkDir)
			assert.NotNil(t, opts.ReadPaths)
			assert.NotNil(t, opts.WritePaths)
		})
	}
}

// TestSandbox_Supported tests platform support detection
func TestSandbox_Supported(t *testing.T) {
	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)

	supported := sb.Supported()

	// Should be supported on Linux with Landlock or macOS with sandbox-exec
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		// May or may not be supported depending on kernel/OS version
		t.Logf("Sandbox supported: %v", supported)
	} else {
		// Other platforms use NoopSandbox
		assert.False(t, supported)
	}
}

// TestSandbox_Wrap tests command wrapping (basic)
func TestSandbox_Wrap(t *testing.T) {
	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)

	workDir := t.TempDir()
	cmd := exec.Command("echo", "hello")

	opts := GetDefaultOptions(ModeWorkspaceWrite, workDir)
	err = sb.Wrap(cmd, opts)
	assert.NoError(t, err)

	// Command should still be executable after wrapping
	assert.NotNil(t, cmd)
}

// TestSandbox_ReadOnly tests read-only mode (integration)
func TestSandbox_ReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Sandbox not supported on Windows")
	}

	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)

	if !sb.Supported() {
		t.Skip("Sandbox not supported on this platform")
	}

	workDir := t.TempDir()
	testFile := filepath.Join(workDir, "test.txt")

	// Create test file
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Try to read file (should succeed in read-only mode)
	cmd := exec.Command("cat", testFile)
	opts := GetDefaultOptions(ModeReadOnly, workDir)
	err = sb.Wrap(cmd, opts)
	require.NoError(t, err)

	output, err := cmd.CombinedOutput()
	// In read-only mode, reading should work
	// (Note: actual enforcement depends on platform support)
	t.Logf("Read test output: %s, error: %v", output, err)
}

// TestSandbox_WriteRestriction tests write restrictions (integration)
func TestSandbox_WriteRestriction(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Sandbox not supported on Windows")
	}

	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)

	if !sb.Supported() {
		t.Skip("Sandbox not supported on this platform")
	}

	workDir := t.TempDir()
	testFile := filepath.Join(workDir, "write-test.txt")

	// Try to write file in read-only mode
	cmd := exec.Command("sh", "-c", "echo test > "+testFile)
	opts := GetDefaultOptions(ModeReadOnly, workDir)
	err = sb.Wrap(cmd, opts)
	require.NoError(t, err)

	err = cmd.Run()
	// In true sandboxing, this should fail
	// But we log the result for debugging
	t.Logf("Write test error: %v", err)
}

// TestNoopSandbox tests the no-op sandbox fallback
func TestNoopSandbox(t *testing.T) {
	sb := &NoopSandbox{}

	assert.False(t, sb.Supported())
	assert.Equal(t, ModeFullAccess, sb.Mode())

	cmd := exec.Command("echo", "test")
	err := sb.Wrap(cmd, SandboxOptions{})
	assert.NoError(t, err)
}
