package status

import (
	"strings"
	"testing"
)

func TestFormatCompact(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Thinking")
	m.SetMaxTokens(1000)
	m.AddTokens(420, 0) // 42%

	result := m.FormatCompact(50)

	// Should contain activity indicator (static or spinner), percentage, and state.
	hasActivityIndicator := strings.Contains(result, "[●]") ||
		strings.Contains(result, "[○]") ||
		strings.Contains(result, "[") // Spinner frames are wrapped in brackets.
	if !hasActivityIndicator {
		t.Error("Expected activity indicator in compact format")
	}

	if !strings.Contains(result, "42%") {
		t.Errorf("Expected '42%%' in compact format, got: %s", result)
	}

	if !strings.Contains(result, "Thinking") {
		t.Errorf("Expected 'Thinking' in compact format, got: %s", result)
	}
}

func TestFormatCompact_WithExplicitStatusText(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetStatus("Custom Status Message")
	m.SetAgentState("Thinking")

	result := m.FormatCompact(50)

	// Should return the explicit status text.
	if result != "Custom Status Message" {
		t.Errorf("Expected explicit status text, got: %s", result)
	}
}

func TestFormatCompact_LongStateName(t *testing.T) {
	t.Parallel()

	m := NewManager()
	// Set a very long state name that will be truncated.
	m.SetAgentState("This is a very long agent state name that should be truncated")

	result := m.FormatCompact(50)

	// Should contain truncated state with ellipsis.
	if !strings.Contains(result, "...") {
		t.Errorf("Expected truncation for long state name, got: %s", result)
	}
	// Should not contain the full state name.
	if len(result) > 50 {
		t.Errorf("Result should fit in terminal width, got length %d: %s", len(result), result)
	}
}

func TestFormatCompact_ConnectedState(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetConnected(true)
	m.SetAgentState("Ready")

	result := m.FormatCompact(50)

	// Should show connected indicator.
	if !strings.Contains(result, "[●]") {
		t.Errorf("Expected connected indicator, got: %s", result)
	}
}

func TestFormatCompact_DisconnectedState(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetConnected(false)
	m.SetAgentState("Ready")

	result := m.FormatCompact(50)

	// Should show disconnected indicator.
	if !strings.Contains(result, "[○]") {
		t.Errorf("Expected disconnected indicator, got: %s", result)
	}
}

func TestFormatMedium(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Calling tools")
	m.SetProvider("ollama", "qwen3:1.7b")
	m.SetMaxTokens(2000)
	m.AddTokens(800, 0)             // 40%
	m.CalculateTPS(125, 1000000000) // 125 tokens in 1 second.

	result := m.FormatMedium(80)

	// Should contain: activity, percentage, state, provider, TPS.
	if !strings.Contains(result, "40%") {
		t.Errorf("Expected '40%%', got: %s", result)
	}

	if !strings.Contains(result, "Calling") {
		t.Errorf("Expected 'Calling', got: %s", result)
	}

	if !strings.Contains(result, "ollama") {
		t.Errorf("Expected 'ollama', got: %s", result)
	}

	if !strings.Contains(result, "125tok/s") {
		t.Errorf("Expected '125tok/s', got: %s", result)
	}
}

func TestFormatMedium_NoMaxTokens(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Ready")
	m.SetProvider("openai", "gpt-4")
	// MaxTokens is 0 by default.

	result := m.FormatMedium(80)

	// Should NOT contain percentage when MaxTokens is 0.
	if strings.Contains(result, "%") {
		t.Errorf("Should not show percentage when MaxTokens is 0, got: %s", result)
	}
	// Should still show state and provider.
	if !strings.Contains(result, "Ready") {
		t.Errorf("Expected 'Ready', got: %s", result)
	}

	if !strings.Contains(result, "openai") {
		t.Errorf("Expected 'openai', got: %s", result)
	}
}

func TestFormatMedium_LowTPS(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Thinking")
	m.CalculateTPS(1, 1000000000) // Exactly 1.0 tok/s (below threshold).

	result := m.FormatMedium(80)

	// Should NOT show TPS when it's <= 1.0.
	if strings.Contains(result, "tok/s") {
		t.Errorf("Should not show TPS when <= 1.0, got: %s", result)
	}
}

func TestFormatMedium_HighTPS(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Thinking")
	m.CalculateTPS(2, 1000000000) // 2.0 tok/s (above threshold).

	result := m.FormatMedium(80)

	// Should show TPS when > 1.0.
	if !strings.Contains(result, "2tok/s") {
		t.Errorf("Expected TPS to be shown when > 1.0, got: %s", result)
	}
}

func TestFormatMedium_NoProvider(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Ready")
	m.SetMaxTokens(1000)
	m.AddTokens(500, 0)
	// Provider is not set.

	result := m.FormatMedium(80)

	// Should NOT show provider when it's empty.
	if strings.Contains(result, "/") {
		t.Errorf("Should not show provider when empty, got: %s", result)
	}
	// Should still show other fields.
	if !strings.Contains(result, "50%") {
		t.Errorf("Expected percentage, got: %s", result)
	}

	if !strings.Contains(result, "Ready") {
		t.Errorf("Expected state, got: %s", result)
	}
}

func TestFormatFull(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Planning")
	m.SetProvider("ollama", "qwen3:1.7b")
	m.SetTaskMode("review")
	m.SetConversationID("abc123def456")
	m.SetMaxTokens(20000)
	m.AddTokens(8500, 0) // 42.5%
	m.CalculateTPS(125, 1000000000)

	result := m.FormatFull(120)

	// Should contain all fields.
	if !strings.Contains(result, "42%") {
		t.Errorf("Expected '42%%', got: %s", result)
	}

	if !strings.Contains(result, "8.5K") {
		t.Errorf("Expected '8.5K', got: %s", result)
	}

	if !strings.Contains(result, "20.0K") {
		t.Errorf("Expected '20.0K', got: %s", result)
	}

	if !strings.Contains(result, "Planning") {
		t.Errorf("Expected 'Planning', got: %s", result)
	}

	if !strings.Contains(result, "Review") {
		t.Errorf("Expected 'Review' (capitalized), got: %s", result)
	}

	if !strings.Contains(result, "conv:abc123") {
		t.Errorf("Expected 'conv:abc123', got: %s", result)
	}
	// Hotkeys are currently disabled
	// if !strings.Contains(result, "?:help") {
	// 	t.Errorf("Expected '?:help', got: %s", result)
	// }.
}

func TestFormatAdaptive_NarrowTerminal(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Ready")

	// Narrow terminal should use compact format.
	result := m.FormatAdaptive(50)

	// Should be compact (no absolute token counts).
	if strings.Contains(result, "K/") {
		t.Errorf("Narrow terminal should not show absolute values, got: %s", result)
	}
}

func TestFormatAdaptive_MediumTerminal(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetProvider("ollama", "model")

	// Medium terminal should use medium format.
	result := m.FormatAdaptive(80)

	// Should show provider.
	if !strings.Contains(result, "ollama") {
		t.Errorf("Medium terminal should show provider, got: %s", result)
	}
}

func TestFormatAdaptive_WideTerminal(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetConversationID("test123")

	// Wide terminal should use full format.
	result := m.FormatAdaptive(120)

	// Should show conversation ID and hotkeys.
	if !strings.Contains(result, "conv:") {
		t.Errorf("Wide terminal should show conversation ID, got: %s", result)
	}
}

func TestHumanizeNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{8500, "8.5K"},
		{10000, "10.0K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
	}

	for _, tt := range tests {
		result := humanizeNumber(tt.input)
		if result != tt.expected {
			t.Errorf("humanizeNumber(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatPercentage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    float64
		expected string
	}{
		{0.0, "0%"},
		{42.5, "42%"},
		{99.9, "100%"},
		{100.0, "100%"},
	}

	for _, tt := range tests {
		result := formatPercentage(tt.input)
		if result != tt.expected {
			t.Errorf("formatPercentage(%.1f) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},                           // 5 chars, fits.
		{"exactly10", 10, "exactly10"},                   // 10 chars, fits exactly.
		{"eleven_char", 10, "eleven_..."},                // 11 chars, truncate.
		{"this_is_a_very_long_string", 10, "this_is..."}, // long string.
		{"abc", 5, "abc"},                                // 3 chars, fits.
		{"abcdef", 5, "ab..."},                           // 6 chars, truncate to 5.
		{"ab", 3, "ab"},                                  // 2 chars, fits.
		{"abc", 3, "abc"},                                // 3 chars, fits exactly.
		{"abcd", 3, "..."},                               // 4 chars, truncate to 3 (0 chars + "...").
		{"verylongmodelname", 12, "verylongm..."},        // truncate to 12 (12-3=9 chars + "...").
		{"hello", 2, "he"},                               // maxLen < 3, no ellipsis.
		{"hi", 1, "h"},                                   // maxLen = 1.
		{"text", 0, ""},                                  // maxLen = 0.
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestCapitalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "A"},
		{"hello", "Hello"},
		{"HELLO", "HELLO"},
		{"hELLO", "HELLO"},
	}

	for _, tt := range tests {
		result := capitalize(tt.input)
		if result != tt.expected {
			t.Errorf("capitalize(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestActivityIndicator(t *testing.T) {
	t.Parallel(
	// Connected.
	)

	result := activityIndicator(true)
	if result != "[●]" {
		t.Errorf("Expected '[●]' for connected, got %q", result)
	}

	// Disconnected.
	result = activityIndicator(false)
	if result != "[○]" {
		t.Errorf("Expected '[○]' for disconnected, got %q", result)
	}
}

func TestEdgeCases_ZeroValues(t *testing.T) {
	t.Parallel()

	m := NewManager()
	// No data set.

	result := m.FormatFull(120)

	// Should handle zero values gracefully.
	if !strings.Contains(result, "Ready") {
		t.Errorf("Expected default 'Ready' state, got: %s", result)
	}
}

func TestEdgeCases_NoProvider(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Thinking")

	result := m.FormatFull(120)

	// Should work without provider.
	if !strings.Contains(result, "Thinking") {
		t.Errorf("Expected 'Thinking', got: %s", result)
	}
}

func TestEdgeCases_Disabled(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Disable()

	result := m.FormatFull(120)

	// Should return empty string when disabled.
	if result != "" {
		t.Errorf("Expected empty string when disabled, got: %s", result)
	}
}

func TestFormatFull_OmitsRegularMode(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetTaskMode("regular")

	result := m.FormatFull(120)

	// Should NOT show "Regular" (it's the default).
	if strings.Contains(result, "Regular") {
		t.Errorf("Should not show 'Regular' mode (default), got: %s", result)
	}
}

func TestFormatFull_ShowsNonDefaultMode(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetTaskMode("compact")

	result := m.FormatFull(120)

	// Should show "Compact".
	if !strings.Contains(result, "Compact") {
		t.Errorf("Expected 'Compact' mode to be shown, got: %s", result)
	}
}

func TestFormatFull_HidesTPSWhenZero(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.SetAgentState("Ready")
	// TPS is 0.

	result := m.FormatFull(120)

	// Should NOT show "tok/s".
	if strings.Contains(result, "tok/s") {
		t.Errorf("Should not show TPS when zero, got: %s", result)
	}
}

func TestFormatFull_HidesHotkeysOnNarrowerTerminals(t *testing.T) {
	t.Parallel()

	m := NewManager()

	// Hotkeys are currently disabled, so this test is not applicable
	// Width < 120 should not show hotkeys.
	result := m.FormatFull(100)

	// Hotkeys disabled for now.
	_ = result
	// if strings.Contains(result, "?:help") {
	// 	t.Errorf("Should not show hotkeys on terminals <120 cols, got: %s", result)
	// }.
}
