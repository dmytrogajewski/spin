package core

import (
	"testing"
)

func TestConfig_MergeCycleDetection(t *testing.T) {
	tests := []struct {
		name  string
		base  *Config
		other *Config
		want  CycleDetectionConfig
	}{
		{
			name: "merge enabled flag",
			base: &Config{
				CycleDetection: CycleDetectionConfig{
					Enabled: false,
				},
			},
			other: &Config{
				CycleDetection: CycleDetectionConfig{
					Enabled: true,
				},
			},
			want: CycleDetectionConfig{
				Enabled: true,
			},
		},
		{
			name: "merge window size",
			base: &Config{
				CycleDetection: CycleDetectionConfig{
					WindowSize: 10,
				},
			},
			other: &Config{
				CycleDetection: CycleDetectionConfig{
					WindowSize: 20,
				},
			},
			want: CycleDetectionConfig{
				WindowSize: 20,
			},
		},
		{
			name: "merge similarity threshold",
			base: &Config{
				CycleDetection: CycleDetectionConfig{
					SimilarityThresh: 0.5,
				},
			},
			other: &Config{
				CycleDetection: CycleDetectionConfig{
					SimilarityThresh: 0.8,
				},
			},
			want: CycleDetectionConfig{
				SimilarityThresh: 0.8,
			},
		},
		{
			name: "merge tool repeat limit",
			base: &Config{
				CycleDetection: CycleDetectionConfig{
					ToolRepeatLimit: 3,
				},
			},
			other: &Config{
				CycleDetection: CycleDetectionConfig{
					ToolRepeatLimit: 5,
				},
			},
			want: CycleDetectionConfig{
				ToolRepeatLimit: 5,
			},
		},
		{
			name: "merge error repeat limit",
			base: &Config{
				CycleDetection: CycleDetectionConfig{
					ErrorRepeatLimit: 2,
				},
			},
			other: &Config{
				CycleDetection: CycleDetectionConfig{
					ErrorRepeatLimit: 4,
				},
			},
			want: CycleDetectionConfig{
				ErrorRepeatLimit: 4,
			},
		},
		{
			name: "merge all fields",
			base: &Config{
				CycleDetection: CycleDetectionConfig{
					Enabled:          false,
					WindowSize:       10,
					SimilarityThresh: 0.5,
					ToolRepeatLimit:  3,
					ErrorRepeatLimit: 2,
				},
			},
			other: &Config{
				CycleDetection: CycleDetectionConfig{
					Enabled:          true,
					WindowSize:       20,
					SimilarityThresh: 0.8,
					ToolRepeatLimit:  5,
					ErrorRepeatLimit: 4,
				},
			},
			want: CycleDetectionConfig{
				Enabled:          true,
				WindowSize:       20,
				SimilarityThresh: 0.8,
				ToolRepeatLimit:  5,
				ErrorRepeatLimit: 4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of base config
			config := &Config{
				CycleDetection: tt.base.CycleDetection,
			}

			// Call mergeCycleDetection
			config.mergeCycleDetection(tt.other)

			// Verify the result
			if config.CycleDetection.Enabled != tt.want.Enabled {
				t.Errorf("mergeCycleDetection() Enabled = %v, want %v", config.CycleDetection.Enabled, tt.want.Enabled)
			}
			if config.CycleDetection.WindowSize != tt.want.WindowSize {
				t.Errorf("mergeCycleDetection() WindowSize = %v, want %v", config.CycleDetection.WindowSize, tt.want.WindowSize)
			}
			if config.CycleDetection.SimilarityThresh != tt.want.SimilarityThresh {
				t.Errorf("mergeCycleDetection() SimilarityThresh = %v, want %v", config.CycleDetection.SimilarityThresh, tt.want.SimilarityThresh)
			}
			if config.CycleDetection.ToolRepeatLimit != tt.want.ToolRepeatLimit {
				t.Errorf("mergeCycleDetection() ToolRepeatLimit = %v, want %v", config.CycleDetection.ToolRepeatLimit, tt.want.ToolRepeatLimit)
			}
			if config.CycleDetection.ErrorRepeatLimit != tt.want.ErrorRepeatLimit {
				t.Errorf("mergeCycleDetection() ErrorRepeatLimit = %v, want %v", config.CycleDetection.ErrorRepeatLimit, tt.want.ErrorRepeatLimit)
			}
		})
	}
}

func TestConfig_MergeSliceFields(t *testing.T) {
	tests := []struct {
		name  string
		base  *Config
		other *Config
		want  []string
	}{
		{
			name: "merge allowed commands",
			base: &Config{
				AllowedCommands: []string{"ls", "pwd"},
			},
			other: &Config{
				AllowedCommands: []string{"cat", "grep"},
			},
			want: []string{"ls", "pwd", "cat", "grep"},
		},
		{
			name: "merge empty allowed commands",
			base: &Config{
				AllowedCommands: []string{},
			},
			other: &Config{
				AllowedCommands: []string{"ls", "cat"},
			},
			want: []string{"ls", "cat"},
		},
		{
			name: "merge with empty other",
			base: &Config{
				AllowedCommands: []string{"ls", "cat"},
			},
			other: &Config{
				AllowedCommands: []string{},
			},
			want: []string{"ls", "cat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of base config
			config := &Config{
				AllowedCommands: make([]string, len(tt.base.AllowedCommands)),
			}
			copy(config.AllowedCommands, tt.base.AllowedCommands)

			// Call mergeSliceFields
			config.mergeSliceFields(tt.other)

			// Verify allowed commands
			if len(config.AllowedCommands) != len(tt.want) {
				t.Errorf("mergeSliceFields() AllowedCommands length = %d, want %d", len(config.AllowedCommands), len(tt.want))
			} else {
				for i, cmd := range config.AllowedCommands {
					if cmd != tt.want[i] {
						t.Errorf("mergeSliceFields() AllowedCommands[%d] = %v, want %v", i, cmd, tt.want[i])
					}
				}
			}
		})
	}
}
