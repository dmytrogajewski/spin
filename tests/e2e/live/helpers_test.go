//go:build e2e_ollama

package live

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	indexHTMLName     = "index.html"
	minIndexBytes     = 512
	defaultTestbedRel = "sources/testbed/spin"
	defaultOllamaURL  = "http://localhost:11434"
	defaultModelName  = "ornith-1.5:35b-262k"
	defaultTimeout    = 45 * time.Minute
	ollamaProbeWait   = 5 * time.Second
	envTestbed        = "SPIN_LIVE_TESTBED"
	envOllamaHost     = "OLLAMA_HOST"
	envModel          = "SPIN_MODEL"
	envTimeout        = "SPIN_LIVE_TIMEOUT"
	spinBinRel        = "build/bin/spin"
)

type execResult struct {
	stdout string
	stderr string
	err    error
}

func testbedRoot() string {
	if v := os.Getenv(envTestbed); v != "" {
		return v
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "spin-testbed")
	}

	return filepath.Join(home, defaultTestbedRel)
}

func ollamaBaseURL() string {
	if v := os.Getenv(envOllamaHost); v != "" {
		return v
	}

	return defaultOllamaURL
}

func configuredModel() string {
	if v := os.Getenv(envModel); v != "" {
		return v
	}

	return defaultModelName
}

func liveTimeout() time.Duration {
	if v := os.Getenv(envTimeout); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d
		}
	}

	return defaultTimeout
}

func requireOllama(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), ollamaProbeWait)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaBaseURL()+"/api/tags", nil)
	if err != nil {
		t.Fatalf("build ollama probe: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("ollama not reachable at %s: %v", ollamaBaseURL(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read ollama tags: %v", err)
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.Unmarshal(body, &tags); err != nil {
		t.Fatalf("decode ollama tags: %v", err)
	}

	want := configuredModel()
	for _, m := range tags.Models {
		if m.Name == want {
			return
		}
	}

	t.Skipf("ollama model %s is not installed", want)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	return filepath.Dir(filepath.Dir(filepath.Dir(wd)))
}

func spinBinary(t *testing.T) string {
	t.Helper()

	return filepath.Join(repoRoot(t), spinBinRel)
}

func createLiveConfig(t *testing.T) string {
	t.Helper()

	content := `version: "2.0"
llm:
  provider: ollama
  model: ` + configuredModel() + `
  base_url: ` + ollamaBaseURL() + `
  context_window: 262144
  max_tokens: 16384
  temperature: 0.7
  timeout: 10m
agent:
  max_turns: 50
  timeout: 1h
  require_approval: false
  cycle_detection:
    enabled: true
security:
  sandbox_mode: workspace-only
protocol:
  enable_mcp: false
  enable_git: false
  enable_shell: true
  shell_timeout: 5m
agents_md:
  enabled: false
`

	path := filepath.Join(t.TempDir(), "spin.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write live config: %v", err)
	}

	return path
}

func runLiveExec(t *testing.T, workDir, prompt string) execResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), liveTimeout())
	defer cancel()

	args := []string{
		"--config-file", createLiveConfig(t),
		"--provider", "ollama",
		"--model", configuredModel(),
		"--cd", workDir,
		"exec",
		"--auto-approve",
		"--timeout", "1h",
		"--debug",
		prompt,
	}

	cmd := exec.CommandContext(ctx, spinBinary(t), args...)
	cmd.Dir = workDir

	var outBuf, errBuf bytes.Buffer

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	t.Logf("running: %s %v", spinBinary(t), args)

	err := cmd.Run()

	return execResult{stdout: outBuf.String(), stderr: errBuf.String(), err: err}
}
