package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Auth manages authentication credentials for LLM providers.
//
// Implementations must be safe for concurrent use by multiple goroutines.
type Auth interface {
	// GetCredential retrieves a credential for a provider.
	//
	// Returns ErrNotAuthenticated if no credential is found.
	// The context can be used for cancellation and timeout control.
	GetCredential(ctx context.Context, provider string) (Credential, error)

	// SetCredential stores a credential for a provider.
	//
	// If a credential already exists, it will be overwritten.
	// Returns ErrInvalidCredential if the credential fails validation.
	SetCredential(ctx context.Context, provider string, cred Credential) error

	// DeleteCredential removes a credential for a provider.
	//
	// This operation is idempotent - deleting a non-existent credential succeeds.
	DeleteCredential(ctx context.Context, provider string) error

	// ListProviders returns all providers with stored credentials.
	//
	// Returns an empty slice if no credentials are stored.
	ListProviders(ctx context.Context) ([]string, error)
}

// Credential represents an authentication credential.
type Credential struct {
	// Type is the credential type (API key, token, etc.)
	Type CredentialType

	// Value is the credential value (e.g., "sk-...", "bearer-...")
	Value string
}

// Validate validates the credential.
//
// Returns an error if:
//   - API key or token type has empty value
//   - Invalid credential type
func (c Credential) Validate() error {
	switch c.Type {
	case CredentialTypeAPIKey, CredentialTypeToken:
		if c.Value == "" {
			return fmt.Errorf("%s credential value cannot be empty", c.Type)
		}
	case CredentialTypeNone:
		// None type can have empty value
	default:
		return fmt.Errorf("invalid credential type: %d", c.Type)
	}
	return nil
}

// CredentialType represents the type of credential.
type CredentialType int

const (
	// CredentialTypeNone indicates no authentication required
	CredentialTypeNone CredentialType = iota

	// CredentialTypeAPIKey indicates API key authentication
	CredentialTypeAPIKey

	// CredentialTypeToken indicates bearer token authentication
	CredentialTypeToken
)

// String returns the string representation of the credential type.
func (ct CredentialType) String() string {
	switch ct {
	case CredentialTypeNone:
		return "none"
	case CredentialTypeAPIKey:
		return "apikey"
	case CredentialTypeToken:
		return "token"
	default:
		return fmt.Sprintf("unknown(%d)", ct)
	}
}

var (
	// ErrNotAuthenticated indicates no credential found for provider
	ErrNotAuthenticated = errors.New("not authenticated")

	// ErrInvalidCredential indicates credential validation failed
	ErrInvalidCredential = errors.New("invalid credential")
)

// Manager implements Auth interface using a Keystore.
type Manager struct {
	keystore Keystore
}

// NewManager creates a new authentication manager.
//
// The keystore is used for persistent credential storage.
// The manager is safe for concurrent use.
func NewManager(keystore Keystore) *Manager {
	return &Manager{keystore: keystore}
}

// GetCredential retrieves a credential for a provider.
func (m *Manager) GetCredential(ctx context.Context, provider string) (Credential, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return Credential{}, err
	}

	// Normalize provider name
	provider = normalizeProvider(provider)

	// Get from keystore
	value, err := m.keystore.Get(credentialKey(provider))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Credential{}, fmt.Errorf("%s: %w", provider, ErrNotAuthenticated)
		}
		return Credential{}, fmt.Errorf("get credential: %w", err)
	}

	// Parse credential
	return parseCredential(value)
}

// SetCredential stores a credential for a provider.
func (m *Manager) SetCredential(ctx context.Context, provider string, cred Credential) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate credential
	if err := cred.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCredential, err)
	}

	// Normalize provider name
	provider = normalizeProvider(provider)

	// Format and store
	value := formatCredential(cred)
	if err := m.keystore.Set(credentialKey(provider), value); err != nil {
		return fmt.Errorf("set credential: %w", err)
	}

	return nil
}

// DeleteCredential removes a credential for a provider.
func (m *Manager) DeleteCredential(ctx context.Context, provider string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return err
	}

	// Normalize provider name
	provider = normalizeProvider(provider)

	// Delete from keystore (idempotent)
	if err := m.keystore.Delete(credentialKey(provider)); err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}

	return nil
}

// ListProviders returns all providers with stored credentials.
func (m *Manager) ListProviders(ctx context.Context) ([]string, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Get all keys from keystore
	keys, err := m.keystore.List()
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	// Filter credential keys and extract provider names
	providers := make([]string, 0)
	prefix := "spin:cred:"

	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			provider := strings.TrimPrefix(key, prefix)
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// credentialKey generates a keystore key for a provider credential.
func credentialKey(provider string) string {
	return "spin:cred:" + provider
}

// normalizeProvider normalizes a provider name (lowercase, trimmed).
func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

// formatCredential formats a credential for storage.
//
// Format: "type:value"
func formatCredential(cred Credential) string {
	return cred.Type.String() + ":" + cred.Value
}

// parseCredential parses a credential from storage format.
//
// Format: "type:value"
func parseCredential(s string) (Credential, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return Credential{}, fmt.Errorf("invalid credential format: %q", s)
	}

	typeName := parts[0]
	value := parts[1]

	var typ CredentialType
	switch typeName {
	case "none":
		typ = CredentialTypeNone
	case "apikey":
		typ = CredentialTypeAPIKey
	case "token":
		typ = CredentialTypeToken
	default:
		return Credential{}, fmt.Errorf("unknown credential type: %q", typeName)
	}

	return Credential{
		Type:  typ,
		Value: value,
	}, nil
}
