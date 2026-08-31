package compact

// Journey: specs/journeys/JOURNEY-010-compact-command-registry.md.

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGoldens(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pipeline := Default()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join("testdata", name)
			cmd := strings.TrimSpace(readTestdata(t, filepath.Join(dir, "cmd")))
			exit := mustAtoi(t, strings.TrimSpace(readTestdata(t, filepath.Join(dir, "exit"))))
			raw := []byte(readTestdata(t, filepath.Join(dir, "raw")))
			want := []byte(readTestdata(t, filepath.Join(dir, "compact")))

			result := pipeline.Apply(cmd, raw, nil, exit)

			if result.ExitCode != exit {
				t.Fatalf("exit = %d, want %d", result.ExitCode, exit)
			}

			if !bytes.Equal(result.Stdout, want) {
				t.Fatalf("stdout mismatch\n got %q\nwant %q", result.Stdout, want)
			}
		})
	}
}

func TestDefault_UnknownPassthrough(t *testing.T) {
	t.Parallel()

	stdout := []byte("raw-bytes")
	result := Default().Apply(unknownCommand, stdout, nil, 0)

	if !bytes.Equal(result.Stdout, stdout) {
		t.Fatalf("stdout = %q, want passthrough", result.Stdout)
	}

	if result.Strategy != StrategyR14 {
		t.Fatalf("strategy = %q, want %q", result.Strategy, StrategyR14)
	}
}

func TestNew_KnownCommandPassthrough(t *testing.T) {
	t.Parallel()

	stdout := []byte("README.md\n")
	result := New().Apply("ls", stdout, nil, 0)

	if !bytes.Equal(result.Stdout, stdout) {
		t.Fatalf("New() must not register table filters")
	}

	if result.Strategy != StrategyR14 {
		t.Fatalf("strategy = %q, want %q", result.Strategy, StrategyR14)
	}
}

func TestDefault_PrefixArgv(t *testing.T) {
	t.Parallel()

	raw := []byte(readTestdata(t, filepath.Join("testdata", "git-status", "raw")))
	want := []byte(readTestdata(t, filepath.Join("testdata", "git-status", "compact")))
	result := Default().Apply("git status --porcelain", raw, nil, 0)

	if !bytes.Equal(result.Stdout, want) {
		t.Fatalf("prefix argv mismatch\n got %q\nwant %q", result.Stdout, want)
	}
}

func TestStrategyR11_NotAFilter(t *testing.T) {
	t.Parallel()

	if StrategyR11 != "R11" {
		t.Fatalf("StrategyR11 = %q, want R11", StrategyR11)
	}

	result := Default().Apply("echo hi", []byte("hi\n"), nil, 0)
	if result.Strategy != StrategyR14 {
		t.Fatalf("rewrite must not be registered, strategy = %q", result.Strategy)
	}
}

func TestPackage_NoNetworkOrGit(t *testing.T) {
	t.Parallel()

	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	execNeedle := "os/" + "exec"
	httpNeedle := "net/" + "http"

	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		body, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}

		if bytes.Contains(body, []byte(execNeedle)) {
			t.Errorf("%s must not import %s", path, execNeedle)
		}

		if bytes.Contains(body, []byte(httpNeedle)) {
			t.Errorf("%s must not import %s", path, httpNeedle)
		}
	}
}

func TestDefault_GoTestNonzeroExit(t *testing.T) {
	t.Parallel()

	raw := []byte(readTestdata(t, filepath.Join("testdata", "gotest", "raw")))
	result := Default().Apply("go test", raw, nil, 1)

	if result.ExitCode != 1 {
		t.Fatalf("exit = %d, want 1", result.ExitCode)
	}
}

func TestDefault_ReadDefaultMinimal(t *testing.T) {
	t.Parallel()

	raw := []byte(readTestdata(t, filepath.Join("testdata", "read-minimal", "raw")))
	want := []byte(readTestdata(t, filepath.Join("testdata", "read-minimal", "compact")))
	result := Default().Apply("read", raw, nil, 0)

	if !bytes.Equal(result.Stdout, want) {
		t.Fatalf("default read level must be minimal")
	}
}

func readTestdata(t *testing.T, path string) string {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(body)
}

func mustAtoi(t *testing.T, text string) int {
	t.Helper()

	value, err := strconv.Atoi(text)
	if err != nil {
		t.Fatalf("atoi %q: %v", text, err)
	}

	return value
}
