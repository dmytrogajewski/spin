// Package sandbox provides platform-specific OS-level isolation for
// command execution in the Spin autonomous coding agent.
//
// The sandbox package implements filesystem and network isolation using
// platform-specific mechanisms:
//
//   - Linux: Landlock LSM (kernel 5.13+)
//   - macOS: Seatbelt (sandbox-exec)
//   - Windows: Not yet implemented (uses NoopSandbox)
//
// # Sandbox Modes
//
// Three restriction levels are available:
//
//   - ModeReadOnly: Allows reading workspace and system paths, blocks all writes
//   - ModeWorkspaceWrite: Allows reading system paths, writing to workspace only
//   - ModeFullAccess: No restrictions (for containerized environments)
//
// # Usage
//
// Create a sandbox and wrap commands before execution:
//
//	sb, err := sandbox.NewSandbox(sandbox.ModeWorkspaceWrite)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if sb.Supported() {
//	    cmd := exec.Command("ls", "/tmp")
//	    opts := sandbox.GetDefaultOptions(sandbox.ModeWorkspaceWrite, "/workspace")
//	    if err := sb.Wrap(cmd, opts); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
//	output, err := cmd.CombinedOutput()
//
// # Platform Support
//
// Sandboxing support varies by platform:
//
//   - Linux: Landlock is available on kernel 5.13+. Falls back to NoopSandbox
//     on older kernels.
//   - macOS: Uses sandbox-exec with Seatbelt profiles. Falls back to NoopSandbox
//     if sandbox-exec is not available.
//   - Windows: Currently not implemented. Always uses NoopSandbox.
//
// # Limitations
//
// Current implementation has some limitations:
//
//   - Linux Landlock integration requires a helper binary for proper pre-exec
//     setup. The current implementation provides the interface but delegates
//     enforcement to future helper binary implementation.
//   - macOS sandbox-exec wrapping is functional but may have performance overhead.
//   - Network blocking is not yet fully implemented on all platforms.
//
// These limitations will be addressed in future releases as the security
// infrastructure matures.
package sandbox
