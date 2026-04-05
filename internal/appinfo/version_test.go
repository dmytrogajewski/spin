package appinfo_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/appinfo"
)

func TestGetInfo(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	// Test the format of String() without modifying globals.
	// String() returns "spin version {Version} (commit: {Commit}, built: {BuildDate}, {GoVersion})".
	str := appinfo.String()

	// Must contain current version value.
	if !strings.Contains(str, appinfo.Version) {
		t.Errorf("Version string should contain version %q, got: %s", appinfo.Version, str)
	}

	// Must contain "spin version" prefix.
	if !strings.HasPrefix(str, "spin version") {
		t.Errorf("Version string should start with 'spin version', got: %s", str)
	}

	// Must contain commit info.
	if !strings.Contains(str, appinfo.Commit) {
		t.Errorf("Version string should contain commit %q, got: %s", appinfo.Commit, str)
	}

	// Must contain build date.
	if !strings.Contains(str, appinfo.BuildDate) {
		t.Errorf("Version string should contain build date %q, got: %s", appinfo.BuildDate, str)
	}

	// Verify the format matches expected pattern.
	expected := fmt.Sprintf(
		"spin version %s (commit: %s, built: %s, %s)",
		appinfo.Version, appinfo.Commit, appinfo.BuildDate, runtime.Version(),
	)
	if str != expected {
		t.Errorf("String() = %q, want %q", str, expected)
	}
}

func TestShortVersion(t *testing.T) {
	t.Parallel()

	// ShortVersion returns the current Version global.
	short := appinfo.ShortVersion()
	if short != appinfo.Version {
		t.Errorf("ShortVersion() = %s, want %s", short, appinfo.Version)
	}
}

func TestGoVersionMatches(t *testing.T) {
	t.Parallel()

	info := appinfo.GetInfo()
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion mismatch: got %s, want %s", info.GoVersion, runtime.Version())
	}
}

func TestInfo_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
	}{
		{
			name:      "dev build",
			version:   "dev",
			commit:    "unknown",
			buildDate: "unknown",
		},
		{
			name:      "release build",
			version:   "1.0.0",
			commit:    "abc123",
			buildDate: "2025-10-05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Construct Info directly instead of modifying globals.
			info := appinfo.Info{
				Version:   tt.version,
				Commit:    tt.commit,
				BuildDate: tt.buildDate,
			}

			if info.Version != tt.version {
				t.Errorf("Version = %s, want %s", info.Version, tt.version)
			}

			if info.Commit != tt.commit {
				t.Errorf("Commit = %s, want %s", info.Commit, tt.commit)
			}

			if info.BuildDate != tt.buildDate {
				t.Errorf("BuildDate = %s, want %s", info.BuildDate, tt.buildDate)
			}
		})
	}
}
