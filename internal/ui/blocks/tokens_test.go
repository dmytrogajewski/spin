package blocks

import (
	"testing"
)

func TestGetTagColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		blockType BlockType
		want      Color
	}{
		{"Execute", BlockTypeExecute, ColorBlue},
		{"Plan", BlockTypePlan, ColorMagenta},
		{"Read", BlockTypeRead, ColorCyan},
		{"Grep", BlockTypeGrep, ColorYellow},
		{"ApplyPatch", BlockTypeApplyPatch, ColorGreen},
		{"Summary", BlockTypeSummary, ColorCyan},
		{"Testing", BlockTypeTesting, ColorBlue},
		{"Notice", BlockTypeNotice, ColorMuted},
		{"Error", BlockTypeError, ColorRed},
		{"Unknown", BlockType("UNKNOWN"), ColorMuted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := GetTagColor(tt.blockType)
			if got != tt.want {
				t.Errorf("GetTagColor(%v) = %v, want %v", tt.blockType, got, tt.want)
			}
		})
	}
}

func TestColorString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		color Color
		want  string
	}{
		{"Reset", ColorReset, "\x1b[0m"},
		{"Bold", ColorBold, "\x1b[1m"},
		{"Blue", ColorBlue, "\x1b[38;5;39m"},
		{"Red", ColorRed, "\x1b[38;5;203m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.color.String()
			if got != tt.want {
				t.Errorf("Color.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSpacingConstants(t *testing.T) {
	t.Parallel()

	if S0 != 0 {
		t.Errorf("S0 = %d, want 0", S0)
	}

	if S1 != 1 {
		t.Errorf("S1 = %d, want 1", S1)
	}

	if S2 != 2 {
		t.Errorf("S2 = %d, want 2", S2)
	}

	if S4 != 4 {
		t.Errorf("S4 = %d, want 4", S4)
	}
}
