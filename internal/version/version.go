// Package version provides version information for Spin.
package version

import (
	"fmt"
	"runtime"
)

// Build-time version information.
// These variables are set via -ldflags during build.
var (
	// Version is the semantic version (e.g., "1.0.0", "dev")
	Version = "dev"

	// Commit is the git commit hash
	Commit = "unknown"

	// BuildDate is the build timestamp
	BuildDate = "unknown"
)

// VersionInfo contains version and build information.
type VersionInfo struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// GetVersionInfo returns the current version information.
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
}

// String returns a formatted version string with all build information.
func String() string {
	info := GetVersionInfo()
	return fmt.Sprintf(
		"spin version %s (commit: %s, built: %s, %s)",
		info.Version,
		info.Commit,
		info.BuildDate,
		info.GoVersion,
	)
}

// ShortVersion returns just the version number.
func ShortVersion() string {
	return Version
}
