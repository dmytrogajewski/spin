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
	if total == 0 {
		t.Fatalf("expected total > 0")
	}

	free, err := d.AvailableVRAM()
	if err != nil {
		t.Fatalf("free: %v", err)
	}
	if free == 0 {
		t.Fatalf("expected free > 0")
	}

	name, err := d.GPUName()
	if err != nil {
		t.Fatalf("name: %v", err)
	}
	if name == "" {
		t.Fatalf("expected name")
	}
}

func TestNvidiaDetector_CommandError(t *testing.T) {
	fr := &fakeRunner{err: errors.New("boom")}
	d := &NvidiaDetector{runner: fr}
	if _, err := d.AvailableVRAM(); err == nil {
		t.Fatalf("expected error")
	}
}
