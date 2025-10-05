package exec

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// ExecArgs holds parsed command-line arguments for exec mode.
type ExecArgs struct {
	Prompt  string        // Task prompt
	Timeout time.Duration // Execution timeout

	// Exec-specific flags
	AutoApprove bool   // Auto-approve dangerous operations
	Format      string // Output format (text, json)
	NoStream    bool   // Disable streaming
	ExitOnError bool   // Exit on first error

	// Global flags (inherited from main CLI)
	Model           string
	Provider        string
	Sandbox         string
	WorkDir         string
	ConfigFile      string
	ConfigOverrides []string
}

// parseArgs parses command-line arguments and stdin.
// If args are empty, reads prompt from stdin (if provided).
func parseArgs(args []string, stdin io.Reader) (*ExecArgs, error) {
	execArgs := &ExecArgs{
		Format:      "text",
		ExitOnError: true,
	}

	flags := pflag.NewFlagSet("spin-exec", pflag.ContinueOnError)

	// Exec-specific flags
	flags.BoolVar(&execArgs.AutoApprove, "auto-approve", false, "Automatically approve all operations (DANGEROUS)")
	flags.DurationVar(&execArgs.Timeout, "timeout", 0, "Maximum execution time (e.g., 5m, 1h)")
	flags.StringVar(&execArgs.Format, "format", "text", "Output format: text or json")
	flags.BoolVar(&execArgs.NoStream, "no-stream", false, "Disable streaming output")
	flags.BoolVar(&execArgs.ExitOnError, "exit-on-error", true, "Exit immediately on first error")

	// Global flags
	flags.StringVar(&execArgs.Model, "model", "", "LLM model to use")
	flags.StringVar(&execArgs.Provider, "provider", "", "LLM provider (ollama, lmstudio, openai, anthropic)")
	flags.StringVar(&execArgs.Sandbox, "sandbox", "", "Sandbox mode (read-only, workspace-write, full-access)")
	flags.StringVar(&execArgs.WorkDir, "cd", "", "Working directory")
	flags.StringVar(&execArgs.ConfigFile, "config-file", "", "Configuration file path")
	flags.StringSliceVarP(&execArgs.ConfigOverrides, "config", "c", nil, "Config overrides (key=value)")

	// Parse flags
	if err := flags.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	// Get remaining args as prompt
	remaining := flags.Args()

	if len(remaining) > 0 {
		// Prompt from command line
		execArgs.Prompt = strings.Join(remaining, " ")
	} else if stdin != nil {
		// Prompt from stdin
		prompt, err := readPromptFromStdin(stdin)
		if err != nil {
			return nil, err
		}
		execArgs.Prompt = prompt
	} else {
		return nil, fmt.Errorf("no prompt provided (use command line args or stdin)")
	}

	// Validate
	if err := validateExecArgs(execArgs); err != nil {
		return nil, err
	}

	return execArgs, nil
}

// readPromptFromStdin reads the task prompt from stdin.
func readPromptFromStdin(stdin io.Reader) (string, error) {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	prompt := string(data)
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("empty prompt from stdin")
	}

	return prompt, nil
}

// validateExecArgs validates parsed arguments.
func validateExecArgs(args *ExecArgs) error {
	// Validate prompt
	if strings.TrimSpace(args.Prompt) == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	// Validate format
	if args.Format != "text" && args.Format != "json" {
		return fmt.Errorf("invalid format: %s (must be 'text' or 'json')", args.Format)
	}

	// Validate timeout
	if args.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	return nil
}

// applyDefaults applies default values to exec args.
func applyDefaults(args *ExecArgs) {
	if args.Format == "" {
		args.Format = "text"
	}
	// ExitOnError defaults to true (already set in parseArgs)
}
