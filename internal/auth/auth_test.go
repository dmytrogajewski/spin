package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var errKeystoreError = errors.New("keystore error")

// TestNewManager tests manager creation.
func TestNewManager(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)

	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.keystore != ks {
		t.Error("Manager keystore not set correctly")
	}
}

// TestManager_GetCredential_Success tests retrieving existing credentials.
func TestManager_GetCredential_Success(t *testing.T) {
	ks := newMockKeystore()
	ks.data["spin:cred:openai"] = "apikey:sk-test-key"

	m := NewManager(ks)
	ctx := context.Background()

	cred, err := m.GetCredential(ctx, "openai")
	if err != nil {
		t.Fatalf("GetCredential() error = %v", err)
	}

	if cred.Type != CredentialTypeAPIKey {
		t.Errorf("Credential.Type = %v, want %v", cred.Type, CredentialTypeAPIKey)
	}

	if cred.Value != "sk-test-key" {
		t.Errorf("Credential.Value = %q, want %q", cred.Value, "sk-test-key")
	}
}

// TestManager_GetCredential_NotFound tests retrieving non-existent credentials.
func TestManager_GetCredential_NotFound(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)
	ctx := context.Background()

	_, err := m.GetCredential(ctx, "nonexistent")
	if err == nil {
		t.Fatal("GetCredential() expected error, got nil")
	}

	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("Error = %v, want ErrNotAuthenticated", err)
	}
}

// TestManager_GetCredential_ContextCanceled tests context cancellation.
func TestManager_GetCredential_ContextCanceled(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := m.GetCredential(ctx, "openai")
	if err == nil {
		t.Fatal("GetCredential() expected context error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error = %v, want context.Canceled", err)
	}
}

// TestManager_GetCredential_ProviderNormalization tests provider name normalization.
func TestManager_GetCredential_ProviderNormalization(t *testing.T) {
	ks := newMockKeystore()
	ks.data["spin:cred:openai"] = "apikey:sk-test"

	m := NewManager(ks)
	ctx := context.Background()

	tests := []struct {
		name     string
		provider string
	}{
		{"lowercase", "openai"},
		{"uppercase", "OPENAI"},
		{"mixed case", "OpenAI"},
		{"with whitespace", "  openai  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred, err := m.GetCredential(ctx, tt.provider)
			if err != nil {
				t.Fatalf("GetCredential(%q) error = %v", tt.provider, err)
			}

			if cred.Value != "sk-test" {
				t.Errorf("Credential.Value = %q, want %q", cred.Value, "sk-test")
			}
		})
	}
}

// TestManager_GetCredential_CredentialTypes tests different credential types.
func TestManager_GetCredential_CredentialTypes(t *testing.T) {
	tests := []struct {
		name      string
		stored    string
		wantType  CredentialType
		wantValue string
		wantErr   bool
	}{
		{
			name:      "API key",
			stored:    "apikey:sk-test-123",
			wantType:  CredentialTypeAPIKey,
			wantValue: "sk-test-123",
		},
		{
			name:      "Token",
			stored:    "token:bearer-xyz",
			wantType:  CredentialTypeToken,
			wantValue: "bearer-xyz",
		},
		{
			name:      "None",
			stored:    "none:",
			wantType:  CredentialTypeNone,
			wantValue: "",
		},
		{
			name:    "Invalid format",
			stored:  "invalid-no-colon",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := newMockKeystore()
			ks.data["spin:cred:test"] = tt.stored

			m := NewManager(ks)
			ctx := context.Background()

			cred, err := m.GetCredential(ctx, "test")
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetCredential() expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("GetCredential() error = %v", err)
			}

			if cred.Type != tt.wantType {
				t.Errorf("Credential.Type = %v, want %v", cred.Type, tt.wantType)
			}

			if cred.Value != tt.wantValue {
				t.Errorf("Credential.Value = %q, want %q", cred.Value, tt.wantValue)
			}
		})
	}
}

// TestManager_SetCredential_Success tests setting credentials.
func TestManager_SetCredential_Success(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)
	ctx := context.Background()

	cred := Credential{
		Type:  CredentialTypeAPIKey,
		Value: "sk-new-key",
	}

	err := m.SetCredential(ctx, "openai", cred)
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	// Verify stored.
	stored, exists := ks.data["spin:cred:openai"]
	if !exists {
		t.Fatal("Credential not stored in keystore")
	}

	if stored != "apikey:sk-new-key" {
		t.Errorf("Stored value = %q, want %q", stored, "apikey:sk-new-key")
	}
}

// TestManager_SetCredential_InvalidCredential tests validation.
func TestManager_SetCredential_InvalidCredential(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)
	ctx := context.Background()

	tests := []struct {
		name string
		cred Credential
	}{
		{
			name: "empty API key",
			cred: Credential{
				Type:  CredentialTypeAPIKey,
				Value: "",
			},
		},
		{
			name: "empty token",
			cred: Credential{
				Type:  CredentialTypeToken,
				Value: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.SetCredential(ctx, "test", tt.cred)
			if err == nil {
				t.Fatal("SetCredential() expected error, got nil")
			}

			if !errors.Is(err, ErrInvalidCredential) {
				t.Errorf("Error = %v, want ErrInvalidCredential", err)
			}
		})
	}
}

// TestManager_SetCredential_Overwrite tests overwriting credentials.
func TestManager_SetCredential_Overwrite(t *testing.T) {
	ks := newMockKeystore()
	ks.data["spin:cred:openai"] = "apikey:old-key"

	m := NewManager(ks)
	ctx := context.Background()

	cred := Credential{
		Type:  CredentialTypeAPIKey,
		Value: "new-key",
	}

	err := m.SetCredential(ctx, "openai", cred)
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	// Verify overwritten.
	stored := ks.data["spin:cred:openai"]
	if stored != "apikey:new-key" {
		t.Errorf("Stored value = %q, want %q", stored, "apikey:new-key")
	}
}

// TestManager_SetCredential_ContextCanceled tests context cancellation.
func TestManager_SetCredential_ContextCanceled(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cred := Credential{
		Type:  CredentialTypeAPIKey,
		Value: "test",
	}

	err := m.SetCredential(ctx, "openai", cred)
	if err == nil {
		t.Fatal("SetCredential() expected context error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error = %v, want context.Canceled", err)
	}
}

// TestManager_DeleteCredential_Success tests deleting credentials.
func TestManager_DeleteCredential_Success(t *testing.T) {
	ks := newMockKeystore()
	ks.data["spin:cred:openai"] = "apikey:test"

	m := NewManager(ks)
	ctx := context.Background()

	err := m.DeleteCredential(ctx, "openai")
	if err != nil {
		t.Fatalf("DeleteCredential() error = %v", err)
	}

	// Verify deleted.
	if _, exists := ks.data["spin:cred:openai"]; exists {
		t.Error("Credential still exists after delete")
	}
}

// TestManager_DeleteCredential_Idempotent tests deleting non-existent credentials.
func TestManager_DeleteCredential_Idempotent(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)
	ctx := context.Background()

	// Delete non-existent credential should not error.
	err := m.DeleteCredential(ctx, "nonexistent")
	if err != nil {
		t.Errorf("DeleteCredential() error = %v, want nil", err)
	}
}

// TestManager_DeleteCredential_ContextCanceled tests context cancellation.
func TestManager_DeleteCredential_ContextCanceled(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.DeleteCredential(ctx, "openai")
	if err == nil {
		t.Fatal("DeleteCredential() expected context error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error = %v, want context.Canceled", err)
	}
}

// TestManager_ListProviders_Empty tests listing when no credentials.
func TestManager_ListProviders_Empty(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)
	ctx := context.Background()

	providers, err := m.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}

	if len(providers) != 0 {
		t.Errorf("ListProviders() = %v, want empty slice", providers)
	}
}

// TestManager_ListProviders_Multiple tests listing multiple providers.
func TestManager_ListProviders_Multiple(t *testing.T) {
	ks := newMockKeystore()
	ks.data["spin:cred:openai"] = "apikey:test1"
	ks.data["spin:cred:ollama"] = "none:"
	ks.data["spin:cred:lmstudio"] = "apikey:test2"
	ks.data["spin:other:key"] = "value" // Should be filtered.

	m := NewManager(ks)
	ctx := context.Background()

	providers, err := m.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}

	want := []string{"lmstudio", "ollama", "openai"}
	if len(providers) != len(want) {
		t.Fatalf("ListProviders() returned %d providers, want %d", len(providers), len(want))
	}

	// Check all expected providers present (order doesn't matter).
	providerMap := make(map[string]bool)
	for _, p := range providers {
		providerMap[p] = true
	}

	for _, w := range want {
		if !providerMap[w] {
			t.Errorf("ListProviders() missing %q", w)
		}
	}
}

// TestManager_ListProviders_ContextCanceled tests context cancellation.
func TestManager_ListProviders_ContextCanceled(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.ListProviders(ctx)
	if err == nil {
		t.Fatal("ListProviders() expected context error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error = %v, want context.Canceled", err)
	}
}

// TestManager_ThreadSafety tests concurrent operations.
func TestManager_ThreadSafety(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)
	ctx := context.Background()

	var wg sync.WaitGroup

	errChan := make(chan error, 100)

	// Concurrent sets.
	for i := range 10 {
		wg.Add(1)

		go func(_ int) {
			defer wg.Done()

			cred := Credential{
				Type:  CredentialTypeAPIKey,
				Value: "test",
			}
			err := m.SetCredential(ctx, "test", cred)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	// Concurrent gets.
	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := m.GetCredential(ctx, "test")
			if err != nil && !errors.Is(err, ErrNotAuthenticated) {
				errChan <- err
			}
		}()
	}

	// Concurrent lists.
	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := m.ListProviders(ctx)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

// TestCredential_Validate tests credential validation.
func TestCredential_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cred    Credential
		wantErr bool
	}{
		{
			name: "valid API key",
			cred: Credential{
				Type:  CredentialTypeAPIKey,
				Value: "sk-test",
			},
			wantErr: false,
		},
		{
			name: "valid token",
			cred: Credential{
				Type:  CredentialTypeToken,
				Value: "bearer-xyz",
			},
			wantErr: false,
		},
		{
			name: "valid none",
			cred: Credential{
				Type:  CredentialTypeNone,
				Value: "",
			},
			wantErr: false,
		},
		{
			name: "empty API key",
			cred: Credential{
				Type:  CredentialTypeAPIKey,
				Value: "",
			},
			wantErr: true,
		},
		{
			name: "empty token",
			cred: Credential{
				Type:  CredentialTypeToken,
				Value: "",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			cred: Credential{
				Type:  CredentialType(999),
				Value: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCredential(tt.cred)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCredential() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCredentialType_String tests string representation.
func TestCredentialType_String(t *testing.T) {
	tests := []struct {
		typ  CredentialType
		want string
	}{
		{CredentialTypeNone, "none"},
		{CredentialTypeAPIKey, "apikey"},
		{CredentialTypeToken, "token"},
		{CredentialType(999), "unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.typ.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestManager_ContextTimeout tests operations with timeout.
func TestManager_ContextTimeout(t *testing.T) {
	ks := newMockKeystore()
	m := NewManager(ks)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Sleep longer than timeout to ensure it triggers.
	time.Sleep(20 * time.Millisecond)

	_, err := m.GetCredential(ctx, "test")
	if err == nil {
		t.Fatal("GetCredential() expected timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Error = %v, want context.DeadlineExceeded", err)
	}
}

// TestManager_KeystoreErrors tests handling of keystore errors.
func TestManager_KeystoreErrors(t *testing.T) {
	ks := &errorKeystore{err: errKeystoreError}
	m := NewManager(ks)
	ctx := context.Background()

	// Get error.
	_, err := m.GetCredential(ctx, "test")
	if err == nil {
		t.Error("GetCredential() expected error, got nil")
	}

	// Set error.
	cred := Credential{Type: CredentialTypeAPIKey, Value: "test"}

	err = m.SetCredential(ctx, "test", cred)
	if err == nil {
		t.Error("SetCredential() expected error, got nil")
	}

	// Delete error.
	err = m.DeleteCredential(ctx, "test")
	if err == nil {
		t.Error("DeleteCredential() expected error, got nil")
	}

	// List error.
	_, err = m.ListProviders(ctx)
	if err == nil {
		t.Error("ListProviders() expected error, got nil")
	}
}

// errorKeystore implements Keystore that always returns errors.
type errorKeystore struct {
	err error
}

func (e *errorKeystore) Get(_ string) (string, error) {
	return "", e.err
}

func (e *errorKeystore) Set(_, _ string) error {
	return e.err
}

func (e *errorKeystore) Delete(_ string) error {
	return e.err
}

func (e *errorKeystore) List() ([]string, error) {
	return nil, e.err
}

// mockKeystore implements Keystore for testing.
type mockKeystore struct {
	data map[string]string
	mu   sync.RWMutex
}

func newMockKeystore() *mockKeystore {
	return &mockKeystore{
		data: make(map[string]string),
	}
}

func (m *mockKeystore) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return "", ErrNotFound
	}

	return value, nil
}

func (m *mockKeystore) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value

	return nil
}

func (m *mockKeystore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)

	return nil
}

func (m *mockKeystore) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}

	return keys, nil
}
