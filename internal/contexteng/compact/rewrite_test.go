package compact

// Journey: specs/journeys/JOURNEY-011-apply-compact-to-shell-exec.md.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var errRTKMissing = errors.New("rtk not found")

func TestRewriteArgv_PrefixesRTKWhenFound(t *testing.T) {
	t.Parallel()

	got, used := RewriteArgv("git status", BackendRTK, func(string) (string, error) {
		return "/fake/rtk", nil
	})
	if !used || got != "rtk git status" {
		t.Fatalf("RewriteArgv = %q used=%v, want rtk-prefixed", got, used)
	}
}

func TestRewriteArgv_MissingRTKFallsBack(t *testing.T) {
	t.Parallel()

	got, used := RewriteArgv("git status", BackendRTK, func(string) (string, error) {
		return "", errRTKMissing
	})
	if used || got != "git status" {
		t.Fatalf("RewriteArgv = %q used=%v, want original", got, used)
	}
}

func TestRewriteArgv_WrongBackendNoPrefix(t *testing.T) {
	t.Parallel()

	got, used := RewriteArgv("git status", "", func(string) (string, error) {
		return "/fake/rtk", nil
	})
	if used || got != "git status" {
		t.Fatalf("empty backend must not rewrite, got %q used=%v", got, used)
	}
}

func TestRewriteArgv_EnvOff(t *testing.T) {
	t.Setenv(EnvName, EnvOff)

	got, used := RewriteArgv("git status", BackendRTK, func(string) (string, error) {
		return "/fake/rtk", nil
	})
	if used || got != "git status" {
		t.Fatalf("SPIN_COMPACT=0 must skip rewrite, got %q used=%v", got, used)
	}
}

func TestRewriteArgv_AlreadyPrefixed(t *testing.T) {
	t.Parallel()

	got, used := RewriteArgv("rtk git status", BackendRTK, func(string) (string, error) {
		return "/fake/rtk", nil
	})
	if !used || got != "rtk git status" {
		t.Fatalf("already prefixed must stay, got %q used=%v", got, used)
	}
}

func TestRewriteArgv_FakePATHBinary(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, BinaryRTK)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir)

	got, used := RewriteArgv("git status", BackendRTK, exec.LookPath)
	if !used || got != "rtk git status" {
		t.Fatalf("fake PATH rtk: got %q used=%v", got, used)
	}
}

func TestShouldApply_SkipsRTKAndEscape(t *testing.T) {
	if !ShouldApply(true, "git status") {
		t.Fatal("enabled git status should apply")
	}

	if ShouldApply(false, "git status") {
		t.Fatal("disabled must not apply")
	}

	if ShouldApply(true, "rtk git status") {
		t.Fatal("rtk-prefixed must not apply")
	}

	t.Setenv(EnvName, EnvOff)

	if ShouldApply(true, "git status") {
		t.Fatal("SPIN_COMPACT=0 must not apply")
	}
}

func TestApply_IdempotentGitStatusExit(t *testing.T) {
	t.Parallel()

	raw := []byte(readTestdata(t, "testdata/git-status/raw"))
	first := Default().Apply("git status", raw, nil, 1)
	second := Default().Apply("git status", first.Stdout, nil, first.ExitCode)

	if second.ExitCode != 1 {
		t.Fatalf("second apply exit = %d, want 1", second.ExitCode)
	}
}
