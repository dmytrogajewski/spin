package appinfo_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/appinfo"
)

func TestGetInfo(t *testing.T) {
	info := appinfo.GetInfo()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	// Should contain Go appinfo.
	if !strings.Contains(info.GoVersion, "go") {
		t.Errorf("GoVersion should contain 'go', got: %s", info.GoVersion)
	}
}

func TestVersionString(t *testing.T) {
	// Save original values.
	origVersion := appinfo.Version
	origCommit := appinfo.Commit
	origBuildDate := appinfo.BuildDate

	// Test with dev appinfo.
	appinfo.Version = "dev"
	appinfo.Commit = "unknown"
	appinfo.BuildDate = "unknown"

	str := appinfo.String()
	if !strings.Contains(str, "dev") {
		t.Errorf("Version string should contain 'dev', got: %s", str)
	}

	if !strings.Contains(str, "unknown") {
		t.Errorf("Version string should contain 'unknown', got: %s", str)
	}

	// Test with release appinfo.
	appinfo.Version = "1.0.0"
	appinfo.Commit = "abc123"
	appinfo.BuildDate = "2025-10-05"

	str = appinfo.String()
	if !strings.Contains(str, "1.0.0") {
		t.Errorf("Version string should contain '1.0.0', got: %s", str)
	}

	if !strings.Contains(str, "abc123") {
		t.Errorf("Version string should contain commit 'abc123', got: %s", str)
	}

	if !strings.Contains(str, "2025-10-05") {
		t.Errorf("Version string should contain build date '2025-10-05', got: %s", str)
	}

	// Restore original values.
	appinfo.Version = origVersion
	appinfo.Commit = origCommit
	appinfo.BuildDate = origBuildDate
}

func TestShortVersion(t *testing.T) {
	// Save original value.
	origVersion := appinfo.Version

	appinfo.Version = "1.2.3"

	short := appinfo.ShortVersion()
	if short != "1.2.3" {
		t.Errorf("ShortVersion() = %s, want '1.2.3'", short)
	}

	appinfo.Version = "dev"

	short = appinfo.ShortVersion()
	if short != "dev" {
		t.Errorf("ShortVersion() = %s, want 'dev'", short)
	}

	// Restore original value.
	appinfo.Version = origVersion
}

func TestGoVersionMatches(t *testing.T) {
	info := appinfo.GetInfo()
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion mismatch: got %s, want %s", info.GoVersion, runtime.Version())
	}
}

func TestInfo_Format(t *testing.T) {
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
			// Save original values.
			origVersion := appinfo.Version
			origCommit := appinfo.Commit
			origBuildDate := appinfo.BuildDate

			appinfo.Version = tt.version
			appinfo.Commit = tt.commit
			appinfo.BuildDate = tt.buildDate

			info := appinfo.GetInfo()
			if info.Version != tt.version {
				t.Errorf("Version = %s, want %s", info.Version, tt.version)
			}

			if info.Commit != tt.commit {
				t.Errorf("Commit = %s, want %s", info.Commit, tt.commit)
			}

			if info.BuildDate != tt.buildDate {
				t.Errorf("BuildDate = %s, want %s", info.BuildDate, tt.buildDate)
			}

			// Restore original values.
			appinfo.Version = origVersion
			appinfo.Commit = origCommit
			appinfo.BuildDate = origBuildDate
		})
	}
}
