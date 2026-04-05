// Package appinfo provides version information for Spin.
package appinfo

import (
	"fmt"
	"runtime"
)

// Build-time version information.
// These variables are set via -ldflags during build.
var (
	// Version is the semantic version (e.g., "1.0.0", "dev").
	Version = "dev"

	// Commit is the git commit hash.
	Commit = "unknown"

	// BuildDate is the build timestamp.
	BuildDate = "unknown"
)

// Info contains version and build information.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// GetInfo returns the current version information.
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
}

// String returns a formatted version string with all build information.
func String() string {
	info := GetInfo()

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
