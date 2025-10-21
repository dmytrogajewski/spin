package security

import (
	"context"
	"testing"

	// Note: Cannot import agent here due to import cycle
	// CommandCache tests have been moved to internal/agent/cache_test.go
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
