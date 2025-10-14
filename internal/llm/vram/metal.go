//go:build darwin

package vram

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// MetalDetector queries system memory as proxy for unified VRAM on Apple Silicon.
type MetalDetector struct{}

// TotalVRAM returns total system memory as VRAM proxy.
func (m *MetalDetector) TotalVRAM() (int64, error) {
	memStr, err := unix.Sysctl("hw.memsize")
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(memStr, 10, 64)
}

// AvailableVRAM approximates as 80% of total (conservative).
func (m *MetalDetector) AvailableVRAM() (int64, error) {
	total, err := m.TotalVRAM()
	if err != nil {
		return 0, err
	}
	return int64(float64(total) * 0.8), nil
}

// GPUName returns "Apple Silicon" + machine model.
func (m *MetalDetector) GPUName() (string, error) {
	model, err := unix.Sysctl("hw.machine")
	if err != nil {
		return "Apple Silicon", err
	}
	return "Apple Silicon " + model, nil
}
