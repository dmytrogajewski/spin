package tools

// RiskLevel represents the risk level of a tool operation.
type RiskLevel int

const (
	// RiskSafe indicates no approval needed.
	RiskSafe RiskLevel = iota
	// RiskLow indicates read operations.
	RiskLow
	// RiskMedium indicates single file modifications.
	RiskMedium
	// RiskHigh indicates multiple file modifications or patches.
	RiskHigh
	// RiskCritical indicates system file modifications or shell commands.
	RiskCritical
)

// String returns the string representation of the risk level.
func (r RiskLevel) String() string {
	switch r {
	case RiskSafe:
		return "safe"
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return unknownStatus
	}
}

// ApprovalNeeds describes approval requirements for an operation.
type ApprovalNeeds struct {
	// Required indicates whether approval is needed.
	Required bool
	// Risk indicates the risk level of the operation.
	Risk RiskLevel
	// Reason provides human-readable explanation.
	Reason string
}

// ToolWithApproval is implemented by tools that can assess their approval needs.
type ToolWithApproval interface {
	Tool
	// CheckApproval assesses whether the operation requires approval.
	CheckApproval(params ToolParameters) ApprovalNeeds
}
