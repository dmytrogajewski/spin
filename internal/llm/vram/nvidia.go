package vram

import (
	"bytes"
	"strconv"
	"strings"
)

// NvidiaDetector queries nvidia-smi for VRAM stats.
type NvidiaDetector struct {
	runner CommandRunner
}

func (d *NvidiaDetector) TotalVRAM() (int64, error) {
	// Query total memory in MiB
	// Some systems have multiple GPUs; take the first line (v1 scope: single GPU).
	out, err := d.runner.Run("nvidia-smi", "--query-gpu=memory.total", "--format=csv,noheader,nounits")
	if err != nil {
		return 0, err
	}
	return parseMiBFirstLine(out)
}

func (d *NvidiaDetector) AvailableVRAM() (int64, error) {
	out, err := d.runner.Run("nvidia-smi", "--query-gpu=memory.free", "--format=csv,noheader,nounits")
	if err != nil {
		return 0, err
	}
	return parseMiBFirstLine(out)
}

func (d *NvidiaDetector) GPUName() (string, error) {
	out, err := d.runner.Run("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
	if err != nil {
		return "", err
	}
	line := firstLine(out)
	return strings.TrimSpace(line), nil
}

func parseMiBFirstLine(out []byte) (int64, error) {
	line := strings.TrimSpace(firstLine(out))
	if line == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return 0, err
	}
	return v * 1024 * 1024, nil
}

func firstLine(out []byte) string {
	idx := bytes.IndexByte(out, '\n')
	if idx == -1 {
		return string(out)
	}
	return string(out[:idx])
}
