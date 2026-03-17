package executor

import (
	"context"
	"slices"

	"github.com/dmytrogajewski/spin/internal/safety"
)

// Environment variable keys for command preparation.
const (
	envPythonUnbuffered = "PYTHONUNBUFFERED"
	envValueEnabled     = "1"
)

// Subcommand that triggers auto-confirm rewriting.
const installSubcommand = "install"

// prepareRule defines a command rewrite rule.
type prepareRule struct {
	programs   []string
	subcommand string
	flag       string
}

// autoConfirmRules maps package managers to their auto-confirm flags.
var autoConfirmRules = []prepareRule{
	{programs: []string{"npm"}, subcommand: installSubcommand, flag: "--yes"},
	{programs: []string{"pip", "pip3"}, subcommand: installSubcommand, flag: "--no-input"},
	{programs: []string{"apt", "apt-get"}, subcommand: installSubcommand, flag: "-y"},
}

// pythonPrograms lists programs that need unbuffered output.
var pythonPrograms = []string{"python", "python3", "pytest"}

// NewPrepareStage creates a stage that rewrites commands for non-interactive execution.
// It auto-confirms package manager installs and unbuffers Python output.
func NewPrepareStage() Stage {
	return func(_ context.Context, pc *PipelineContext) error {
		cmd := pc.Command

		applyAutoConfirm(cmd.Program, cmd.Args, &cmd.Args)
		applyPythonUnbuffered(cmd)

		return nil
	}
}

// applyAutoConfirm appends auto-confirm flags to matching package manager commands.
func applyAutoConfirm(program string, args []string, out *[]string) {
	if len(args) == 0 {
		return
	}

	for _, rule := range autoConfirmRules {
		if !slices.Contains(rule.programs, program) {
			continue
		}

		if args[0] != rule.subcommand {
			return
		}

		if slices.Contains(args, rule.flag) {
			return
		}

		*out = append(*out, rule.flag)

		return
	}
}

// applyPythonUnbuffered sets PYTHONUNBUFFERED=1 for Python-related commands.
func applyPythonUnbuffered(cmd *safety.Command) {
	if !slices.Contains(pythonPrograms, cmd.Program) {
		return
	}

	if cmd.Env == nil {
		cmd.Env = make(map[string]string)
	}

	cmd.Env[envPythonUnbuffered] = envValueEnabled
}
