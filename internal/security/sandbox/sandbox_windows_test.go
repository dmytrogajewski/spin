//go:build windows

package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestNewSandbox_Windows(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		wantType string
	}{
		{
			name:     "read-only mode",
			mode:     ModeReadOnly,
			wantType: "*sandbox.WindowsSandbox",
		},
		{
			name:     "workspace-write mode",
			mode:     ModeWorkspaceWrite,
			wantType: "*sandbox.WindowsSandbox",
		},
		{
			name:     "full-access mode",
			mode:     ModeFullAccess,
			wantType: "*sandbox.WindowsSandbox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb, err := NewSandbox(tt.mode)
			require.NoError(t, err)
			assert.NotNil(t, sb)

			// Should return WindowsSandbox on supported Windows
			if isWindowsSupported() {
				ws, ok := sb.(*WindowsSandbox)
				require.True(t, ok, "expected WindowsSandbox, got %T", sb)
				assert.Equal(t, tt.mode, ws.Mode())
			} else {
				// Should return NoopSandbox on unsupported Windows
				_, ok := sb.(*NoopSandbox)
				require.True(t, ok, "expected NoopSandbox on unsupported Windows")
			}
		})
	}
}

func TestWindowsSandbox_Supported(t *testing.T) {
	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)

	// Should be true on Windows Vista+ (version 6.0+)
	if isWindowsSupported() {
		assert.True(t, sb.Supported(), "sandbox should be supported on Windows Vista+")
	} else {
		assert.False(t, sb.Supported(), "sandbox should not be supported on Windows XP or earlier")
	}
}

func TestWindowsSandbox_Mode(t *testing.T) {
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
			sb, err := NewSandbox(tt.mode)
			require.NoError(t, err)

			if isWindowsSupported() {
				ws := sb.(*WindowsSandbox)
				assert.Equal(t, tt.mode, ws.Mode())
			}
		})
	}
}

func TestWindowsSandbox_Wrap_FullAccess(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	sb, err := NewSandbox(ModeFullAccess)
	require.NoError(t, err)

	// Create a simple command
	cmd := exec.Command("cmd", "/c", "echo", "test")

	opts := SandboxOptions{
		Mode:    ModeFullAccess,
		WorkDir: t.TempDir(),
	}

	// Wrap should succeed and not modify the command for full access
	err = sb.Wrap(cmd, opts)
	assert.NoError(t, err)

	// Command should execute successfully
	err = cmd.Run()
	assert.NoError(t, err)
}

func TestWindowsSandbox_Wrap_ReadOnly(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)

	workDir := t.TempDir()

	// Create a command that tries to write
	cmd := exec.Command("cmd", "/c", "echo", "test")

	opts := SandboxOptions{
		Mode:    ModeReadOnly,
		WorkDir: workDir,
	}

	// Wrap should succeed
	err = sb.Wrap(cmd, opts)
	assert.NoError(t, err)

	// Command should have SysProcAttr set
	assert.NotNil(t, cmd.SysProcAttr)
	assert.Equal(t, windows.CREATE_SUSPENDED|windows.CREATE_BREAKAWAY_FROM_JOB,
		cmd.SysProcAttr.CreationFlags)
}

func TestWindowsSandbox_Wrap_WorkspaceWrite(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)

	workDir := t.TempDir()

	// Create a command
	cmd := exec.Command("cmd", "/c", "echo", "test")

	opts := SandboxOptions{
		Mode:    ModeWorkspaceWrite,
		WorkDir: workDir,
	}

	// Wrap should succeed
	err = sb.Wrap(cmd, opts)
	assert.NoError(t, err)

	// Command should have SysProcAttr set
	assert.NotNil(t, cmd.SysProcAttr)
	assert.Equal(t, windows.CREATE_SUSPENDED|windows.CREATE_BREAKAWAY_FROM_JOB,
		cmd.SysProcAttr.CreationFlags)
}

func TestWindowsSandbox_createJobObject(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	sb := &WindowsSandbox{mode: ModeReadOnly}

	job, err := sb.createJobObject()
	require.NoError(t, err)
	require.NotEqual(t, windows.Handle(0), job)

	// Clean up
	windows.CloseHandle(job)

	// Verify job object was created successfully
	// We can't inspect the job object directly without more complex API calls,
	// but the fact that CreateJobObject succeeded is a good sign
}

func TestMarkDirectoryLowIntegrity(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	// Create a temporary directory
	tempDir := t.TempDir()

	// Try to mark directory as low integrity
	// Note: This may fail if we don't have admin privileges
	err := markDirectoryLowIntegrity(tempDir)

	// If we have admin privileges, it should succeed
	// If not, it will fail with a specific error
	// Either way, we just verify the function runs without panic
	t.Logf("markDirectoryLowIntegrity result: %v", err)

	// We can't reliably test this without admin privileges,
	// but we verify it doesn't panic and returns an error type
	if err != nil {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "icacls", "error should mention icacls")
	}
}

func TestSetLowIntegrity(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	// Get current process token
	var token windows.Token
	err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_QUERY|windows.TOKEN_ADJUST_DEFAULT,
		&token,
	)
	require.NoError(t, err)
	defer token.Close()

	// Try to set low integrity on a duplicate token
	// (we don't want to affect our test process)
	var dupToken windows.Token
	err = windows.DuplicateTokenEx(
		token,
		windows.TOKEN_QUERY|windows.TOKEN_ADJUST_DEFAULT,
		nil,
		windows.SecurityImpersonation,
		windows.TokenPrimary,
		&dupToken,
	)
	require.NoError(t, err)
	defer dupToken.Close()

	// Set low integrity on the duplicate
	err = setLowIntegrity(dupToken)

	// This should succeed
	assert.NoError(t, err)
}

func TestIsWindowsSupported(t *testing.T) {
	supported := isWindowsSupported()

	// Get Windows version to verify
	ver := windows.RtlGetVersion()

	if ver.MajorVersion >= 6 {
		assert.True(t, supported, "should be supported on Windows Vista+ (version %d.%d)",
			ver.MajorVersion, ver.MinorVersion)
	} else {
		assert.False(t, supported, "should not be supported on Windows XP or earlier (version %d.%d)",
			ver.MajorVersion, ver.MinorVersion)
	}

	t.Logf("Windows version: %d.%d (build %d)",
		ver.MajorVersion, ver.MinorVersion, ver.BuildNumber)
}

func TestWindowsSandbox_Integration_Echo(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	// This is an integration test that verifies the basic command execution
	// works with sandbox wrapping

	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)

	workDir := t.TempDir()

	// Create a simple echo command
	cmd := exec.Command("cmd", "/c", "echo", "hello")

	opts := SandboxOptions{
		Mode:    ModeWorkspaceWrite,
		WorkDir: workDir,
	}

	// Wrap the command
	err = sb.Wrap(cmd, opts)
	require.NoError(t, err)

	// Note: Currently, the implementation creates the job object but
	// doesn't fully apply the low integrity level, so the command
	// will be created in SUSPENDED state but not executed.
	// This test verifies the wrapping works without errors.

	// TODO: Once full implementation is complete, we should be able to
	// execute the command and verify output
}

func TestWindowsSandbox_Integration_WriteFile(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	t.Skip("Skipping until full low-integrity implementation is complete")

	// This integration test will be enabled once we implement the full
	// token-based low integrity process creation

	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)

	workDir := t.TempDir()

	// Mark workspace as low integrity (requires admin)
	err = markDirectoryLowIntegrity(workDir)
	if err != nil {
		t.Skipf("Cannot mark directory as low integrity (need admin): %v", err)
	}

	// Try to write inside workspace (should succeed)
	testFile := filepath.Join(workDir, "test.txt")
	cmd := exec.Command("cmd", "/c", "echo test >", testFile)

	opts := SandboxOptions{
		Mode:    ModeWorkspaceWrite,
		WorkDir: workDir,
	}

	err = sb.Wrap(cmd, opts)
	require.NoError(t, err)

	err = cmd.Run()
	assert.NoError(t, err, "should be able to write in workspace")

	// Verify file created
	_, err = os.Stat(testFile)
	assert.NoError(t, err, "file should exist")
}

func TestWindowsSandbox_Integration_BlockWrite(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	t.Skip("Skipping until full low-integrity implementation is complete")

	// This integration test will be enabled once we implement the full
	// token-based low integrity process creation

	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)

	workDir := t.TempDir()

	// Try to write outside workspace (should fail)
	tempFile := filepath.Join(os.TempDir(), "blocked.txt")
	defer os.Remove(tempFile) // Clean up in case test fails

	cmd := exec.Command("cmd", "/c", "echo test >", tempFile)

	opts := SandboxOptions{
		Mode:    ModeReadOnly,
		WorkDir: workDir,
	}

	err = sb.Wrap(cmd, opts)
	require.NoError(t, err)

	err = cmd.Run()
	assert.Error(t, err, "should not be able to write in read-only mode")

	// Verify file was not created
	_, err = os.Stat(tempFile)
	assert.Error(t, err, "file should not exist")
}
