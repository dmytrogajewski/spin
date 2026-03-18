package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestParsePrompt_FromArgs(t *testing.T) {
	t.Parallel()

	prompt, err := parsePrompt([]string{"hello world"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prompt != "hello world" {
		t.Errorf("got %q, want %q", prompt, "hello world")
	}
}

func TestParsePrompt_FromReader(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("prompt from stdin")

	prompt, err := parsePrompt(nil, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prompt != "prompt from stdin" {
		t.Errorf("got %q, want %q", prompt, "prompt from stdin")
	}
}

func TestParsePrompt_EmptyReader(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("")

	_, err := parsePrompt(nil, r)
	if err == nil {
		t.Fatal("expected error for empty input")
	}

	if !errors.Is(err, ErrNoPromptProvidedUseCommandLine) {
		t.Errorf("got %v, want %v", err, ErrNoPromptProvidedUseCommandLine)
	}
}

func TestParsePrompt_ArgsOverrideReader(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("should be ignored")

	prompt, err := parsePrompt([]string{"from args"}, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prompt != "from args" {
		t.Errorf("got %q, want %q", prompt, "from args")
	}
}

func TestCreateExecUI_WithBuffer(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	ui, err := createExecUI(&buf)
	if err != nil {
		t.Fatalf("createExecUI with buffer: %v", err)
	}

	if ui == nil {
		t.Fatal("expected non-nil UI")
	}
}

func TestNewExecCmd_Flags(t *testing.T) {
	t.Parallel()

	cmd := newExecCmd()

	flags := []string{"auto-approve", "timeout", "format", "no-stream", "exit-on-error", "debug"}

	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("flag --%s not found", flagName)
		}
	}
}

func TestNewExecCmd_Help(t *testing.T) {
	t.Parallel()

	cmd := newExecCmd()
	cmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "exec") {
		t.Error("help should mention 'exec'")
	}

	if !strings.Contains(output, "non-interactive") && !strings.Contains(output, "Non-interactive") {
		t.Errorf("help should mention non-interactive, got: %s", output)
	}
}

func TestParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		wantErr bool
	}{
		{"5m", false},
		{"1h", false},
		{"30s", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			_, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestMockTTY(t *testing.T) {
	t.Parallel()

	m := &mockTTY{width: 120, height: 30}

	if err := m.Enter(); err != nil {
		t.Errorf("Enter() = %v", err)
	}

	if err := m.Exit(); err != nil {
		t.Errorf("Exit() = %v", err)
	}

	w, h := m.Size()
	if w != 120 || h != 30 {
		t.Errorf("Size() = (%d, %d), want (120, 30)", w, h)
	}

	// OnResize should not panic.
	m.OnResize(func(_, _ int) {})
}

func TestResolveSessionID_WithStorage(t *testing.T) {
	t.Parallel()

	// When storage is nil, should generate a prefix-based ID.
	id := resolveSessionID(nil, "/tmp", "test")
	if !strings.HasPrefix(id, "test-") {
		t.Errorf("expected prefix 'test-', got: %s", id)
	}
}
