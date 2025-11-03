package agent

import (
	"testing"
)

func TestNewBuilder_CreatesBuilder(t *testing.T) {
	builder := NewBuilder()

	if builder == nil {
		t.Fatal("NewBuilder() returned nil")
	}
}

func TestBuilder_WithConfig(t *testing.T) {
	cfg := &Config{
		Model: "test-model",
	}

	builder := NewBuilder().WithConfig(cfg)

	if builder.config != cfg {
		t.Error("WithConfig() did not store config")
	}
}

func TestBuilder_FluentInterface(t *testing.T) {
	cfg := &Config{Model: "test"}

	builder := NewBuilder().
		WithConfig(cfg).
		WithWorkingDir("/test").
		WithEmitter(nil).
		WithApprovalHandler(nil)

	if builder == nil {
		t.Error("Fluent interface broke chain")
	}
	if builder.config != cfg {
		t.Error("Config not set")
	}
	if builder.workingDir != "/test" {
		t.Error("WorkingDir not set")
	}
}
