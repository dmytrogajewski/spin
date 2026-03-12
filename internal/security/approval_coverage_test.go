package security

import (
	"context"
	"testing"
	// Note: Cannot import agent here due to import cycle
	// CommandCache tests have been moved to internal/agent/cache_test.go.
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
			operation:    NewOperation(&Command{Program: "ls"}, "test operation", "/tmp"),
			wantApproved: false,
			wantErr:      true,
		},
		{
			name: "handler approves",
			service: &ApprovalService{
				handler: func(ctx context.Context, req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  true,
						Reason:    "approved by handler",
					}
				},
			},
			operation:    NewOperation(&Command{Program: "mkdir", Args: []string{"testdir"}}, "create directory", "/tmp"),
			wantApproved: true,
			wantErr:      false,
		},
		{
			name: "handler denies",
			service: &ApprovalService{
				handler: func(ctx context.Context, req ApprovalRequest) ApprovalResponse {
					return ApprovalResponse{
						RequestID: req.ID,
						Approved:  false,
						Reason:    "denied by handler",
					}
				},
			},
			operation:    NewOperation(&Command{Program: "rm", Args: []string{"-rf", "/"}}, "dangerous operation", "/tmp"),
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
