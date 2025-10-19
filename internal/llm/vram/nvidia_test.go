package vram

import (
	"errors"
	"testing"
)

type fakeRunner struct {
	out map[string][]byte
	err error
}

func (f *fakeRunner) Run(name string, args ...string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := name + " " + joinArgs(args)
	return f.out[key], nil
}

func joinArgs(a []string) string {
	s := ""
	for i, v := range a {
		if i > 0 {
			s += " "
		}
		s += v
	}
	return s
}

func TestNvidiaDetector_ParseMiB(t *testing.T) {
	fr := &fakeRunner{out: map[string][]byte{
		"nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits": []byte("24576\n24576\n"),
		"nvidia-smi --query-gpu=memory.free --format=csv,noheader,nounits":  []byte("8192\n8192\n"),
		"nvidia-smi --query-gpu=name --format=csv,noheader":                 []byte("RTX 4090\n"),
	}}

	d := &NvidiaDetector{runner: fr}
	total, err := d.TotalVRAM()
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	expectedTotal := int64(24576) * 1024 * 1024 // Convert MiB to bytes
	if total != expectedTotal {
		t.Fatalf("expected total %d bytes (24576 MiB), got %d", expectedTotal, total)
	}

	free, err := d.AvailableVRAM()
	if err != nil {
		t.Fatalf("free: %v", err)
	}
	expectedFree := int64(8192) * 1024 * 1024 // Convert MiB to bytes
	if free != expectedFree {
		t.Fatalf("expected free %d bytes (8192 MiB), got %d", expectedFree, free)
	}

	name, err := d.GPUName()
	if err != nil {
		t.Fatalf("name: %v", err)
	}
	if name != "RTX 4090" {
		t.Fatalf("expected name 'RTX 4090', got %q", name)
	}
}

func TestNvidiaDetector_CommandError(t *testing.T) {
	fr := &fakeRunner{err: errors.New("boom")}
	d := &NvidiaDetector{runner: fr}
	if _, err := d.AvailableVRAM(); err == nil {
		t.Fatalf("expected error")
	}
}
