package sandbox

import (
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

// TestSandbox_Wrap tests sandbox creation (not wrapping, which would restrict the test process)
func TestSandbox_Wrap(t *testing.T) {
	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)

	// Verify sandbox was created
	assert.NotNil(t, sb)
	assert.Equal(t, ModeWorkspaceWrite, sb.Mode())

	// NOTE: We cannot actually call Wrap() in tests because it would restrict
	// the test process itself. Real-world usage calls Wrap() just before cmd.Run()
	// in a context where restricting the current process is acceptable.
}

// TestSandbox_ReadOnly tests read-only mode sandbox creation
func TestSandbox_ReadOnly(t *testing.T) {
	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)
	assert.NotNil(t, sb)
	assert.Equal(t, ModeReadOnly, sb.Mode())
}

// TestSandbox_ModeWorkspaceWrite tests workspace write mode sandbox creation
func TestSandbox_ModeWorkspaceWrite(t *testing.T) {
	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)
	assert.NotNil(t, sb)
	assert.Equal(t, ModeWorkspaceWrite, sb.Mode())
}

// TestNoopSandbox tests the no-op sandbox fallback
func TestNoopSandbox(t *testing.T) {
	sb := &NoopSandbox{}

	assert.False(t, sb.Supported())
	assert.Equal(t, ModeFullAccess, sb.Mode())

	// Verify Wrap succeeds (does nothing)
	err := sb.Wrap(nil, SandboxOptions{})
	assert.NoError(t, err)
}
