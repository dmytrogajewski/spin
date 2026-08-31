package compact

// Journey: specs/journeys/JOURNEY-009-compact-pipeline-core.md.

import (
	"bytes"
	"errors"
	"math"
	"slices"
	"testing"
	"time"
)

var errFilterFailed = errors.New("filter failed")

const (
	unknownCommand    = "not-a-registered-compact-command"
	bytesPerKiB       = 1024
	fixtureSizeKiB    = 64
	fixtureSizeBytes  = fixtureSizeKiB * bytesPerKiB
	compactP99Percent = 99
	compactP99Samples = 100
	compactP99Limit   = 15 * time.Millisecond
)

func TestApply_UnknownCommandPreservesZeroExit(t *testing.T) {
	t.Parallel()

	pipeline := New()
	result := pipeline.Apply(unknownCommand, nil, nil, 0)

	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestApply_UnknownCommandPreservesNonzeroExit(t *testing.T) {
	t.Parallel()

	const wantExit = 2

	pipeline := New()
	result := pipeline.Apply(unknownCommand, nil, nil, wantExit)

	if result.ExitCode != wantExit {
		t.Fatalf("exit code = %d, want %d", result.ExitCode, wantExit)
	}
}

func TestApply_UnknownCommandPassthroughBytes(t *testing.T) {
	t.Parallel()

	stdout := []byte("out-bytes")
	stderr := []byte("err-bytes")
	pipeline := New()
	result := pipeline.Apply(unknownCommand, stdout, stderr, 0)

	if !bytes.Equal(result.Stdout, stdout) {
		t.Fatalf("stdout = %q, want %q", result.Stdout, stdout)
	}

	if !bytes.Equal(result.Stderr, stderr) {
		t.Fatalf("stderr = %q, want %q", result.Stderr, stderr)
	}

	if result.Strategy != StrategyR14 {
		t.Fatalf("strategy = %q, want %q", result.Strategy, StrategyR14)
	}
}

func TestApply_UnknownCommandZeroReduction(t *testing.T) {
	t.Parallel()

	stdout := []byte("out-bytes")
	stderr := []byte("err-bytes")
	pipeline := New()
	result := pipeline.Apply(unknownCommand, stdout, stderr, 0)
	wantBytes := len(stdout) + len(stderr)

	if result.Ledger.ReductionPct != 0 {
		t.Fatalf("reduction = %v, want 0", result.Ledger.ReductionPct)
	}

	if result.Ledger.BytesIn != wantBytes {
		t.Fatalf("bytes in = %d, want %d", result.Ledger.BytesIn, wantBytes)
	}

	if result.Ledger.BytesOut != wantBytes {
		t.Fatalf("bytes out = %d, want %d", result.Ledger.BytesOut, wantBytes)
	}
}

func TestApply_LedgerCeilBytesOverFour(t *testing.T) {
	t.Parallel()

	// 5 bytes → ceil(5/4) = 2. Identity: saved = 2-2 = 0.
	stdout := []byte("12345")
	result := New().Apply(unknownCommand, stdout, nil, 0)
	wantTokens := 2

	if result.Ledger.TokensIn != wantTokens {
		t.Fatalf("tokens in = %d, want %d", result.Ledger.TokensIn, wantTokens)
	}

	if result.Ledger.TokensOut != wantTokens {
		t.Fatalf("tokens out = %d, want %d", result.Ledger.TokensOut, wantTokens)
	}

	if result.Ledger.TokensSaved != 0 {
		t.Fatalf("tokens saved = %d, want 0", result.Ledger.TokensSaved)
	}
}

func TestApply_FilterErrorFailSafe(t *testing.T) {
	t.Parallel()

	stdout := []byte("keep-out")
	stderr := []byte("keep-err")

	const wantExit = 3

	pipeline := New()
	pipeline.SetFilter("boom", func(_ string, _, _ []byte) ([]byte, []byte, error) {
		return []byte("nope"), []byte("nope"), errFilterFailed
	})

	result := pipeline.Apply("boom", stdout, stderr, wantExit)

	if !bytes.Equal(result.Stdout, stdout) {
		t.Fatalf("stdout = %q, want %q", result.Stdout, stdout)
	}

	if !bytes.Equal(result.Stderr, stderr) {
		t.Fatalf("stderr = %q, want %q", result.Stderr, stderr)
	}

	if result.Strategy != StrategyR12 {
		t.Fatalf("strategy = %q, want %q", result.Strategy, StrategyR12)
	}

	if result.ExitCode != wantExit {
		t.Fatalf("exit code = %d, want %d", result.ExitCode, wantExit)
	}
}

func TestApply_FilterPanicFailSafe(t *testing.T) {
	t.Parallel()

	stdout := []byte("keep-out")
	stderr := []byte("keep-err")

	const wantExit = 7

	pipeline := New()
	pipeline.SetFilter("boom", func(_ string, _, _ []byte) ([]byte, []byte, error) {
		panic("filter panic")
	})

	result := pipeline.Apply("boom", stdout, stderr, wantExit)

	if !bytes.Equal(result.Stdout, stdout) {
		t.Fatalf("stdout = %q, want %q", result.Stdout, stdout)
	}

	if !bytes.Equal(result.Stderr, stderr) {
		t.Fatalf("stderr = %q, want %q", result.Stderr, stderr)
	}

	if result.Strategy != StrategyR12 {
		t.Fatalf("strategy = %q, want %q", result.Strategy, StrategyR12)
	}

	if result.ExitCode != wantExit {
		t.Fatalf("exit code = %d, want %d", result.ExitCode, wantExit)
	}
}

func TestApply_ShrinkingFilterLedger(t *testing.T) {
	t.Parallel()

	// 5 in, 1 out → ceil(5/4)-ceil(1/4) = 2-1 = 1.
	const (
		wantBytesIn     = 5
		wantBytesOut    = 1
		wantTokensSaved = 1
	)

	pipeline := New()
	pipeline.SetFilter("shrink", func(_ string, _, _ []byte) ([]byte, []byte, error) {
		return []byte("x"), nil, nil
	})

	result := pipeline.Apply("shrink", []byte("12345"), nil, 0)

	if result.Ledger.BytesIn != wantBytesIn {
		t.Fatalf("bytes in = %d, want %d", result.Ledger.BytesIn, wantBytesIn)
	}

	if result.Ledger.BytesOut != wantBytesOut {
		t.Fatalf("bytes out = %d, want %d", result.Ledger.BytesOut, wantBytesOut)
	}

	if result.Ledger.TokensSaved != wantTokensSaved {
		t.Fatalf("tokens saved = %d, want %d", result.Ledger.TokensSaved, wantTokensSaved)
	}
}

func TestApply_FilterMutatesThenErrors(t *testing.T) {
	t.Parallel()

	stdout := []byte("original")
	pipeline := New()
	pipeline.SetFilter("mut", func(_ string, out, _ []byte) ([]byte, []byte, error) {
		out[0] = 'X'

		return out, nil, errFilterFailed
	})

	result := pipeline.Apply("mut", stdout, nil, 0)

	if string(result.Stdout) != "original" {
		t.Fatalf("stdout = %q, want original bytes before mutation", result.Stdout)
	}

	if result.Strategy != StrategyR12 {
		t.Fatalf("strategy = %q, want %q", result.Strategy, StrategyR12)
	}
}

func TestApply_EmptyStdioLedger(t *testing.T) {
	t.Parallel()

	result := New().Apply(unknownCommand, nil, nil, 0)

	if result.Ledger.BytesIn != 0 {
		t.Fatalf("bytes in = %d, want 0", result.Ledger.BytesIn)
	}

	if result.Ledger.BytesOut != 0 {
		t.Fatalf("bytes out = %d, want 0", result.Ledger.BytesOut)
	}

	if result.Ledger.ReductionPct != 0 {
		t.Fatalf("reduction = %v, want 0", result.Ledger.ReductionPct)
	}
}

func TestByteReductionPct_MatchesLedgerBytes(t *testing.T) {
	t.Parallel()

	if got := ByteReductionPct(1000, 280); got != 72 {
		t.Fatalf("ByteReductionPct(1000, 280) = %v, want 72", got)
	}

	if got := ByteReductionPct(100, 100); got != 0 {
		t.Fatalf("identity = %v, want 0", got)
	}
}

func TestApply_NilPipelinePassthrough(t *testing.T) {
	t.Parallel()

	const wantExit = 2

	var pipeline *Pipeline

	stdout := []byte("raw")
	result := pipeline.Apply(unknownCommand, stdout, nil, wantExit)

	if string(result.Stdout) != "raw" {
		t.Fatalf("stdout = %q, want raw", result.Stdout)
	}

	if result.Strategy != StrategyR14 {
		t.Fatalf("strategy = %q, want %q", result.Strategy, StrategyR14)
	}

	if result.ExitCode != wantExit {
		t.Fatalf("exit code = %d, want %d", result.ExitCode, wantExit)
	}
}

func TestApply_BinaryPassthrough(t *testing.T) {
	t.Parallel()

	stdout := []byte{0x00, 0x01, 0x02, 0xff}
	result := New().Apply(unknownCommand, stdout, nil, 0)

	if !bytes.Equal(result.Stdout, stdout) {
		t.Fatalf("stdout = %v, want %v", result.Stdout, stdout)
	}
}

func TestApply_UnknownCommandP99(t *testing.T) {
	t.Parallel()

	fixture := bytes.Repeat([]byte("x"), fixtureSizeBytes)
	pipeline := New()
	durations := make([]time.Duration, compactP99Samples)

	for i := range compactP99Samples {
		start := time.Now()
		result := pipeline.Apply(unknownCommand, fixture, nil, 0)
		durations[i] = time.Since(start)

		if len(result.Stdout) != fixtureSizeBytes {
			t.Fatalf("stdout len = %d, want %d", len(result.Stdout), fixtureSizeBytes)
		}
	}

	p99 := percentile(durations, compactP99Percent)
	if p99 >= compactP99Limit {
		t.Fatalf("p99 %s is not < %s", p99, compactP99Limit)
	}
}

func percentile(durations []time.Duration, pct int) time.Duration {
	sorted := slices.Clone(durations)
	slices.Sort(sorted)

	idx := max(int(math.Ceil(float64(len(sorted))*float64(pct)/percentScale))-1, 0)
	idx = min(idx, len(sorted)-1)

	return sorted[idx]
}
