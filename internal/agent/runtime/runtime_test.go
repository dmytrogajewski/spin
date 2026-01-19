package runtime

import (
	"testing"
)

// TestBuiltinRuntime_ImplementsToolRegistrar verifies BuiltinRuntime implements ToolRegistrar.
func TestBuiltinRuntime_ImplementsToolRegistrar(t *testing.T) {
	var _ ToolRegistrar = (*BuiltinRuntime)(nil)
}

// TestACPRuntime_ImplementsToolRegistrar verifies ACPRuntime implements ToolRegistrar.
func TestACPRuntime_ImplementsToolRegistrar(t *testing.T) {
	var _ ToolRegistrar = (*ACPRuntime)(nil)
}

// TestBuiltinRuntime_ImplementsNotificationProvider verifies BuiltinRuntime implements NotificationProvider.
func TestBuiltinRuntime_ImplementsNotificationProvider(t *testing.T) {
	var _ NotificationProvider = (*BuiltinRuntime)(nil)
}

// TestACPRuntime_ImplementsNotificationProvider verifies ACPRuntime implements NotificationProvider.
func TestACPRuntime_ImplementsNotificationProvider(t *testing.T) {
	var _ NotificationProvider = (*ACPRuntime)(nil)
}

// TestBuiltinRuntime_ImplementsApprovalProvider verifies BuiltinRuntime implements ApprovalProvider.
func TestBuiltinRuntime_ImplementsApprovalProvider(t *testing.T) {
	var _ ApprovalProvider = (*BuiltinRuntime)(nil)
}

// TestACPRuntime_ImplementsApprovalProvider verifies ACPRuntime implements ApprovalProvider.
func TestACPRuntime_ImplementsApprovalProvider(t *testing.T) {
	var _ ApprovalProvider = (*ACPRuntime)(nil)
}

// TestBuiltinRuntime_ImplementsSessionProvider verifies BuiltinRuntime implements SessionProvider.
func TestBuiltinRuntime_ImplementsSessionProvider(t *testing.T) {
	var _ SessionProvider = (*BuiltinRuntime)(nil)
}

// TestACPRuntime_ImplementsSessionProvider verifies ACPRuntime implements SessionProvider.
func TestACPRuntime_ImplementsSessionProvider(t *testing.T) {
	var _ SessionProvider = (*ACPRuntime)(nil)
}

// TestBuiltinRuntime_ImplementsTerminalProvider verifies BuiltinRuntime implements TerminalProvider.
func TestBuiltinRuntime_ImplementsTerminalProvider(t *testing.T) {
	var _ TerminalProvider = (*BuiltinRuntime)(nil)
}

// TestACPRuntime_ImplementsTerminalProvider verifies ACPRuntime implements TerminalProvider.
func TestACPRuntime_ImplementsTerminalProvider(t *testing.T) {
	var _ TerminalProvider = (*ACPRuntime)(nil)
}

// TestBuiltinRuntime_ImplementsRuntime verifies BuiltinRuntime implements the composite Runtime interface.
func TestBuiltinRuntime_ImplementsRuntime(t *testing.T) {
	var _ Runtime = (*BuiltinRuntime)(nil)
}

// TestACPRuntime_ImplementsRuntime verifies ACPRuntime implements the composite Runtime interface.
func TestACPRuntime_ImplementsRuntime(t *testing.T) {
	var _ Runtime = (*ACPRuntime)(nil)
}

// TestRuntime_TypeAssertionToToolRegistrar verifies Runtime can be type-asserted to ToolRegistrar.
func TestRuntime_TypeAssertionToToolRegistrar(t *testing.T) {
	var rt Runtime = &BuiltinRuntime{}
	_, ok := rt.(ToolRegistrar)
	if !ok {
		t.Error("Runtime should be type-assertable to ToolRegistrar")
	}
}

// TestRuntime_TypeAssertionToNotificationProvider verifies Runtime can be type-asserted to NotificationProvider.
func TestRuntime_TypeAssertionToNotificationProvider(t *testing.T) {
	var rt Runtime = &BuiltinRuntime{}
	_, ok := rt.(NotificationProvider)
	if !ok {
		t.Error("Runtime should be type-assertable to NotificationProvider")
	}
}

// TestRuntime_TypeAssertionToApprovalProvider verifies Runtime can be type-asserted to ApprovalProvider.
func TestRuntime_TypeAssertionToApprovalProvider(t *testing.T) {
	var rt Runtime = &BuiltinRuntime{}
	_, ok := rt.(ApprovalProvider)
	if !ok {
		t.Error("Runtime should be type-assertable to ApprovalProvider")
	}
}

// TestRuntime_TypeAssertionToSessionProvider verifies Runtime can be type-asserted to SessionProvider.
func TestRuntime_TypeAssertionToSessionProvider(t *testing.T) {
	var rt Runtime = &BuiltinRuntime{}
	_, ok := rt.(SessionProvider)
	if !ok {
		t.Error("Runtime should be type-assertable to SessionProvider")
	}
}

// TestRuntime_TypeAssertionToTerminalProvider verifies Runtime can be type-asserted to TerminalProvider.
func TestRuntime_TypeAssertionToTerminalProvider(t *testing.T) {
	var rt Runtime = &BuiltinRuntime{}
	_, ok := rt.(TerminalProvider)
	if !ok {
		t.Error("Runtime should be type-assertable to TerminalProvider")
	}
}
