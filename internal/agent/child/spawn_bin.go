package child

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	envSpinBin  = "SPIN_BIN"
	spinRelPath = "build/bin/spin"
	spinName    = "spin"
)

var errEmptyBinary = errors.New("child: binary path is empty")

// ResolveBinary returns SPIN_BIN, a walked build/bin/spin, the current executable, or "spin".
func ResolveBinary() string {
	if bin, ok := FindSpinBinary(); ok {
		return bin
	}

	return spinName
}

// FindSpinBinary locates a spin executable for process spawn.
func FindSpinBinary() (string, bool) {
	if bin := os.Getenv(envSpinBin); bin != "" && fileExists(bin) {
		return bin, true
	}

	if found, ok := FindRepoBinary(); ok {
		return found, true
	}

	if exe, err := os.Executable(); err == nil && fileExists(exe) {
		return exe, true
	}

	return "", false
}

// FindRepoBinary walks from the working directory to locate build/bin/spin.
func FindRepoBinary() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		candidate := filepath.Join(dir, spinRelPath)
		if fileExists(candidate) {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}

		dir = parent
	}
}

func fileExists(path string) bool {
	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) {
		return false
	}

	info, err := os.Stat(clean)

	return err == nil && !info.IsDir()
}
