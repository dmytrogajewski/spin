package agentsmd

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if !cfg.Enabled {
		t.Error("DefaultConfig().Enabled = false, want true")
	}

	if cfg.Path != "" {
		t.Errorf("DefaultConfig().Path = %v, want empty string", cfg.Path)
	}

	if cfg.MaxSize != 100*1024 {
		t.Errorf("DefaultConfig().MaxSize = %d, want %d", cfg.MaxSize, 100*1024)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Enabled: true,
				MaxSize: 100 * 1024,
			},
			wantErr: false,
		},
		{
			name: "negative maxsize becomes zero",
			config: &Config{
				Enabled: true,
				MaxSize: -100,
			},
			wantErr: false,
		},
		{
			name: "zero maxsize is valid",
			config: &Config{
				Enabled: true,
				MaxSize: 0,
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Negative MaxSize should become 0 after validation.
			if tt.config.MaxSize < 0 {
				t.Errorf("MaxSize = %d after Validate(), want >= 0", tt.config.MaxSize)
			}
		})
	}
}
