package theme

import (
	"testing"
)

func TestDarkTheme(t *testing.T) {
	theme := NewDarkTheme()

	// Test neutral colors
	if theme.Fg() == "" {
		t.Error("Dark theme Fg should not be empty")
	}
	if theme.Bg() == "" {
		t.Error("Dark theme Bg should not be empty")
	}
	if theme.Muted() == "" {
		t.Error("Dark theme Muted should not be empty")
	}
	if theme.Border() == "" {
		t.Error("Dark theme Border should not be empty")
	}
	if theme.Shadow() == "" {
		t.Error("Dark theme Shadow should not be empty")
	}

	// Test accent colors
	if theme.Blue() == "" {
		t.Error("Dark theme Blue should not be empty")
	}
	if theme.Green() == "" {
		t.Error("Dark theme Green should not be empty")
	}
	if theme.Yellow() == "" {
		t.Error("Dark theme Yellow should not be empty")
	}
	if theme.Red() == "" {
		t.Error("Dark theme Red should not be empty")
	}
	if theme.Magenta() == "" {
		t.Error("Dark theme Magenta should not be empty")
	}
	if theme.Cyan() == "" {
		t.Error("Dark theme Cyan should not be empty")
	}
}

func TestLightTheme(t *testing.T) {
	theme := NewLightTheme()

	// Test neutral colors
	if theme.Fg() == "" {
		t.Error("Light theme Fg should not be empty")
	}
	if theme.Bg() == "" {
		t.Error("Light theme Bg should not be empty")
	}
	if theme.Muted() == "" {
		t.Error("Light theme Muted should not be empty")
	}
	if theme.Border() == "" {
		t.Error("Light theme Border should not be empty")
	}
	if theme.Shadow() == "" {
		t.Error("Light theme Shadow should not be empty")
	}

	// Test accent colors
	if theme.Blue() == "" {
		t.Error("Light theme Blue should not be empty")
	}
	if theme.Green() == "" {
		t.Error("Light theme Green should not be empty")
	}
	if theme.Yellow() == "" {
		t.Error("Light theme Yellow should not be empty")
	}
	if theme.Red() == "" {
		t.Error("Light theme Red should not be empty")
	}
	if theme.Magenta() == "" {
		t.Error("Light theme Magenta should not be empty")
	}
	if theme.Cyan() == "" {
		t.Error("Light theme Cyan should not be empty")
	}
}

func TestEightColorTheme(t *testing.T) {
	theme := NewEightColorTheme()

	// Test neutral colors fallback
	if theme.Fg() == "" {
		t.Error("8-color theme Fg should not be empty")
	}
	if theme.Bg() == "" {
		t.Error("8-color theme Bg should not be empty")
	}
	if theme.Muted() == "" {
		t.Error("8-color theme Muted should not be empty")
	}
	if theme.Border() == "" {
		t.Error("8-color theme Border should not be empty")
	}

	// Test accent colors fallback
	if theme.Blue() == "" {
		t.Error("8-color theme Blue should not be empty")
	}
	if theme.Green() == "" {
		t.Error("8-color theme Green should not be empty")
	}
	if theme.Yellow() == "" {
		t.Error("8-color theme Yellow should not be empty")
	}
	if theme.Red() == "" {
		t.Error("8-color theme Red should not be empty")
	}
	if theme.Magenta() == "" {
		t.Error("8-color theme Magenta should not be empty")
	}
	if theme.Cyan() == "" {
		t.Error("8-color theme Cyan should not be empty")
	}
}

func TestHexToANSI256(t *testing.T) {
	tests := []struct {
		name string
		hex  string
		min  int
		max  int
	}{
		{"Black", "#000000", 16, 16},
		{"White", "#ffffff", 231, 231},
		{"Dark blue-gray", "#0b0e12", 232, 235},     // Grayscale range
		{"Light blue", "#5aa6ff", 31, 75},           // Blue range
		{"Green", "#57d98d", 42, 121},               // Green range
		{"Yellow", "#f5c156", 178, 221},             // Yellow range
		{"Red", "#ff6b6b", 167, 210},                // Red range
		{"Dark gray", "#2d3640", 16, 239},           // Dark colors (can be cube or grayscale)
		{"Light gray", "#9aa4b2", 102, 250},         // Medium gray-blue (can be cube or grayscale)
		{"Magenta", "#d08bff", 135, 177},            // Magenta range
		{"Cyan", "#7adcf3", 80, 123},                // Cyan range
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hexToANSI256(tt.hex)
			// Check if result is in expected range
			if result < tt.min || result > tt.max {
				t.Errorf("hexToANSI256(%s) = %d, want range [%d-%d]", tt.hex, result, tt.min, tt.max)
			}
		})
	}
}

func TestDetectTerminalCapabilities(t *testing.T) {
	// Test that detection doesn't panic
	capability := DetectTerminalCapabilities()

	// Should return one of the known capabilities
	validCapabilities := []TerminalCapability{
		TerminalCapability8Color,
		TerminalCapability256Color,
		TerminalCapabilityTrueColor,
	}

	found := false
	for _, valid := range validCapabilities {
		if capability == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("DetectTerminalCapabilities() returned unknown capability: %d", capability)
	}
}

func TestNewTheme(t *testing.T) {
	tests := []struct {
		name       string
		themeName  string
		capability TerminalCapability
		wantType   string
	}{
		{"Dark 256", "dark", TerminalCapability256Color, "dark"},
		{"Light 256", "light", TerminalCapability256Color, "light"},
		{"Dark 8-color", "dark", TerminalCapability8Color, "8color"},
		{"Light 8-color", "light", TerminalCapability8Color, "8color"},
		{"Unknown defaults to dark", "unknown", TerminalCapability256Color, "dark"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := NewTheme(tt.themeName, tt.capability)
			if theme == nil {
				t.Fatal("NewTheme returned nil")
			}
			// Just verify it returns a valid theme
			if theme.Fg() == "" {
				t.Error("Theme Fg should not be empty")
			}
		})
	}
}

func TestGetThemeFromEnv(t *testing.T) {
	// Test that it returns a valid theme
	theme := GetThemeFromEnv()
	if theme == nil {
		t.Fatal("GetThemeFromEnv returned nil")
	}
	if theme.Fg() == "" {
		t.Error("Theme from env should have valid colors")
	}
}
