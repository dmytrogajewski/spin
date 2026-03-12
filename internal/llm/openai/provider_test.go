package openai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewProvider tests provider creation and validation.
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test",
				Model:   "gpt-4",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing base URL",
			cfg: Config{
				APIKey:  "sk-test",
				Model:   "gpt-4",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "missing model",
			cfg: Config{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			cfg: Config{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test",
				Model:   "gpt-4",
				Timeout: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, p)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p)
				assert.Equal(t, tt.cfg.Model, p.model)
			}
		})
	}
}

// TestProvider_Capabilities tests that capabilities are correctly reported.
func TestProvider_Capabilities(t *testing.T) {
	p := &Provider{
		model: "gpt-4",
	}

	caps := p.Capabilities()
	assert.True(t, caps.Streaming, "should support streaming")
	assert.True(t, caps.FunctionCalling, "should support function calling")
}

// TestProvider_Name tests provider name.
func TestProvider_Name(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, "openai-compatible", p.Name())
}

// TestProvider_Close tests cleanup.
func TestProvider_Close(t *testing.T) {
	p := &Provider{}
	err := p.Close()
	assert.NoError(t, err)
}

// TestProvider_Models tests model listing.
func TestProvider_Models(t *testing.T) {
	t.Skip("Models() requires SDK API call - covered by integration tests")
}

// Note: Complete() and Stream() require real API calls or complex mocking
// These are tested in integration tests or with the old test suite approach
// For now, we verify the SDK integration compiles and basic methods work.
