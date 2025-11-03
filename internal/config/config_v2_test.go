package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigV2_Validate_MinimalValid tests that a minimal valid v2 config passes validation.
// This is the first step in the new v2.0 config structure.
func TestConfigV2_Validate_MinimalValid(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider:    "ollama",
			Model:       "qwen",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     30 * time.Second,
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "minimal valid config should pass validation")
}

// TestConfigV2_Validate_LLMProviderRequired tests that validation fails when LLM.Provider is empty.
// Kills mutant: removing the provider check would make this test fail.
func TestConfigV2_Validate_LLMProviderRequired(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider: "", // Empty provider should fail
			Model:    "qwen",
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "empty LLM provider should fail validation")
	assert.Contains(t, err.Error(), "provider", "error should mention provider field")
	assert.Contains(t, err.Error(), "required", "error should indicate field is required")
}

// TestConfigV2_Validate_LLMModelRequired tests that validation fails when LLM.Model is empty.
// Kills mutant: removing the model check would make this test fail.
func TestConfigV2_Validate_LLMModelRequired(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider: "ollama",
			Model:    "", // Empty model should fail
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "empty LLM model should fail validation")
	assert.Contains(t, err.Error(), "model", "error should mention model field")
	assert.Contains(t, err.Error(), "required", "error should indicate field is required")
}

// TestConfigV2_Validate_LLMFieldRanges tests validation of numeric field ranges.
// Kills mutants: removing range checks would make these tests fail.
func TestConfigV2_Validate_LLMFieldRanges(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConfigV2
		wantErr string
	}{
		{
			name: "temperature too low",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: -0.1,
					MaxTokens:   4096,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "temperature",
		},
		{
			name: "temperature too high",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 2.1,
					MaxTokens:   4096,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "temperature",
		},
		{
			name: "max_tokens zero",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   0,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "max_tokens",
		},
		{
			name: "max_tokens negative",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   -100,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "max_tokens",
		},
		{
			name: "timeout zero",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   4096,
					Timeout:     0,
				},
			},
			wantErr: "timeout",
		},
		{
			name: "timeout negative",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   4096,
					Timeout:     -5 * time.Second,
				},
			},
			wantErr: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			require.Error(t, err, "validation should fail for %s", tt.name)
			assert.Contains(t, err.Error(), tt.wantErr, "error should mention %s", tt.wantErr)
		})
	}
}

// TestConfigV2_Validate_LLMValidRanges tests that valid values pass validation.
func TestConfigV2_Validate_LLMValidRanges(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider:    "ollama",
			Model:       "qwen",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     5 * time.Minute,
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "valid config should pass validation")
}
