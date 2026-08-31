package tools

// Journey: specs/journeys/JOURNEY-011-apply-compact-to-shell-exec.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
)

const gitStatusPorcelain = "M  staged.go\n M unstaged.go\n?? new.go\n"

func gitStatusCompact() string {
	return string(compact.Default().Apply("git status", []byte(gitStatusPorcelain), nil, 0).Stdout)
}

func TestShellCommandTool_Execute_CompactsGitStatus(t *testing.T) {
	t.Parallel()

	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{Stdout: gitStatusPorcelain, ExitCode: 0}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{"operation": "execute", "command": "git status"})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	want := gitStatusCompact()
	if result.Output != want {
		t.Fatalf("Output = %q, want compact %q", result.Output, want)
	}

	led := compact.Default().Apply("git status", []byte(gitStatusPorcelain), nil, 0).Ledger
	if result.Metadata[compact.MetaBytesIn] != led.BytesIn || result.Metadata[compact.MetaBytesOut] != led.BytesOut {
		t.Fatalf("ledger meta in=%v out=%v, want %d/%d",
			result.Metadata[compact.MetaBytesIn], result.Metadata[compact.MetaBytesOut], led.BytesIn, led.BytesOut)
	}
}

func TestShellCommandTool_Execute_PreservesNonzeroExit(t *testing.T) {
	t.Parallel()

	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{Stdout: gitStatusPorcelain, ExitCode: 1}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{"operation": "execute", "command": "git status"})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Success || result.ExitCode != 1 {
		t.Fatalf("Success=%v ExitCode=%d, want fail/1", result.Success, result.ExitCode)
	}

	if result.Output != gitStatusCompact() {
		t.Fatalf("Output = %q, want compact", result.Output)
	}
}

func TestShellCommandTool_Execute_EnvOffSkipsCompact(t *testing.T) {
	t.Setenv(compact.EnvName, compact.EnvOff)

	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{Stdout: gitStatusPorcelain, ExitCode: 0}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{"operation": "execute", "command": "git status"})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != gitStatusPorcelain {
		t.Fatalf("SPIN_COMPACT=0 Output = %q, want raw", result.Output)
	}
}

func TestShellCommandTool_Execute_DisabledSkipsCompact(t *testing.T) {
	t.Parallel()

	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{Stdout: gitStatusPorcelain, ExitCode: 0}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)
	tool.SetCompactEnabled(false)

	params, err := FromMap(map[string]any{"operation": "execute", "command": "git status"})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != gitStatusPorcelain {
		t.Fatalf("disabled Output = %q, want raw", result.Output)
	}
}

func TestShellCommandTool_Execute_RTKPrefixedSkipsApply(t *testing.T) {
	t.Parallel()

	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{Stdout: gitStatusPorcelain, ExitCode: 0}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{"operation": "execute", "command": "rtk git status"})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != gitStatusPorcelain {
		t.Fatalf("rtk-prefixed Output = %q, want raw (no double-compact)", result.Output)
	}
}
