package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// fixtureOpt configures a fixture test run.
type fixtureOpt func(*fixtureConfig)

type fixtureConfig struct {
	workDir     string
	autoApprove bool
	timeout     time.Duration
	execTimeout string // --timeout flag value (e.g. "1s").
	exitOnError *bool  // --exit-on-error flag (nil = default).
}

func withAutoApprove() fixtureOpt {
	return func(c *fixtureConfig) { c.autoApprove = true }
}

func withWorkDir(dir string) fixtureOpt {
	return func(c *fixtureConfig) { c.workDir = dir }
}

func withTimeout(d time.Duration) fixtureOpt {
	return func(c *fixtureConfig) { c.timeout = d }
}

func withExecTimeout(val string) fixtureOpt {
	return func(c *fixtureConfig) { c.execTimeout = val }
}

func withExitOnError(val bool) fixtureOpt {
	return func(c *fixtureConfig) { c.exitOnError = &val }
}

// runFixtureExec runs `spin exec` with a fixture file driving the LLM responses.
// fixtureName is relative to tests/e2e/fixtures/ (e.g. "simple_response.jsonl").
func runFixtureExec(t *testing.T, fixtureName, prompt string, opts ...fixtureOpt) execResult {
	t.Helper()

	cfg := &fixtureConfig{
		timeout: 60 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Resolve fixture path (relative to this test file).
	fixturePath, err := filepath.Abs(filepath.Join("fixtures", fixtureName))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}

	if _, err := os.Stat(fixturePath); err != nil {
		t.Fatalf("fixture not found: %s", fixturePath)
	}

	// Create temp config.
	configPath := createFixtureConfig(t)

	// Determine workdir.
	workDir := cfg.workDir
	if workDir == "" {
		workDir = t.TempDir()
	}

	// Build command args.
	args := []string{
		"--config-file", configPath,
		"--cd", workDir,
		"exec",
	}

	if cfg.autoApprove {
		args = append(args, "--auto-approve")
	}

	if cfg.execTimeout != "" {
		args = append(args, "--timeout", cfg.execTimeout)
	}

	if cfg.exitOnError != nil {
		if *cfg.exitOnError {
			args = append(args, "--exit-on-error")
		} else {
			args = append(args, "--exit-on-error=false")
		}
	}

	args = append(args, prompt)

	// Run spin binary.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)

	cmd.Env = append(os.Environ(), "SPIN_TEST_FIXTURE="+fixturePath)

	var outBuf, errBuf bytes.Buffer

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()

	return execResult{
		stdout: outBuf.String(),
		stderr: errBuf.String(),
		err:    runErr,
	}
}

// createFixtureConfig creates a minimal config for fixture-driven exec tests.
func createFixtureConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "spin.yaml")
	config := `llm:
  provider: test-llm
  model: fixture
  temperature: 0
  max_tokens: 4096
`

	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write fixture config: %v", err)
	}

	return configPath
}

// setupFixtureWorkDir creates a temporary directory with the given files.
func setupFixtureWorkDir(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	for name, content := range files {
		path := filepath.Join(dir, name)

		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("create dir for %s: %v", name, err)
		}

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return dir
}

// ansiRegex matches ANSI escape sequences.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// combinedOutput returns ANSI-stripped stdout+stderr.
func combinedOutput(r execResult) string {
	return stripANSI(r.stdout + r.stderr)
}

// assertOutputContains checks that the combined output contains the given substring.
func assertOutputContains(t *testing.T, r execResult, substr string) {
	t.Helper()

	output := combinedOutput(r)
	if !strings.Contains(output, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, output)
	}
}

// assertNoError checks that the exec result has no error.
func assertNoError(t *testing.T, r execResult) {
	t.Helper()

	if r.err != nil {
		t.Errorf("unexpected error: %v\nstdout: %s\nstderr: %s", r.err, r.stdout, r.stderr)
	}
}

// runFixtureExecStdin runs spin exec with the prompt piped via stdin.
func runFixtureExecStdin(t *testing.T, fixtureName, prompt string, opts ...fixtureOpt) execResult {
	t.Helper()

	cfg := &fixtureConfig{
		timeout: 60 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	fixturePath, err := filepath.Abs(filepath.Join("fixtures", fixtureName))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}

	configPath := createFixtureConfig(t)

	workDir := cfg.workDir
	if workDir == "" {
		workDir = t.TempDir()
	}

	args := []string{
		"--config-file", configPath,
		"--cd", workDir,
		"exec",
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)

	cmd.Env = append(os.Environ(), "SPIN_TEST_FIXTURE="+fixturePath)
	cmd.Stdin = strings.NewReader(prompt)

	var outBuf, errBuf bytes.Buffer

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()

	return execResult{
		stdout: outBuf.String(),
		stderr: errBuf.String(),
		err:    runErr,
	}
}
