// Package auth provides authentication and credential management for LLM providers.
//
// The package defines a vendor-agnostic interface for storing and retrieving
// authentication credentials (API keys, tokens) using platform-specific secure
// storage backends (Keychain, Secret Service, Credential Manager).
//
// # Basic Usage
//
// Create an authentication manager with a keystore:
//
//	keystore := newMockKeystore() // or platform-specific keystore
//	manager := auth.NewManager(keystore)
//
// Store a credential
//
//	cred := auth.Credential{
//		Type:  auth.CredentialTypeAPIKey,
//		Value: "sk-...",
//	}
//	err := manager.SetCredential(ctx, "openai", cred)
//
// Retrieve a credential
//
//	cred, err := manager.GetCredential(ctx, "openai")
//	if errors.Is(err, auth.ErrNotAuthenticated) {
//
// No credential found
//
//	}
//
// # Credential Types
//
// The package supports three credential types:
//
//   - CredentialTypeAPIKey: API key authentication (e.g., "sk-...")
//   - CredentialTypeToken: Bearer token authentication
//   - CredentialTypeNone: No authentication required (for local providers)
//
// # Provider Names
//
// Provider names are normalized to lowercase and trimmed of whitespace.
// This ensures consistent credential lookup regardless of casing:
//
//	manager.SetCredential(ctx, "OpenAI", cred)
//	manager.GetCredential(ctx, "openai") // Returns same credential
//
// # Security Considerations
//
//   - Credentials are never logged or printed
//   - Error messages do not expose credential values
//   - Credentials are validated before storage
//   - Keystore operations are wrapped with context for timeout control
//
// # Error Handling
//
// The package defines two primary error types:
//
//   - ErrNotAuthenticated: Returned when no credential exists for a provider
//   - ErrInvalidCredential: Returned when credential validation fails
//
// Keystore errors are wrapped and propagated with context.
//
// # Thread Safety
//
// The Manager implementation is safe for concurrent use by multiple goroutines.
// Keystore implementations must also be thread-safe.
package auth
