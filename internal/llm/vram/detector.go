package vram

import (
	"bytes"
	"errors"
	"os/exec"
)

// Detector reports GPU VRAM capabilities.
// All methods should be fast and non-blocking for UX.
type Detector interface {
	TotalVRAM() (int64, error)     // bytes
	AvailableVRAM() (int64, error) // bytes
	GPUName() (string, error)
}

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// defaultRunner runs system commands.
type defaultRunner struct{}

func (d *defaultRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// NewDetector attempts to auto-detect the best available detector for the
// current platform. Order: NVIDIA → AMD → CPU fallback.
// MetalDetector is only available on macOS and should be constructed directly by callers.
// A nil runner uses the default system runner.
func NewDetector(runner CommandRunner) Detector {
	if runner == nil {
		runner = &defaultRunner{}
	}
	if hasNvidia(runner) {
		return &NvidiaDetector{runner: runner}
	}
	if hasAMD(runner) {
		return &AMDDetector{runner: runner}
	}
	return &CPUFallback{}
}

func hasNvidia(runner CommandRunner) bool {
	out, err := runner.Run("nvidia-smi", "--help")
	return err == nil && len(bytes.TrimSpace(out)) >= 0
}

func hasAMD(runner CommandRunner) bool {
	out, err := runner.Run("rocm-smi", "--help")
	return err == nil && len(bytes.TrimSpace(out)) >= 0
}

// CPUFallback is returned when no GPU detector is available.
type CPUFallback struct{}

func (c *CPUFallback) TotalVRAM() (int64, error)     { return 0, nil }
func (c *CPUFallback) AvailableVRAM() (int64, error) { return 0, nil }
func (c *CPUFallback) GPUName() (string, error)      { return "cpu", nil }

// ErrNotImplemented is returned by detectors that are stubs on the
// current platform (e.g., MetalDetector on non-macOS hosts).
var ErrNotImplemented = errors.New("not implemented on this platform")
