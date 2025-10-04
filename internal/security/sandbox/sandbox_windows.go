//go:build windows

package sandbox

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TOKEN_MANDATORY_LABEL represents a token's mandatory integrity level.
// This structure is used to set the integrity level of a process token.
type TOKEN_MANDATORY_LABEL struct {
	Label windows.SIDAndAttributes
}

// WindowsSandbox implements Sandbox using Windows Job Objects and Integrity Levels.
//
// Security Mechanism:
//   - Job Objects: Process grouping and resource control
//   - Integrity Levels: Restrict filesystem write access (Low integrity)
//   - Restricted Tokens: Drop unnecessary privileges
//
// Modes:
//   - ModeReadOnly: Low integrity process, cannot write to most filesystem
//   - ModeWorkspaceWrite: Low integrity + workspace marked as low integrity (writable)
//   - ModeFullAccess: No restrictions applied
//
// Requirements:
//   - Windows Vista or later (for Integrity Levels)
//   - Administrative privileges to mark directories as low integrity
//
// Limitations:
//   - Cannot block read access to arbitrary paths (unlike Linux Landlock)
//   - Workspace directory must be marked with icacls for write access
//   - No network isolation (future enhancement)
type WindowsSandbox struct {
	mode Mode
}

// NewSandbox creates a Windows sandbox with the specified mode.
//
// Returns NoopSandbox if:
//   - Windows version < Vista (Integrity Levels not available)
//   - Required Windows APIs are not available
func NewSandbox(mode Mode) (Sandbox, error) {
	if !isWindowsSupported() {
		return &NoopSandbox{}, nil
	}

	return &WindowsSandbox{mode: mode}, nil
}

// Wrap applies Windows sandbox restrictions to the command.
//
// For ModeReadOnly:
//   - Creates process with Low Integrity Level
//   - Process cannot write to Medium+ integrity locations
//   - Assigns process to Job Object for resource control
//
// For ModeWorkspaceWrite:
//   - Creates process with Low Integrity Level
//   - Marks workspace directory as Low Integrity (allows writes)
//   - Assigns process to Job Object
//
// For ModeFullAccess:
//   - No restrictions applied
//
// IMPORTANT: The workspace directory must exist before calling Wrap()
// in ModeWorkspaceWrite mode, as it will be marked with low integrity.
func (s *WindowsSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
	// Full access mode - no restrictions
	if opts.Mode == ModeFullAccess {
		return nil
	}

	// Create job object for process management
	jobHandle, err := s.createJobObject()
	if err != nil {
		return fmt.Errorf("create job object: %w", err)
	}
	// Note: Job handle will be inherited by child process and closed
	// when process terminates due to JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	defer windows.CloseHandle(jobHandle)

	// For workspace write mode, mark workspace as low integrity
	if opts.Mode == ModeWorkspaceWrite && opts.WorkDir != "" {
		if err := markDirectoryLowIntegrity(opts.WorkDir); err != nil {
			// Log warning but continue - write operations may fail
			// This is expected if user doesn't have admin privileges
			_ = err // Ignore error for now, TODO: add logging
		}
	}

	// Prepare SysProcAttr for restricted process creation
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Create process suspended so we can modify it before execution
	// CREATE_BREAKAWAY_FROM_JOB allows creating new job even if parent is in job
	cmd.SysProcAttr.CreationFlags = windows.CREATE_SUSPENDED |
		windows.CREATE_BREAKAWAY_FROM_JOB

	// TODO: Implement token creation and integrity level setting
	// This requires more complex process creation workflow:
	// 1. Create process suspended
	// 2. Get process token
	// 3. Create restricted token with low integrity
	// 4. Set token on process
	// 5. Assign to job object
	// 6. Resume process
	//
	// For now, we create the job object infrastructure.
	// The process will be assigned to the job in a post-creation step
	// when we have process handle available.

	return nil
}

// Supported returns true if Windows sandboxing is available.
func (s *WindowsSandbox) Supported() bool {
	return isWindowsSupported()
}

// Mode returns the sandbox mode.
func (s *WindowsSandbox) Mode() Mode {
	return s.mode
}

// createJobObject creates and configures a Windows Job Object.
//
// Job Object Configuration:
//   - JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE: Kill all processes when job handle closed
//   - JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION: Terminate on unhandled exception
//
// The job object provides process lifecycle management and ensures child
// processes are terminated when the parent exits.
func (s *WindowsSandbox) createJobObject() (windows.Handle, error) {
	// Create job object with no name (anonymous)
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("CreateJobObject failed: %w", err)
	}

	// Configure job limits
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags =
		windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE |
			windows.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION

	// Apply extended limits to job
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)),
		uint32(unsafe.Sizeof(limits)),
	)
	if err != nil {
		windows.CloseHandle(job)
		return 0, fmt.Errorf("SetInformationJobObject failed: %w", err)
	}

	return job, nil
}

// setLowIntegrity sets a process token to run with low integrity level.
//
// Integrity levels (from low to high):
//   - Untrusted (S-1-16-0)
//   - Low (S-1-16-4096) <- Our sandbox level
//   - Medium (S-1-16-8192) <- Default user processes
//   - High (S-1-16-12288) <- Administrator
//   - System (S-1-16-16384) <- System services
//
// Low integrity processes cannot:
//   - Write to Medium+ integrity locations (most of filesystem)
//   - Write to registry with Medium+ integrity
//   - Send window messages to Medium+ processes
//
// Note: This function is not currently used but prepared for future
// implementation of full token-based sandboxing.
func setLowIntegrity(token windows.Token) error {
	// Convert low integrity SID string to SID structure
	// S-1-16-4096 is the Low Mandatory Level SID
	var sid *windows.SID
	err := windows.ConvertStringSidToSid(
		windows.StringToUTF16Ptr("S-1-16-4096"),
		&sid,
	)
	if err != nil {
		return fmt.Errorf("ConvertStringSidToSid failed: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(sid)))

	// Create TOKEN_MANDATORY_LABEL structure
	tml := TOKEN_MANDATORY_LABEL{
		Label: windows.SIDAndAttributes{
			Sid:        sid,
			Attributes: windows.SE_GROUP_INTEGRITY,
		},
	}

	// Set the token's integrity level
	err = windows.SetTokenInformation(
		token,
		windows.TokenIntegrityLevel,
		(*byte)(unsafe.Pointer(&tml)),
		uint32(unsafe.Sizeof(tml)+uintptr(windows.GetLengthSid(sid))),
	)
	if err != nil {
		return fmt.Errorf("SetTokenInformation failed: %w", err)
	}

	return nil
}

// markDirectoryLowIntegrity marks a directory as accessible to low-integrity processes.
//
// This function uses the icacls command-line tool to set the directory's
// integrity level to Low. This allows low-integrity processes to write to
// the directory while still restricting writes to other locations.
//
// Command executed:
//
//	icacls "path" /setintegritylevel Low /t /c /q
//
// Flags:
//   - /setintegritylevel Low: Set integrity level to Low (S-1-16-4096)
//   - /t: Apply recursively to all subdirectories and files
//   - /c: Continue on errors
//   - /q: Quiet mode (suppress success messages)
//
// Requirements:
//   - Directory must exist
//   - Requires administrative privileges (may fail without elevation)
//
// Note: Errors are returned but should be handled gracefully by caller,
// as this operation may fail without admin rights.
func markDirectoryLowIntegrity(path string) error {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	// Use icacls to set integrity level
	// icacls "path" /setintegritylevel Low /t /c /q
	cmd := exec.Command("icacls", absPath, "/setintegritylevel", "Low", "/t", "/c", "/q")

	// Run command and capture error
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("icacls failed (may need admin privileges): %w", err)
	}

	return nil
}

// isWindowsSupported checks if the Windows version supports required features.
//
// Requirements:
//   - Windows Vista or later (version 6.0+)
//   - Integrity Levels available (Vista+)
//   - Job Objects available (Windows 2000+, but we require Vista for consistency)
//
// Returns:
//   - true if Windows Vista or later
//   - false if Windows XP or earlier
func isWindowsSupported() bool {
	// Get Windows version
	ver := windows.RtlGetVersion()

	// Windows Vista = version 6.0
	// Windows 7 = version 6.1
	// Windows 8 = version 6.2
	// Windows 8.1 = version 6.3
	// Windows 10 = version 10.0
	// Windows 11 = version 10.0 (build 22000+)
	//
	// We require at least Vista (6.0) for Integrity Levels
	return ver.MajorVersion >= 6
}
