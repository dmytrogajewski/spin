package tools

import (
	"context"
	"testing"
)

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskSafe, "safe"},
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.expected {
				t.Errorf("RiskLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestApprovalNeeds_NoApprovalRequired(t *testing.T) {
	needs := ApprovalNeeds{
		Required: false,
		Risk:     RiskSafe,
		Reason:   "",
	}

	if needs.Required {
		t.Error("ApprovalNeeds.Required should be false for safe operations")
	}

	if needs.Risk != RiskSafe {
		t.Errorf("ApprovalNeeds.Risk = %v, want RiskSafe", needs.Risk)
	}

	if needs.Reason != "" {
		t.Errorf("ApprovalNeeds.Reason = %q, want empty string", needs.Reason)
	}
}

// mockToolWithApproval is a test tool that implements ToolWithApproval.
type mockToolWithApproval struct {
	checkApprovalFunc func(ToolParameters) ApprovalNeeds
}

func (m *mockToolWithApproval) Name() string        { return "mock_tool" }
func (m *mockToolWithApproval) Description() string { return "Mock tool for testing" }
func (m *mockToolWithApproval) Schema() ToolSchema  { return ToolSchema{} }
func (m *mockToolWithApproval) Execute(_ context.Context, _ ToolParameters) (ToolResult, error) {
	return ToolResult{Success: true}, nil
}
func (m *mockToolWithApproval) CheckApproval(params ToolParameters) ApprovalNeeds {
	if m.checkApprovalFunc != nil {
		return m.checkApprovalFunc(params)
	}

	return ApprovalNeeds{Required: false, Risk: RiskSafe}
}

func TestToolWithApproval_Interface(t *testing.T) {
	tool := &mockToolWithApproval{
		checkApprovalFunc: func(_ ToolParameters) ApprovalNeeds {
			return ApprovalNeeds{
				Required: true,
				Risk:     RiskHigh,
				Reason:   "test reason",
			}
		},
	}

	// Verify it implements ToolWithApproval.
	var _ ToolWithApproval = tool

	// Call CheckApproval.
	needs := tool.CheckApproval(ToolParameters{})
	if !needs.Required {
		t.Error("CheckApproval should return Required=true")
	}

	if needs.Risk != RiskHigh {
		t.Errorf("CheckApproval Risk = %v, want RiskHigh", needs.Risk)
	}

	if needs.Reason != "test reason" {
		t.Errorf("CheckApproval Reason = %q, want %q", needs.Reason, "test reason")
	}
}
