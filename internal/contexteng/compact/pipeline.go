// Package compact implements the RTK compact pipeline (R12–R15) and the
// per-command filter registry (R1–R10). R11 rewrite is argv-level (see RewriteArgv).
package compact

import (
	"bytes"
	"strings"
)

// Strategy identifiers recorded on Result.
const (
	// StrategyR11 names the auto-rewrite hook (argv prefix, not a stdio filter).
	StrategyR11 = "R11"
	// StrategyR12 is fail-safe raw output after a filter error or panic.
	StrategyR12 = "R12"
	// StrategyR14 is unknown-command passthrough (R14).
	StrategyR14 = "R14"
)

// Filter transforms stdout/stderr for one command.
// An error (or panic) triggers R12 fail-safe: original bytes are returned.
type Filter func(cmd string, stdout, stderr []byte) (compactedStdout, compactedStderr []byte, err error)

// Result is the output of Pipeline.Apply.
type Result struct {
	// Stdout is the (possibly compacted) standard output.
	Stdout []byte
	// Stderr is the (possibly compacted) standard error.
	Stderr []byte
	// ExitCode is the original process exit code (never rewritten).
	ExitCode int
	// Strategy is the RTK rule that produced the bytes (R12 or R14 in this step).
	Strategy string
	// Ledger is the R15 ceil(bytes/4) savings record.
	Ledger Ledger
}

// Pipeline applies compact filters. This step has none in production.
type Pipeline struct {
	filters map[string]Filter
}

// New returns an empty pipeline (all commands unknown).
func New() *Pipeline {
	return &Pipeline{filters: make(map[string]Filter)}
}

// SetFilter registers a command filter. Default() uses this for the spec table;
// tests inject failing filters to prove R12.
func (p *Pipeline) SetFilter(cmd string, filter Filter) {
	if p == nil {
		return
	}

	if p.filters == nil {
		p.filters = make(map[string]Filter)
	}

	p.filters[cmd] = filter
}

// Apply returns compacted stdio. Exit code is always the input exit code.
func (p *Pipeline) Apply(cmd string, stdout, stderr []byte, exitCode int) (result Result) {
	result = identity(stdout, stderr, exitCode, StrategyR14)
	filter := p.filterFor(cmd)

	if filter == nil {
		return result
	}

	origStdout := bytes.Clone(stdout)
	origStderr := bytes.Clone(stderr)

	defer func() {
		if recover() != nil {
			result = identity(origStdout, origStderr, exitCode, StrategyR12)
		}
	}()

	outStdout, outStderr, err := filter(cmd, stdout, stderr)
	if err != nil {
		return identity(origStdout, origStderr, exitCode, StrategyR12)
	}

	return Result{
		Stdout:   outStdout,
		Stderr:   outStderr,
		ExitCode: exitCode,
		Ledger:   account(origStdout, origStderr, outStdout, outStderr),
	}
}

func (p *Pipeline) filterFor(cmd string) Filter {
	if p == nil {
		return nil
	}

	if filter, found := p.filters[cmd]; found {
		return filter
	}

	return p.prefixFilter(cmd)
}

func (p *Pipeline) prefixFilter(cmd string) Filter {
	bestLen := 0

	var best Filter

	for key, filter := range p.filters {
		if !commandPrefixed(cmd, key) {
			continue
		}

		if len(key) > bestLen {
			best = filter
			bestLen = len(key)
		}
	}

	return best
}

func commandPrefixed(cmd, key string) bool {
	return strings.HasPrefix(cmd, key+" ")
}

func identity(stdout, stderr []byte, exitCode int, strategy string) Result {
	return Result{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Strategy: strategy,
		Ledger:   account(stdout, stderr, stdout, stderr),
	}
}
