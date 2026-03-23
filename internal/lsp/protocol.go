// Package lsp provides a lifecycle manager for Language Server Protocol servers.
// It handles lazy process startup, JSON-RPC 2.0 communication, and response caching.
package lsp

import (
	"errors"

	"github.com/dmytrogajewski/spin/pkg/protocol/jsonrpc"
)

// SymbolKind classifies the kind of a programming symbol.
type SymbolKind int

const (
	// SymbolFunction represents a function symbol.
	SymbolFunction SymbolKind = iota + 1
	// SymbolMethod represents a method symbol.
	SymbolMethod
	// SymbolVariable represents a variable symbol.
	SymbolVariable
	// SymbolConstant represents a constant symbol.
	SymbolConstant
	// SymbolType represents a type symbol.
	SymbolType
	// SymbolInterface represents an interface symbol.
	SymbolInterface
	// SymbolPackage represents a package symbol.
	SymbolPackage
	// SymbolField represents a struct field symbol.
	SymbolField
	// SymbolProperty represents a property symbol.
	SymbolProperty
)

// symbolKindNames maps SymbolKind values to their string representations.
var symbolKindNames = [...]string{
	SymbolFunction:  "Function",
	SymbolMethod:    "Method",
	SymbolVariable:  "Variable",
	SymbolConstant:  "Constant",
	SymbolType:      "Type",
	SymbolInterface: "Interface",
	SymbolPackage:   "Package",
	SymbolField:     "Field",
	SymbolProperty:  "Property",
}

// String returns the string representation of a SymbolKind.
func (sk SymbolKind) String() string {
	if sk >= SymbolFunction && int(sk) < len(symbolKindNames) {
		return symbolKindNames[sk]
	}

	return "Unknown"
}

// DiagnosticSeverity classifies the severity of a diagnostic.
type DiagnosticSeverity int

const (
	// SeverityError indicates an error diagnostic.
	SeverityError DiagnosticSeverity = iota + 1
	// SeverityWarning indicates a warning diagnostic.
	SeverityWarning
	// SeverityInformation indicates an informational diagnostic.
	SeverityInformation
	// SeverityHint indicates a hint diagnostic.
	SeverityHint
)

// severityNames maps DiagnosticSeverity values to their string representations.
var severityNames = [...]string{
	SeverityError:       "Error",
	SeverityWarning:     "Warning",
	SeverityInformation: "Information",
	SeverityHint:        "Hint",
}

// String returns the string representation of a DiagnosticSeverity.
func (ds DiagnosticSeverity) String() string {
	if ds >= SeverityError && int(ds) < len(severityNames) {
		return severityNames[ds]
	}

	return "Unknown"
}

// Position represents a zero-based line and character offset in a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a span between two positions in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a specific document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Symbol represents a named programming symbol with its kind and location.
type Symbol struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"container_name,omitempty"`
}

// Reference represents a reference to a symbol.
type Reference struct {
	Location     Location `json:"location"`
	IsDefinition bool     `json:"is_definition"`
}

// TextEdit represents a text edit operation.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"new_text"`
}

// WorkspaceEdit represents a set of edits across multiple files.
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

// Diagnostic represents a compiler or linter diagnostic.
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Source   string             `json:"source"`
	Message  string             `json:"message"`
}

// Common errors for the lsp package.
var (
	// ErrUnsupportedLanguage is returned when no language config matches the file extension.
	ErrUnsupportedLanguage = errors.New("unsupported language")

	// ErrServerNotFound is returned when the language server binary is not in PATH.
	ErrServerNotFound = errors.New("language server binary not found")

	// ErrServerNotInitialized is returned when calling methods before Initialize().
	ErrServerNotInitialized = errors.New("server not initialized")

	// ErrServerDead is returned when the server process has exited.
	ErrServerDead = errors.New("server process has exited")

	// ErrRequestTimeout is returned when a JSON-RPC request times out.
	ErrRequestTimeout = errors.New("request timeout")

	// ErrTransportClosed is returned when the transport is closed.
	ErrTransportClosed = jsonrpc.ErrTransportClosed
)
