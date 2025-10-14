package vram

import (
	"regexp"
	"strconv"
)

// AMDDetector queries rocm-smi for VRAM stats.
type AMDDetector struct {
	runner CommandRunner
}

var rocmMemRE = regexp.MustCompile(`VRAM Total Memory:\s*(\d+)\s*MiB|Total VRAM Memory:\s*(\d+)\s*MiB`)
var rocmFreeRE = regexp.MustCompile(`VRAM Free Memory:\s*(\d+)\s*MiB|VRAM Usage:\s*\d+/(\d+)\s*MiB`)

func (d *AMDDetector) TotalVRAM() (int64, error) {
	out, err := d.runner.Run("rocm-smi", "--showmeminfo", "vram")
	if err != nil {
		return 0, err
	}
	return parseROCMFirstMiB(out, rocmMemRE)
}

func (d *AMDDetector) AvailableVRAM() (int64, error) {
	out, err := d.runner.Run("rocm-smi", "--showmeminfo", "vram")
	if err != nil {
		return 0, err
	}
	// Prefer explicit free; fall back to parsing usage second group if needed.
	if v, err := parseROCMFirstMiB(out, rocmFreeRE); err == nil && v > 0 {
		return v, nil
	}
	return 0, nil
}

func (d *AMDDetector) GPUName() (string, error) {
	out, err := d.runner.Run("rocm-smi", "--showproductname")
	if err != nil {
		return "", err
	}
	// Very loose parse; return first product name occurrence if present.
	re := regexp.MustCompile(`GPU\[\d+\]\s*:\s*(.+)`) // e.g., GPU[0]: Radeon XXX
	m := re.FindSubmatch(out)
	if len(m) >= 2 {
		return string(m[1]), nil
	}
	return "amd-gpu", nil
}

func parseROCMFirstMiB(out []byte, re *regexp.Regexp) (int64, error) {
	m := re.FindSubmatch(out)
	if len(m) == 0 {
		return 0, nil
	}
	// m can have two alternatives; pick the first non-empty group.
	var s string
	for i := 1; i < len(m); i++ {
		if len(m[i]) > 0 {
			s = string(m[i])
			break
		}
	}
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return v * 1024 * 1024, nil
}
