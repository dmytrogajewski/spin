package security

// validator_patterns.go defines pattern initialization for command classification.
//
// This file contains pattern definitions for:
//   - Forbidden commands (catastrophic operations)
//   - Dangerous commands (destructive operations)
//   - Interactive commands (write operations)
//   - Safe commands (read-only operations)

// initializeForbiddenPatterns sets up forbidden command patterns.
func (v *Validator) initializeForbiddenPatterns() {
	v.forbiddenPatterns = []Pattern{
		// Catastrophic deletion patterns.
		{
			Program:     "rm",
			ArgPatterns: []string{"-rf", "/"},
			Description: "Attempting to delete root filesystem",
		},
		{
			Program:     "rm",
			ArgPatterns: []string{"-rf", "/*"},
			Description: "Attempting to delete root filesystem contents",
		},
		{
			Program:     "rm",
			ArgPatterns: []string{"-rf", "~"},
			Description: "Attempting to delete home directory",
		},
		{
			Program:     "rm",
			ArgPatterns: []string{"-rf", "$HOME"},
			Description: "Attempting to delete home directory",
		},
		// Fork bomb - matches the actual fork bomb syntax.
		{
			Program:     ":()",
			Description: "Fork bomb detected",
		},
		// Piping to shell (common RCE vector).
		{
			Program:           "curl",
			ForbiddenPatterns: []string{"|", "bash", "sh"},
			Description:       "Piping curl output to shell",
		},
		{
			Program:           "wget",
			ForbiddenPatterns: []string{"|", "bash", "sh"},
			Description:       "Piping wget output to shell",
		},
		// Dangerous chmod.
		{
			Program:     "chmod",
			ArgPatterns: []string{"-R", "777", "/"},
			Description: "Setting insecure permissions on root",
		},
		// Disk operations.
		{
			Program:     "dd",
			ArgPatterns: []string{"of=/dev/sda"},
			Description: "Writing to system disk",
		},
		{
			Program:     "mkfs",
			Description: "Formatting filesystem",
		},
		// Sudo with dangerous operations.
		{
			Program:     "sudo",
			ArgPatterns: []string{"rm", "-rf", "/"},
			Description: "Sudo with dangerous deletion",
		},
	}
}

// initializeDangerousPatterns sets up dangerous command patterns.
func (v *Validator) initializeDangerousPatterns() {
	v.initializeDangerousFilePatterns()
	v.initializeDangerousPrivilegePatterns()
	v.initializeDangerousNetworkPatterns()
	v.initializeDangerousProcessPatterns()
	v.initializeDangerousPackagePatterns()
}

// initializeDangerousFilePatterns sets up dangerous file and git operation patterns.
func (v *Validator) initializeDangerousFilePatterns() {
	v.dangerousPatterns["rm"] = []Pattern{
		{Program: "rm", ArgPatterns: []string{"-rf"}, Description: "Recursive force delete"},
		{Program: "rm", ArgPatterns: []string{"-r"}, Description: "Recursive delete"},
	}
	v.dangerousPatterns["rmdir"] = []Pattern{
		{Program: "rmdir", Description: "Remove directory"},
	}
	v.dangerousPatterns["chmod"] = []Pattern{
		{Program: "chmod", ArgPatterns: []string{"+x"}, Description: "Make file executable"},
		{Program: "chmod", Description: "Change file permissions"},
	}
	v.dangerousPatterns["git"] = []Pattern{
		{Program: "git", ArgPatterns: []string{"reset", "--hard"}, Description: "Hard reset git repository"},
		{Program: "git", ArgPatterns: []string{"push", "--force"}, Description: "Force push to git repository"},
		{Program: "git", ArgPatterns: []string{"clean", "-fd"}, Description: "Force clean git repository"},
	}
}

// initializeDangerousPrivilegePatterns sets up dangerous privilege escalation patterns.
func (v *Validator) initializeDangerousPrivilegePatterns() {
	v.dangerousPatterns["sudo"] = []Pattern{
		{Program: "sudo", Description: "Execute command as root"},
	}
	v.dangerousPatterns["su"] = []Pattern{
		{Program: "su", Description: "Switch user"},
	}
}

// initializeDangerousNetworkPatterns sets up dangerous network operation patterns.
func (v *Validator) initializeDangerousNetworkPatterns() {
	v.dangerousPatterns["curl"] = []Pattern{
		{Program: "curl", ArgPatterns: []string{"-X", "POST"}, Description: "HTTP POST request"},
		{Program: "curl", ArgPatterns: []string{"-X", "PUT"}, Description: "HTTP PUT request"},
		{Program: "curl", ArgPatterns: []string{"-X", "DELETE"}, Description: "HTTP DELETE request"},
	}
	v.dangerousPatterns["wget"] = []Pattern{
		{Program: "wget", ArgPatterns: []string{"-O"}, Description: "Download file with output"},
	}
}

// initializeDangerousProcessPatterns sets up dangerous process control patterns.
func (v *Validator) initializeDangerousProcessPatterns() {
	v.dangerousPatterns["kill"] = []Pattern{
		{Program: "kill", ArgPatterns: []string{"-9"}, Description: "Force kill process"},
	}
	v.dangerousPatterns["killall"] = []Pattern{
		{Program: "killall", Description: "Kill all matching processes"},
	}
	v.dangerousPatterns["pkill"] = []Pattern{
		{Program: "pkill", Description: "Kill processes by pattern"},
	}
}

// initializeDangerousPackagePatterns sets up dangerous package manager patterns.
func (v *Validator) initializeDangerousPackagePatterns() {
	v.dangerousPatterns["apt"] = []Pattern{
		{Program: "apt", Description: "System package management"},
	}
	v.dangerousPatterns["yum"] = []Pattern{
		{Program: "yum", Description: "System package management"},
	}
	v.dangerousPatterns["brew"] = []Pattern{
		{Program: "brew", Description: "System package management"},
	}
}

// initializeInteractivePatterns sets up interactive command patterns.
func (v *Validator) initializeInteractivePatterns() {
	v.initializeInteractiveFilePatterns()
	v.initializeInteractiveGitPatterns()
	v.initializeInteractiveBuildPatterns()
}

// initializeInteractiveFilePatterns sets up interactive file operation patterns.
func (v *Validator) initializeInteractiveFilePatterns() {
	v.interactivePatterns["mkdir"] = []Pattern{
		{Program: "mkdir", Description: "Create directory"},
	}
	v.interactivePatterns["touch"] = []Pattern{
		{Program: "touch", Description: "Create or update file"},
	}
	v.interactivePatterns["cp"] = []Pattern{
		{Program: "cp", Description: "Copy file"},
	}
	v.interactivePatterns["mv"] = []Pattern{
		{Program: "mv", Description: "Move file"},
	}
	v.interactivePatterns["echo"] = []Pattern{
		{Program: "echo", ArgPatterns: []string{">"}, Description: "Write to file via echo redirect"},
	}
}

// initializeInteractiveGitPatterns sets up interactive git write patterns.
func (v *Validator) initializeInteractiveGitPatterns() {
	v.interactivePatterns["git"] = []Pattern{
		{Program: "git", ArgPatterns: []string{"add"}, Description: "Stage files in git"},
		{Program: "git", ArgPatterns: []string{"commit"}, Description: "Commit changes to git"},
		{Program: "git", ArgPatterns: []string{"checkout"}, Description: "Switch git branch"},
		{Program: "git", ArgPatterns: []string{"branch"}, Description: "Create git branch"},
	}
}

// initializeInteractiveBuildPatterns sets up interactive package manager and build tool patterns.
func (v *Validator) initializeInteractiveBuildPatterns() {
	v.interactivePatterns["npm"] = []Pattern{
		{Program: "npm", ArgPatterns: []string{"install"}, Description: "Install npm package"},
	}
	v.interactivePatterns["go"] = []Pattern{
		{Program: "go", ArgPatterns: []string{"get"}, Description: "Install Go package"},
	}
	v.interactivePatterns["pip"] = []Pattern{
		{Program: "pip", ArgPatterns: []string{"install"}, Description: "Install Python package"},
	}
	v.interactivePatterns["make"] = []Pattern{
		{Program: "make", Description: "Run makefile target"},
	}
	v.interactivePatterns["cargo"] = []Pattern{
		{Program: "cargo", ArgPatterns: []string{"build"}, Description: "Build Rust project"},
	}
}

// initializeSafePatterns sets up safe command patterns.
func (v *Validator) initializeSafePatterns() {
	v.initializeSafeFilePatterns()
	v.initializeSafeGitPatterns()
	v.initializeSafeSystemPatterns()
}

// initializeSafeFilePatterns sets up safe file reading patterns.
func (v *Validator) initializeSafeFilePatterns() {
	v.safePatterns["ls"] = []Pattern{{Program: "ls", Description: "List files"}}
	v.safePatterns["cat"] = []Pattern{{Program: "cat", Description: "Read file"}}
	v.safePatterns["head"] = []Pattern{{Program: "head", Description: "Read first lines of file"}}
	v.safePatterns["tail"] = []Pattern{{Program: "tail", Description: "Read last lines of file"}}
	v.safePatterns["grep"] = []Pattern{{Program: "grep", Description: "Search in file"}}
	v.safePatterns["find"] = []Pattern{{Program: "find", Description: "Find files"}}
	v.safePatterns["tree"] = []Pattern{{Program: "tree", Description: "Show directory tree"}}
	v.safePatterns["file"] = []Pattern{{Program: "file", Description: "Determine file type"}}
	v.safePatterns["stat"] = []Pattern{{Program: "stat", Description: "Show file statistics"}}
	v.safePatterns["wc"] = []Pattern{{Program: "wc", Description: "Count words/lines/bytes"}}
}

// initializeSafeGitPatterns sets up safe git read-only patterns.
func (v *Validator) initializeSafeGitPatterns() {
	v.safePatterns["git"] = []Pattern{
		{Program: "git", ArgPatterns: []string{"status"}, Description: "Show git status"},
		{Program: "git", ArgPatterns: []string{"log"}, Description: "Show git log"},
		{Program: "git", ArgPatterns: []string{"diff"}, Description: "Show git diff"},
		{Program: "git", ArgPatterns: []string{"show"}, Description: "Show git commit"},
	}
}

// initializeSafeSystemPatterns sets up safe system info patterns.
func (v *Validator) initializeSafeSystemPatterns() {
	v.safePatterns["pwd"] = []Pattern{{Program: "pwd", Description: "Print working directory"}}
	v.safePatterns["whoami"] = []Pattern{{Program: "whoami", Description: "Show current user"}}
	v.safePatterns["which"] = []Pattern{{Program: "which", Description: "Find program path"}}
	v.safePatterns["date"] = []Pattern{{Program: "date", Description: "Show date/time"}}
	v.safePatterns["echo"] = []Pattern{{Program: "echo", Description: "Print text"}}
}
