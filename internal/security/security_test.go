package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name            string
		validator       *Validator
		approvalService *ApprovalService
		wantNil         bool
	}{
		{
			name:            "with both dependencies",
			validator:       NewValidator(),
			approvalService: NewApprovalServiceWithConfig(ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: nil}),
			wantNil:         false,
		},
		{
			name:            "with nil validator",
			validator:       nil,
			approvalService: NewApprovalServiceWithConfig(ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: nil}),
			wantNil:         false, // Service allows nil validator.
		},
		{
			name:            "with nil approval service",
			validator:       NewValidator(),
			approvalService: nil,
			wantNil:         false, // Service allows nil approval.
		},
		{
			name:            "with both nil",
			validator:       nil,
			approvalService: nil,
			wantNil:         false, // Service allows nil deps.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.validator, tt.approvalService)

			if tt.wantNil {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
			}
		})
	}
}

func TestService_ValidateCommand(t *testing.T) {
	validator := NewValidator()
	svc := NewService(validator, nil)

	tests := []struct {
		name           string
		cmd            *Command
		wantClass      CommandClass
		wantErr        bool
		setupValidator func(*Validator)
	}{
		{
			name: "safe command",
			cmd: &Command{
				Raw:     "echo hello",
				Program: "echo",
				Args:    []string{"hello"},
			},
			wantClass: CommandSafe,
			wantErr:   false,
		},
		{
			name: "forbidden command - rm rf with path",
			cmd: &Command{
				Raw:     "rm -rf /tmp",
				Program: "rm",
				Args:    []string{"-rf", "/tmp"},
			},
			wantClass: CommandForbidden,
			wantErr:   false,
		},
		{
			name: "unverified command - rm single file",
			cmd: &Command{
				Raw:     "rm file.txt",
				Program: "rm",
				Args:    []string{"file.txt"},
			},
			wantClass: CommandUnverified, // rm without flags is unverified.
			wantErr:   false,
		},
		{
			name: "interactive command",
			cmd: &Command{
				Raw:     "mkdir testdir",
				Program: "mkdir",
				Args:    []string{"testdir"},
			},
			wantClass: CommandInteractive,
			wantErr:   false,
		},
		{
			name:      "nil command",
			cmd:       nil,
			wantClass: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupValidator != nil {
				tt.setupValidator(validator)
			}

			result, err := svc.ValidateCommand(tt.cmd)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantClass, result.Classification)
		})
	}
}

func TestService_ValidateCommand_NilValidator(t *testing.T) {
	svc := NewService(nil, nil)

	cmd := &Command{
		Raw:     "echo test",
		Program: "echo",
		Args:    []string{"test"},
	}

	_, err := svc.ValidateCommand(cmd)
	assert.Error(t, err, "should error when validator is nil")
	assert.Contains(t, err.Error(), "validator not configured")
}

func TestService_NeedsApproval(t *testing.T) {
	validator := NewValidator()
	svc := NewService(validator, nil)

	tests := []struct {
		name         string
		cmd          *Command
		wantApproval bool
	}{
		{
			name: "safe command - no approval",
			cmd: &Command{
				Raw:     "ls -la",
				Program: "ls",
				Args:    []string{"-la"},
			},
			wantApproval: false,
		},
		{
			name: "forbidden command - should be blocked (no approval)",
			cmd: &Command{
				Raw:     "rm -rf /",
				Program: "rm",
				Args:    []string{"-rf", "/"},
			},
			wantApproval: false, // Forbidden commands should be blocked, not approved.
		},
		{
			name: "interactive command - needs approval",
			cmd: &Command{
				Raw:     "mkdir testdir",
				Program: "mkdir",
				Args:    []string{"testdir"},
			},
			wantApproval: true,
		},
		{
			name: "safe command - no approval",
			cmd: &Command{
				Raw:     "cat file.txt",
				Program: "cat",
				Args:    []string{"file.txt"},
			},
			wantApproval: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needs := svc.NeedsApproval(tt.cmd)
			assert.Equal(t, tt.wantApproval, needs)
		})
	}
}

func TestService_NeedsApproval_NilValidator(t *testing.T) {
	svc := NewService(nil, nil)

	cmd := &Command{
		Raw:     "rm -rf /",
		Program: "rm",
		Args:    []string{"-rf", "/"},
	}

	needs := svc.NeedsApproval(cmd)
	assert.False(t, needs, "should return false when validator is nil")
}

func TestService_RequestApproval(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name         string
		operation    Operation
		setupHandler func() ApprovalHandler
		wantApproved bool
		wantErr      bool
		errContains  string
	}{
		{
			name: "approval granted",
			operation: NewOperation(&Command{
				Raw:     "rm -rf /tmp/test",
				Program: "rm",
				Args:    []string{"-rf", "/tmp/test"},
			}, "dangerous operation", "/tmp"),
			setupHandler: func() ApprovalHandler {
				return func(_ context.Context, req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  true,
						Reason:    "user approved",
						Timestamp: time.Now(),
					}
				}
			},
			wantApproved: true,
			wantErr:      false,
		},
		{
			name: "approval denied",
			operation: NewOperation(&Command{
				Raw:     "rm -rf /",
				Program: "rm",
				Args:    []string{"-rf", "/"},
			}, "extremely dangerous", "/"),
			setupHandler: func() ApprovalHandler {
				return func(_ context.Context, req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  false,
						Reason:    "too dangerous",
						Timestamp: time.Now(),
					}
				}
			},
			wantApproved: false,
			wantErr:      false,
		},
		{
			name: "no approval handler",
			operation: NewOperation(&Command{
				Raw:     "rm test.txt",
				Program: "rm",
				Args:    []string{"test.txt"},
			}, "needs approval", "/tmp"),
			setupHandler: func() ApprovalHandler {
				return nil
			},
			wantApproved: false,
			wantErr:      true,
			errContains:  "no approval handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			approvalService := NewApprovalServiceWithConfig(ApprovalServiceConfig{Handler: handler, Emitter: nil, Validator: validator})
			svc := NewService(validator, approvalService)

			ctx := context.Background()
			approved, err := svc.RequestApproval(ctx, tt.operation)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantApproved, approved)
		})
	}
}

func TestService_RequestApproval_NilApprovalService(t *testing.T) {
	validator := NewValidator()
	svc := NewService(validator, nil)

	operation := NewOperation(&Command{
		Raw:     "rm test",
		Program: "rm",
		Args:    []string{"test"},
	}, "needs approval", "/tmp")

	ctx := context.Background()
	approved, err := svc.RequestApproval(ctx, operation)

	assert.False(t, approved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "approval service not configured")
}

func TestService_ValidateAndApprove(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name          string
		cmd           *Command
		setupHandler  func() ApprovalHandler
		wantApproved  bool
		wantErr       bool
		shouldRequest bool // Whether approval request is expected.
	}{
		{
			name: "safe command - no approval needed",
			cmd: &Command{
				Raw:     "ls -la",
				Program: "ls",
				Args:    []string{"-la"},
			},
			setupHandler: func() ApprovalHandler {
				return func(_ context.Context, _ ApprovalRequest) ApprovalResponse {
					panic("should not be called for safe commands")
				}
			},
			wantApproved:  true,
			wantErr:       false,
			shouldRequest: false,
		},
		{
			name: "forbidden command - blocked without approval",
			cmd: &Command{
				Raw:     "rm -rf /",
				Program: "rm",
				Args:    []string{"-rf", "/"},
			},
			setupHandler: func() ApprovalHandler {
				return func(_ context.Context, _ ApprovalRequest) ApprovalResponse {
					panic("should not be called for forbidden commands")
				}
			},
			wantApproved:  false,
			wantErr:       false,
			shouldRequest: false, // Forbidden commands are blocked, not approved.
		},
		{
			name: "interactive command - approval granted",
			cmd: &Command{
				Raw:     "mkdir testdir",
				Program: "mkdir",
				Args:    []string{"testdir"},
			},
			setupHandler: func() ApprovalHandler {
				return func(_ context.Context, req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  true,
						Timestamp: time.Now(),
					}
				}
			},
			wantApproved:  true,
			wantErr:       false,
			shouldRequest: true,
		},
		{
			name: "interactive command - approval denied",
			cmd: &Command{
				Raw:     "mkdir sensitive_dir",
				Program: "mkdir",
				Args:    []string{"sensitive_dir"},
			},
			setupHandler: func() ApprovalHandler {
				return func(_ context.Context, req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  false,
						Reason:    "user denied",
						Timestamp: time.Now(),
					}
				}
			},
			wantApproved:  false,
			wantErr:       false,
			shouldRequest: true,
		},
		{
			name: "unverified command - approval granted",
			cmd: &Command{
				Raw:     "somecommand arg",
				Program: "somecommand",
				Args:    []string{"arg"},
			},
			setupHandler: func() ApprovalHandler {
				return func(_ context.Context, req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  true,
						Timestamp: time.Now(),
					}
				}
			},
			wantApproved:  true,
			wantErr:       false,
			shouldRequest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			approvalService := NewApprovalServiceWithConfig(ApprovalServiceConfig{Handler: handler, Emitter: nil, Validator: validator})
			svc := NewService(validator, approvalService)

			ctx := context.Background()
			approved, err := svc.ValidateAndApprove(ctx, tt.cmd, "/tmp")

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantApproved, approved)
		})
	}
}

// Benchmark tests.
func TestService_ValidateAndApprove_ApprovalError(t *testing.T) {
	validator := NewValidator()
	// No approval service configured - will cause error.
	svc := NewService(validator, nil)

	// Interactive command that needs approval.
	cmd := &Command{
		Raw:     "mkdir testdir",
		Program: "mkdir",
		Args:    []string{"testdir"},
	}

	ctx := context.Background()
	approved, err := svc.ValidateAndApprove(ctx, cmd, "/tmp")

	assert.False(t, approved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "approval request failed")
}

func TestService_ValidateAndApprove_NilValidator(t *testing.T) {
	svc := NewService(nil, nil)

	cmd := &Command{
		Raw:     "echo test",
		Program: "echo",
		Args:    []string{"test"},
	}

	ctx := context.Background()
	approved, err := svc.ValidateAndApprove(ctx, cmd, "/tmp")

	assert.False(t, approved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestService_ValidateAndApprove_NilCommand(t *testing.T) {
	validator := NewValidator()
	svc := NewService(validator, nil)

	ctx := context.Background()
	approved, err := svc.ValidateAndApprove(ctx, nil, "/tmp")

	assert.False(t, approved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func BenchmarkService_ValidateCommand(b *testing.B) {
	validator := NewValidator()
	svc := NewService(validator, nil)

	cmd := &Command{
		Raw:     "ls -la",
		Program: "ls",
		Args:    []string{"-la"},
	}

	b.ResetTimer()

	for range b.N {
		_, _ = svc.ValidateCommand(cmd)
	}
}

func BenchmarkService_NeedsApproval(b *testing.B) {
	validator := NewValidator()
	svc := NewService(validator, nil)

	cmd := &Command{
		Raw:     "rm -rf /tmp",
		Program: "rm",
		Args:    []string{"-rf", "/tmp"},
	}

	b.ResetTimer()

	for range b.N {
		_ = svc.NeedsApproval(cmd)
	}
}

func TestService_ApprovalService(t *testing.T) {
	validator := NewValidator()
	approvalService := NewApprovalServiceWithConfig(ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: validator})
	svc := NewService(validator, approvalService)

	retrieved := svc.ApprovalService()
	assert.Equal(t, approvalService, retrieved)
}

func TestService_ApprovalService_Nil(t *testing.T) {
	validator := NewValidator()
	svc := NewService(validator, nil)

	retrieved := svc.ApprovalService()
	assert.Nil(t, retrieved)
}

func TestService_Validator(t *testing.T) {
	validator := NewValidator()
	approvalService := NewApprovalServiceWithConfig(ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: validator})
	svc := NewService(validator, approvalService)

	retrieved := svc.Validator()
	assert.Equal(t, validator, retrieved)
}

func TestService_Validator_Nil(t *testing.T) {
	approvalService := NewApprovalServiceWithConfig(ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: nil})
	svc := NewService(nil, approvalService)

	retrieved := svc.Validator()
	assert.Nil(t, retrieved)
}
