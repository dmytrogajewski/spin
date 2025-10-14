package vram

import (
	"fmt"
)

// ModelRequirements captures the chosen parameters for best-fit loading.
type ModelRequirements struct {
	Quantization    string
	ContextLength   int
	NumGPULayers    int
	MinVRAM         int64 // bytes
	RecommendedVRAM int64 // bytes
}

// RequirementsCalculator computes model requirements given a detector.
type RequirementsCalculator struct {
	detector      Detector
	headroomBytes int64
	layerCount    int
	kvBytesPerTok int64
}

// NewRequirementsCalculator creates a calculator with sensible defaults.
// headroomBytes reserves VRAM for system processes.
func NewRequirementsCalculator(detector Detector, headroomBytes int64) *RequirementsCalculator {
	return &RequirementsCalculator{
		detector:      detector,
		headroomBytes: headroomBytes,
		layerCount:    32,
		kvBytesPerTok: 2, // simplistic estimation
	}
}

// Calculate selects the best-fit quantization and context length.
// modelSizeBytes is the unquantized parameter size in bytes.
// contextLen is the desired context length; may be reduced if needed.
func (c *RequirementsCalculator) Calculate(modelSizeBytes int64, contextLen int) (*ModelRequirements, error) {
	if c.detector == nil {
		return nil, fmt.Errorf("detector is required")
	}
	available, err := c.detector.AvailableVRAM()
	if err != nil {
		return nil, err
	}
	if available > c.headroomBytes {
		available -= c.headroomBytes
	} else {
		available = 0
	}

	// Try quantizations in descending quality
	quantOrder := []string{"f16", "q8_0", "q4_0"}
	quantFactor := map[string]float64{
		"f16":  1.0,
		"q8_0": 0.5,
		"q4_0": 0.25,
	}

	// Attempt with requested context, then reduce to floor if needed
	tryContext := func(ctxLen int) (*ModelRequirements, bool) {
		kv := int64(ctxLen) * int64(c.layerCount) * c.kvBytesPerTok
		for _, q := range quantOrder {
			modelVRAM := int64(float64(modelSizeBytes) * quantFactor[q])
			total := modelVRAM + kv
			if total <= available {
				return &ModelRequirements{
					Quantization:    q,
					ContextLength:   ctxLen,
					NumGPULayers:    -1,
					MinVRAM:         total,
					RecommendedVRAM: total + (1 << 30), // +1GiB headroom
				}, true
			}
		}
		return nil, false
	}

	if reqs, ok := tryContext(contextLen); ok {
		return reqs, nil
	}
	if contextLen > 2048 {
		if reqs, ok := tryContext(2048); ok {
			return reqs, nil
		}
	}

	// Fallback: partial GPU layers and minimal context
	return &ModelRequirements{
		Quantization:    "q4_0",
		ContextLength:   2048,
		NumGPULayers:    16,
		MinVRAM:         0,
		RecommendedVRAM: available,
	}, nil
}
