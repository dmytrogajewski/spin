//go:build darwin

package vram

import (
	"strings"
	"testing"
	"time"
)

func TestMetalDetector_TotalVRAM(t *testing.T) {
	d := &MetalDetector{}
	total, err := d.TotalVRAM()
	if err != nil {
		t.Fatalf("TotalVRAM error: %v", err)
	}
	if total <= 0 {
		t.Errorf("TotalVRAM = %d, want >0", total)
	}
}

func TestMetalDetector_AvailableVRAM(t *testing.T) {
	d := &MetalDetector{}
	avail, err := d.AvailableVRAM()
	if err != nil {
		t.Fatalf("AvailableVRAM error: %v", err)
	}
	total, _ := d.TotalVRAM()
	want := int64(float64(total) * 0.8)
	if avail != want {
		t.Errorf("AvailableVRAM = %d, want %d (80%% of total)", avail, want)
	}
}

func TestMetalDetector_GPUName(t *testing.T) {
	d := &MetalDetector{}
	name, err := d.GPUName()
	if err != nil {
		t.Fatalf("GPUName error: %v", err)
	}
	if name == "" || !strings.Contains(name, "Apple") {
		t.Errorf("GPUName = %q, want containing 'Apple'", name)
	}
}

func BenchmarkMetalDetector_TotalVRAM(b *testing.B) {
	d := &MetalDetector{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := d.TotalVRAM()
		if err != nil {
			b.Fatal(err)
		}
	}
	if b.Elapsed() > 500*time.Millisecond {
		b.Errorf("Detection time exceeded 500ms: %v", b.Elapsed())
	}
}
