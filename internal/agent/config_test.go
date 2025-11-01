package agent

import "testing"

// Test ACE config is present in Config struct
func TestConfig_ACE_Field(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.ACE.Enabled {
		t.Error("ACE should be enabled by default")
	}
}

// Test ACE config has default playbook path
func TestConfig_ACE_PlaybookPath(t *testing.T) {
	cfg := DefaultConfig()

	expected := "~/.spin/ace/playbooks/default.json"
	if cfg.ACE.PlaybookPath != expected {
		t.Errorf("ACE.PlaybookPath = %q, want %q", cfg.ACE.PlaybookPath, expected)
	}
}

// Test ACE config has all required fields with correct defaults
func TestConfig_ACE_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"Enabled", cfg.ACE.Enabled, true},
		{"PlaybookPath", cfg.ACE.PlaybookPath, "~/.spin/ace/playbooks/default.json"},
		{"TrajectoryPath", cfg.ACE.TrajectoryPath, "~/.spin/ace/trajectories/"},
		{"Retrieval.TopK", cfg.ACE.Retrieval.TopK, 100},
		{"Retrieval.MinScore", cfg.ACE.Retrieval.MinScore, 0.3},
		{"ItemizedLearning.Enabled", cfg.ACE.ItemizedLearning.Enabled, true},
		{"ItemizedLearning.ParseFeedback", cfg.ACE.ItemizedLearning.ParseFeedback, true},
		{"ItemizedLearning.UpdateAsync", cfg.ACE.ItemizedLearning.UpdateAsync, true},
		{"Generation.Enabled", cfg.ACE.Generation.Enabled, true},
		{"Generation.AutoReflect", cfg.ACE.Generation.AutoReflect, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// Test ACE config validation
func TestConfig_ACE_Validation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid TopK (zero)",
			modify: func(c *Config) {
				c.ACE.Retrieval.TopK = 0
			},
			wantErr: true,
			errMsg:  "top_k must be > 0",
		},
		{
			name: "invalid TopK (negative)",
			modify: func(c *Config) {
				c.ACE.Retrieval.TopK = -1
			},
			wantErr: true,
			errMsg:  "top_k must be > 0",
		},
		{
			name: "invalid MinScore (negative)",
			modify: func(c *Config) {
				c.ACE.Retrieval.MinScore = -0.1
			},
			wantErr: true,
			errMsg:  "min_score must be between 0 and 1",
		},
		{
			name: "invalid MinScore (too high)",
			modify: func(c *Config) {
				c.ACE.Retrieval.MinScore = 1.5
			},
			wantErr: true,
			errMsg:  "min_score must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Provider = "openai"
			cfg.Model = "gpt-4"
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
