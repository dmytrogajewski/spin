// Package vram provides GPU VRAM detection and model parameter auto-tuning
// utilities for local LLM providers (e.g., Ollama, LM Studio).
//
// The detector sub-components query platform-specific tools to estimate
// available VRAM without introducing vendor lock-in. The calculator
// estimates best-fit model parameters (quantization preference, context
// length, GPU layer split) to maximize model quality while ensuring the
// model can load successfully on the detected hardware.
package vram
