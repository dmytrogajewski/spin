package appserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/security"
)

func TestServer_New(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				WorkspacePath: "/tmp",
				Version:       "1.0.0",
				Provider:      llm.NewMockProvider("test"),
				Executor:      createTestExecutor(t),
				Validator:     security.NewValidator(),
				Environment:   &agent.Environment{WorkDir: "/tmp"},
			},
			wantErr: false,
		},
		{
			name: "minimal config",
			config: Config{
				WorkspacePath: "/tmp",
				Version:       "1.0.0",
				Provider:      llm.NewMockProvider("test"),
				Executor:      createTestExecutor(t),
			},
			wantErr: false,
		},
		{
			name: "invalid workspace path",
			config: Config{
				WorkspacePath: "",
				Version:       "1.0.0",
				Provider:      llm.NewMockProvider("test"),
				Executor:      createTestExecutor(t),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := New(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("New() expected error, got nil")
				}
				if server != nil {
					t.Error("New() expected nil server on error, got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("New() unexpected error: %v", err)
				}
				if server == nil {
					t.Error("New() expected server, got nil")
				}
				if server != nil {
					if server.workspacePath != tt.config.WorkspacePath {
						t.Errorf("New() workspacePath = %q, want %q", server.workspacePath, tt.config.WorkspacePath)
					}
					if server.processor == nil {
						t.Error("New() processor is nil")
					}
					if server.jsonrpcServer == nil {
						t.Error("New() jsonrpcServer is nil")
					}
				}
			}
		})
	}
}

func TestServer_Serve(t *testing.T) {
	// Create a test server
	config := Config{
		WorkspacePath: t.TempDir(),
		Version:       "1.0.0",
		Provider:      llm.NewMockProvider("test"),
		Executor:      createTestExecutor(t),
		Validator:     security.NewValidator(),
		Environment:   &agent.Environment{WorkDir: t.TempDir()},
	}

	server, err := New(config)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid JSON-RPC request",
			input:   `{"jsonrpc":"2.0","method":"initialize","params":{"workspacePath":"/tmp","config":{}},"id":1}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `invalid json`,
			wantErr: false, // JSON-RPC server handles invalid JSON gracefully
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			reader := strings.NewReader(tt.input)
			writer := &strings.Builder{}

			err := server.Serve(ctx, reader, writer)

			if tt.wantErr {
				if err == nil {
					t.Error("Serve() expected error, got nil")
				}
			} else {
				// For these tests, we expect either no error or context cancellation
				if err != nil && err != context.DeadlineExceeded {
					t.Errorf("Serve() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestServer_Serve_ContextCancellation(t *testing.T) {
	// Create a test server
	config := Config{
		WorkspacePath: t.TempDir(),
		Version:       "1.0.0",
		Provider:      llm.NewMockProvider("test"),
		Executor:      createTestExecutor(t),
		Validator:     security.NewValidator(),
		Environment:   &agent.Environment{WorkDir: t.TempDir()},
	}

	server, err := New(config)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	reader := strings.NewReader(`{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}`)
	writer := &strings.Builder{}

	err = server.Serve(ctx, reader, writer)

	// Should get context cancellation error
	if err == nil {
		t.Error("Serve() expected context cancellation error, got nil")
	}
}

func TestServer_Serve_ProcessorOutput(t *testing.T) {
	// Create a test server
	config := Config{
		WorkspacePath: t.TempDir(),
		Version:       "1.0.0",
		Provider:      llm.NewMockProvider("test"),
		Executor:      createTestExecutor(t),
		Validator:     security.NewValidator(),
		Environment:   &agent.Environment{WorkDir: t.TempDir()},
	}

	server, err := New(config)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Test that processor output is set
	writer := &strings.Builder{}
	reader := strings.NewReader("")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// This should set the output writer on the processor
	err = server.Serve(ctx, reader, writer)

	// The main thing we're testing is that SetOutput was called
	// We can't easily verify this without exposing internal state,
	// but the test ensures the method doesn't panic
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Serve() unexpected error: %v", err)
	}
}

// Helper function to create a test executor
func createTestExecutor(t *testing.T) *agent.Executor {
	executor, err := agent.NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create test executor: %v", err)
	}
	return executor
}
