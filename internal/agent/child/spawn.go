package child

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const (
	argA2A   = "a2a"
	argSpec  = "--spec"
	argStdio = "--stdio"
)

// ErrChildCrashed indicates the child process exited before a successful A2A exchange.
var ErrChildCrashed = errors.New("child: process crashed")

// Process is a spawned `spin a2a` (or helper) child with stdio bound to A2A.
type Process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr bytes.Buffer
	client *a2a.Client
	task   *a2a.Task
}

// StartSpec starts `bin a2a --spec name --stdio` as an OS process.
func StartSpec(ctx context.Context, bin, specName, workDir string) (*Process, error) {
	return Start(ctx, bin, []string{argA2A, argSpec, specName, argStdio}, workDir)
}

// Start starts bin with args. Child stdout is the RPC stream; stderr is captured.
func Start(ctx context.Context, bin string, args []string, workDir string) (*Process, error) {
	if bin == "" {
		return nil, errEmptyBinary
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	proc := &Process{cmd: cmd}

	if err := proc.bindPipes(); err != nil {
		return nil, err
	}

	if startErr := cmd.Start(); startErr != nil {
		return nil, fmt.Errorf("child start: %w", startErr)
	}

	client, cardErr := a2a.NewClient(proc.stdout, proc.stdin)
	if cardErr != nil {
		_, _ = proc.recordCrash()

		return proc, fmt.Errorf("child card: %w", cardErr)
	}

	proc.client = client

	return proc, nil
}

func (proc *Process) bindPipes() error {
	stdin, err := proc.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("child stdin: %w", err)
	}

	stdout, err := proc.cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()

		return fmt.Errorf("child stdout: %w", err)
	}

	proc.stdin = stdin
	proc.stdout = stdout
	proc.cmd.Stderr = &proc.stderr

	return nil
}

// PID returns the child process id, or 0 if the process was not started.
func (proc *Process) PID() int {
	if proc == nil || proc.cmd == nil || proc.cmd.Process == nil {
		return 0
	}

	return proc.cmd.Process.Pid
}

// Task returns the last completed, failed, or crash-synthesized Task.
func (proc *Process) Task() *a2a.Task {
	if proc == nil {
		return nil
	}

	return proc.task
}

// SignalTERM sends SIGTERM after tasks/cancel.
func (proc *Process) SignalTERM() error {
	if proc == nil || proc.cmd == nil || proc.cmd.Process == nil {
		return nil
	}

	if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("child sigterm: %w", err)
	}

	return nil
}

// Close kills the child and reaps it.
func (proc *Process) Close() error {
	if proc == nil || proc.cmd == nil {
		return nil
	}

	if proc.cmd.Process != nil {
		_ = proc.cmd.Process.Kill()
	}

	proc.reap()

	return nil
}

func (proc *Process) reap() {
	if proc.cmd != nil {
		_ = proc.cmd.Wait()
	}
}
