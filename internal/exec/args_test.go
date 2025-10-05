package exec

import (
	"strings"
	"testing"
	"time"
)

func TestParseArgsFromCmdLine(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *ExecArgs
		wantErr bool
	}{
		{
			name: "simple prompt",
			args: []string{"test prompt"},
			want: &ExecArgs{
				Prompt:      "test prompt",
				Format:      "text",
				ExitOnError: true,
			},
			wantErr: false,
		},
		{
			name: "prompt with flags",
			args: []string{"--timeout", "5m", "--auto-approve", "run tests"},
			want: &ExecArgs{
				Prompt:      "run tests",
				AutoApprove: true,
				Timeout:     5 * time.Minute,
				Format:      "text",
				ExitOnError: true,
			},
			wantErr: false,
		},
		{
			name: "json format",
			args: []string{"--format", "json", "analyze code"},
			want: &ExecArgs{
				Prompt:      "analyze code",
				Format:      "json",
				ExitOnError: true,
			},
			wantErr: false,
		},
		{
			name: "no stream flag",
			args: []string{"--no-stream", "task"},
			want: &ExecArgs{
				Prompt:      "task",
				NoStream:    true,
				Format:      "text",
				ExitOnError: true,
			},
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    []string{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "only flags no prompt",
			args:    []string{"--timeout", "5m"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Prompt != tt.want.Prompt {
				t.Errorf("parseArgs() Prompt = %v, want %v", got.Prompt, tt.want.Prompt)
			}
			if got.AutoApprove != tt.want.AutoApprove {
				t.Errorf("parseArgs() AutoApprove = %v, want %v", got.AutoApprove, tt.want.AutoApprove)
			}
			if got.Timeout != tt.want.Timeout {
				t.Errorf("parseArgs() Timeout = %v, want %v", got.Timeout, tt.want.Timeout)
			}
			if got.Format != tt.want.Format {
				t.Errorf("parseArgs() Format = %v, want %v", got.Format, tt.want.Format)
			}
			if got.NoStream != tt.want.NoStream {
				t.Errorf("parseArgs() NoStream = %v, want %v", got.NoStream, tt.want.NoStream)
			}
		})
	}
}

func TestParseArgsFromStdin(t *testing.T) {
	tests := []struct {
		name    string
		stdin   string
		want    string
		wantErr bool
	}{
		{
			name:    "simple stdin",
			stdin:   "test prompt from stdin",
			want:    "test prompt from stdin",
			wantErr: false,
		},
		{
			name:    "multiline stdin",
			stdin:   "line 1\nline 2\nline 3",
			want:    "line 1\nline 2\nline 3",
			wantErr: false,
		},
		{
			name:    "empty stdin",
			stdin:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			stdin:   "   \n\t\n  ",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.stdin)
			got, err := parseArgs([]string{}, reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() with stdin error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Prompt != tt.want {
				t.Errorf("parseArgs() from stdin = %q, want %q", got.Prompt, tt.want)
			}
		})
	}
}

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "seconds",
			args:    []string{"--timeout", "30s", "task"},
			want:    30 * time.Second,
			wantErr: false,
		},
		{
			name:    "minutes",
			args:    []string{"--timeout", "5m", "task"},
			want:    5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "hours",
			args:    []string{"--timeout", "2h", "task"},
			want:    2 * time.Hour,
			wantErr: false,
		},
		{
			name:    "combined",
			args:    []string{"--timeout", "1h30m", "task"},
			want:    90 * time.Minute,
			wantErr: false,
		},
		{
			name:    "invalid format",
			args:    []string{"--timeout", "invalid", "task"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "negative duration",
			args:    []string{"--timeout", "-5m", "task"},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() timeout error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Timeout != tt.want {
				t.Errorf("parseArgs() Timeout = %v, want %v", got.Timeout, tt.want)
			}
		})
	}
}

func TestParseInvalidFormat(t *testing.T) {
	args := []string{"--format", "xml", "task"}
	_, err := parseArgs(args, nil)
	if err == nil {
		t.Error("parseArgs() should error on invalid format")
	}
}

func TestParseGlobalFlags(t *testing.T) {
	args := []string{
		"--model", "llama3.1",
		"--provider", "ollama",
		"--sandbox", "workspace-write",
		"--cd", "/tmp/test",
		"task prompt",
	}

	got, err := parseArgs(args, nil)
	if err != nil {
		t.Fatalf("parseArgs() unexpected error: %v", err)
	}

	if got.Model != "llama3.1" {
		t.Errorf("Model = %v, want llama3.1", got.Model)
	}
	if got.Provider != "ollama" {
		t.Errorf("Provider = %v, want ollama", got.Provider)
	}
	if got.Sandbox != "workspace-write" {
		t.Errorf("Sandbox = %v, want workspace-write", got.Sandbox)
	}
	if got.WorkDir != "/tmp/test" {
		t.Errorf("WorkDir = %v, want /tmp/test", got.WorkDir)
	}
}
