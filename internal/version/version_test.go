package version_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/version"
)

func TestGetVersionInfo(t *testing.T) {
	info := version.GetVersionInfo()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	// Should contain Go version
	if !strings.Contains(info.GoVersion, "go") {
		t.Errorf("GoVersion should contain 'go', got: %s", info.GoVersion)
	}
}

func TestVersionString(t *testing.T) {
	// Save original values
	origVersion := version.Version
	origCommit := version.Commit
	origBuildDate := version.BuildDate

	// Test with dev version
	version.Version = "dev"
	version.Commit = "unknown"
	version.BuildDate = "unknown"

	str := version.String()
	if !strings.Contains(str, "dev") {
		t.Errorf("Version string should contain 'dev', got: %s", str)
	}
	if !strings.Contains(str, "unknown") {
		t.Errorf("Version string should contain 'unknown', got: %s", str)
	}

	// Test with release version
	version.Version = "1.0.0"
	version.Commit = "abc123"
	version.BuildDate = "2025-10-05"

	str = version.String()
	if !strings.Contains(str, "1.0.0") {
		t.Errorf("Version string should contain '1.0.0', got: %s", str)
	}
	if !strings.Contains(str, "abc123") {
		t.Errorf("Version string should contain commit 'abc123', got: %s", str)
	}
	if !strings.Contains(str, "2025-10-05") {
		t.Errorf("Version string should contain build date '2025-10-05', got: %s", str)
	}

	// Restore original values
	version.Version = origVersion
	version.Commit = origCommit
	version.BuildDate = origBuildDate
}

func TestShortVersion(t *testing.T) {
	// Save original value
	origVersion := version.Version

	version.Version = "1.2.3"
	short := version.ShortVersion()
	if short != "1.2.3" {
		t.Errorf("ShortVersion() = %s, want '1.2.3'", short)
	}

	version.Version = "dev"
	short = version.ShortVersion()
	if short != "dev" {
		t.Errorf("ShortVersion() = %s, want 'dev'", short)
	}

	// Restore original value
	version.Version = origVersion
}

func TestGoVersionMatches(t *testing.T) {
	info := version.GetVersionInfo()
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion mismatch: got %s, want %s", info.GoVersion, runtime.Version())
	}
}

func TestVersionInfo_Format(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
		want      string
	}{
		{
			name:      "dev build",
			version:   "dev",
			commit:    "unknown",
			buildDate: "unknown",
			want:      "dev",
		},
		{
			name:      "release build",
			version:   "1.0.0",
			commit:    "abc123",
			buildDate: "2025-10-05",
			want:      "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origVersion := version.Version
			origCommit := version.Commit
			origBuildDate := version.BuildDate

			version.Version = tt.version
			version.Commit = tt.commit
			version.BuildDate = tt.buildDate

			info := version.GetVersionInfo()
			if info.Version != tt.version {
				t.Errorf("Version = %s, want %s", info.Version, tt.version)
			}
			if info.Commit != tt.commit {
				t.Errorf("Commit = %s, want %s", info.Commit, tt.commit)
			}
			if info.BuildDate != tt.buildDate {
				t.Errorf("BuildDate = %s, want %s", info.BuildDate, tt.buildDate)
			}

			// Restore original values
			version.Version = origVersion
			version.Commit = origCommit
			version.BuildDate = origBuildDate
		})
	}
}
