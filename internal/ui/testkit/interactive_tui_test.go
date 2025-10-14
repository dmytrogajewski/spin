package testkit

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/prompt"
)

// InteractiveTUITest provides a comprehensive testing framework for interactive TUI flows.
//
// This framework allows testing complete user interaction flows:
// - User types characters
// - Prompt updates in real-time
// - Status bar updates
// - Output scrolls correctly
// - Sticky bottom stays fixed
//
// Usage:
//
//	test := NewInteractiveTUITest(t)
//	test.TypeString("hello world")
//	test.PressEnter()
//	test.AssertPromptShows("hello world")
//	test.AssertOutputContains("hello world")
type InteractiveTUITest struct {
	t              *testing.T
	out            *bytes.Buffer
	fakeTTY        *FakeTTY
	keyboard       *FakeKeyboard
	promptModel    *prompt.Model
	promptRenderer *prompt.Renderer
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewInteractiveTUITest creates a new interactive TUI test harness.
func NewInteractiveTUITest(t *testing.T) *InteractiveTUITest {
	t.Helper()

	out := &bytes.Buffer{}
	fakeTTY := NewFakeTTY(80, 24)
	keyboard := NewFakeKeyboard()
	promptModel := prompt.NewModel(100)
	promptRenderer := prompt.NewRenderer(out, 80, "> ")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	return &InteractiveTUITest{
		t:              t,
		out:            out,
		fakeTTY:        fakeTTY,
		keyboard:       keyboard,
		promptModel:    promptModel,
		promptRenderer: promptRenderer,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Cleanup should be called in defer to clean up resources.
func (it *InteractiveTUITest) Cleanup() {
	it.cancel()
	it.keyboard.Close()
}

// TypeString simulates typing a string.
func (it *InteractiveTUITest) TypeString(s string) {
	it.t.Helper()
	it.keyboard.InjectString(s)
	time.Sleep(10 * time.Millisecond) // Simulate human typing speed
}

// PressEnter simulates pressing Enter key.
func (it *InteractiveTUITest) PressEnter() {
	it.t.Helper()
	it.keyboard.InjectEnter()
	time.Sleep(20 * time.Millisecond) // Wait for processing
}

// PressBackspace simulates pressing Backspace.
func (it *InteractiveTUITest) PressBackspace() {
	it.t.Helper()
	it.keyboard.InjectBackspace()
	time.Sleep(10 * time.Millisecond)
}

// PressCtrlC simulates pressing Ctrl-C.
func (it *InteractiveTUITest) PressCtrlC() {
	it.t.Helper()
	it.keyboard.InjectCtrlC()
	time.Sleep(20 * time.Millisecond)
}

// PressCtrlD simulates pressing Ctrl-D.
func (it *InteractiveTUITest) PressCtrlD() {
	it.t.Helper()
	it.keyboard.InjectCtrlD()
	time.Sleep(20 * time.Millisecond)
}

// GetOutput returns all output written so far.
func (it *InteractiveTUITest) GetOutput() string {
	return it.out.String()
}

// GetOutputBuffer returns the output buffer for passing to adapters.
func (it *InteractiveTUITest) GetOutputBuffer() *bytes.Buffer {
	return it.out
}

// ClearOutput clears the output buffer.
func (it *InteractiveTUITest) ClearOutput() {
	it.out.Reset()
}

// AssertPromptShows verifies the prompt displays the expected text.
func (it *InteractiveTUITest) AssertPromptShows(expected string) {
	it.t.Helper()

	output := it.out.String()

	// Prompt format: "> text"
	expectedPrompt := "> " + expected

	if !strings.Contains(output, expectedPrompt) {
		it.t.Errorf("Expected prompt to show %q, but output is:\n%s", expectedPrompt, output)
	}
}

// AssertOutputContains verifies output contains expected text.
func (it *InteractiveTUITest) AssertOutputContains(expected string) {
	it.t.Helper()

	output := it.out.String()

	if !strings.Contains(output, expected) {
		it.t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
	}
}

// AssertOutputDoesNotContain verifies output does NOT contain text.
func (it *InteractiveTUITest) AssertOutputDoesNotContain(unexpected string) {
	it.t.Helper()

	output := it.out.String()

	if strings.Contains(output, unexpected) {
		it.t.Errorf("Expected output to NOT contain %q, but it does:\n%s", unexpected, output)
	}
}

// AssertStickyBottomPresent verifies sticky bottom area is rendered.
func (it *InteractiveTUITest) AssertStickyBottomPresent() {
	it.t.Helper()

	output := it.out.String()

	// Check for ANSI positioning sequences that indicate sticky area
	// \x1b[s = save cursor
	// \x1b[line;colH = absolute positioning
	// \x1b[u = restore cursor

	if !strings.Contains(output, "\x1b[s") {
		it.t.Error("Expected sticky bottom (save cursor sequence not found)")
	}

	if !strings.Contains(output, "\x1b[u") {
		it.t.Error("Expected sticky bottom (restore cursor sequence not found)")
	}
}

// AssertStatusBarVisible verifies status bar is visible in output.
func (it *InteractiveTUITest) AssertStatusBarVisible() {
	it.t.Helper()

	output := it.out.String()

	// Status bar should contain indicator like [●] or percentage
	if !strings.Contains(output, "[●]") && !strings.Contains(output, "%") {
		it.t.Error("Expected status bar to be visible")
	}
}

// AssertPromptAtBottom verifies prompt is rendered at bottom line.
func (it *InteractiveTUITest) AssertPromptAtBottom() {
	it.t.Helper()

	output := it.out.String()

	// Should contain ANSI sequence to position at terminal height
	// Format: \x1b[24;1H for line 24 (in 24-line terminal)
	bottomLine := fmt.Sprintf("\x1b[%d;1H", it.fakeTTY.height)

	if !strings.Contains(output, bottomLine) {
		it.t.Errorf("Expected prompt at bottom line %d, output:\n%s", it.fakeTTY.height, output)
	}
}

// WaitForOutput waits for output to contain expected text.
func (it *InteractiveTUITest) WaitForOutput(expected string, timeout time.Duration) bool {
	it.t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(it.out.String(), expected) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// DumpOutput dumps output for debugging (only on test failure).
func (it *InteractiveTUITest) DumpOutput() {
	it.t.Helper()

	if it.t.Failed() {
		it.t.Logf("=== OUTPUT DUMP ===\n%s\n=== END ===", it.out.String())
	}
}
