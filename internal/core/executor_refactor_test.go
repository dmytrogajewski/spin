package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestExecutor_errorResult tests error result creation helper
func TestExecutor_errorResult(t *testing.T) {
	executor := &Executor{}
	cmd := &Command{
		Program: "test",
		Args:    []string{"arg1"},
	}
	err := assert.AnError

	result := executor.errorResult(cmd, err)

	assert.NotNil(t, result)
	assert.Equal(t, cmd, result.Command)
	assert.Equal(t, err, result.Error)
	assert.NotZero(t, result.StartedAt)
	assert.NotZero(t, result.CompletedAt)
	assert.Equal(t, time.Duration(0), result.Duration, "Error results have 0 duration")
	assert.Equal(t, -1, result.ExitCode, "Error results have exit code -1")
}

// TestExecutor_checkCache tests cache checking helper
func TestExecutor_checkCache(t *testing.T) {
	cmd := &Command{Program: "echo", Args: []string{"hello"}}

	t.Run("no_cache", func(t *testing.T) {
		executor := &Executor{} // No cache configured
		result := executor.checkCache(cmd)
		assert.Nil(t, result, "Should return nil when cache not configured")
	})

	t.Run("cache_miss", func(t *testing.T) {
		cache := NewCommandCache(1*time.Minute, 1024)
		executor := &Executor{cache: cache}
		result := executor.checkCache(cmd)
		assert.Nil(t, result, "Should return nil on cache miss")
	})

	t.Run("cache_hit", func(t *testing.T) {
		cache := NewCommandCache(1*time.Minute, 1024)
		executor := &Executor{cache: cache}

		// Pre-populate cache
		cachedResult := &Result{
			Command:  cmd,
			Stdout:   "cached output",
			ExitCode: 0,
		}
		cache.Set(cache.Key(cmd), cachedResult)

		result := executor.checkCache(cmd)
		assert.NotNil(t, result, "Should return cached result")
		assert.Equal(t, "cached output", result.Stdout)
	})
}

// TestExecutor_cacheResultIfEligible tests result caching helper
func TestExecutor_cacheResultIfEligible(t *testing.T) {
	cmd := &Command{Program: "echo", Args: []string{"hello"}}
	result := &Result{
		Command:  cmd,
		Stdout:   "output",
		ExitCode: 0,
	}

	t.Run("no_cache", func(t *testing.T) {
		executor := &Executor{} // No cache
		executor.cacheResultIfEligible(cmd, result)
		// Should not panic, just no-op
	})

	t.Run("cache_error_result", func(t *testing.T) {
		cache := NewCommandCache(1*time.Minute, 1024)
		executor := &Executor{cache: cache}

		errorResult := &Result{
			Command:  cmd,
			Error:    assert.AnError,
			ExitCode: 1,
		}

		executor.cacheResultIfEligible(cmd, errorResult)

		// Error results should not be cached
		cached, ok := cache.Get(cache.Key(cmd))
		assert.False(t, ok, "Error results should not be cached")
		assert.Nil(t, cached, "Cached value should be nil")
	})

	t.Run("cache_success_result", func(t *testing.T) {
		cache := NewCommandCache(1*time.Minute, 1024)
		executor := &Executor{cache: cache}

		executor.cacheResultIfEligible(cmd, result)

		// Success result should be cached
		cached, ok := cache.Get(cache.Key(cmd))
		assert.True(t, ok, "Should find cached result")
		assert.Equal(t, result, cached, "Cached result should match")
	})
}

// TestExecutor_validateCommand tests validation pipeline
func TestExecutor_validateCommand(t *testing.T) {
	executor := &Executor{}
	opts := DefaultExecuteOptions()

	tests := []struct {
		name    string
		cmd     *Command
		wantErr bool
		errType error
	}{
		{
			name:    "valid_command",
			cmd:     &Command{Program: "echo", Args: []string{"test"}},
			wantErr: false,
		},
		{
			name:    "nil_command",
			cmd:     nil,
			wantErr: true,
			errType: ErrNilCommand,
		},
		{
			name:    "empty_program",
			cmd:     &Command{Program: "", Args: []string{}},
			wantErr: true,
			errType: ErrEmptyProgram,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateCommand(tt.cmd, opts)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExecutor_requestApprovalIfNeeded tests approval flow
func TestExecutor_requestApprovalIfNeeded(t *testing.T) {
	cmd := &Command{
		Program: "rm",
		Args:    []string{"-rf", "/"},
		Raw:     "rm -rf /",
		WorkDir: "/tmp",
	}
	opts := DefaultExecuteOptions()
	opts.WorkDir = "/tmp"
	ctx := context.Background()

	t.Run("no_approval_service", func(t *testing.T) {
		executor := &Executor{} // No approval service
		err := executor.requestApprovalIfNeeded(ctx, cmd, opts)
		assert.NoError(t, err, "Should not error when no approval service")
	})

	t.Run("with_approval_service_approved", func(t *testing.T) {
		approvalService := NewApprovalService(func(req ApprovalRequest) ApprovalResponse {
			return ApprovalResponse{
				RequestID: req.ID,
				Approved:  true,
			}
		})
		validator := NewValidator()
		executor := &Executor{
			approvalService: approvalService,
			validator:       validator,
			workDir:         "/tmp",
		}

		err := executor.requestApprovalIfNeeded(ctx, cmd, opts)
		assert.NoError(t, err, "Should not error when approved")
	})

	t.Run("safe_command_no_approval_needed", func(t *testing.T) {
		safeCmd := &Command{
			Program: "ls",
			Args:    []string{},
			Raw:     "ls",
			WorkDir: "/tmp",
		}

		approvalService := NewApprovalService(func(req ApprovalRequest) ApprovalResponse {
			t.Fatal("Should not request approval for safe command")
			return ApprovalResponse{}
		})
		validator := NewValidator()
		executor := &Executor{
			approvalService: approvalService,
			validator:       validator,
			workDir:         "/tmp",
		}

		err := executor.requestApprovalIfNeeded(ctx, safeCmd, opts)
		assert.NoError(t, err, "Safe commands should not error")
	})
}
