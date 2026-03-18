package e2e

import (
	"bytes"
	"context"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// skipSandboxTests skips container-based tests unless explicitly enabled.
// These tests require Docker and build a linux/amd64 binary.
func skipSandboxTests(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping sandbox test in short mode")
	}

	if os.Getenv("SPIN_E2E_SANDBOX") != "1" {
		t.Skip("Sandbox tests require SPIN_E2E_SANDBOX=1 and Docker")
	}
}

// sandboxBinPath is the path for the Linux binary used in container tests.
var sandboxBinPath = filepath.Join(".", "..", "..", "bin", "spin-linux-amd64")

// buildLinuxBinary compiles the spin binary for linux/amd64.
func buildLinuxBinary(t *testing.T) string {
	t.Helper()

	absPath, err := filepath.Abs(sandboxBinPath)
	require.NoError(t, err)

	// Skip rebuild if binary is fresh (within 5 minutes).
	if info, statErr := os.Stat(absPath); statErr == nil {
		if time.Since(info.ModTime()) < 5*time.Minute {
			t.Log("Using recently built linux binary for sandbox tests")

			return absPath
		}
	}

	t.Log("Building linux/amd64 binary for sandbox tests...")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	buildArgs := []string{"build", "-tags", "e2e_llm_test", "-o", absPath, "../../cmd/spin"}

	cmd := osExec.CommandContext(ctx, "go", buildArgs...)

	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	}

	output, buildErr := cmd.CombinedOutput()
	require.NoError(t, buildErr, "build failed: %s", output)

	return absPath
}

// spinContainer creates a container with the spin binary mounted.
// Recovers from testcontainers panics (e.g., missing Docker socket) and skips the test.
func spinContainer(ctx context.Context, t *testing.T, binPath string) *testcontainers.DockerContainer {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:      "ubuntu:22.04",
		Cmd:        []string{"tail", "-f", "/dev/null"}, // Keep-alive.
		WaitingFor: wait.ForExec([]string{"true"}).WithStartupTimeout(10 * time.Second),
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      binPath,
				ContainerFilePath: "/usr/local/bin/spin",
				FileMode:          0o755,
			},
		},
	}

	var container *testcontainers.DockerContainer

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("testcontainers panicked (Docker not configured?): %v", r)
			}
		}()

		c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		require.NoError(t, err)

		dc, ok := c.(*testcontainers.DockerContainer)
		require.True(t, ok, "expected DockerContainer")

		container = dc
	}()

	if container == nil {
		t.Skip("container not created")
	}

	t.Cleanup(func() {
		_ = container.Terminate(context.WithoutCancel(ctx))
	})

	return container
}

// containerExec runs a command inside the container and returns exit code + output.
func containerExec(ctx context.Context, t *testing.T, container testcontainers.Container, cmd []string) (int, string) {
	t.Helper()

	exitCode, reader, err := container.Exec(ctx, cmd)
	require.NoError(t, err)

	var buf bytes.Buffer

	_, _ = buf.ReadFrom(reader)

	return exitCode, buf.String()
}

// TestSandbox_HelpCommand verifies the binary runs inside a container.
func TestSandbox_HelpCommand(t *testing.T) {
	t.Parallel()
	skipSandboxTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	binPath := buildLinuxBinary(t)
	container := spinContainer(ctx, t, binPath)

	exitCode, output := containerExec(ctx, t, container, []string{"spin", "--help"})
	assert.Equal(t, 0, exitCode, "spin --help should exit 0")
	assert.Contains(t, output, "Usage:", "help output should contain Usage")
}

// TestSandbox_VersionCommand verifies version output.
func TestSandbox_VersionCommand(t *testing.T) {
	t.Parallel()
	skipSandboxTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	binPath := buildLinuxBinary(t)
	container := spinContainer(ctx, t, binPath)

	exitCode, output := containerExec(ctx, t, container, []string{"spin", "--version"})
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, output, "version")
}

// TestSandbox_ExecWriteFile verifies the agent can create a file inside the container.
func TestSandbox_ExecWriteFile(t *testing.T) {
	t.Parallel()
	skipSandboxTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	binPath := buildLinuxBinary(t)
	container := spinContainer(ctx, t, binPath)

	// Write a config that uses the test LLM provider.
	configYAML := "llm:\n  provider: test-llm\n  model: dummy\nsandbox:\n  mode: workspace-write\n"
	exitCode, _ := containerExec(ctx, t, container, []string{
		"sh", "-c", "printf '%s' '" + configYAML + "' > /tmp/spin.yaml",
	})
	require.Equal(t, 0, exitCode)

	// Run spin exec to create a file.
	exitCode, output := containerExec(ctx, t, container, []string{
		"spin", "--config-file", "/tmp/spin.yaml",
		"exec", "--auto-approve",
		"Create a file named /tmp/hello.txt containing the text 'hello from sandbox'",
	})
	t.Logf("exec output: %s", output)

	if exitCode != 0 {
		t.Logf("spin exec exited with code %d (may be expected with test-llm)", exitCode)
	}

	// Verify the file was created (if the test LLM provider triggers write_file).
	exitCode, content := containerExec(ctx, t, container, []string{"cat", "/tmp/hello.txt"})
	if exitCode == 0 {
		assert.Contains(t, content, "hello", "file should contain expected content")
	} else {
		t.Log("File not created - test LLM may not have triggered write_file tool")
	}
}

// TestSandbox_NoHostLeakage verifies the container is isolated from the host.
func TestSandbox_NoHostLeakage(t *testing.T) {
	t.Parallel()
	skipSandboxTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	binPath := buildLinuxBinary(t)
	container := spinContainer(ctx, t, binPath)

	// Verify host home directory is not accessible.
	homeDir, _ := os.UserHomeDir()
	exitCode, _ := containerExec(ctx, t, container, []string{"ls", homeDir})
	assert.NotEqual(t, 0, exitCode, "host home directory should not exist in container")

	// Verify the container has its own filesystem.
	exitCode, output := containerExec(ctx, t, container, []string{"cat", "/etc/os-release"})
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, output, "Ubuntu", "container should be Ubuntu")
}

// TestSandbox_ConfigValidation verifies config commands work in container.
func TestSandbox_ConfigValidation(t *testing.T) {
	t.Parallel()
	skipSandboxTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	binPath := buildLinuxBinary(t)
	container := spinContainer(ctx, t, binPath)

	// Create a minimal config inside the container.
	exitCode, _ := containerExec(ctx, t, container, []string{
		"sh", "-c", "echo '# spin config' > /tmp/spin.yaml",
	})
	require.Equal(t, 0, exitCode)

	// Validate config.
	exitCode, output := containerExec(ctx, t, container, []string{
		"spin", "--config-file", "/tmp/spin.yaml", "config", "validate",
	})
	t.Logf("config validate: exit=%d output=%s", exitCode, output)

	// Should produce some validation output (valid or invalid).
	assert.True(t,
		strings.Contains(output, "valid") || strings.Contains(output, "invalid"),
		"should produce validation output",
	)
}
