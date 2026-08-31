package child

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	runtimeSubdir = "spin/a2a"
	pidExt        = ".pid"
	sockExt       = ".sock"
)

// RuntimeDir is XDG_RUNTIME_DIR/spin/a2a, or a spin-prefixed temp path.
func RuntimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, runtimeSubdir)
	}

	return filepath.Join(os.TempDir(), runtimeSubdir)
}

// PidPath is dir/<pid>.pid.
func PidPath(dir string, pid int) string {
	return filepath.Join(dir, strconv.Itoa(pid)+pidExt)
}

// SockPath is dir/<pid>.sock.
func SockPath(dir string, pid int) string {
	return filepath.Join(dir, strconv.Itoa(pid)+sockExt)
}

// WritePidFile writes dir/<pid>.pid containing the pid.
func WritePidFile(dir string, pid int) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("child mkdir runtime: %w", err)
	}

	if err := os.WriteFile(PidPath(dir, pid), []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		return fmt.Errorf("child write pid: %w", err)
	}

	return nil
}

// ReapOnStart reaps stale pid/socket files under RuntimeDir.
func ReapOnStart() error {
	return ReapStale(RuntimeDir())
}

// ReapStale SIGTERMs live orphans and removes pid/socket files.
func ReapStale(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("child reap: %w", err)
	}

	for _, entry := range entries {
		reapDeadPid(dir, entry.Name())
	}

	return nil
}

func reapDeadPid(dir, name string) {
	if filepath.Ext(name) != pidExt {
		return
	}

	pid, err := strconv.Atoi(name[:len(name)-len(pidExt)])
	if err != nil {
		return
	}

	if proc, findErr := os.FindProcess(pid); findErr == nil && pidAlive(pid) {
		_ = proc.Signal(syscall.SIGTERM)
	}

	_ = os.Remove(filepath.Join(dir, name))
	_ = os.Remove(SockPath(dir, pid))
}

func pidAlive(pid int) bool {
	proc, err := os.FindProcess(pid)

	return err == nil && proc.Signal(syscall.Signal(0)) == nil
}
