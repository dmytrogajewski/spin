package vram

import (
	"errors"
	"testing"
)

// mockRunner implements CommandRunner for testing
type mockRunner struct {
	outputs map[string][]byte
	errors  map[string]error
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
		if len(args) > 1 {
			key = name + " " + args[0] + " " + args[1]
		}
		if len(args) > 2 {
			key = name + " " + args[0] + " " + args[1] + " " + args[2]
		}
	}
	if err, exists := m.errors[key]; exists {
		return nil, err
	}
	if output, exists := m.outputs[key]; exists {
		return output, nil
	}
	// Default behavior for unmocked commands
	if name == "nvidia-smi" || name == "rocm-smi" {
		return nil, errors.New("command not found")
	}
	return []byte("mock output"), nil
}

func TestNewDetector(t *testing.T) {
	tests := []struct {
		name   string
		runner CommandRunner
		want   string // Expected detector type
	}{
		{
			name: "nil runner uses default",
			runner: &mockRunner{
				errors: map[string]error{
					"nvidia-smi --help": errors.New("not found"),
					"rocm-smi --help":   errors.New("not found"),
				},
			},
			want: "cpu", // Will fall back to CPU
		},
		{
			name: "nvidia available",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"nvidia-smi --help": []byte("NVIDIA System Management Interface"),
					"nvidia-smi --query-gpu=name --format=csv,noheader": []byte("GeForce RTX 3080"),
				},
			},
			want: "GeForce RTX 3080",
		},
		{
			name: "amd available",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --help":            []byte("ROCm System Management Interface"),
					"rocm-smi --showproductname": []byte("GPU[0]: Radeon RX 6800 XT"),
				},
			},
			want: "Radeon RX 6800 XT",
		},
		{
			name: "cpu fallback",
			runner: &mockRunner{
				errors: map[string]error{
					"nvidia-smi --help": errors.New("not found"),
					"rocm-smi --help":   errors.New("not found"),
				},
			},
			want: "cpu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector(tt.runner)

			// Test that we get the expected detector type
			gpuName, err := detector.GPUName()
			if err != nil {
				t.Errorf("NewDetector().GPUName() error = %v", err)
				return
			}

			if gpuName != tt.want {
				t.Errorf("NewDetector().GPUName() = %v, want %v", gpuName, tt.want)
			}
		})
	}
}

func TestHasNvidia(t *testing.T) {
	tests := []struct {
		name   string
		runner CommandRunner
		want   bool
	}{
		{
			name: "nvidia available",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"nvidia-smi --help": []byte("NVIDIA System Management Interface"),
				},
			},
			want: true,
		},
		{
			name: "nvidia not available",
			runner: &mockRunner{
				errors: map[string]error{
					"nvidia-smi --help": errors.New("not found"),
				},
			},
			want: false,
		},
		{
			name: "nvidia returns empty output",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"nvidia-smi --help": []byte(""),
				},
			},
			want: true, // Empty output is still considered success
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasNvidia(tt.runner)
			if result != tt.want {
				t.Errorf("hasNvidia() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestHasAMD(t *testing.T) {
	tests := []struct {
		name   string
		runner CommandRunner
		want   bool
	}{
		{
			name: "amd available",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --help": []byte("ROCm System Management Interface"),
				},
			},
			want: true,
		},
		{
			name: "amd not available",
			runner: &mockRunner{
				errors: map[string]error{
					"rocm-smi --help": errors.New("not found"),
				},
			},
			want: false,
		},
		{
			name: "amd returns empty output",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --help": []byte(""),
				},
			},
			want: true, // Empty output is still considered success
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAMD(tt.runner)
			if result != tt.want {
				t.Errorf("hasAMD() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestCPUFallback(t *testing.T) {
	fallback := &CPUFallback{}

	// Test TotalVRAM
	total, err := fallback.TotalVRAM()
	if err != nil {
		t.Errorf("CPUFallback.TotalVRAM() error = %v", err)
	}
	if total != 0 {
		t.Errorf("CPUFallback.TotalVRAM() = %v, want 0", total)
	}

	// Test AvailableVRAM
	available, err := fallback.AvailableVRAM()
	if err != nil {
		t.Errorf("CPUFallback.AvailableVRAM() error = %v", err)
	}
	if available != 0 {
		t.Errorf("CPUFallback.AvailableVRAM() = %v, want 0", available)
	}

	// Test GPUName
	name, err := fallback.GPUName()
	if err != nil {
		t.Errorf("CPUFallback.GPUName() error = %v", err)
	}
	if name != "cpu" {
		t.Errorf("CPUFallback.GPUName() = %v, want 'cpu'", name)
	}
}

func TestDefaultRunner(t *testing.T) {
	runner := &defaultRunner{}

	// Test with a command that should exist on most systems
	output, err := runner.Run("echo", "test")
	if err != nil {
		t.Errorf("defaultRunner.Run() error = %v", err)
	}
	if string(output) != "test\n" {
		t.Errorf("defaultRunner.Run() = %v, want 'test\\n'", string(output))
	}

	// Test with a command that doesn't exist
	_, err = runner.Run("nonexistent-command-12345")
	if err == nil {
		t.Error("defaultRunner.Run() expected error for nonexistent command, got nil")
	}
}

func TestErrNotImplemented(t *testing.T) {
	if ErrNotImplemented == nil {
		t.Error("ErrNotImplemented should not be nil")
	}
	if ErrNotImplemented.Error() != "not implemented on this platform" {
		t.Errorf("ErrNotImplemented.Error() = %v, want 'not implemented on this platform'", ErrNotImplemented.Error())
	}
}

func TestAMDDetector_TotalVRAM(t *testing.T) {
	tests := []struct {
		name    string
		runner  CommandRunner
		want    int64
		wantErr bool
	}{
		{
			name: "successful VRAM detection",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --showmeminfo vram": []byte("VRAM Total Memory: 8192 MiB"),
				},
			},
			want:    8192 * 1024 * 1024, // Convert MiB to bytes
			wantErr: false,
		},
		{
			name: "alternative format",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --showmeminfo vram": []byte("Total VRAM Memory: 4096 MiB"),
				},
			},
			want:    4096 * 1024 * 1024, // Convert MiB to bytes
			wantErr: false,
		},
		{
			name: "command error",
			runner: &mockRunner{
				errors: map[string]error{
					"rocm-smi --showmeminfo vram": errors.New("command failed"),
				},
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "no match found",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --showmeminfo vram": []byte("No VRAM info available"),
				},
			},
			want:    0,
			wantErr: false, // parseROCMFirstMiB returns 0, nil when no match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &AMDDetector{runner: tt.runner}
			got, err := detector.TotalVRAM()

			if tt.wantErr {
				if err == nil {
					t.Error("TotalVRAM() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("TotalVRAM() unexpected error: %v", err)
				}
			}

			if got != tt.want {
				t.Errorf("TotalVRAM() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAMDDetector_GPUName(t *testing.T) {
	tests := []struct {
		name    string
		runner  CommandRunner
		want    string
		wantErr bool
	}{
		{
			name: "successful GPU name detection",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --showproductname": []byte("GPU[0]: Radeon RX 6800 XT"),
				},
			},
			want:    "Radeon RX 6800 XT",
			wantErr: false,
		},
		{
			name: "command error",
			runner: &mockRunner{
				errors: map[string]error{
					"rocm-smi --showproductname": errors.New("command failed"),
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "empty output",
			runner: &mockRunner{
				outputs: map[string][]byte{
					"rocm-smi --showproductname": []byte(""),
				},
			},
			want:    "amd-gpu", // AMDDetector returns default name for empty output
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &AMDDetector{runner: tt.runner}
			got, err := detector.GPUName()

			if tt.wantErr {
				if err == nil {
					t.Error("GPUName() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GPUName() unexpected error: %v", err)
				}
			}

			if got != tt.want {
				t.Errorf("GPUName() = %q, want %q", got, tt.want)
			}
		})
	}
}
