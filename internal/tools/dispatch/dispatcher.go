// Package dispatch provides a generic operation dispatcher for multi-operation tools.
//
// It eliminates the repeated pattern of operation-map lookup, validation,
// and dispatch found across git_operation, memory, scratchpad, and shell_command tools.
package dispatch

import (
	"context"
	"fmt"
	"strings"
)

// Handler processes an operation for tool T with given parameters.
type Handler[T any] func(ctx context.Context, tool T, params Params) (Result, error)

// Params abstracts parameter access for tool execution.
type Params interface {
	GetStringOr(key, fallback string) string
	GetIntOr(key string, fallback int) int
}

// Result is the output of an operation handler.
type Result struct {
	Content string
	IsError bool
}

// OK creates a successful result.
func OK(content string) (Result, error) {
	return Result{Content: content}, nil
}

// Errorf creates an error result (not a Go error — a tool-level error).
func Errorf(format string, args ...any) (Result, error) {
	return Result{Content: fmt.Sprintf(format, args...), IsError: true}, nil
}

// Dispatcher routes operations to registered handlers for tool type T.
type Dispatcher[T any] struct {
	handlers map[string]Handler[T]
	names    []string
}

// New creates a new Dispatcher.
func New[T any]() *Dispatcher[T] {
	return &Dispatcher[T]{
		handlers: make(map[string]Handler[T]),
	}
}

// Register adds a handler for the named operation.
func (d *Dispatcher[T]) Register(name string, handler Handler[T]) *Dispatcher[T] {
	d.handlers[name] = handler

	d.names = append(d.names, name)

	return d
}

// Operations returns the registered operation names in registration order.
func (d *Dispatcher[T]) Operations() []string {
	return d.names
}

// Dispatch extracts the operation from params and executes the matching handler.
// Returns an error result if the operation is missing or unknown.
func (d *Dispatcher[T]) Dispatch(ctx context.Context, tool T, operationKey string, params Params) (Result, error) {
	operation := params.GetStringOr(operationKey, "")
	if operation == "" {
		return Errorf("operation parameter is required")
	}

	handler, exists := d.handlers[operation]
	if !exists {
		return Errorf("unknown operation: %s (valid: %s)", operation, strings.Join(d.names, ", "))
	}

	return handler(ctx, tool, params)
}
