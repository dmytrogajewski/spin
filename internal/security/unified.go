package security

import (
	"context"
	"errors"
	"time"
)

// UnifiedSecurity provides a consolidated security system that combines
// authentication, validation, and approval into a single interface.
type UnifiedSecurity struct {
	// Authentication
	authManager AuthManager

	// Validation
	validator CommandValidator

	// Approval
	approvalHandler ApprovalHandler

	// Configuration
	config SecurityConfig
}

// SecurityConfig configures the unified security system.
type SecurityConfig struct {
	// RequireApprovalForDangerous requires approval for dangerous commands
	RequireApprovalForDangerous bool

	// RequireApprovalForInteractive requires approval for interactive commands
	RequireApprovalForInteractive bool

	// ApprovalTimeout is the timeout for approval requests
	ApprovalTimeout time.Duration

	// SandboxMode enables sandboxed execution
	SandboxMode bool

	// AllowedCommands is a whitelist of allowed commands
	AllowedCommands []string

	// ForbiddenCommands is a blacklist of forbidden commands
	ForbiddenCommands []string
}

// AuthManager provides authentication management.
type AuthManager interface {
	// GetCredential retrieves a credential for a provider
	GetCredential(ctx context.Context, provider string) (Credential, error)

	// SetCredential stores a credential for a provider
	SetCredential(ctx context.Context, provider string, cred Credential) error

	// DeleteCredential removes a credential for a provider
	DeleteCredential(ctx context.Context, provider string) error

	// ListProviders returns all providers with stored credentials
	ListProviders(ctx context.Context) ([]string, error)
}

// Credential represents an authentication credential.
type Credential struct {
	Type  CredentialType `json:"type"`
	Value string         `json:"value"`
}

// CredentialType represents the type of credential.
type CredentialType int

const (
	CredentialTypeAPIKey CredentialType = iota
	CredentialTypeToken
	CredentialTypeBearer
	CredentialTypeBasic
)

// CommandValidator provides command validation.
type CommandValidator interface {
	// ValidateCommand validates a command and returns its classification
	ValidateCommand(cmd string) (CommandClassification, error)

	// IsAllowed checks if a command is allowed by the security policy
	IsAllowed(cmd string) bool
}

// CommandClassification represents the safety classification of a command.
type CommandClassification int

const (
	ClassificationSafe CommandClassification = iota
	ClassificationInteractive
	ClassificationDangerous
	ClassificationForbidden
	ClassificationUnknown
)

// ApprovalHandler handles approval requests.
type ApprovalHandler interface {
	// RequestApproval requests approval for an operation
	RequestApproval(req ApprovalRequest) ApprovalResponse
}

// ApprovalRequest represents a request for approval.
type ApprovalRequest struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Reason    string    `json:"reason"`
	WorkDir   string    `json:"work_dir"`
	Timestamp time.Time `json:"timestamp"`
}

// ApprovalResponse represents the response to an approval request.
type ApprovalResponse struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// SecurityResult represents the result of a security check.
type SecurityResult struct {
	Allowed          bool                  `json:"allowed"`
	Reason           string                `json:"reason,omitempty"`
	Classification   CommandClassification `json:"classification"`
	RequiresApproval bool                  `json:"requires_approval"`
	Approved         bool                  `json:"approved,omitempty"`
}

// Errors
var (
	ErrNotAuthenticated  = errors.New("not authenticated")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrCommandForbidden  = errors.New("command forbidden")
	ErrApprovalDenied    = errors.New("approval denied")
	ErrApprovalTimeout   = errors.New("approval timeout")
)
