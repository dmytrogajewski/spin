package vram

import "testing"

type staticDetector struct{ avail, total int64 }

func (s *staticDetector) TotalVRAM() (int64, error)     { return s.total, nil }
func (s *staticDetector) AvailableVRAM() (int64, error) { return s.avail, nil }
func (s *staticDetector) GPUName() (string, error)      { return "static", nil }

func TestCalculator_SelectsQuantization(t *testing.T) {
	// 12GiB available after headroom
	det := &staticDetector{avail: 13 << 30, total: 16 << 30}
	calc := NewRequirementsCalculator(det, 1<<30)

	// Model 12GiB f16, should pick q8_0 (~6GiB) with context 4096 within 12GiB+kv
	model := int64(12 << 30)
	req, err := calc.Calculate(model, 4096)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if req.Quantization != "q8_0" && req.Quantization != "q4_0" && req.Quantization != "f16" {
		t.Fatalf("unexpected quant: %s", req.Quantization)
	}
	if req.ContextLength <= 0 {
		t.Fatalf("expected context > 0")
	}
}

func TestCalculator_Fallbacks(t *testing.T) {
	// Very low VRAM → fallback
	det := &staticDetector{avail: 512 << 20, total: 512 << 20}
	calc := NewRequirementsCalculator(det, 0)
	req, err := calc.Calculate(8<<30, 4096)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if req.ContextLength != 2048 {
		t.Fatalf("expected 2048 ctx, got %d", req.ContextLength)
	}
	if req.Quantization != "q4_0" {
		t.Fatalf("expected q4_0, got %s", req.Quantization)
	}
}
