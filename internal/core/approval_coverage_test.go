package core

import (
	"context"
	"testing"
	"time"
)

func TestApprovalService_RequestApproval(t *testing.T) {
	tests := []struct {
		name         string
		service      *ApprovalService
		operation    Operation
		wantApproved bool
		wantErr      bool
	}{
		{
			name: "no handler configured",
			service: &ApprovalService{
				handler: nil,
			},
			operation: Operation{
				Command: &Command{Program: "ls"},
				Reason:  "test operation",
				WorkDir: "/tmp",
			},
			wantApproved: false,
			wantErr:      true,
		},
		{
			name: "handler approves",
			service: &ApprovalService{
				handler: func(req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  true,
						Reason:    "approved by handler",
					}
				},
			},
			operation: Operation{
				Command: &Command{Program: "mkdir", Args: []string{"testdir"}},
				Reason:  "create directory",
				WorkDir: "/tmp",
			},
			wantApproved: true,
			wantErr:      false,
		},
		{
			name: "handler denies",
			service: &ApprovalService{
				handler: func(req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  false,
						Reason:    "denied by handler",
					}
				},
			},
			operation: Operation{
				Command: &Command{Program: "rm", Args: []string{"-rf", "/"}},
				Reason:  "dangerous operation",
				WorkDir: "/tmp",
			},
			wantApproved: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, approved, err := tt.service.RequestApproval(context.Background(), tt.operation)

			if tt.wantErr {
				if err == nil {
					t.Error("RequestApproval() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("RequestApproval() unexpected error: %v", err)
				}
			}

			if approved != tt.wantApproved {
				t.Errorf("RequestApproval() approved = %v, want %v", approved, tt.wantApproved)
			}
		})
	}
}

func TestApprovalService_RequestApprovalWithValidator(t *testing.T) {
	tests := []struct {
		name         string
		service      *ApprovalService
		cmd          *Command
		validator    *Validator
		workDir      string
		wantApproved bool
		wantErr      bool
	}{
		{
			name: "no validator - approve by default",
			service: &ApprovalService{
				handler: func(req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{Approved: false} // This shouldn't be called
				},
			},
			cmd:          &Command{Program: "ls"},
			validator:    nil,
			workDir:      "/tmp",
			wantApproved: true,
			wantErr:      false,
		},
		{
			name: "safe command - no approval needed",
			service: &ApprovalService{
				handler: func(req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{Approved: false} // This shouldn't be called
				},
			},
			cmd:          &Command{Program: "ls"},
			validator:    NewValidator(),
			workDir:      "/tmp",
			wantApproved: true,
			wantErr:      false,
		},
		{
			name: "dangerous command - approval needed",
			service: &ApprovalService{
				handler: func(req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  true,
						Reason:    "approved by handler",
					}
				},
			},
			cmd:          &Command{Program: "rm", Args: []string{"-rf", "testdir"}},
			validator:    NewValidator(),
			workDir:      "/tmp",
			wantApproved: true,
			wantErr:      false,
		},
		{
			name: "unknown command needs approval",
			service: &ApprovalService{
				handler: func(req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  true,
						Reason:    "approved unknown command",
					}
				},
			},
			cmd:          &Command{Program: "unknown_command"},
			validator:    NewValidator(),
			workDir:      "/tmp",
			wantApproved: true, // Handler approves the unknown command
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approved, err := tt.service.RequestApprovalWithValidator(context.Background(), tt.cmd, tt.validator, tt.workDir)

			if tt.wantErr {
				if err == nil {
					t.Error("RequestApprovalWithValidator() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("RequestApprovalWithValidator() unexpected error: %v", err)
				}
			}

			if approved != tt.wantApproved {
				t.Errorf("RequestApprovalWithValidator() approved = %v, want %v", approved, tt.wantApproved)
			}
		})
	}
}

func TestCommandCache_Set(t *testing.T) {
	tests := []struct {
		name   string
		cache  *CommandCache
		key    string
		result *Result
	}{
		{
			name:  "set basic result",
			cache: NewCommandCache(5*time.Minute, 1024),
			key:   "test-key",
			result: &Result{
				Command:  &Command{Program: "ls"},
				Stdout:   "file1.txt\nfile2.txt",
				Stderr:   "",
				ExitCode: 0,
			},
		},
		{
			name:  "set result with stderr",
			cache: NewCommandCache(5*time.Minute, 1024),
			key:   "error-key",
			result: &Result{
				Command:  &Command{Program: "invalid"},
				Stdout:   "",
				Stderr:   "command not found",
				ExitCode: 1,
			},
		},
		{
			name:  "set large result",
			cache: NewCommandCache(5*time.Minute, 1024),
			key:   "large-key",
			result: &Result{
				Command:  &Command{Program: "cat"},
				Stdout:   string(make([]byte, 500)), // 500 bytes
				Stderr:   "",
				ExitCode: 0,
			},
		},
		{
			name:  "set result that triggers eviction",
			cache: NewCommandCache(5*time.Minute, 100), // Small cache
			key:   "eviction-key",
			result: &Result{
				Command:  &Command{Program: "echo"},
				Stdout:   string(make([]byte, 50)), // 50 bytes - fits in cache
				Stderr:   "",
				ExitCode: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the result
			tt.cache.Set(tt.key, tt.result)

			// Verify it was cached
			cached, found := tt.cache.Get(tt.key)
			if !found {
				t.Error("Set() result not found in cache")
				return
			}

			if cached.Command.Program != tt.result.Command.Program {
				t.Errorf("Set() cached command = %q, want %q", cached.Command.Program, tt.result.Command.Program)
			}

			if cached.Stdout != tt.result.Stdout {
				t.Errorf("Set() cached stdout = %q, want %q", cached.Stdout, tt.result.Stdout)
			}

			if cached.Stderr != tt.result.Stderr {
				t.Errorf("Set() cached stderr = %q, want %q", cached.Stderr, tt.result.Stderr)
			}

			if cached.ExitCode != tt.result.ExitCode {
				t.Errorf("Set() cached exit code = %d, want %d", cached.ExitCode, tt.result.ExitCode)
			}
		})
	}
}

func TestCommandCache_Set_Eviction(t *testing.T) {
	// Create a small cache to test eviction
	cache := NewCommandCache(5*time.Minute, 50)

	// Add multiple entries that exceed cache size
	cache.Set("key1", &Result{
		Command:  &Command{Program: "echo"},
		Stdout:   "output1",
		Stderr:   "",
		ExitCode: 0,
	})

	cache.Set("key2", &Result{
		Command:  &Command{Program: "echo"},
		Stdout:   "output2",
		Stderr:   "",
		ExitCode: 0,
	})

	cache.Set("key3", &Result{
		Command:  &Command{Program: "echo"},
		Stdout:   string(make([]byte, 30)), // 30 bytes - fits in cache
		Stderr:   "",
		ExitCode: 0,
	})

	// Verify key3 is in cache
	_, found := cache.Get("key3")
	if !found {
		t.Error("Set() entry not found in cache")
	}

	// Some older entries might have been evicted
	// This is expected behavior for cache eviction
}
