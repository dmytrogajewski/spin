package vram

import "testing"

func TestAMDDetector_Parse(t *testing.T) {
	sample := []byte(`
======================= ROCm System Management Interface =======================
GPU[0]          : VRAM Total Memory: 16384 MiB
GPU[0]          : VRAM Free Memory:  8192 MiB
    `)
	fr := &fakeRunner{out: map[string][]byte{
		"rocm-smi --showmeminfo vram": sample,
		"rocm-smi --showproductname":  []byte("GPU[0]: Radeon 7900 XT\n"),
	}}
	d := &AMDDetector{runner: fr}
	total, _ := d.TotalVRAM()
	if total == 0 {
		t.Fatalf("expected total > 0")
	}
	free, _ := d.AvailableVRAM()
	if free == 0 {
		t.Fatalf("expected free > 0")
	}
	name, _ := d.GPUName()
	if name == "" {
		t.Fatalf("expected name")
	}
}
