package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionCommand(t *testing.T) {
	cmd := newCompletionCmd()

	if !strings.HasPrefix(cmd.Use, "completion") {
		t.Errorf("Completion command Use = %s, should start with 'completion'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Completion command Short description should not be empty")
	}
}

func TestCompletionCommand_Bash(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.SetArgs([]string{"completion", "bash"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("completion bash failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "bash completion") {
		t.Error("Bash completion should contain 'bash completion'")
	}
}

func TestCompletionCommand_Zsh(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.SetArgs([]string{"completion", "zsh"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("completion zsh failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "zsh completion") {
		t.Error("Zsh completion should contain 'zsh completion'")
	}
}

func TestCompletionCommand_Fish(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.SetArgs([]string{"completion", "fish"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("completion fish failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "fish completion") {
		t.Error("Fish completion should contain 'fish completion'")
	}
}

func TestCompletionCommand_Powershell(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.SetArgs([]string{"completion", "powershell"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("completion powershell failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "powershell completion") {
		t.Error("Powershell completion should contain 'powershell completion'")
	}
}

func TestCompletionCommand_InvalidShell(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.SetArgs([]string{"completion", "invalid"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	err := rootCmd.Execute()
	if err == nil {
		t.Error("completion with invalid shell should return error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "unsupported shell") {
		t.Errorf("Error should mention unsupported shell, got: %s", errMsg)
	}
}

func TestCompletionCommand_NoArgs(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.SetArgs([]string{"completion"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	err := rootCmd.Execute()
	if err == nil {
		t.Error("completion with no args should return error")
	}
}
