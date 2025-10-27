package main

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  bool
		wantName string
		wantArgs []string
	}{
		{
			name:     "slash command with no args",
			input:    "/mode",
			wantCmd:  true,
			wantName: "/mode",
			wantArgs: []string{},
		},
		{
			name:     "slash command with one arg",
			input:    "/mode review",
			wantCmd:  true,
			wantName: "/mode",
			wantArgs: []string{"review"},
		},
		{
			name:     "slash command with multiple args",
			input:    "/mode review test",
			wantCmd:  true,
			wantName: "/mode",
			wantArgs: []string{"review", "test"},
		},
		{
			name:     "slash command with leading/trailing whitespace",
			input:    "  /help  ",
			wantCmd:  true,
			wantName: "/help",
			wantArgs: []string{},
		},
		{
			name:    "regular message",
			input:   "Write a test for the auth module",
			wantCmd: false,
		},
		{
			name:    "message with slash in middle (not a command)",
			input:   "Use the /api endpoint to fetch data",
			wantCmd: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantCmd: false,
		},
		{
			name:    "whitespace only",
			input:   "   \t  ",
			wantCmd: false,
		},
		{
			name:    "just slash",
			input:   "/",
			wantCmd: false,
		},
		{
			name:    "slash with whitespace only",
			input:   "/  ",
			wantCmd: false,
		},
		{
			name:     "help command",
			input:    "/help",
			wantCmd:  true,
			wantName: "/help",
			wantArgs: []string{},
		},
		{
			name:     "exit command",
			input:    "/exit",
			wantCmd:  true,
			wantName: "/exit",
			wantArgs: []string{},
		},
		{
			name:     "quit command",
			input:    "/quit",
			wantCmd:  true,
			wantName: "/quit",
			wantArgs: []string{},
		},
		{
			name:     "case insensitive command",
			input:    "/MODE REVIEW",
			wantCmd:  true,
			wantName: "/mode",
			wantArgs: []string{"review"},
		},
		{
			name:     "unknown command",
			input:    "/unknown",
			wantCmd:  true,
			wantName: "/unknown",
			wantArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommand(tt.input)

			if got.isCommand != tt.wantCmd {
				t.Errorf("isCommand = %v, want %v", got.isCommand, tt.wantCmd)
			}

			if !tt.wantCmd {
				// For non-commands, just verify rawInput is preserved
				if got.rawInput != tt.input {
					t.Errorf("rawInput = %q, want %q", got.rawInput, tt.input)
				}
				return
			}

			// For commands, verify all fields
			if got.command != tt.wantName {
				t.Errorf("command = %q, want %q", got.command, tt.wantName)
			}

			if len(got.args) != len(tt.wantArgs) {
				t.Errorf("args length = %d, want %d; args = %v, want = %v",
					len(got.args), len(tt.wantArgs), got.args, tt.wantArgs)
				return
			}

			for i, arg := range got.args {
				// Args should be lowercase
				wantArg := tt.wantArgs[i]
				if arg != wantArg {
					t.Errorf("args[%d] = %q, want %q", i, arg, wantArg)
				}
			}

			if got.rawInput != tt.input {
				t.Errorf("rawInput = %q, want %q", got.rawInput, tt.input)
			}
		})
	}
}

func TestGetModeDescription(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{
			mode: "regular",
			want: "Full-featured mode with all tools (16K tokens)",
		},
		{
			mode: "review",
			want: "Read-only mode for code analysis (12K tokens)",
		},
		{
			mode: "compact",
			want: "Quick queries with minimal tools (4K tokens)",
		},
		{
			mode: "planning",
			want: "Task planning and decomposition (4K tokens)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := getModeDescription(tt.mode)
			if got != tt.want {
				t.Errorf("getModeDescription(%s) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestGetModeDescription_Unknown(t *testing.T) {
	// Test that unknown modes return empty string (graceful degradation)
	got := getModeDescription("unknown")
	if got != "" {
		t.Errorf("getModeDescription(unknown) = %q, want empty string", got)
	}
}
