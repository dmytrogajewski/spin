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
	// If a credential already exists, it is overwritten.
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
func (t CredentialType) String() string {
	switch t {
	case CredentialTypeNone:
		return "none"
	case CredentialTypeAPIKey:
		return "apikey"
	case CredentialTypeToken:
		return "token"
	default:
		return fmt.Sprintf("unknown(%d)", t)
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
	if err := validateCredential(cred); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCredential, err)
	}

	// Normalize provider name
	provider = normalizeProvider(provider)

	// Format credential
	value := formatCredential(cred)

	// Store in keystore
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

	// Delete from keystore
	if err := m.keystore.Delete(credentialKey(provider)); err != nil {
		if errors.Is(err, ErrNotFound) {
			// Idempotent - no error if not found
			return nil
		}
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

	// List all keys
	keys, err := m.keystore.List()
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	// Extract provider names from credential keys
	providers := make([]string, 0, len(keys))
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

// parseCredential parses a credential from storage format.
//
// Format: "type:value"
func parseCredential(s string) (Credential, error) {
	typeName, value, err := splitCredentialString(s)
	if err != nil {
		return Credential{}, err
	}

	typ, err := parseCredentialType(typeName)
	if err != nil {
		return Credential{}, err
	}

	return Credential{
		Type:  typ,
		Value: value,
	}, nil
}

// splitCredentialString splits a credential string into type and value.
func splitCredentialString(s string) (string, string, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid credential format: %q", s)
	}
	return parts[0], parts[1], nil
}

// parseCredentialType parses a credential type from string.
func parseCredentialType(typeName string) (CredentialType, error) {
	switch typeName {
	case "none":
		return CredentialTypeNone, nil
	case "apikey":
		return CredentialTypeAPIKey, nil
	case "token":
		return CredentialTypeToken, nil
	default:
		return CredentialTypeNone, fmt.Errorf("unknown credential type: %q", typeName)
	}
}

// validateCredential validates a credential.
func validateCredential(cred Credential) error {
	// Validate type
	switch cred.Type {
	case CredentialTypeNone:
		// None is valid but value should be empty
		if cred.Value != "" {
			return errors.New("none credential must have empty value")
		}
	case CredentialTypeAPIKey, CredentialTypeToken:
		// API key and token must have non-empty value
		if cred.Value == "" {
			return errors.New("credential value cannot be empty")
		}
	default:
		return fmt.Errorf("unknown credential type: %d", cred.Type)
	}

	return nil
}

// formatCredential formats a credential for storage.
func formatCredential(cred Credential) string {
	var typeName string
	switch cred.Type {
	case CredentialTypeNone:
		typeName = "none"
	case CredentialTypeAPIKey:
		typeName = "apikey"
	case CredentialTypeToken:
		typeName = "token"
	default:
		typeName = "none"
	}

	return typeName + ":" + cred.Value
}
