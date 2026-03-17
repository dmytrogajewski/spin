package lsp_test

// Journey: specs/journeys/JOURNEY-R8.1.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

func TestSymbolKind_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind lsp.SymbolKind
		want string
	}{
		{name: "function", kind: lsp.SymbolFunction, want: "Function"},
		{name: "method", kind: lsp.SymbolMethod, want: "Method"},
		{name: "variable", kind: lsp.SymbolVariable, want: "Variable"},
		{name: "constant", kind: lsp.SymbolConstant, want: "Constant"},
		{name: "type", kind: lsp.SymbolType, want: "Type"},
		{name: "interface", kind: lsp.SymbolInterface, want: "Interface"},
		{name: "package", kind: lsp.SymbolPackage, want: "Package"},
		{name: "field", kind: lsp.SymbolField, want: "Field"},
		{name: "property", kind: lsp.SymbolProperty, want: "Property"},
		{name: "unknown_zero", kind: lsp.SymbolKind(0), want: "Unknown"},
		{name: "unknown_high", kind: lsp.SymbolKind(99), want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.kind.String())
		})
	}
}

func TestDiagnosticSeverity_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity lsp.DiagnosticSeverity
		want     string
	}{
		{name: "error", severity: lsp.SeverityError, want: "Error"},
		{name: "warning", severity: lsp.SeverityWarning, want: "Warning"},
		{name: "information", severity: lsp.SeverityInformation, want: "Information"},
		{name: "hint", severity: lsp.SeverityHint, want: "Hint"},
		{name: "unknown_zero", severity: lsp.DiagnosticSeverity(0), want: "Unknown"},
		{name: "unknown_high", severity: lsp.DiagnosticSeverity(99), want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.severity.String())
		})
	}
}
