package sandbox

// GetDefaultOptions returns default sandbox options for the given mode.
func GetDefaultOptions(mode Mode, workDir string) SandboxOptions {
	opts := SandboxOptions{
		Mode:    mode,
		WorkDir: workDir,
	}

	switch mode {
	case ModeReadOnly:
		// Read-only: can read workspace and system paths, but no writes
		opts.ReadPaths = []string{
			workDir,
			"/usr",
			"/bin",
			"/lib",
			"/lib64",
			"/etc",
		}
		opts.WritePaths = []string{}
		opts.BlockNetwork = true

	case ModeWorkspaceWrite:
		// Workspace write: can read system paths, write to workspace
		opts.ReadPaths = []string{
			workDir,
			"/usr",
			"/bin",
			"/lib",
			"/lib64",
			"/etc",
		}
		opts.WritePaths = []string{workDir}
		opts.BlockNetwork = true

	case ModeFullAccess:
		// Full access: no restrictions (for containers)
		opts.ReadPaths = []string{"/"}
		opts.WritePaths = []string{"/"}
		opts.BlockNetwork = false
	}

	return opts
}
