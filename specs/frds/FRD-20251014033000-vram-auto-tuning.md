# FRD-20251014033000: VRAM Auto-Tuning

**Feature:** Feature 3 from advanced roadmap — VRAM Auto-Tuning  
**Status:** Draft (Implementation Ready)  
**Date:** 2025-10-14 03:30:00  
**Author:** Spin Implementation Agent  
**Related:** `specs/advanced-features-20251012/ROADMAP.md` (Feature 3), `specs/advanced-features-20251012/RESEARCH.md` (Feature 3)

---

## 1. Executive Summary

Automatically detect available GPU VRAM and select optimal model parameters (quantization, context length, GPU layer split) before loading local LLMs (Ollama, LM Studio). Goal: zero "out of VRAM" errors, best-fit quality, fast detection (<500ms), provider-agnostic design.

---

## 2. Problem Statement

- Users manually guess VRAM-related parameters (`num_ctx`, quantization variant, GPU layers).  
- Misconfiguration causes load failures or suboptimal quality.  
- We need a portable, fast auto-tuner that works across NVIDIA/AMD/Metal and integrates cleanly with providers.

---

## 3. Scope

In-scope (v1):
- VRAM detection (NVIDIA via `nvidia-smi`, AMD via `rocm-smi`, Apple via Metal; CPU fallback).
- Requirements calculator: pick quantization (f16→q8_0→q4_0), adjust context length, optionally set GPU layers.
- Ollama integration: apply computed parameters to generation options and validate before load.
- YAML config flags and headroom.
- Tests (≥90% for new code), race clean, lint clean.

Out-of-scope (v1):
- Dynamic runtime retuning mid-session.  
- Embedding-based or ML-driven tuning.  
- Vendor-specific advanced tricks beyond documented options.

---

## 4. Requirements

### 4.1 Functional
- Detect available VRAM on NVIDIA, AMD, Metal; fallback to CPU.  
- Compute best-fit: try f16 → q8_0 → q4_0; reduce context length if needed; finally partial GPU layers.  
- Integrate with Ollama provider to set `num_ctx`, GPU layers; warn if model likely won’t fit.  
- Configurable via YAML: enable/disable, headroom MiB.

### 4.2 Non-Functional
- Detection < 500ms typical.  
- Provider-agnostic core (no vendor lock).  
- Clear logs/events on warnings.  
- Tests ≥90% coverage for new packages; `-race` clean; `make lint` clean.

---

## 5. Design

### 5.1 Package Layout
```
internal/llm/vram/
  detector.go       # Detector interface + NewDetector auto-select
  nvidia.go         # NVIDIA (nvidia-smi) implementation
  amd.go            # AMD (rocm-smi) implementation
  metal.go          # Apple Metal implementation
  calculator.go     # Compute ModelRequirements from VRAM + model size
  doc.go            # Godoc
```

### 5.2 Interfaces
```go
// Detector reports GPU VRAM capabilities.
type Detector interface {
    TotalVRAM() (int64, error)     // bytes
    AvailableVRAM() (int64, error) // bytes
    GPUName() (string, error)
}

type Calculator interface {
    Calculate(modelSizeBytes int64, contextLen int) (*ModelRequirements, error)
}

type ModelRequirements struct {
    Quantization    string // "f16", "q8_0", "q4_0"
    ContextLength   int
    NumGPULayers    int // -1 = auto/all; smaller means partial offload
    MinVRAM         int64
    RecommendedVRAM int64
}
```

### 5.3 Algorithms
- Detection: shell out to `nvidia-smi --query-gpu=memory.free --format=csv,noheader,nounits` (MiB), `rocm-smi --showmeminfo vram` (parse), Metal via cgo shim or sysctl (approx).  
- Calculation:
  - Quantization factors: f16=1.0, q8_0≈0.5, q4_0≈0.25 (configurable constants).  
  - KV cache estimate: simple formula based on context length and assumed layers; apply headroom.  
  - Iterate quality→size, then reduce context (≥2048 floor), then set partial GPU layers.

### 5.4 Integration (Ollama)
- Pre-load hook `AutoTune(ctx)` in `internal/llm/ollama/provider.go`:  
  1) Query model metadata (size).  
  2) Run calculator.  
  3) Apply: `num_ctx`, GPU layers; note: quantization is encoded in model tag (informative in logs/warning).  
  4) Emit info/warning events to TUI.

---

## 6. Configuration
```yaml
llm:
  auto_tune: true
  vram:
    detect: true
    headroom_mib: 1024  # reserved VRAM for system
```

---

## 7. Testing Strategy

Unit:
- `nvidia.go`: parse typical and edge `nvidia-smi` outputs.  
- `amd.go`: parse `rocm-smi` outputs.  
- `calculator.go`: quantization selection, context reduction, GPU layers.

Integration:
- Ollama auto-tune path with mocked detector and model metadata.  
- Warning when model cannot fit even at q4_0 with minimal context.

E2E (manual):
- NVIDIA 8GB/16GB/24GB; CPU-only host. Verify loads succeed with auto-tune on.

Targets:
- Coverage ≥90% for `internal/llm/vram`.  
- Detection <500ms (mocked timing + manual spot checks).  
- Race detector clean.

---

## 8. Acceptance Criteria
- VRAM detection works on NVIDIA, AMD, and Apple Silicon (Metal) with CPU fallback.  
- Quantization selection prefers quality and fits available VRAM.  
- Model loading success with auto-tune enabled; user warnings when not feasible.  
- YAML config toggles behavior; defaults sensible.  
- Tests pass, lint clean.

---

## 9. Risks & Mitigations
- Platform command availability → Detect and fallback; log clear guidance.  
- Approximation error in KV cache → Conservative headroom; configurable.  
- Metal detection complexity → Start with approximate; refine later.

---

## 10. Plan & Effort

1) Implement detectors + tests (NVIDIA/AMD/CPU), stub Metal.  
2) Implement calculator + tests.  
3) Wire into Ollama provider + integration tests.  
4) Add YAML config, docs, examples.  
5) Lint, race, coverage, docs polish.

ETA: ~1 week.

---

## 11. References
- `specs/advanced-features-20251012/ROADMAP.md` (Feature 3)  
- `specs/advanced-features-20251012/RESEARCH.md` (Feature 3)  
- `docs/packages/llm.md`


