package core

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestManager_WithManagerToolRegistry(t *testing.T) {
	registry := tools.NewRegistry()

	option := WithManagerToolRegistry(registry)

	// Create a minimal manager to test the option
	m := &Manager{}

	// Apply the option
	option(m)

	if m.toolRegistry == nil {
		t.Error("WithManagerToolRegistry() did not set toolRegistry")
	}
}

func TestManager_WithManagerApprovalHandler(t *testing.T) {
	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{Approved: true}
	}

	option := WithManagerApprovalHandler(handler)

	// Create a minimal manager to test the option
	m := &Manager{}

	// Apply the option
	option(m)

	if m.approvalHandler == nil {
		t.Error("WithManagerApprovalHandler() did not set approvalHandler")
	}
}

func TestManager_OptionsChaining(t *testing.T) {
	registry := tools.NewRegistry()
	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{Approved: true}
	}

	// Test that options can be chained
	options := []ManagerOption{
		WithManagerToolRegistry(registry),
		WithManagerApprovalHandler(handler),
	}

	m := &Manager{}

	for _, opt := range options {
		opt(m)
	}

	if m.toolRegistry == nil {
		t.Error("Chained options: toolRegistry not set")
	}
	if m.approvalHandler == nil {
		t.Error("Chained options: approvalHandler not set")
	}
}

func TestManager_WithManagerToolRegistry_NilRegistry(t *testing.T) {
	option := WithManagerToolRegistry(nil)

	m := &Manager{}
	option(m)

	// Should handle nil gracefully
	if m.toolRegistry != nil {
		t.Error("WithManagerToolRegistry(nil) should set toolRegistry to nil")
	}
}

func TestManager_WithManagerApprovalHandler_NilHandler(t *testing.T) {
	option := WithManagerApprovalHandler(nil)

	m := &Manager{}
	option(m)

	// Should handle nil gracefully
	if m.approvalHandler != nil {
		t.Error("WithManagerApprovalHandler(nil) should set approvalHandler to nil")
	}
}
