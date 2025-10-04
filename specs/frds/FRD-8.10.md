# FRD-8.10: Windows Sandbox Implementation

**Feature**: Windows Sandbox Support for Command Isolation
**Status**: Pending Implementation
**Priority**: Low (Platform-specific)
**Created**: 2025-10-04
**Related**: [specs/security-modules.md](../security-modules.md#windows-stub)

---

## Overview

Implement Windows-specific sandbox support for the Spin agent framework to provide filesystem and process isolation when executing commands on Windows platforms. This complements the existing Linux Landlock and macOS Seatbelt implementations.

## Background

Currently, the Windows sandbox implementation is a stub that returns a `NoopSandbox`, providing no isolation:

```go
//go:build windows

package sandbox

// NewSandbox creates a Windows sandbox (currently not implemented).
// Returns a NoopSandbox as Windows sandboxing support is planned for future releases.
func NewSandbox(mode Mode) (Sandbox, error) {
	return &NoopSandbox{}, nil
}
```

This FRD defines the implementation of Windows-specific sandboxing using available Windows security features.

## Goals

1. Implement filesystem isolation for Windows processes
2. Provide read-only and workspace-write modes
3. Maintain API compatibility with existing Sandbox interface
4. Use native Windows security features (Job Objects, Low-Integrity processes)
5. Gracefully degrade on older Windows versions

## Non-Goals

- Network isolation (future enhancement)
- Full AppContainer implementation (complex, future enhancement)
- Windows Defender Application Guard integration

## Windows Security Mechanisms

### Available Options

1. **Job Objects** - Process grouping and resource limits
2. **Integrity Levels** - Low-integrity processes with restricted access
3. **Restricted Tokens** - Token-based access control
4. **AppContainer** (Windows 8+) - Full application sandboxing (complex)

### Recommended Approach: Job Objects + Integrity Levels

For this implementation, we'll use:
- **Job Objects**: To control and limit child process resources
- **Low Integrity Level**: To restrict filesystem write access
- **Restricted Token**: To drop unnecessary privileges

## API Design

### Windows Sandbox Structure

```go
//go:build windows

package sandbox

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WindowsSandbox implements Sandbox using Windows Job Objects and Integrity Levels
type WindowsSandbox struct {
	mode Mode
}

// NewSandbox creates a Windows sandbox
func NewSandbox(mode Mode) (Sandbox, error) {
	// Check Windows version
	if !isWindowsSupported() {
		return &NoopSandbox{}, nil
	}

	return &WindowsSandbox{mode: mode}, nil
}

// Wrap applies Windows sandbox restrictions to command
func (s *WindowsSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
	// Implementation details below
}

// Supported returns true if Windows sandboxing is available
func (s *WindowsSandbox) Supported() bool {
	return isWindowsSupported()
}

// Mode returns the sandbox mode
func (s *WindowsSandbox) Mode() Mode {
	return s.mode
}
```

### Implementation Strategy

#### Mode: ModeReadOnly

For read-only mode:
1. Create process with **Low Integrity Level** (prevents writes to medium+ integrity locations)
2. Use **Restricted Token** to drop write permissions
3. Create **Job Object** with UI restrictions

#### Mode: ModeWorkspaceWrite

For workspace-write mode:
1. Create process with **Low Integrity Level**
2. Mark workspace directory as **Low Integrity** (allows writes)
3. Use **Job Object** for resource limits

#### Mode: ModeFullAccess

For full access mode:
- No restrictions applied
- Normal process creation

### Detailed Implementation

```go
// Wrap applies Windows sandbox restrictions to command
func (s *WindowsSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
	if opts.Mode == ModeFullAccess {
		return nil // No restrictions
	}

	// Prepare SysProcAttr for process creation
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Create job object
	jobHandle, err := s.createJobObject()
	if err != nil {
		return fmt.Errorf("create job object: %w", err)
	}

	// Set up process creation flags
	cmd.SysProcAttr.CreationFlags = windows.CREATE_SUSPENDED |
		windows.CREATE_BREAKAWAY_FROM_JOB

	// Store job handle for post-creation setup
	// Note: We'll use a goroutine to assign process to job after creation

	return nil
}

// createJobObject creates and configures a Windows Job Object
func (s *WindowsSandbox) createJobObject() (windows.Handle, error) {
	// Create job object
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("CreateJobObject: %w", err)
	}

	// Configure job limits
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags =
		windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE |
		windows.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION

	// Apply limits
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)),
		uint32(unsafe.Sizeof(limits)),
	)
	if err != nil {
		windows.CloseHandle(job)
		return 0, fmt.Errorf("SetInformationJobObject: %w", err)
	}

	return job, nil
}

// setLowIntegrity sets the process to run with low integrity level
func setLowIntegrity(token windows.Token) error {
	// Get low integrity SID
	var sid *windows.SID
	err := windows.ConvertStringSidToSid(
		windows.StringToUTF16Ptr("S-1-16-4096"), // Low Integrity
		&sid,
	)
	if err != nil {
		return fmt.Errorf("ConvertStringSidToSid: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(sid)))

	// Create TOKEN_MANDATORY_LABEL
	tml := windows.TOKEN_MANDATORY_LABEL{
		Label: windows.SID_AND_ATTRIBUTES{
			Sid:        sid,
			Attributes: windows.SE_GROUP_INTEGRITY,
		},
	}

	// Set token integrity level
	err = windows.SetTokenInformation(
		token,
		windows.TokenIntegrityLevel,
		(*byte)(unsafe.Pointer(&tml)),
		uint32(unsafe.Sizeof(tml)+uintptr(windows.GetLengthSid(sid))),
	)
	if err != nil {
		return fmt.Errorf("SetTokenInformation: %w", err)
	}

	return nil
}

// markDirectoryLowIntegrity marks a directory as accessible to low-integrity processes
func markDirectoryLowIntegrity(path string) error {
	// This allows low-integrity processes to write to the directory
	// Implementation uses icacls command or direct SACL modification

	// Using icacls: icacls "path" /setintegritylevel Low
	cmd := exec.Command("icacls", path, "/setintegritylevel", "Low", "/t", "/c", "/q")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("icacls: %w", err)
	}

	return nil
}

// isWindowsSupported checks if Windows version supports required features
func isWindowsSupported() bool {
	// Job Objects available since Windows 2000
	// Integrity Levels available since Windows Vista
	// Check for Windows Vista or later (version 6.0+)

	ver := windows.RtlGetVersion()
	return ver.MajorVersion >= 6
}
```

## Implementation Plan

### Phase 1: Job Object Support
1. Implement `createJobObject()` function
2. Configure job limits (kill on close, exception handling)
3. Test process creation with job assignment

### Phase 2: Integrity Level Support
1. Implement `setLowIntegrity()` function
2. Create process with low integrity token
3. Test read restrictions

### Phase 3: Workspace Write Support
1. Implement `markDirectoryLowIntegrity()` function
2. Allow writes to workspace directory
3. Test workspace write permissions

### Phase 4: Testing
1. Create unit tests (mock-based for cross-platform development)
2. Create integration tests (Windows-only)
3. Test on various Windows versions (Windows 10, 11, Server)

## Testing Strategy

### Unit Tests (Cross-Platform)

```go
//go:build windows

func TestWindowsSandbox_NewSandbox(t *testing.T) {
	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)
	assert.NotNil(t, sb)

	ws, ok := sb.(*WindowsSandbox)
	require.True(t, ok)
	assert.Equal(t, ModeReadOnly, ws.Mode())
}

func TestWindowsSandbox_Supported(t *testing.T) {
	sb, _ := NewSandbox(ModeReadOnly)

	// Should be true on Windows Vista+
	if isWindowsSupported() {
		assert.True(t, sb.Supported())
	}
}
```

### Integration Tests (Windows-Only)

```go
//go:build windows

func TestWindowsSandbox_ReadRestriction(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	sb, err := NewSandbox(ModeReadOnly)
	require.NoError(t, err)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create test file outside workspace
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Try to write (should fail with low integrity)
	cmd := exec.Command("cmd", "/c", fmt.Sprintf("echo test > %s\\blocked.txt", tempDir))
	opts := GetDefaultOptions(ModeReadOnly, "C:\\workspace")
	require.NoError(t, sb.Wrap(cmd, opts))

	err = cmd.Run()
	assert.Error(t, err, "should not be able to write in read-only mode")
}

func TestWindowsSandbox_WorkspaceWrite(t *testing.T) {
	if !isWindowsSupported() {
		t.Skip("Windows Vista+ required")
	}

	sb, err := NewSandbox(ModeWorkspaceWrite)
	require.NoError(t, err)

	workDir := t.TempDir()

	// Mark workspace as low integrity
	require.NoError(t, markDirectoryLowIntegrity(workDir))

	// Try to write inside workspace (should succeed)
	testFile := filepath.Join(workDir, "test.txt")
	cmd := exec.Command("cmd", "/c", fmt.Sprintf("echo test > %s", testFile))
	opts := GetDefaultOptions(ModeWorkspaceWrite, workDir)
	require.NoError(t, sb.Wrap(cmd, opts))

	err = cmd.Run()
	assert.NoError(t, err, "should be able to write in workspace")

	// Verify file created
	assert.FileExists(t, testFile)
}
```

## Security Considerations

### Integrity Levels

Windows integrity levels:
- **Untrusted** (S-1-16-0): Browser sandboxes
- **Low** (S-1-16-4096): Protected Mode IE, our sandbox
- **Medium** (S-1-16-8192): Normal user processes (default)
- **High** (S-1-16-12288): Administrator processes
- **System** (S-1-16-16384): System services

Low-integrity processes **cannot**:
- Write to Medium+ integrity locations (most of filesystem)
- Write to registry keys with Medium+ integrity
- Send window messages to Medium+ processes

Low-integrity processes **can**:
- Read most filesystem locations
- Write to directories marked as Low integrity
- Execute within Job Object constraints

### Limitations

1. **Less Restrictive than Landlock**: Cannot block read access to arbitrary paths
2. **Workspace Directory Must Be Marked**: Requires `icacls` to mark workspace as Low integrity
3. **Requires Windows Vista+**: Not available on Windows XP/2000
4. **No Network Isolation**: Requires additional mechanisms (future work)

### Fallback Behavior

On unsupported Windows versions (< Vista):
- `NewSandbox()` returns `NoopSandbox`
- No isolation applied
- Agent continues to function (degraded security)

## Success Criteria

1. ✅ Windows sandbox created successfully on Windows Vista+
2. ✅ Read-only mode prevents writes outside workspace
3. ✅ Workspace-write mode allows writes to workspace directory
4. ✅ Full-access mode applies no restrictions
5. ✅ Job objects properly limit process resources
6. ✅ Tests pass on Windows 10/11
7. ✅ Graceful fallback on unsupported versions
8. ✅ API compatibility with Linux/macOS implementations

## Implementation Checklist

- [ ] Create `sandbox_windows.go` with WindowsSandbox struct
- [ ] Implement `NewSandbox()` for Windows
- [ ] Implement `Wrap()` with Job Object creation
- [ ] Implement `createJobObject()` helper
- [ ] Implement `setLowIntegrity()` helper
- [ ] Implement `markDirectoryLowIntegrity()` helper
- [ ] Implement `isWindowsSupported()` version check
- [ ] Create unit tests in `sandbox_windows_test.go`
- [ ] Create integration tests for read/write modes
- [ ] Test on Windows 10
- [ ] Test on Windows 11
- [ ] Test on Windows Server
- [ ] Update documentation
- [ ] Update `missing.md` with completion status

## References

- [Windows Job Objects](https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects)
- [Mandatory Integrity Control](https://docs.microsoft.com/en-us/windows/win32/secauthz/mandatory-integrity-control)
- [golang.org/x/sys/windows](https://pkg.go.dev/golang.org/x/sys/windows)
- [Security Modules Spec](../security-modules.md)
- [Linux Landlock Implementation](../../internal/security/sandbox/sandbox_linux.go)

---

**Next Steps**: Implement `WindowsSandbox.Wrap()` following the design above, focusing on Job Objects and Integrity Levels for filesystem isolation.
